// Shard of Spring remake。
//
// M2:世界地圖場景(docs/spec/05-world-scene.md)
// M3:隊伍、角色與存檔(docs/spec/06-party-and-save.md)。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/render"
	"shardofspring/internal/world"
)

var (
	cgaBlack = color.RGBA{0x00, 0x00, 0x00, 0xff}
	cgaCyan  = color.RGBA{0x55, 0xff, 0xff, 0xff}
	cgaWhite = color.RGBA{0xff, 0xff, 0xff, 0xff}
	// 未解地形的佔位色。docs/spec/05 §4:**未解的東西在畫面上要刺眼** ——
	// 用黑色或海洋填會讓那 66 格偽裝成正常地形。
	missing = color.RGBA{0xff, 0x00, 0xff, 0xff}
)

type Game struct {
	savePath string // <assets>/save/GROUPS.DAT
	slot     int

	world *world.Map
	party world.State
	tiles map[int]*ebiten.Image // 地形值 → 圖;沒有來源的值不在裡面
	// noSrc 記下畫面上出現過的未解地形值,顯示在提示列。
	// 讓未解項目在**執行時**也看得見,不是只在文件裡。
	noSrc map[int]bool

	// M3:隊伍與存檔(docs/spec/06)
	members []original.Character // 依 GROUPS.DAT 的成員槽順序
	group   original.Group
	panel   *render.Painter
	// 載入時發現的不一致,畫在提示列。⚠ 不自行修正(docs/spec/06 §1)。
	warnings []string
	// 最近一次存檔的結果,畫在提示列。
	saveMsg string

	// M4:戰鬥(docs/spec/07)
	monsters []original.Monster
	items    map[int]combat.Item
	rand     *combat.SeededRand
	field    *combat.Field // nil = 不在戰鬥中
	// M10:戰場(docs/spec/12)
	points combat.Points
	actor  int // 目前輪到的隊員索引;−1 = 沒有人能動

	// M5:迷宮(docs/spec/08)
	assets      string
	mazeData    []original.MazeEntry
	mazeTiles   map[int]*ebiten.Image
	level       *mazeLevel // nil = 不在迷宮中
	mazeState   maze.State
	overlay     string // 非空 = 敘述覆蓋層開著
	overlayFont *render.Painter

	// M8:城鎮與名冊(docs/spec/11)
	chars  []original.Character
	roster *rosterState
	create *createState // 非 nil = 建立角色的畫面蓋在名冊上

	shops     []original.Shop
	townSites []original.TownSite // TOWNDATA.BIN 的座標表(docs/re/53 §2)
	itemList  []original.Item
	town      *townState
	rumors    map[int]string // 酒館傳聞:位移 36 → 文字(docs/re/138 §4)

	// M11:聲音(docs/spec/13)。nil = 音訊關閉,遊戲照常跑。
	sound *sound

	// M6:法術(docs/spec/09)
	spells   []original.Spell
	castUnit int
	castList []original.Spell
	// cursor 非 nil = 施法的**選格階段**(手冊 p.34 的 I/J/K/M + 空白鍵)
	cursor *castCursor
}

