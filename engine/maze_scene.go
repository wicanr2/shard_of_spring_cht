package main

import (
	"fmt"
	"path/filepath"

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
	return lv, nil
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
		g.overlay = "離開迷宮……"
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

	lh := p.LineHeight()
	p.Draw(dst, fmt.Sprintf("地城 DG%d　(%d, %d)　朝向 %d　能見度 %d",
		lv.entry.MazeFile, g.mazeState.Major, g.mazeState.Minor,
		g.mazeState.Facing, g.mazeState.Visibility),
		float64(layout.Message.X+ui.PanelPad),
		float64(layout.Message.Y+ui.PanelPad)+lh*0)
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
