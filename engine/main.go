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
	"shardofspring/internal/music"
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
	// 迷宮機關的互動狀態(docs/re/161 §3 的五個目標編號)。
	prompt *mazePrompt
	// tombs 記著踩過哪幾座墓(事件 701–704),餵 Eldron 謎題的進度旗標。
	tombs map[int]bool
	// clanRewarded:謎題解過了。⚠ **原版把這個旗標存在哪未解**
	// (docs/re/162 §5)—— 所以它只活在記憶體裡,存檔不會記得。
	clanRewarded bool

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

	// M12:遊戲外殼(docs/spec/15-game-shell.md)。標題/主選單/隊伍選擇/全滅/結局。
	shell     *shellState
	titleFont *render.Painter // 標題用 32px(docs/spec/04 §4)

	// bossFight:目前這場戰鬥是不是迷宮事件目標 533(maze.TargetFinalBoss,
	// 最終首領 Siriadne)引發的劇情戰鬥。
	//
	// ⚠ **目前沒有任何呼叫端會把它設成 true** —— 那場戰鬥的怪物組成未解
	// (docs/re/161 §4),maze_prompt.go 的 fireTrigger 明講「不要在這裡
	// 自己安排一場戰鬥,那會是編的,不是原版的」。這個欄位只是結局畫面
	// (docs/spec/15 §7)的介接點:一旦 RE 解出怪物組成,呼叫端在觸發
	// 那場戰鬥時把它設成 true,勝負判定與轉場已經現成,不必再動這裡。
	bossFight bool

	// testKeys 是測試用的假輸入佇列(docs/spec/15 §9)。
	// nil = 正式執行,Update() 照舊呼叫 inpututil 讀真正的鍵盤;
	// 非 nil(即使是空切片)= 測試模式,底下改用這個切片當「這一格剛按下的鍵」。
	//
	// ⚠ 只為了這件事才加這層:ebitengine 在沒有 X11 display 的環境
	// 連套件初始化都會 panic(glfw.Init 需要 display),而 inpututil 的
	// 「剛按下」狀態本來就是靠 RunGame 的事件迴圈在更新 —— 不跑 RunGame,
	// 不管有沒有 display,永遠讀不到任何鍵。沒有這層接縫,
	// docs/spec/15 §9 要求的「不開視窗、直接呼叫 Update()」測試做不到。
	testKeys []ebiten.Key
}

// pressedKeys / pressed 包一層 inpututil,讓 g.testKeys 非 nil 時可以
// 用假輸入取代真正的鍵盤(見上面 testKeys 的說明)。
func (g *Game) pressedKeys() []ebiten.Key {
	if g.testKeys != nil {
		return g.testKeys
	}
	return inpututil.AppendJustPressedKeys(nil)
}

func (g *Game) pressed(k ebiten.Key) bool {
	if g.testKeys != nil {
		for _, tk := range g.testKeys {
			if tk == k {
				return true
			}
		}
		return false
	}
	return inpututil.IsKeyJustPressed(k)
}

