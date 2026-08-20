package magic

import (
	"testing"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
)

// docs/re/230 §3.1:移轉術的欄4 與欄5 都是 3 → 效力恆等於投入 → **1:1 轉移**。
func TestTransferenceIsOneForOne(t *testing.T) {
	s := original.Spell{Name: "移轉術", Effect: EffTransference, Power: 3, UnitCost: 3}
	for _, invest := range []int{1, 3, 9, 20} {
		tgt := combat.Unit{SP: 0}
		r := Apply(s, invest, &combat.Unit{}, []*combat.Unit{&tgt})
		if tgt.SP != invest {
			t.Errorf("投入 %d,目標得到 %d 點法力,應為 %d", invest, tgt.SP, invest)
		}
		if r.Unresolved {
			t.Errorf("投入 %d:類別 13 已經解出來了,不該再標未解", invest)
		}
	}
}

// 那道 `min(量, 投入)` 在移轉術上永遠不觸發,但它擋的是
// 「欄4 > 欄5 的法術憑空生出法力」—— 拿一個假造的比例驗它還在
// (docs/re/230 §3.1:一道永遠不觸發的上限,要問它防的是什麼)。
func TestTransferenceNeverCreatesPoints(t *testing.T) {
	// 欄4 = 10、欄5 = 1 → 效力 = 10 × 投入,若沒有那道上限就會生出法力。
	s := original.Spell{Name: "假", Effect: EffTransference, Power: 10, UnitCost: 1}
	tgt := combat.Unit{SP: 0}
	Apply(s, 4, &combat.Unit{}, []*combat.Unit{&tgt})
	if tgt.SP != 4 {
		t.Errorf("轉移量 %d 超過投入的 4 —— min(量, 投入) 那道上限沒接上", tgt.SP)
	}
}

// docs/re/230 §3:寫回之前夾三次 —— 下限 3、最大值、硬上限 255。
func TestRestoreClampsThreeWays(t *testing.T) {
	for _, c := range []struct {
		name          string
		newValue, max int
		want          int
	}{
		{"低於下限", 1, 20, RestoreFloor},
		{"下限剛好", 3, 20, 3},
		{"一般", 12, 20, 12},
		{"超過最大值", 99, 20, 20},
		{"沒有最大值時只夾 255", 999, 0, RestoreCap},
		{"最大值比 255 大也夾 255", 999, 9999, RestoreCap},
	} {
		if got := Restore(c.newValue, c.max); got != c.want {
			t.Errorf("%s:Restore(%d, %d) = %d,應為 %d",
				c.name, c.newValue, c.max, got, c.want)
		}
	}
	// 下限**是 3 不是 0** —— 這一條專門釘住那個容易被寫成 0 的數字。
	if RestoreFloor != 3 {
		t.Errorf("下限 %d,原版是 3(CAMP 0x11FFE)", RestoreFloor)
	}
	// 255 與 docs/re/218 §1 的 ds:6F10 是同一個數字。
	if RestoreCap != 255 {
		t.Errorf("硬上限 %d,原版的 ds:6F10 是 255", RestoreCap)
	}
}
