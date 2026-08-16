package main

// 訊息面板與指令提示列的共用畫法。docs/spec/14-remake-worklist.md §4(C3)。
//
// 先前城鎮、名冊、迷宮、戰鬥各自寫一份「從 layout.Message 左上角開始、
// 折成 30 欄、一行一行往下畫」,四份逐字相同;而施法與道具選單各自寫一份
// 「先用黑色蓋掉底下的訊息」。收斂成這裡的三個函式。
//
// ⚠ 30 這個欄數是**版面推出來的**(layout.MsgCols),不是手感 ——
// 面板 340 px 扣內距 16×2 = 308 px,20 px 的字剛好 30 欄。

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/layout"
	"shardofspring/internal/ui"
)

// drawMessage 把一段文字折行後畫進訊息面板。空字串什麼都不畫。
func (g *Game) drawMessage(dst *ebiten.Image, text string) {
	if text == "" {
		return
	}
	g.drawMessageLines(dst, ui.Wrap(text, layout.MsgCols))
}

// drawMessageLines 畫**已經折好**的行(戰鬥的訊息是「日誌的最後幾行」,
// 折行與取尾要一起算,不能只給一段文字)。超出面板高度的部分不畫。
func (g *Game) drawMessageLines(dst *ebiten.Image, lines []string) {
	p := g.panel
	if p == nil || len(lines) == 0 {
		return
	}
	lh := p.LineHeight()
	x := float64(layout.Message.X + ui.PanelPad)
	y := float64(layout.Message.Y + ui.PanelPad)
	bottom := float64(layout.Message.Bottom()) - ui.PanelPad
	for _, ln := range lines {
		if y+lh > bottom {
			return
		}
		p.Draw(dst, ln, x, y)
		y += lh
	}
}

// messageLines 回傳訊息面板放得下幾行。
func (g *Game) messageLines() int {
	if g.panel == nil {
		return 0
	}
	return int((float64(layout.Message.H) - 2*ui.PanelPad) / g.panel.LineHeight())
}

// clearMessage 用底色蓋掉訊息面板的內容,給蓋在上面的選單用。
//
// ⚠ 兩層字疊在一起是「看得到但讀不出來」,而那比缺字更難察覺。
// 內縮 2 px 是為了留下外框那一圈線。
func clearMessage(dst *ebiten.Image) {
	rc := layout.Message
	vector.DrawFilledRect(dst, float32(rc.X+2), float32(rc.Y+2),
		float32(rc.W-4), float32(rc.H-4), cgaBlack, false)
}

// promptOrigin 回傳指令提示列的左上角起點。
func promptOrigin() (x, y float64) {
	return float64(layout.Prompt.X + ui.PanelPad), float64(layout.Prompt.Y + ui.PanelPad)
}

// activeScene 回傳「現在沒有按任何鍵的話,這一格會由誰接手」——
// 也就是畫面上該顯示誰的指令提示列。
//
// ⚠ 用空的 Input 問 —— 唯一會因輸入而改變答案的是 N)ames 熱鍵
// (它只在真的按下 N 那一格接手),而熱鍵不該搶走提示列。
// **這就是「提示列跟著實際接手者走」的實作**:先前提示列自己寫了一份
// switch 去猜現在是哪個畫面,兩份判斷各自演化,結果戰鬥那一行停在
// 「空白鍵:推進一回合」——那個鍵在 M10 之後已經不存在了。
func (g *Game) activeScene() Scene {
	for _, s := range g.inputChain() {
		if s.Handles(Input{}) {
			return s
		}
	}
	return worldScene{g}
}
