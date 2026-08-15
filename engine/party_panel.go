package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/ui"
)

func (g *Game) drawParty(dst *ebiten.Image) {
	if g.panel == nil {
		return
	}
	x0 := float64(layout.Party.X + ui.PanelPad)
	y := float64(layout.Party.Y + ui.PanelPad)
	lh := g.panel.LineHeight()
	at := func(c float64) float64 { return x0 + c*ui.ColUnit }

	// 標題列
	g.panel.Draw(dst, "#", at(ui.ColNum), y)
	g.panel.Draw(dst, "狀", at(ui.ColStatus), y)
	g.panel.Draw(dst, "名稱", at(ui.ColName), y)
	g.panel.DrawRight(dst, "HP", at(ui.ColHP), y)
	g.panel.DrawRight(dst, "SP", at(ui.ColSP), y)
	y += lh

	for i, c := range g.members {
		g.panel.Draw(dst, fmt.Sprint(i+1), at(ui.ColNum), y)
		// 狀態 0(正常)顯示空白 —— 五人都正常時整欄空著,異常才跳出來
		g.panel.Draw(dst, c.StatusName(), at(ui.ColStatus), y)
		g.panel.Draw(dst, c.Name, at(ui.ColName), y)
		g.panel.DrawRight(dst, fmt.Sprint(c.HP), at(ui.ColHP), y)
		g.panel.DrawRight(dst, fmt.Sprint(c.SP), at(ui.ColSP), y)
		y += lh
	}

	y += lh * 0.5
	g.panel.Draw(dst, fmt.Sprintf("金幣：%.0f", g.group.Gold), at(ui.ColNum), y)
	y += lh
	g.panel.Draw(dst, fmt.Sprintf("補給：%d", g.group.Provisions), at(ui.ColNum), y)

	// 提示列:可用按鍵 + 載入時發現的不一致(docs/spec/06 §1)——
	// 不自行修正,但也不安靜吞掉。
	px := float64(layout.Prompt.X + ui.PanelPad)
	py := float64(layout.Prompt.Y + ui.PanelPad)
	switch {
	case g.field != nil:
		g.panel.Draw(dst, "空白鍵：推進一回合　　C：施法（固定投一級）　　ESC：離開", px, py)
	case g.roster != nil && g.roster.open:
		g.panel.Draw(dst, "↑↓：選　　J：入隊　　D：離隊　　X：刪除　　ESC：返回", px, py)
	case g.town != nil && g.town.mode != townClosed:
		g.panel.Draw(dst, "字母：選項　　+／-：翻頁　　ESC：返回／離開城鎮", px, py)
	case g.level != nil:
		g.panel.Draw(dst, "方向鍵／1234：移動　　ESC：離開地城　　S：存檔", px, py)
	default:
		g.panel.Draw(dst, "方向鍵／1234：移動　　N：名冊　　S：存檔", px, py)
	}
	py += lh
	// 未解項在**執行時**也要看得見(docs/spec/07 §3),不是只寫在文件裡。
	if g.field != nil {
		for i, u := range combat.Unresolved {
			g.panel.Draw(dst, "⚠ "+u, px+460, float64(layout.Prompt.Y+ui.PanelPad)+float64(i)*lh)
		}
	}
	// 治療池的擲骰面數是佔位(docs/re/155 §2.3),只在池邊那一刻標出來。
	if g.prompt != nil && g.prompt.kind == promptPool {
		for i, u := range maze.Unresolved {
			g.panel.Draw(dst, u, px+460, float64(layout.Prompt.Y+ui.PanelPad)+float64(i)*lh)
		}
	}
	if g.saveMsg != "" {
		g.panel.Draw(dst, g.saveMsg, px, py)
		py += lh
	}
	for _, w := range g.warnings {
		g.panel.Draw(dst, "⚠ "+w, px, py)
		py += lh
	}
}
