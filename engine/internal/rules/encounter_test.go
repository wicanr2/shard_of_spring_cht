package rules

import "testing"

// 世界地圖三區的分界(docs/re/169 §2)。邊界值寫錯不會有症狀,所以逐格釘。
func TestWorldZoneBoundaries(t *testing.T) {
	for _, c := range []struct{ x, y, want int }{
		{0, 0, 1}, {70, 60, 1}, // x ≤ 70 且 y ≤ 60
		{71, 60, 3}, {102, 0, 3}, // x > 70 且 y ≤ 60
		{0, 61, 6}, {71, 61, 6}, // y > 60 一律 6(x 不再參與)
	} {
		if got := WorldZone(c.x, c.y); got != c.want {
			t.Errorf("(%d,%d) → 區域 %d,應為 %d", c.x, c.y, got, c.want)
		}
	}
	// ISLANDA 拱門那一格在分界線上 —— 它自己屬於區域 1
	if WorldZone(70, 57) != 1 || WorldZone(71, 57) != 3 {
		t.Error("x = 70 的拱門格與它東邊一格應分屬不同區域")
	}
}

// 迷宮逐座指定,而第 5 座再按樓層座標分兩區(docs/re/169 §3)。
func TestMazeZone(t *testing.T) {
	for _, c := range []struct{ no, mx, my, want int }{
		{1, 0, 0, 1}, {2, 0, 0, 1},
		{3, 0, 0, 3}, {6, 0, 0, 3},
		{4, 0, 0, 5},
		{5, 24, 24, 2}, // 塔的西北角
		{5, 25, 24, 8}, // 邊界:25 不算「< 25」
		{5, 24, 25, 8},
		{12, 0, 0, 9},
		{7, 0, 0, 6}, {11, 0, 0, 6}, {13, 0, 0, 6}, // 未列到的走預設
	} {
		if got := MazeZone(c.no, c.mx, c.my); got != c.want {
			t.Errorf("迷宮 %d (%d,%d) → 區域 %d,應為 %d",
				c.no, c.mx, c.my, got, c.want)
		}
	}
}

// 判定是**絕對值** —— 原版有 neg,單邊的寫法會讓低階怪出現在高階區。
func TestZoneAcceptsIsSymmetric(t *testing.T) {
	for _, c := range []struct {
		zone, tier int
		want       bool
	}{
		{3, 2, true}, {3, 3, true}, {3, 4, true},
		{3, 1, false}, {3, 5, false},
		{1, 13, false}, // 階級 13 只出現在區域 12–14,而區域最大是 9
	} {
		if got := ZoneAccepts(c.zone, c.tier); got != c.want {
			t.Errorf("區域 %d / 階級 %d → %v,應為 %v",
				c.zone, c.tier, got, c.want)
		}
	}
}

// 七個實際用到的區域,每一個都必須有候選 —— 原版的重擲沒有上限,
// 一個區域湊不到就會當掉(docs/re/169 §1.1)。
//
// ⚠ 這條測試用的是**出貨資料的階級分佈**(docs/formats/03 欄9:1–10、13),
// 不是硬編的名單 —— 欄位讀錯時它會失敗。
func TestEveryZoneHasCandidates(t *testing.T) {
	// MONSTERS.DAT 的欄9 分佈(74 筆)
	tiers := map[int]int{1: 10, 2: 6, 3: 8, 4: 6, 5: 6, 6: 4, 7: 12, 8: 4, 9: 5, 10: 12, 13: 1}
	for _, zone := range []int{1, 2, 3, 5, 6, 8, 9} {
		n := 0
		for tier, count := range tiers {
			if ZoneAccepts(zone, tier) {
				n += count
			}
		}
		if n == 0 {
			t.Errorf("區域 %d 沒有候選怪物 —— 原版會在這裡無限重擲", zone)
		}
	}
	// ⚠ 階級 13 只有 Siriadne,而區域最大是 9 → 她永遠不會隨機出現。
	// 她由事件 533 的腳本放上場(docs/re/161 §4)。
	for zone := 1; zone <= 9; zone++ {
		if ZoneAccepts(zone, 13) {
			t.Errorf("區域 %d 挑得到階級 13 —— 最終首領不該隨機出現", zone)
		}
	}
	// 但階級 10 在區域 9 挑得到 —— 只有 13 是例外
	if !ZoneAccepts(9, 10) {
		t.Error("區域 9 應該挑得到階級 10(元素生物、Great Dragon 那一批)")
	}
}

// ── 一場遭遇有幾隻(docs/re/225 §2)────────────────────────────────────

// TestEncounterCountStaysWithinTheCap:上限那一行是**規則**不是保險絲。
//
// `cap = 7` 時算式本身擲得出 8(`INT(7×RND×0.5)` 最大 3,加 3.5 加 1 = 7.5 → 8),
// 而實測十二場的最大值是 7 —— 少了上限,難度會整批偏高一級。
func TestEncounterCountStaysWithinTheCap(t *testing.T) {
	for _, capCol := range []int{4, 5, 6, 7} {
		seen := map[int]bool{}
		for i := 0; i <= 100; i++ {
			n := EncounterCount(capCol, float64(i)/101)
			if n < 1 || n > capCol {
				t.Errorf("cap %d:擲出 %d 隻,應落在 1…%d", capCol, n, capCol)
			}
			seen[n] = true
		}
		if len(seen) < 2 {
			t.Errorf("cap %d:整段 RND 只擲得出 %v —— 那不是一條隨機算式", capCol, seen)
		}
	}
}

// TestEncounterCountMatchesTheFieldSamples:十二場實測的隻數都在算式的值域內。
//
// 實測(docs/re/225 §8)區域 1:cap 6 的列給 4–6、cap 7 的列給 5–7。
func TestEncounterCountMatchesTheFieldSamples(t *testing.T) {
	rangeOf := func(capCol int) (lo, hi int) {
		lo, hi = 99, 0
		for i := 0; i <= 200; i++ {
			n := EncounterCount(capCol, float64(i)/201)
			if n < lo {
				lo = n
			}
			if n > hi {
				hi = n
			}
		}
		return
	}
	for _, c := range []struct{ capCol, lo, hi int }{{6, 4, 6}, {7, 5, 7}} {
		lo, hi := rangeOf(c.capCol)
		if lo != c.lo || hi != c.hi {
			t.Errorf("cap %d 的值域是 %d…%d,實測對應的是 %d…%d",
				c.capCol, lo, hi, c.lo, c.hi)
		}
	}
}

// TestEncounterRunCanTakeEverything:一次就放滿是**常見**情況,不是邊界。
//
// 實測十二場有七場清一色 —— 那是「連放」擲出整數群的結果。
// ⛔ 若把 EncounterRun 夾到 1,清一色就只能靠「四欄剛好同一隻」發生。
func TestEncounterRunCanTakeEverything(t *testing.T) {
	if n := EncounterRun(7, 0, 0.99); n < 7 {
		t.Errorf("RND 接近 1 時應該一次放滿(≥7),得到 %d", n)
	}
	// 已放的數量會把它壓下去 —— 這是原版的減項,不是保護。
	if n := EncounterRun(7, 6, 0.1); n > 1 {
		t.Errorf("已放 6 隻、RND 很小時不該再放 %d 隻", n)
	}
}
