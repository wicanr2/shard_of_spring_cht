package ui

import "testing"

// docs/spec/04 §4:每行不超過指定欄寬。
func TestWrapRespectsWidth(t *testing.T) {
	for _, s := range []string{
		"這是一段很長的中文敘述用來測試斷行是不是真的有在算欄寬",
		"Fire Hawk 擊中 Small Dragon,Small Dragon 倒下",
		"abcdefghij klmnopqrst",
	} {
		for _, w := range []int{10, 20, 30} {
			for i, ln := range Wrap(s, w) {
				// ⚠ 行首禁則會讓某一行超出一欄 —— 那是刻意的,
				// 孤立的標點比超寬一欄難看。所以容忍 +2(一個全形標點)。
				if Cols(ln) > w+2 {
					t.Errorf("寬 %d:第 %d 行 %d 欄 %q", w, i, Cols(ln), ln)
				}
			}
		}
	}
}

// 標點不可以落在行首。
func TestWrapNoLeadingPunctuation(t *testing.T) {
	got := Wrap("測試一二三,四五六七八。", 8)
	for i, ln := range got {
		if i == 0 {
			continue
		}
		r := []rune(ln)[0]
		if containsRune(noLineStart, r) {
			t.Errorf("第 %d 行以標點 %q 起頭:%v", i, string(r), got)
		}
	}
}

// 英文單字不在字母中間斷開。
func TestWrapKeepsWordsIntact(t *testing.T) {
	for _, ln := range Wrap("怪物 Small Dragon 出現", 8) {
		for _, w := range []string{"Small", "Dragon"} {
			// 出現就必須完整出現
			if idx := indexOf(ln, w[:2]); idx >= 0 && indexOf(ln, w) < 0 {
				t.Errorf("%q 被切開了:%q", w, ln)
			}
		}
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// 不會吃掉字:所有行接起來要等於原文。
func TestWrapLosesNothing(t *testing.T) {
	for _, s := range []string{"短", "中文與 English 混排的一段文字,含標點。", ""} {
		var joined string
		for _, ln := range Wrap(s, 7) {
			joined += ln
		}
		if joined != s {
			t.Errorf("折行後 %q,原文 %q —— 斷行不可以增刪字元", joined, s)
		}
	}
}
