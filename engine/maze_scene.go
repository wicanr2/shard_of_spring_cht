package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/ui"
)

// 迷宮場景。docs/spec/08-maze-scene.md。

// mazeLevel 是載進來的一層。
type mazeLevel struct {
	// index 是這一層在 `MAZEDATA` 裡的**入口編號** —— 地城名靠它查
	// (docs/re/222:名字在 MENU.EXE 的兩串 DATA,索引就是入口編號)。
	// ⚠ 不能用 `MazeFile` 代替:入口 0/1/4/6 共用 DG1、7–11 共用 DG6。
	index  int
	entry  original.MazeEntry
	grid   *original.Maze
	events []original.Event
	text   map[int]string
}

// enterMaze 從世界地圖進入迷宮。回 false 表示這一格沒有入口。
func (g *Game) enterMaze(x, y int) bool {
	for i, e := range g.mazeData {
		if e.WorldX != x || e.WorldY != y {
			continue
		}
		lv, err := g.loadLevel(i, e)
		if err != nil {
			g.warnings = append(g.warnings, err.Error())
			return false
		}
		g.level = lv
		g.mazeState = maze.State{
			Major: e.StartMajor, Minor: e.StartMinor,
			Facing: maze.Facing(e.Facing),
		}
		g.syncMazeNum()
		g.overlay = ""
		return true
	}
	return false
}

func (g *Game) loadLevel(i int, e original.MazeEntry) (*mazeLevel, error) {
	lv := &mazeLevel{index: i, entry: e}
	if err := readJSON(filepath.Join(g.assets, "data",
		fmt.Sprintf("maze%d.json", e.MazeFile)), &lv.grid); err != nil {
		return nil, fmt.Errorf("讀 DG%d:%w", e.MazeFile, err)
	}
	if err := readJSON(filepath.Join(g.assets, "data",
		fmt.Sprintf("events%d.json", e.TextFile)), &lv.events); err != nil {
		return nil, fmt.Errorf("讀 DE%d:%w", e.TextFile, err)
	}
	if err := readJSON(filepath.Join(g.assets, "data",
		fmt.Sprintf("dtext%d.json", e.TextFile)), &lv.text); err != nil {
		return nil, fmt.Errorf("讀 DT%d:%w", e.TextFile, err)
	}
	// docs/re/181 §4 + docs/spec/18 §1 第 3 項:事件表每次都是重新從檔案讀的
	// (上面三個 readJSON),所以任何一次性事件的作廢都只活在**上一個**
	// lv.events 裡 —— 這裡把存檔記著的作廢清單重新套用一次,否則走出迷宮
	// 再走回來,寶箱/劇情戰鬥就復活了。
	g.applyDisabledEvents(lv)
	// 拉利斯之門(docs/re/205):唸過 `DAZA REVELI` 之後那一格從擋路變成可通行。
	// ⚠ 與作廢清單同一個理由放在這裡 —— 迷宮格線每次都是重讀的,
	// 改在別處會在「走出去再走回來」之後失效,而門會**默默地關回去**。
	maze.OpenGate(lv.grid, e.MazeFile, g.group.GateOpen)
	return lv, nil
}

// applyDisabledEvents 把 g.disabledEvents 記著的、屬於這個事件檔的作廢目標
// 重新套用到剛讀出來的 lv.events。key 是 original.MazeEntry.TextFile
// (DE<N>EFF.BIN 的 N),不是 MazeFile —— 兩者在 DG5/DG51 這種情況下不同
// (docs/re/161 §6 的訂正)。
func (g *Game) applyDisabledEvents(lv *mazeLevel) {
	key := strconv.Itoa(lv.entry.TextFile)
	for _, target := range g.disabledEvents[key] {
		maze.DisableTarget(lv.events, target)
	}
}

// disableMazeEvent 把打贏一場由迷宮事件觸發的戰鬥(目標 204/533)這件事
// 記下來,讓這個目標**這次立刻**不再觸發(maze.DisableTarget 改
// g.level.events),並且記進 g.disabledEvents,讓離開迷宮再回來
// (loadLevel 會整個重讀)、以及存檔讀回之後都不會復活
// (docs/re/181 §4、docs/spec/18 §3.2)。
//
// g.level == nil 時什麼都不做 —— docs/spec/17 的腳本戰鬥測試會直接呼叫
// startScriptedCombat 而不經過完整的迷宮載入流程,這個保護讓那些測試
// 不會因為這裡的新行為而 panic 或誤動到不存在的迷宮狀態。
func (g *Game) disableMazeEvent(target int) {
	if g.level == nil {
		return
	}
	maze.DisableTarget(g.level.events, target)
	key := strconv.Itoa(g.level.entry.TextFile)
	for _, t := range g.disabledEvents[key] {
		if t == target {
			return // 已經記過,不重複加
		}
	}
	if g.disabledEvents == nil {
		g.disabledEvents = map[string][]int{}
	}
	g.disabledEvents[key] = append(g.disabledEvents[key], target)
}

