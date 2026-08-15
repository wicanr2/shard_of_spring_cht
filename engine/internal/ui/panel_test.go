package ui

import (
	"fmt"
	"testing"
)

// docs/spec/06 §7 驗收 7:整列 ≤ 30 欄,而且名稱欄不會撞到 HP 欄。
//
// ⚠ 這裡測的是**版面預算**,不是「畫得出來」。
// 畫得出來但兩欄重疊,在畫面上是「數字黏在名字後面」——
// 看得出來但很容易被當成字型問題。
func TestPartyRowFits(t *testing.T) {
	// 名稱欄位是 10 bytes(docs/formats/01 位移 2–11),
	// 最壞情況是 10 個半形字元;中文名的話 5 個全形字也是 10 欄。
	for _, c := range []struct {
		name string
		hp   int
	}{
		{"Richtatha", 15},   // 出貨資料裡最長的
		{"0123456789", 999}, // 欄位上限 + 三位數 HP
		{"中文名字五", 999},      // 5 個全形 = 10 欄
	} {
		if slack := PanelSlack(c.name, len(fmt.Sprint(c.hp))); slack < 1 {
			t.Errorf("名稱 %q(%d 欄)+ HP %d:剩 %d 欄,至少要留 1 欄間隔",
				c.name, Cols(c.name), c.hp, slack)
		}
	}
	if int(ColSP) > 30 {
		t.Errorf("最右欄在第 %d 欄,側欄只有 30 欄(docs/spec/04 §5)", int(ColSP))
	}
}

// 全形算 2 欄。這條是 Cols 的行為約定,錯了會讓上面那條測試失效。
func TestColsCountsFullWidthAsTwo(t *testing.T) {
	for _, c := range []struct {
		s string
		n int
	}{
		{"abc", 3}, {"中文", 4}, {"中a", 3}, {"", 0},
		{"金幣：", 6}, // 全形冒號也算 2 欄
	} {
		if got := Cols(c.s); got != c.n {
			t.Errorf("Cols(%q) = %d,應為 %d", c.s, got, c.n)
		}
	}
}
