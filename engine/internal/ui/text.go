// Package ui 是文字排版與繪製。規格:docs/spec/04-display-layout.md §4。
package ui

import (
	"strings"
	"unicode"
)

// Cols 回傳一段字串的**顯示欄寬**:全形算 2,半形算 1。
//
// 側欄與提示列的容量全部以「欄」為單位(docs/spec/04 §5),
// ⚠ **不是以「字」為單位** —— 我曾經拿字數去比欄數,得到「放得下」的錯誤結論。
func Cols(s string) int {
	n := 0
	for _, r := range s {
		n += RuneCols(r)
	}
	return n
}

// RuneCols 回傳單一字元的顯示欄寬。
//
// 判準:Unicode 的 East Asian Wide / Fullwidth 算 2,其餘算 1。
// 這裡用區段判斷而不是拉整張 EastAsianWidth 表 —— 本專案的文本只有
// 中文、ASCII 與常見標點,而**多引一個相依是要付維護成本的**。
func RuneCols(r rune) int {
	switch {
	case r < 0x1100:
		return 1
	case unicode.Is(unicode.Han, r), // 漢字
		r >= 0x3000 && r <= 0x303F,  // CJK 標點
		r >= 0xFF01 && r <= 0xFF60,  // 全形 ASCII
		r >= 0xFFE0 && r <= 0xFFE6,  // 全形符號
		r >= 0x3040 && r <= 0x30FF,  // 假名
		r >= 0xAC00 && r <= 0xD7A3:  // 諺文
		return 2
	}
	return 1
}

// PadTo 把字串補空白到指定欄寬;超過就**不截斷**,原樣回傳。
//
// ⚠ 故意不截斷:截斷會讓「排版爆掉」變成「看起來正常但少了字」,
// 而後者查不出來。放不下要在版面規格裡解決,不是在繪製時砍掉
// (docs/spec/04 §5 的商店清單就是這樣被發現的)。
func PadTo(s string, cols int) string {
	if n := Cols(s); n < cols {
		return s + strings.Repeat(" ", cols-n)
	}
	return s
}

// PadLeft 把字串靠右補到指定欄寬(數字欄用)。
func PadLeft(s string, cols int) string {
	if n := Cols(s); n < cols {
		return strings.Repeat(" ", cols-n) + s
	}
	return s
}

// ---------------------------------------------------------------------------
// 隊伍狀態欄的欄位位置。docs/spec/06-party-and-save.md §5。
//
// ⚠ **用像素位置排欄,不用空白補位** —— 字型是比例字,補空白只在
// 等寬終端機裡對齊。這裡的常數是「第幾欄」,由 render 換算成像素。
// ---------------------------------------------------------------------------

const (
	PanelPad = 16
	// ColUnit 是 20 px 字下一「欄」的像素寬(半形寬)。
	ColUnit = 10.0

	ColNum    = 0.0  // 編號:左對齊
	ColStatus = 2.0  // 狀態:左對齊
	ColName   = 5.0  // 名稱:左對齊
	ColHP     = 24.0 // HP:**右**對齊到這一欄
	ColSP     = 30.0 // SP:右對齊
)

// PanelSlack 回傳「名稱欄尾」與「HP 欄頭」之間還剩幾欄;負值表示重疊。
//
// 側欄總容量 30 欄(docs/spec/04 §5),而真正會爆的是這兩欄之間:
// 名稱長度可變、HP 右對齊。**放不下要改版面,不要在繪製時截字** ——
// 截掉之後畫面看起來正常,但少了字。
func PanelSlack(name string, hpDigits int) int {
	return int(ColHP) - hpDigits - (int(ColName) + Cols(name))
}