// resumeMaze 套用存檔裡「上次存在迷宮裡的哪一格」(docs/spec/18 §3.2
// MazeFile、驗收 4)。由 shell_scene.go 的 selectParty 在 loadParty 成功後呼叫。
//
// ⚠ 只在選到的隊伍剛好是存檔的 Active 那一隊時才套用 —— g.pendingMazeFile
// 是單一一組欄位(不像 Group.MazeX/MazeY 每隊各有一份),選別隊時沒有對應
// 的意義,見 Game 結構裡 pendingActive 的說明。不管有沒有套用,結束時都要
// 清掉三個 pending 欄位,避免下一次選隊伍誤套用到這次的殘留值。
func (g *Game) resumeMaze(slot int) {
	file, facing := g.pendingMazeFile, g.pendingMazeFacing
	active := g.pendingActive
	g.pendingActive, g.pendingMazeFile, g.pendingMazeFacing = 0, "", 0

	if file == "" || slot != active {
		return
	}
	n, err := strconv.Atoi(file)
	if err != nil {
		g.warnings = append(g.warnings, "存檔的迷宮編號讀不懂:"+file)
		return
	}
	var entry original.MazeEntry
	idx, found := -1, false
	// ⚠ 存檔只記**迷宮檔號**(GROUPS.DAT 位移 83),不記入口編號 ——
	// 而入口 0/1/4/6 共用 DG1、7–11 共用 DG6,所以讀檔回到這幾個地城時
	// 取的是第一個相符的入口,**名字可能不是原本那一個**。
	// 原版怎麼記這件事沒讀(docs/re/222 §4)。
	for i, e := range g.mazeData {
		if e.MazeFile == n {
			entry, idx, found = e, i, true
			break
		}
	}
	if !found {
		g.warnings = append(g.warnings, fmt.Sprintf("存檔記著在 DG%d,但找不到這個地城", n))
		return
	}
	lv, err := g.loadLevel(idx, entry)
	if err != nil {
		g.warnings = append(g.warnings, "重新載入迷宮失敗:"+err.Error())
		return
	}
	g.level = lv
	g.mazeState = maze.State{
		Major: g.group.MazeX, Minor: g.group.MazeY,
		Facing: maze.Facing(facing),
	}
	g.syncMazeNum()
	g.overlay = ""
}

// stepMaze 處理一次方向輸入,並在**實際位移之後**掃事件(docs/spec/08 §4)。
func (g *Game) stepMaze(dir maze.Facing) {
	if g.level == nil {
		return
	}
	// 迷宮裡走一步同樣是一個動作 —— 時鐘要走、光源要燒(docs/re/149、204)。
	// ⚠ 先推進再判位移:原版轉身也算一格(world.Step 的說明),
	// 而迷宮的 Step 把轉身與位移合在一個回傳值裡。
	g.party.Tick()
	switch g.mazeState.Step(dir, g.level.grid) {
	case maze.Left:
		// 走出邊界 → 回世界地圖。原版在這一刻印 `Leaving maze ..`
		// (docs/re/147:實跑從入口那一格往外走一步就出去了)。
		g.level = nil
		g.syncMazeNum() // 走出地城:火把熄掉、能見度換回天色(docs/re/204 §2)
		g.overlay = "離開地城……" // MAZEMOVE:88
		return
	case maze.Moved:
		g.walkGait++ // 同世界地圖:轉身不翻步態
	default:
		return
	}
	g.fireTrigger(maze.Scan(g.level.events, g.mazeState, g.level.text))

	// docs/formats/02 位移 25:歸零時觸發遭遇檢查 —— **地城裡也會遭遇**。
	// 手冊「在野外或地城中冒險時」;原版實跑在拉利斯走第三步就撞上一群
	// (2026-08-18,workplace/dosbox/shots/q3b-p3.png)。
	//
	// ⚠ 先前只有世界地圖那一條路呼叫 `startCombat`,而 `startCombat`
	// **早就寫好了迷宮分支**(`rules.MazeZone`)—— 規則齊了、沒有人叫它。
	// 症狀是地城變成純散步,而任何測試都不會紅。
	//
	// ⚠ 進了事件(傳送、跨層)那幾條路會在 fireTrigger 裡 return,
	// 所以這一段只在「單純走了一步」時跑到,與世界地圖同一個位置。
	if g.level != nil && g.field == nil && g.party.Encounter == 0 {
		g.startCombat()
	}
}