func (g *Game) Update() error {
	// 覆蓋層開著時吃掉所有按鍵(docs/spec/04 §3:出現時遊戲暫停)。
	// 這個機制同時被 A6 按鍵表(docs/spec/15 §8)借用 —— 不專屬迷宮敘述。
	if g.overlay != "" {
		if len(g.pressedKeys()) > 0 {
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
		if g.pressed(ebiten.KeyEscape) && g.field.Outcome() != combat.Ongoing {
			outcome := g.field.Outcome()
			g.field = nil
			switch {
			case outcome == combat.PartyDead:
				// A4 全滅(docs/spec/15 §6)。**不能帶著死掉的隊伍回世界
				// 地圖繼續走路** —— 直接進全滅畫面,按鍵後回主選單。
				// 死亡曲(music.Userlib)已經在 endTurn() 判定 PartyDead
				// 的那一刻放過(combat_scene.go),這裡不重放。
				if g.shell != nil {
					g.shell.mode = shellWipe
				}
			case outcome == combat.MonstersDead && g.bossFight:
				// A5 結局(docs/spec/15 §7):這場戰鬥是迷宮事件目標 533
				// (maze.TargetFinalBoss)引發的劇情戰鬥,打贏了。
				// ⚠ g.bossFight 目前沒有任何呼叫端會設成 true —— 見它
				// 在 Game 結構裡的說明,這裡只是接介面,不是宣告
				// 「已經打得到最終首領」。
				g.bossFight = false
				if g.shell != nil {
					g.shell.mode = shellEnding
				}
				g.play(music.Ending)
			default:
				// 遭遇倒數重置。⚠ **重置值未解** —— 原版每次遭遇後填什麼
				// 沒有讀到。這裡沿用出貨存檔的量級,是佔位。
				g.party.Encounter = 54
			}
		}
		return nil
	}

	if (g.shell == nil || g.shell.mode == shellPlaying) && g.pressed(ebiten.KeyN) {
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

	// 名冊中。docs/spec/15 §4:ESC 回哪裡由 g.shell.mode 決定,
	// rosterKey 本身不記「是從哪裡打開的」——見 openMainMenu 的說明。
	if g.roster != nil && g.roster.open {
		for _, k := range g.pressedKeys() {
			g.rosterKey(k)
		}
		return nil
	}

	// 外殼接管畫面:標題 / 主選單 / 隊伍選擇 / 全滅 / 結局(docs/spec/15)。
	// ⚠ 放在 create/roster 之後 —— C)har Utilities 蓋在主選單上時,
	// 上面兩個檢查已經先接手,不會落到這裡;roster 關閉後(ESC)
	// g.shell.mode 沒被動過,下一輪 Update() 才會再落到這裡,
	// 也就自然「回到主選單」。
	if g.shell != nil && g.shell.mode != shellPlaying {
		return g.shellUpdate()
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

	// 迷宮機關問問題中(蓋在迷宮之上)
	if g.prompt != nil {
		g.promptRunes(ebiten.AppendInputChars(nil))
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			g.promptKey(k)
			if g.prompt == nil {
				break
			}
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

	// 外殼接管畫面時,把整個畫布讓給它(docs/spec/15)。
	// drawOverlay 仍要呼叫 —— A6 按鍵表(P 鍵)借用的是敘述覆蓋層的機制,
	// 蓋在主選單上。
	if g.shell != nil && g.shell.mode != shellPlaying {
		g.drawShell(screen)
		g.drawOverlay(screen)
		return
	}

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
	g.drawPrompt(screen)

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
	// docs/spec/15 §1:預設走標題 → 主選單 → 隊伍選擇。-slot 只是除錯捷徑。
	slot := flag.Int("slot", 0,
		"除錯捷徑:跳過標題/主選單/隊伍選擇,直接進第幾隊(1–5)。0 = 預設,走標題流程")
	fontPath := flag.String("font", "", "字型檔;留空則依序試內建候選")
	seed := flag.Uint64("seed", 1, "戰鬥亂數種子(docs/spec/07 §2:同種子可重現)")
	enc := flag.Int("encounter", -1, "覆寫遭遇倒數(除錯用,只在搭配 -slot 時生效)")
	x := flag.Int("x", -1, "覆寫起始 x(除錯用,只在搭配 -slot 時生效)")
	y := flag.Int("y", -1, "覆寫起始 y")
	flag.Parse()

	g, err := loadStatic(*assets, *fontPath, *seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "載入失敗:", err)
		fmt.Fprintln(os.Stderr, "請先跑:go run ./cmd/convert -in <原版> -out assets")
		os.Exit(1)
	}

	if *slot > 0 {
		// 除錯捷徑(docs/spec/15 §1):略過標題/主選單/隊伍選擇,直接進遊戲。
		if err := g.loadParty(*slot); err != nil {
			fmt.Fprintln(os.Stderr, "載入隊伍失敗:", err)
			os.Exit(1)
		}
		// 除錯用的座標/遭遇倒數覆寫。**預設不覆寫** —— 存檔裡的值才是真相。
		if *x >= 0 {
			g.party.X = *x
		}
		if *y >= 0 {
			g.party.Y = *y
		}
		if *enc >= 0 {
			g.party.Encounter = *enc
		}
		g.shell.mode = shellPlaying
	} else {
		g.openTitle() // A1(docs/spec/15 §3)
	}

	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	ebiten.SetWindowTitle("春之石 Shard of Spring")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// loadStatic 讀開局前就需要的靜態資產:世界地圖、圖塊、迷宮資料、規則表、
// 名冊、字型、音訊。**不載入任何一支隊伍** —— docs/spec/15 §1:
// 隊伍要選了才載入,所以拆成「載入靜態資產」與「載入某一支隊伍」兩段
// (後者是 loadParty)。
func loadStatic(dir, fontPath string, seed uint64) (*Game, error) {
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

	// 名冊獨立於隊伍槽:C)har Utilities 從主選單就能進去,不必先 L)oad
	// 一支隊伍(docs/spec/15 §4)。GROUPS.DAT 的路徑也在這裡定案,
	// 隊伍選擇畫面(A3)與 loadParty 都靠它。
	g.savePath = filepath.Join(dir, "save", "GROUPS.DAT")
	cb, err := os.ReadFile(filepath.Join(dir, "save", "CHARS.DAT"))
	if err != nil {
		return nil, err
	}
	g.chars, err = original.ParseChars(cb)
	if err != nil {
		return nil, err
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
	// 標題用 32px(docs/spec/04 §4)。
	g.titleFont = render.NewPainter(src, 32, cgaWhite)
	g.initSound() // docs/spec/13:失敗只記警告,不影響遊戲
	g.shell = &shellState{mode: shellTitle}
	return g, nil
}

// loadParty 選一支隊伍進遊戲。docs/spec/15 §5 = 開局的唯一入口。
//
// ⚠ 讀的是 <assets>/save/ 底下的**複本**,不是 game/sharspri/ ——
// 存檔要寫回去,而原版目錄是唯讀的(CLAUDE.md §8)。
// 呼叫前 g.chars 必須已經由 loadStatic 讀好 —— CHARS.DAT 與隊伍槽無關。
func (g *Game) loadParty(slot int) error {
	if slot < 1 || slot > original.GroupSlots {
		return fmt.Errorf("存檔槽 %d 超出 1–%d", slot, original.GroupSlots)
	}
	gb, err := os.ReadFile(g.savePath)
	if err != nil {
		return err
	}
	groups, err := original.ParseGroups(gb)
	if err != nil {
		return err
	}
	grp := groups[slot-1]
	if grp.Blank() {
		return fmt.Errorf("第 %d 隊還沒建立(記錄整份是空白)", slot)
	}

	g.slot = slot
	g.members = nil
	// 以 GROUPS.DAT 的成員槽為準(docs/spec/06 §1)
	for _, id := range grp.MemberIDs() {
		if id < 1 || id > len(g.chars) {
			g.warnings = append(g.warnings,
				fmt.Sprintf("成員編號 %d 超出名冊 1–%d", id, len(g.chars)))
			continue
		}
		c := g.chars[id-1]
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
