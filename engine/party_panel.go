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
	g.panel.Draw(dst, fmt.Sprintf("食糧：%d", g.group.Provisions), at(ui.ColNum), y)
	// 時刻與朝向。**原版右下角常駐一個框寫 `Hour N   Facing ↓`**
	// (2026-08-18 對照原版,docs/spec/14 §12-C)——
	// ⚠ 兩個都是玩家每一步都要用的資訊:時刻決定天色與視野
	// (docs/re/213),朝向決定按鍵會轉身還是前進(「先轉再走」)。
	// 先前只出現在**視窗標題列**,遊戲畫面上讀不到。
	y += lh
	g.panel.Draw(dst, fmt.Sprintf("時刻：%d 時　朝向：%s",
		g.party.Clock.Hour, facingArrow(g.facingNow())), at(ui.ColNum), y)

	// 提示列:**由實際會接手按鍵的場景自己說**(message.go 的 activeScene)。
	// ⚠ 先前這裡是一個 switch,自己判斷「現在是哪個畫面」—— 與 Update() 的
	// 派工是兩份各自演化的判斷,戰鬥那一行因此停在早就移除的
	// 「空白鍵:推進一回合」,而畫面上看不出那是舊的。
	px, py := promptOrigin()
	if line := g.activeScene().Prompt(); line != "" {
		g.panel.Draw(dst, line, px, py)
		py += lh
	}
	// 未解項在**執行時**也要看得見(docs/spec/07 §3),不是只寫在文件裡。
	//
	// ⚠ **畫在下一行,不要跟指令列擠同一行。** 先前是 `px+460` 同一行,
	// 而指令列本身就超過 460 px —— 兩段字直接疊在一起,誰都讀不出來。
	// 這是拍 README 截圖時才看見的:UI 的重疊在測試裡沒有症狀。
	// ⚠ 折行到提示列的欄寬 —— 這些說明比一行長,不折就畫到框外,
	// 而畫到框外的字照樣印得出來(只是壓在畫面邊緣)。
	notice := func(s string) {
		for _, ln := range ui.Wrap(s, layout.PromptCols) {
			g.panel.Draw(dst, ln, px, py)
			py += lh
		}
	}
	if g.field != nil {
		for _, u := range combat.Unresolved {
			notice(u) // ⚠ 常數本身已經帶了「⚠ 」,不要再加一個
		}
	}
	// 治療池的擲骰面數是佔位(docs/re/155 §2.3),只在池邊那一刻標出來。
	if g.prompt != nil && g.prompt.kind == promptPool {
		for _, u := range maze.Unresolved {
			notice(u)
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