func (g *Game) Update() error {
	// 覆蓋層開著時吃掉所有按鍵(docs/spec/04 §3:出現時遊戲暫停)
	if g.overlay != "" {
		if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
			g.overlay = ""
		}
		return nil
	}

	// 戰鬥中:方向鍵不移動,空白鍵推一回合、ESC 離開結束的戰鬥。
	if g.field != nil {
		// 施法的選格階段:I/J/K/M 移動、空白鍵施放、ESC 取消
		if g.cursor != nil {
			for _, k := range inpututil.AppendJustPressedKeys(nil) {
				if g.cursorKey(k) {
					break
				}
			}
			return nil
		}
		// 施法選單開著時只吃字母
		if len(g.castList) > 0 {
			for i := 0; i < len(g.castList) && i < 26; i++ {
				if inpututil.IsKeyJustPressed(ebiten.KeyA + ebiten.Key(i)) {
					g.pickSpell(i)
					break
				}
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				g.castList = nil
			}
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			g.openCast()
			return nil
		}
		// M10 之後戰鬥由戰場操作(docs/spec/12);
		// ⚠ 先前的「空白鍵推一整回合」已經移除 —— 兩套並存會讓同一場戰鬥
		// 有兩種規則(一套算行動點數、一套不算),而畫面上分不出剛才用了哪一套。
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			if g.boardKey(k) {
				return nil
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) &&
			g.field.Outcome() != combat.Ongoing {
			g.field = nil
			// 遭遇倒數重置。⚠ **重置值未解** —— 原版每次遭遇後填什麼
			// 沒有讀到。這裡沿用出貨存檔的量級,是佔位。
			g.party.Encounter = 54
		}
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.openRoster() // N)ames —— 名冊(docs/spec/11 §5)
		return nil
	}

	dirs := map[ebiten.Key]int{
		ebiten.KeyUp: 1, ebiten.KeyRight: 2, ebiten.KeyDown: 3, ebiten.KeyLeft: 4,
		ebiten.KeyDigit1: 1, ebiten.KeyDigit2: 2, ebiten.KeyDigit3: 3, ebiten.KeyDigit4: 4,
	}

	// 建立角色中(蓋在名冊之上)
	if g.create != nil {
		g.createRunes(ebiten.AppendInputChars(nil))
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			g.createKey(k)
			if g.create == nil {
				break
			}
		}
		return nil
	}

	// 名冊中
	if g.roster != nil && g.roster.open {
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			g.rosterKey(k)
		}
		return nil
	}

	// 城鎮中
	if g.town != nil && g.town.mode != townClosed {
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			if k == ebiten.KeyEscape && g.town.mode == townBuildings {
				g.town = nil
				break
			}
			g.townKey(k)
		}
		return nil
	}

	// 迷宮中
	if g.level != nil {
		for key, d := range dirs {
			if inpututil.IsKeyJustPressed(key) {
				g.stepMaze(maze.Facing(d))
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			// ⚠ 原版怎麼離開迷宮**未解**(docs/re/146 §2)。
			// 實跑已經排除 `ESC` 與 `E`(兩者在原版的迷宮裡都沒有作用),
			// `Q` 是離開遊戲。這裡沿用 ESC 是**本引擎的選擇**,不是原版行為。
			g.level = nil
		}
		return nil
	}

	for key, d := range dirs {
		if inpututil.IsKeyJustPressed(key) {
			if g.party.Step(world.Facing(d), g.world) == world.Moved {
				// 踩到地城入口 → 進迷宮(docs/spec/08 §6)
				if g.enterMaze(g.party.X, g.party.Y) {
					return nil
				}
				// 踩到城鎮 → 進城(docs/spec/11 §2)
				if v := g.world.At(g.party.X, g.party.Y); v >= 30 && v <= 32 {
					if g.enterTown(g.party.X, g.party.Y) {
						return nil
					}
				}
				// docs/formats/02 位移 25:歸零時觸發遭遇檢查
				if g.party.Encounter == 0 {
					g.startCombat()
				}
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		// ⚠ **存檔也推進時鐘一格**(docs/re/149:只按 S 不移動,位移 33 仍 +1)。
		// 這不是直覺的行為 —— 但三次量測都吻合「每個動作一格」。
		g.party.Tick()
		if err := g.save(); err != nil {
			g.saveMsg = "存檔失敗：" + err.Error()
		} else {
			g.saveMsg = fmt.Sprintf("已存到第 %d 隊(%s)", g.slot, g.savePath)
		}
	}
	return nil
}

// save 把目前狀態寫回 GROUPS.DAT。docs/spec/06 §6。
//
// ⚠ 寫的是 <assets>/save/ 的複本,**不碰 game/sharspri/**(CLAUDE.md §8)。
// ⚠ 只覆寫已解的欄位;未解的位置由 Group.Bytes() 從 Raw 原樣保留。
// syncMember 把一位隊伍成員的變化寫回名冊。
//
// ⚠ `g.members` 是 `g.chars` 的**複本**,不是指標 ——
// 戰鬥扣的血、拿到的經驗全部只改到複本。少了這一步,
// 存檔會把「打完仗之前」的名冊寫回去,而且**不會報錯**:
// 檔案長度對、欄位合法、原版也開得起來,只是進度沒了。
func (g *Game) syncMember(c original.Character) {
	if c.ID >= 1 && c.ID <= len(g.chars) {
		g.chars[c.ID-1] = c
	}
}

func (g *Game) syncMembers() {
	for _, c := range g.members {
		g.syncMember(c)
	}
}

// save 寫回存檔。**兩個檔一起寫。**
//
// 原版的存檔常式(`USERLIB` 槽 34)在同一段裡開了兩個檔:
// `#1` = `CHARS.DAT`(記錄 94)、`#2` = `GROUPS.DAT`(記錄 90),docs/re/80 §1。
// 只寫其中一個會讓兩份資料對不上 —— 隊伍位置前進了,成員的血量卻回到上一次。
func (g *Game) save() error {
	g.syncMembers()
	if err := g.writeChars(); err != nil {
		return err
	}
	b, err := os.ReadFile(g.savePath)
	if err != nil {
		return err
	}
	groups, err := original.ParseGroups(b)
	if err != nil {
		return err
	}
	grp := g.group
	grp.WorldX, grp.WorldY = g.party.X, g.party.Y
	grp.Facing = int(g.party.Facing)
	grp.Month, grp.Day = g.party.Clock.Month, g.party.Clock.Day
	grp.Hour, grp.Sub = g.party.Clock.Hour, g.party.Clock.Sub
	grp.Encounter = g.party.Encounter
	groups[g.slot-1] = grp

	out := make([]byte, 0, len(b))
	for _, x := range groups {
		out = append(out, x.Bytes()...)
	}
	if len(out) != len(b) {
		return fmt.Errorf("寫出 %d bytes,原檔 %d", len(out), len(b))
	}
	return os.WriteFile(g.savePath, out, 0o644)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(cgaBlack)

	// 戰鬥與迷宮各自接管主視野。
	inCombat := g.field != nil
	inMaze := g.level != nil && !inCombat
	inTown := g.town != nil && g.town.mode != townClosed && !inCombat && !inMaze
	inRoster := (g.roster != nil && g.roster.open) || g.create != nil

	// 9×9 視野,隊伍固定在正中央(docs/spec/05 §3、§4)。
	const half = layout.ViewTiles / 2
	for vy := 0; !inCombat && !inMaze && !inTown && !inRoster && vy < layout.ViewTiles; vy++ {
		for vx := 0; vx < layout.ViewTiles; vx++ {
			mx, my := g.party.X-half+vx, g.party.Y-half+vy
			v := g.world.At(mx, my)
			px := float32(layout.View.X + vx*layout.TileDst)
			py := float32(layout.View.Y + vy*layout.TileDst)

			// 值 11(海洋,全圖 55.63%)原版**一個像素都不畫**
			// (docs/re/132 §1),顯示的就是底色。這裡同樣什麼都不做 ——
			// 畫一張「海的圖」會讓畫面比原版多東西。
			if src, _ := original.WorldTileOrigin(v); src == original.SrcBackdrop {
				continue
			}

			if img, ok := g.tiles[v]; ok {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(layout.ArtScale, layout.ArtScale)
				op.GeoM.Translate(float64(px), float64(py))
				// 最近鄰 —— 整數倍放大不該有插值(docs/spec/04 §1)。
				op.Filter = ebiten.FilterNearest
				screen.DrawImage(img, op)
			} else {
				g.noSrc[v] = true
				vector.DrawFilledRect(screen, px, py,
					layout.TileDst, layout.TileDst, missing, false)
			}
		}
	}

	if !inCombat && !inMaze && !inTown && !inRoster {
		// 隊伍所在格的框
		c := float32(layout.View.X + half*layout.TileDst)
		r := float32(layout.View.Y + half*layout.TileDst)
		vector.StrokeRect(screen, c, r, layout.TileDst, layout.TileDst, 3, cgaWhite, false)
	}

	frame := func(rc layout.Rect) {
		vector.StrokeRect(screen, float32(rc.X), float32(rc.Y),
			float32(rc.W), float32(rc.H), 2, cgaWhite, false)
	}
	frame(layout.View)
	frame(layout.Party)
	frame(layout.Message)
	frame(layout.Prompt)

	g.drawParty(screen)
	g.drawMaze(screen)
	g.drawTown(screen)
	g.drawRoster(screen)
	g.drawCreate(screen)
	g.drawCombat(screen)
	g.drawCastMenu(screen)
	g.drawOverlay(screen)

	// M2 還沒有字型(docs/spec/04 §4 的 TTF 是 M3 之後),
	// 所以狀態暫時走 Ebitengine 的除錯字。**這不是最終呈現。**
	ebiten.SetWindowTitle(fmt.Sprintf(
		"春之石 — (%d,%d) 朝向%d  月%d 日%d 時%d  遭遇倒數%d  未解地形%v",
		g.party.X, g.party.Y, g.party.Facing,
		g.party.Clock.Month, g.party.Clock.Day, g.party.Clock.Hour,
		g.party.Encounter, keys(g.noSrc)))
}

func keys(m map[int]bool) []int {
	out := []int{}
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (g *Game) Layout(int, int) (int, int) { return layout.ScreenW, layout.ScreenH }

func main() {
	assets := flag.String("assets", "assets", "資產資料夾(由 cmd/convert 產生)")
	slot := flag.Int("slot", 5, "隊伍存檔槽 1–5(出貨磁片組好的是 #5)")
	fontPath := flag.String("font", "", "字型檔;留空則依序試內建候選")
	seed := flag.Uint64("seed", 1, "戰鬥亂數種子(docs/spec/07 §2:同種子可重現)")
	enc := flag.Int("encounter", -1, "覆寫遭遇倒數(除錯用)")
	x := flag.Int("x", -1, "覆寫起始 x(除錯用;預設用存檔裡的座標)")
	y := flag.Int("y", -1, "覆寫起始 y")
	flag.Parse()

	g, err := load(*assets, *slot, *fontPath, *seed, *x, *y, *enc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "載入失敗:", err)
		fmt.Fprintln(os.Stderr, "請先跑:go run ./cmd/convert -in <原版> -out assets")
		os.Exit(1)
	}

	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	ebiten.SetWindowTitle("春之石 Shard of Spring")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func load(dir string, slot int, fontPath string, seed uint64, x, y, enc int) (*Game, error) {
	// ⚠ 每個欄位要有自己的 tag。寫成 `W, H int `json:"w"`` 會讓兩個欄位
	// 共用同一個 tag,H 永遠是 0 —— 而 JSON 解碼**不會報錯**。
	var wm struct {
		W     int   `json:"w"`
		H     int   `json:"h"`
		Cells []int `json:"cells"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "data", "worldmap.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &wm); err != nil {
		return nil, err
	}
	if wm.W != world.W || wm.H != world.H {
		return nil, fmt.Errorf("地圖尺寸 %d×%d,規格說 %d×%d", wm.W, wm.H, world.W, world.H)
	}

	tiles := loadTiles(filepath.Join(dir, "gfx", "world"), 38)

	g := &Game{
		world:  &world.Map{Cells: wm.Cells},
		tiles:  tiles,
		assets: dir,
		noSrc:  map[int]bool{},
	}
	if err := readJSON(filepath.Join(dir, "data", "mazedata.json"), &g.mazeData); err != nil {
		return nil, err
	}
	// 迷宮圖塊:MAZEITEM.PIC 第 k 行 = 格值 k(偏移 0)。
	g.mazeTiles = loadTiles(filepath.Join(dir, "gfx", "maze"), 32)
	if err := g.loadParty(dir, slot); err != nil {
		return nil, err
	}
	// 除錯用的座標覆寫。**預設不覆寫** —— 存檔裡的座標才是真相。
	if x >= 0 {
		g.party.X = x
	}
	if y >= 0 {
		g.party.Y = y
	}
	if enc >= 0 {
		g.party.Encounter = enc
	}

	if err := g.loadCombat(dir, seed); err != nil {
		return nil, err
	}

	src, path, err := render.LoadFont(fontPath)
	if err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr, "字型:", path)
	g.panel = render.NewPainter(src, 20, cgaWhite)
	// 敘述覆蓋層用 24 px(docs/spec/04 §4 的主要閱讀字級)。
	g.overlayFont = render.NewPainter(src, 24, cgaWhite)
	g.initSound() // docs/spec/13:失敗只記警告,不影響遊戲
	return g, nil
}

// loadParty 讀 CHARS.DAT / GROUPS.DAT。docs/spec/06。
//
// ⚠ 讀的是 <assets>/save/ 底下的**複本**,不是 game/sharspri/ ——
// 存檔要寫回去,而原版目錄是唯讀的(CLAUDE.md §8)。
func (g *Game) loadParty(dir string, slot int) error {
	if slot < 1 || slot > original.GroupSlots {
		return fmt.Errorf("存檔槽 %d 超出 1–%d", slot, original.GroupSlots)
	}
	cb, err := os.ReadFile(filepath.Join(dir, "save", "CHARS.DAT"))
	if err != nil {
		return err
	}
	chars, err := original.ParseChars(cb)
	if err != nil {
		return err
	}
	gb, err := os.ReadFile(filepath.Join(dir, "save", "GROUPS.DAT"))
	if err != nil {
		return err
	}
	groups, err := original.ParseGroups(gb)
	if err != nil {
		return err
	}
	g.slot = slot
	g.savePath = filepath.Join(dir, "save", "GROUPS.DAT")
	grp := groups[slot-1]
	if grp.Blank() {
		return fmt.Errorf("第 %d 隊還沒建立(記錄整份是空白)—— 出貨磁片組好的是第 5 隊", slot)
	}

	g.chars = chars
	// 以 GROUPS.DAT 的成員槽為準(docs/spec/06 §1)
	for _, id := range grp.MemberIDs() {
		if id < 1 || id > len(chars) {
			g.warnings = append(g.warnings,
				fmt.Sprintf("成員編號 %d 超出名冊 1–%d", id, len(chars)))
			continue
		}
		c := chars[id-1]
		if !c.Occupied() {
			g.warnings = append(g.warnings, fmt.Sprintf("成員編號 %d 指向空槽", id))
			continue
		}
		// CHARS.DAT 位移 1 應該回指同一隊。不一致只記錄,**不自行修正**。
		if p, ok := c.InParty(); !ok || p != slot {
			g.warnings = append(g.warnings,
				fmt.Sprintf("%s 在第 %d 隊的成員表裡,自己的隊伍欄卻是 %q",
					c.Name, slot, string(c.Party)))
		}
		g.members = append(g.members, c)
	}

	g.group = grp
	g.party = world.State{
		X: grp.WorldX, Y: grp.WorldY, Facing: world.Facing(grp.Facing),
		Clock: world.Clock{
			Sub: grp.Sub, Hour: grp.Hour, Day: grp.Day, Month: grp.Month,
		},
		Encounter: grp.Encounter,
	}
	return nil
}

// loadCombat 讀 M4 需要的資料表。docs/spec/07。
func (g *Game) loadCombat(dir string, seed uint64) error {
	var monsters []original.Monster
	if err := readJSON(filepath.Join(dir, "data", "monsters.json"), &monsters); err != nil {
		return err
	}
	var items []struct {
		Index int `json:"index"`
		Col4  int `json:"col4"`
		Col5  int `json:"col5"`
	}
	if err := readJSON(filepath.Join(dir, "data", "items.json"), &items); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(dir, "data", "spells.json"), &g.spells); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(dir, "data", "townsites.json"), &g.townSites); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(dir, "data", "shops.json"), &g.shops); err != nil {
		return err
	}
	// 傳聞。⚠ 找不到就是找不到(docs/re/138 §4:10 段對 11 個索引),
	// 畫面上會明講,不拿別段頂替。
	g.rumors = map[int]string{}
	_ = readJSON(filepath.Join(dir, "data", "rumors.json"), &g.rumors)
	if err := readJSON(filepath.Join(dir, "data", "items.json"), &g.itemList); err != nil {
		return err
	}
	g.items = map[int]combat.Item{}
	for _, it := range items {
		// ⚠ 欄4/欄5 是型別相依的(docs/re/74)。這裡原樣搬,
		// 由 combat 依「這是武器還是防具」去解讀 —— 不在載入時分類。
		g.items[it.Index] = combat.Item{Main: it.Col4, Bonus: it.Col5}
	}
	g.monsters = monsters
	g.rand = combat.NewRand(seed)
	return nil
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// loadTiles 讀 <dir>/t00.png … t<max>.png。讀不到的編號不進 map ——
// 執行期會畫佔位符,讓「沒有來源」在畫面上看得見(docs/spec/05 §4)。
func loadTiles(dir string, max int) map[int]*ebiten.Image {
	out := map[int]*ebiten.Image{}
	for v := 0; v <= max; v++ {
		f, err := os.Open(filepath.Join(dir, fmt.Sprintf("t%02d.png", v)))
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err == nil {
			out[v] = ebiten.NewImageFromImage(img)
		}
	}
	return out
}

// writeChars 把名冊寫回 <assets>/save/CHARS.DAT。
//
// ⚠ 與 GROUPS.DAT 一樣,**只覆寫已解的欄位** —— 未解的位置由 Raw 原樣保留
// (docs/spec/06 §3 的同一條原則)。
func (g *Game) writeChars() error {
	path := filepath.Join(g.assets, "save", "CHARS.DAT")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(b) != original.CharRecLen*original.CharSlots {
		return fmt.Errorf("CHARS.DAT 長度 %d 不對", len(b))
	}
	out := append([]byte(nil), b...)
	for i, c := range g.chars {
		copy(out[i*original.CharRecLen:], c.Bytes())
	}
	return os.WriteFile(path, out, 0o644)
}