func (g *Game) fireTrigger(t maze.Trigger) {
	switch t.Kind {
	case maze.KindTeleport:
		g.mazeState.Major, g.mazeState.Minor = t.Major, t.Minor
		return
	case maze.KindCrossLevel:
		g.crossLevel(t.Number)
		return
	}
	// 五個機關同時也有 DT 文字,原版是**先印文字再做事**(docs/re/161 §3)。
	g.overlay = t.Text
	// 六個定點道具其中五個是「踩到就給」(docs/re/202);第六個是謎題的獎賞。
	// ⚠ **先印房間敘述再給東西** —— 原版的順序就是這樣(handler 開頭是
	// `mov ax, ds:3532` 取 DT 文字,`call sub_1393E` 在最後一行)。
	if item, ok := maze.LootEvents[t.Number]; ok {
		g.overlay += "　" + g.giveMazeItem(item)
	}
	if g.tombs == nil {
		g.tombs = map[int]bool{}
	}
	for _, n := range maze.TombTargets {
		if t.Number == n {
			g.tombs[n] = true
		}
	}
	if t.Kind == maze.KindScript {
		// 目標 204 / 533 —— 腳本戰鬥(docs/spec/17-scripted-fights.md)。
		// 怪物組成已經從 ds:372C 起的清單解出來(docs/re/180):204 是
		// 1 隻 Hill Giant,533 是 2 隻 Great Dragon + 1 隻 Siriadne !。
		// 上面已經把 DT 文字放進 g.overlay(原版先印文字再做事,
		// docs/re/161 §3)——玩家關掉那個覆蓋層之後,g.field 已經是
		// 這場戰鬥,直接進戰鬥畫面。
		//
		// 查不到腳本清單(docs/re/180 §6 其餘 13 處寫入點還沒盤到)
		// 就明講,⛔ 不自己編一場戰鬥出來頂替。
		if !g.startScriptedCombat(t.Number) {
			g.warnings = append(g.warnings,
				fmt.Sprintf("劇情事件 %d 沒有腳本怪物清單(docs/re/180 §6 尚未盤到)", t.Number))
		}
		return
	}
	g.openPrompt(t)
}

// crossLevel 走跨關卡的樓梯。
//
// ⚠ **目標編號 → 關卡的對應未解**(docs/spec/08 §5)。這裡用
// 「同一個地城的另一個檔」去找,找不到就**明講**,不要靜默留在原地 ——
// 靜默留下的話,玩家看到的是「樓梯壞掉」,而那查不出原因。
func (g *Game) crossLevel(target int) {
	cur := g.level.entry
	for i, e := range g.mazeData {
		if e.MazeFile == cur.MazeFile {
			continue
		}
		// DG5 ↔ DG51:檔號其中一個是另一個的前綴(5 → 51)
		if !relatedMazeFile(cur.MazeFile, e.MazeFile) {
			continue
		}
		lv, err := g.loadLevel(i, e)
		if err != nil {
			g.warnings = append(g.warnings, err.Error())
			return
		}
		g.level = lv
		g.syncMazeNum()
		g.mazeState.Major, g.mazeState.Minor = e.StartMajor, e.StartMinor
		// MAZEMOVE:29/32「One momement..」—— 原版在**載入另一層的迷宮檔**
		// 之前印這一句(`0x123AC` / `0x12462`,兩處都緊接著組檔名再呼叫載入)。
		// ⚠ 引擎讀檔是瞬間的,所以它與結果排在同一行,不是一個會停留的畫面。
		g.overlay = mazeOneMoment + fmt.Sprintf("你走到了另一層(DG%d)。", e.MazeFile)
		return
	}
	g.overlay = fmt.Sprintf("跨關卡目標未解(編號 %d)—— docs/spec/08 §5", target)
}

