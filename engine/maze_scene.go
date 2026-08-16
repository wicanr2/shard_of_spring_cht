package main

import (
	"fmt"
	"path/filepath"
	"strconv"

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
	entry  original.MazeEntry
	grid   *original.Maze
	events []original.Event
	text   map[int]string
}

// enterMaze 從世界地圖進入迷宮。回 false 表示這一格沒有入口。
func (g *Game) enterMaze(x, y int) bool {
	for _, e := range g.mazeData {
		if e.WorldX != x || e.WorldY != y {
			continue
		}
		lv, err := g.loadLevel(e)
		if err != nil {
			g.warnings = append(g.warnings, err.Error())
			return false
		}
		g.level = lv
		g.mazeState = maze.State{
			Major: e.StartMajor, Minor: e.StartMinor,
			Facing:     maze.Facing(e.Facing),
			Visibility: g.group.VisLit,
		}
		g.overlay = ""
		return true
	}
	return false
}

func (g *Game) loadLevel(e original.MazeEntry) (*mazeLevel, error) {
	lv := &mazeLevel{entry: e}
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
	found := false
	for _, e := range g.mazeData {
		if e.MazeFile == n {
			entry, found = e, true
			break
		}
	}
	if !found {
		g.warnings = append(g.warnings, fmt.Sprintf("存檔記著在 DG%d,但找不到這個地城", n))
		return
	}
	lv, err := g.loadLevel(entry)
	if err != nil {
		g.warnings = append(g.warnings, "重新載入迷宮失敗:"+err.Error())
		return
	}
	g.level = lv
	g.mazeState = maze.State{
		Major: g.group.MazeX, Minor: g.group.MazeY,
		Facing: maze.Facing(facing), Visibility: g.group.VisLit,
	}
	g.overlay = ""
}

// stepMaze 處理一次方向輸入,並在**實際位移之後**掃事件(docs/spec/08 §4)。
func (g *Game) stepMaze(dir maze.Facing) {
	if g.level == nil {
		return
	}
	switch g.mazeState.Step(dir, g.level.grid) {
	case maze.Left:
		// 走出邊界 → 回世界地圖。原版在這一刻印 `Leaving maze ..`
		// (docs/re/147:實跑從入口那一格往外走一步就出去了)。
		g.level = nil
		g.overlay = "離開地城……" // MAZEMOVE:88
		return
	case maze.Moved:
	default:
		return
	}
	g.fireTrigger(maze.Scan(g.level.events, g.mazeState, g.level.text))
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
	for _, e := range g.mazeData {
		if e.MazeFile == cur.MazeFile {
			continue
		}
		// DG5 ↔ DG51:檔號其中一個是另一個的前綴(5 → 51)
		if !relatedMazeFile(cur.MazeFile, e.MazeFile) {
			continue
		}
		lv, err := g.loadLevel(e)
		if err != nil {
			g.warnings = append(g.warnings, err.Error())
			return
		}
		g.level = lv
		g.mazeState.Major, g.mazeState.Minor = e.StartMajor, e.StartMinor
		g.overlay = fmt.Sprintf("你走到了另一層(DG%d)。", e.MazeFile)
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
	const half = layout.ViewTiles / 2
	for vy := 0; vy < layout.ViewTiles; vy++ {
		for vx := 0; vx < layout.ViewTiles; vx++ {
			aM := g.mazeState.Major - half + vx
			am := g.mazeState.Minor - half + vy
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
	// 隊伍所在格
	c := float32(layout.View.X + half*layout.TileDst)
	r := float32(layout.View.Y + half*layout.TileDst)
	vector.StrokeRect(dst, c, r, layout.TileDst, layout.TileDst, 3, cgaWhite, false)

	// ⚠ 自己斷成兩行,不要靠折行 —— 這段固定超過訊息面板的 30 欄,
	// 而 ui.Wrap 是按欄數硬斷的,會把「能見度」從中間切開。
	g.drawMessageLines(dst, []string{
		fmt.Sprintf("地城 DG%d　(%d, %d)", lv.entry.MazeFile,
			g.mazeState.Major, g.mazeState.Minor),
		fmt.Sprintf("朝向 %d　能見度 %d", g.mazeState.Facing, g.mazeState.Visibility),
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
	for _, ln := range ui.Wrap(g.overlay, cols) {
		g.overlayFont.Draw(dst, ln, float64(rc.X+pad), y)
		y += lh
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
