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
		r >= 0x3000 && r <= 0x303F, // CJK 標點
		r >= 0xFF01 && r <= 0xFF60, // 全形 ASCII
		r >= 0xFFE0 && r <= 0xFFE6, // 全形符號
		r >= 0x3040 && r <= 0x30FF, // 假名
		r >= 0xAC00 && r <= 0xD7A3: // 諺文
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

	// CombatNameCols 是戰鬥單位列的名稱欄寬(docs/spec/10 §4)。
	//
	// ⚠ 24 而不是 12:九個頭目的譯名帶 `中文(English)` 加註
	// (glossary 硬規則 4),最長的 `希瑞雅妮！(Siriadne !)` 是 22 欄。
	// 主視野有 60 欄,**該放寬的是預算不是譯名** ——
	// 原版的 16 bytes 限制來自它要寫回 MONSTERS.DAT,remake 不寫回。
	CombatNameCols = 24
)

// PanelSlack 回傳「名稱欄尾」與「HP 欄頭」之間還剩幾欄;負值表示重疊。
//
// 側欄總容量 30 欄(docs/spec/04 §5),而真正會爆的是這兩欄之間:
// 名稱長度可變、HP 右對齊。**放不下要改版面,不要在繪製時截字** ——
// 截掉之後畫面看起來正常,但少了字。
func PanelSlack(name string, hpDigits int) int {
	return int(ColHP) - hpDigits - (int(ColName) + Cols(name))
}

// ---------------------------------------------------------------------------
// 斷行。docs/spec/04-display-layout.md §4。
// ---------------------------------------------------------------------------

// 行首禁則:這些字元不可以出現在一行的開頭。
const noLineStart = "。,、；：?!」』）〕】》〉…—～%‰℃,.;:?!)]}"

// HangTolerance 是懸掛標點允許超出的欄數(一個全形字寬)。
// 超過就改成把上一行最後一個字拉下來(見 Wrap)。
const HangTolerance = 2

// 行尾禁則:這些字元不可以出現在一行的結尾。
const noLineEnd = "「『（〔【《〈$#([{"

// Wrap 把一段文字折成每行不超過 cols 欄。
//
// 中文允許任意位置斷行,但要避頭尾(docs/spec/04 §4):
// 斷點若讓標點落到行首、或讓開括號落到行尾,就把前一個字一起推到下一行。
//
// ⚠ **英文與數字視為不可分割**,不在字母中間斷行(§4 中英混排第 1 點)。
func Wrap(s string, cols int) []string {
	if cols <= 0 {
		return []string{s}
	}
	rs := []rune(s)
	var out []string
	line := []rune{}
	w := 0
	flush := func() {
		if len(line) > 0 {
			out = append(out, string(line))
			line, w = nil, 0
		}
	}
	for i := 0; i < len(rs); i++ {
		// 拉丁字母/數字連成一個不可分割的單位
		j := i
		for j < len(rs) && isWordRune(rs[j]) {
			j++
		}
		var tok []rune
		if j > i {
			tok = rs[i:j]
			i = j - 1
		} else {
			tok = rs[i : i+1]
		}
		tw := Cols(string(tok))
		if w+tw > cols && w > 0 {
			// 行尾禁則:最後一個字不可以留在行尾
			for len(line) > 0 && containsRune(noLineEnd, line[len(line)-1]) {
				tok = append([]rune{line[len(line)-1]}, tok...)
				line = line[:len(line)-1]
				w -= RuneCols(tok[0])
			}
			flush()
		}
		// 行首禁則:標點不可以起頭。做法是**懸掛**到上一行 ——
		// 但只掛得下一個全形寬(HangTolerance)。
		//
		// ⚠ 連續的收尾標點(例如 `！」`)會一路掛過去,把上一行撐爆。
		// 掛不下時改用另一種避頭尾:把上一行的**最後一個字拉下來**,
		// 讓標點跟著它到新的一行。
		if w == 0 && len(out) > 0 && len(tok) == 1 && containsRune(noLineStart, tok[0]) {
			prev := out[len(out)-1]
			if Cols(prev)+RuneCols(tok[0]) <= cols+HangTolerance {
				out[len(out)-1] = prev + string(tok)
				continue
			}
			pr := []rune(prev)
			if len(pr) > 1 {
				out[len(out)-1] = string(pr[:len(pr)-1])
				line = append(line, pr[len(pr)-1])
				w += RuneCols(pr[len(pr)-1])
			}
		}
		line = append(line, tok...)
		w += tw
	}
	flush()
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '\'' || r == '-' || r == '+'
}

func containsRune(set string, r rune) bool {
	for _, c := range set {
		if c == r {
			return true
		}
	}
	return false
}