// relatedMazeFile:檔號 5 與 51 是同一個地城的兩半(docs/re/60 §3)。
// ⚠ 這是**從檔名推的**,不是查表讀出來的。
func relatedMazeFile(a, b int) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)[:min(len(fmt.Sprint(b)), len(fmt.Sprint(a)))] ||
		fmt.Sprint(b) == fmt.Sprint(a)[:min(len(fmt.Sprint(a)), len(fmt.Sprint(b)))]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// drawMaze 畫 9×9 的迷宮視野。
func (g *Game) drawMaze(dst *ebiten.Image) {
	lv, p := g.level, g.panel
	// g.field != nil:一場腳本戰鬥正在迷宮裡進行(docs/spec/17)——
	// main.go 的 Draw() 沒有幫忙互斥這兩層(它只在世界地圖那圈用
	// inCombat/inMaze 互斥,drawMaze/drawCombat 各自的呼叫沒有比較),
	// 這裡自己讓開,否則迷宮格線會跟戰場畫在同一塊視野裡疊在一起。
	// 隨機遭遇目前不會在迷宮裡觸發(combat_scene.go startCombat 的
	// 呼叫端只有世界地圖那一條路),所以這個分支只影響腳本戰鬥。
	if lv == nil || p == nil || g.field != nil {
		return
	}
	// 生效能見度由隊伍狀態現算(有光 → 位移 59、無光 → 位移 61,
	// docs/re/204 §2)。**畫之前抓一次**,不要在 mazeState 裡另存一份 ——
	// 兩份會漂,而漂掉的症狀是「火把燒完了視野卻沒變」。
	g.mazeState.Visibility = g.party.Visibility
	const half = layout.ViewTiles / 2
	for vy := 0; vy < layout.ViewTiles; vy++ {
		for vx := 0; vx < layout.ViewTiles; vx++ {
			// ⚠ **Major 畫在垂直軸** —— Major 是南北、Minor 是東西
			// (docs/re/224)。畫反的話地圖看起來仍然像一座迷宮,
			// 只是整張轉置,而**事件的方向欄會全部對不上**。
			aM := g.mazeState.Major - half + vy
			am := g.mazeState.Minor - half + vx
			px := float32(layout.View.X + vx*layout.TileDst)
			py := float32(layout.View.Y + vy*layout.TileDst)
			if !maze.Visible(g.mazeState, aM, am) {
				continue // 超出能見度 —— 畫成底色
			}
			id, drawn := original.MazeDrawn(lv.grid.At(aM, am))
			if !drawn {
				continue
			}
			if img, ok := g.mazeTiles[id]; ok {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(layout.ArtScale, layout.ArtScale)
				op.GeoM.Translate(float64(px), float64(py))
				op.Filter = ebiten.FilterNearest
				dst.DrawImage(img, op)
			} else {
				g.noSrc[id] = true
				vector.DrawFilledRect(dst, px, py,
					layout.TileDst, layout.TileDst, missing, false)
			}
		}
	}
	// 隊伍所在格:人形(walk.go)。⚠ 地城用**另一組色盤**(0x0E),
	// 所以讀的是 walk-maze 那一份 —— 用世界地圖那份畫出來仍然是一個人形,
	// 只是顏色不對。
	c := float64(layout.View.X + half*layout.TileDst)
	r := float64(layout.View.Y + half*layout.TileDst)
	if !drawWalk(dst, g.walkArtMaze(), c, r) {
		vector.StrokeRect(dst, float32(c), float32(r),
			layout.TileDst, layout.TileDst, 3, cgaWhite, false)
	}

	// ⚠ 紮營時那個框借給營地選單(town_scene.go 的 drawCampInPlace)——
	// 兩邊都畫的話,兩層字直接疊在一起(QA 2026-08-18)。
	if g.campInPlace() {
		return
	}

	// ⚠ 自己斷成兩行,不要靠折行 —— 這段固定超過訊息面板的 30 欄,
	// 而 ui.Wrap 是按欄數硬斷的,會把「能見度」從中間切開。
	g.drawMessageLines(dst, []string{
		// ⚠ **不印格座標與朝向編號** —— 那是內部狀態,原版在這個位置印的是
		// **地城名**(docs/re/222)。
		g.dungeonName(lv),
		fmt.Sprintf("能見度 %d", g.mazeState.Visibility),
	})
}

// drawOverlay 畫敘述覆蓋層。docs/spec/04 §3:置中於**畫布**,不是主視野。
func (g *Game) drawOverlay(dst *ebiten.Image) {
	if g.overlay == "" || g.panel == nil {
		return
	}
	rc := layout.Overlay
	vector.DrawFilledRect(dst, float32(rc.X), float32(rc.Y),
		float32(rc.W), float32(rc.H), cgaBlack, false)
	vector.StrokeRect(dst, float32(rc.X), float32(rc.Y),
		float32(rc.W), float32(rc.H), 2, cgaWhite, false)

	// 內距 32 → 文字區 736 px;24 px 字 → 每行 30 個全形字 = 60 欄
	const pad, cols = 32, 60
	lh := g.overlayFont.LineHeight()
	y := float64(rc.Y + pad)
	// **先照 "\n" 切段,再對每一段折行。**
	// ⚠ `ui.Wrap` 不認得換行字元,會把它當成一個寬度 1 的普通字元夾在行中間 ——
	// 所以要有人先切。切在這裡而不是要求呼叫端避開換行,是因為
	// 按鍵表那張小鍵盤九宮格**必須逐列對齊**(docs/spec/15 §8)。
	for _, para := range strings.Split(g.overlay, "\n") {
		if para == "" {
			y += lh // 空行就是空行,不要被 Wrap 吃掉
			continue
		}
		for _, ln := range ui.Wrap(para, cols) {
			g.overlayFont.Draw(dst, ln, float64(rc.X+pad), y)
			y += lh
		}
	}
	g.overlayFont.Draw(dst, "（按任意鍵繼續）",
		float64(rc.X+pad), float64(rc.Y+rc.H-pad)-lh)
}

// drawPrompt 畫迷宮機關的問題。與覆蓋層同一個框,但**不吃「按任意鍵」** ——
// 它在等特定的輸入。
func (g *Game) drawPrompt(dst *ebiten.Image) {
	if g.prompt == nil || g.overlay != "" || g.panel == nil {
		return
	}
	rc := layout.Overlay
	vector.DrawFilledRect(dst, float32(rc.X), float32(rc.Y),
		float32(rc.W), float32(rc.H), cgaBlack, false)
	vector.StrokeRect(dst, float32(rc.X), float32(rc.Y),
		float32(rc.W), float32(rc.H), 2, cgaWhite, false)

	const pad, cols = 32, 60
	lh := g.overlayFont.LineHeight()
	y := float64(rc.Y + pad)
	for _, ln := range g.prompt.lines() {
		for _, w := range ui.Wrap(ln, cols) {
			g.overlayFont.Draw(dst, w, float64(rc.X+pad), y)
			y += lh
		}
	}
	if g.prompt.kind == promptPool {
		for i, m := range g.members {
			g.overlayFont.Draw(dst,
				fmt.Sprintf("%d) %-10s %3d/%3d", i+1, m.Name, m.HP, m.MaxHP),
				float64(rc.X+pad), y)
			y += lh
		}
	}
	g.overlayFont.Draw(dst, "（ESC 離開）",
		float64(rc.X+pad), float64(rc.Y+rc.H-pad)-lh)
}

// mazeOneMoment 是 MAZEMOVE:29/32「One momement..」。
//
// 原版在跨層時要換讀一個 `DG*MAZE.SQZ`,所以先印一句「稍候」。
// ⚠ 兩列是**同一句在兩個分支各出現一次**(迷宮 5 的兩個方向),
// 引擎只有一支 crossLevel,兩列共用它。
const mazeOneMoment = "稍候……　"

// dungeonName 回傳這一層的地城名。
//
// 名字來自 `MENU.EXE` 的兩串 `DATA`(docs/re/222),`cmd/convert` 轉成
// `data/dungeons.json`,索引就是 `MAZEDATA` 的入口編號。
//
// ⚠ 讀不到就退回 `地城 DGn` —— ⛔ 不拿別的名字頂替。原版在畫面左上角
// 印的是名字,而編號是內部狀態;但**印錯名字比印編號糟**。
func (g *Game) dungeonName(lv *mazeLevel) string {
	if lv.index >= 0 && lv.index < len(g.dungeonNames) {
		if n := g.dungeonNames[lv.index]; n != "" {
			return n
		}
	}
	return fmt.Sprintf("地城 DG%d", lv.entry.MazeFile)
}
