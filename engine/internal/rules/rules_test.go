package rules

import "testing"

// 這些測試把 docs/manual/raw/scan-048-056.md 抄錄的表**原樣**寫進來,
// 再與 rules.go 的公式比對。
//
// ⚠ 測試裡的表是**獨立的一份抄寫**,不是從 rules.go 生出來的 ——
// 否則它只會證明「公式等於自己」。

func TestSpeedTableMatchesManual(t *testing.T) {
	// 手冊 p.48 EFFECTS OF SPEED:{速度, 最多移動, 最多攻擊}
	table := [][3]int{
		{3, 1, 1}, {4, 2, 1}, {5, 2, 1}, {6, 3, 2}, {7, 3, 2}, {8, 4, 2},
		{9, 4, 3}, {10, 5, 3}, {11, 5, 3}, {12, 6, 4}, {13, 6, 4}, {14, 7, 4},
		{15, 7, 5}, {16, 8, 5}, {17, 8, 5}, {18, 9, 6}, {19, 9, 6}, {20, 10, 6},
	}
	for _, r := range table {
		if got := MaxMoves(r[0]); got != r[1] {
			t.Errorf("速度 %d 的移動數 = %d,手冊 %d", r[0], got, r[1])
		}
		if got := MaxAttacks(r[0]); got != r[2] {
			t.Errorf("速度 %d 的攻擊數 = %d,手冊 %d", r[0], got, r[2])
		}
	}
}

func TestStrengthBonusMatchesManual(t *testing.T) {
	// 手冊 p.48 BONUS DAMAGE BY STRENGTH,STR 3–20
	want := []int{-2, -2, -1, -1, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6}
	for i, w := range want {
		str := i + 3
		if got := StrengthBonus(str); got != w {
			t.Errorf("力量 %d 的加值 = %d,手冊 %d", str, got, w)
		}
	}
}

// 負數要向下取整 —— Go 的 / 往零取整,BASIC 的 INT 往下。
// 這個差別只在 STR ≤ 6 顯現,所以拿掉 floorDiv 之後上面那個表仍會有兩列過。
func TestStrengthBonusRoundsDown(t *testing.T) {
	if got := StrengthBonus(6); got != -1 {
		t.Fatalf("力量 6 應為 −1(INT(−0.5) 向下),得 %d", got)
	}
}

func TestHitChanceMatchesManual(t *testing.T) {
	// 手冊 p.49 % CHANCE TO HIT BY SKILL:{技巧, 正面, 背後}
	table := [][3]int{
		{3, 12, 24}, {4, 16, 28}, {5, 20, 32}, {6, 24, 36}, {7, 28, 40},
		{8, 32, 44}, {9, 36, 48}, {10, 40, 52}, {11, 44, 56}, {12, 48, 60},
		{13, 52, 64}, {14, 56, 68}, {15, 60, 72}, {16, 64, 76}, {17, 68, 80},
		{18, 72, 84}, {19, 76, 88}, {20, 80, 92},
	}
	for _, r := range table {
		if got := HitChance(r[0], false); got != r[1] {
			t.Errorf("技巧 %d 正面命中 = %d,手冊 %d", r[0], got, r[1])
		}
		if got := HitChance(r[0], true); got != r[2] {
			t.Errorf("技巧 %d 背後命中 = %d,手冊 %d", r[0], got, r[2])
		}
	}
}

func TestHPGainMatchesManual(t *testing.T) {
	// 手冊 p.49 MAX H.P. GAIN PER LEVEL:{體質, 戰士, 巫師}
	table := [][3]int{
		{3, 3, 2}, {4, 3, 2}, {5, 4, 3}, {6, 5, 3}, {7, 5, 4}, {8, 6, 4},
		{9, 7, 4}, {10, 7, 5}, {11, 8, 5}, {12, 9, 6}, {13, 9, 6}, {14, 10, 7},
		{15, 11, 7}, {16, 11, 7}, {17, 12, 8}, {18, 13, 8}, {19, 13, 9}, {20, 14, 9},
	}
	for _, r := range table {
		if got := MaxHPGain(r[0], false); got != r[1] {
			t.Errorf("體質 %d 戰士生命成長 = %d,手冊 %d", r[0], got, r[1])
		}
		if got := MaxHPGain(r[0], true); got != r[2] {
			t.Errorf("體質 %d 巫師生命成長 = %d,手冊 %d", r[0], got, r[2])
		}
	}
}

func TestSPGainMatchesManual(t *testing.T) {
	// 手冊 p.48,只列到 INT 19
	want := []int{3, 4, 4, 5, 5, 6, 6, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12}
	for i, w := range want {
		in := i + 3
		if got := MaxSPGain(in); got != w {
			t.Errorf("智力 %d 的法力成長 = %d,手冊 %d", in, got, w)
		}
	}
}

// 把手冊 p.48 那張 20×10 的矩陣整張展開,逐格比對。
// 這同時驗證了 rules.go 那句「每一格只取決於 INT − LV」。
func TestDispelMatrixMatchesManual(t *testing.T) {
	const dash = -1
	matrix := [][10]int{
		{3, dash, dash, dash, dash, dash, dash, dash, dash, dash},
		{7, 3, dash, dash, dash, dash, dash, dash, dash, dash},
		{10, 7, 3, dash, dash, dash, dash, dash, dash, dash},
		{14, 10, 7, 3, dash, dash, dash, dash, dash, dash},
		{18, 14, 10, 7, 3, dash, dash, dash, dash, dash},
		{21, 18, 14, 10, 7, 3, dash, dash, dash, dash},
		{25, 21, 18, 14, 10, 7, 3, dash, dash, dash},
		{28, 25, 21, 18, 14, 10, 7, 3, dash, dash},
		{32, 28, 25, 21, 18, 14, 10, 7, 3, dash},
		{36, 32, 28, 25, 21, 18, 14, 10, 7, 3},
		{39, 36, 32, 28, 25, 21, 18, 14, 10, 7},
		{43, 39, 36, 32, 28, 25, 21, 18, 14, 10},
		{46, 43, 39, 36, 32, 28, 25, 21, 18, 14},
		{50, 46, 43, 39, 36, 32, 28, 25, 21, 18},
		{54, 50, 46, 43, 39, 36, 32, 28, 25, 21},
		{57, 54, 50, 46, 43, 39, 36, 32, 28, 25},
		{61, 57, 54, 50, 46, 43, 39, 36, 32, 28},
		{64, 61, 57, 54, 50, 46, 43, 39, 36, 32},
		{68, 64, 61, 57, 54, 50, 46, 43, 39, 36},
		{72, 68, 64, 61, 57, 54, 50, 46, 43, 39},
	}
	for i, row := range matrix {
		intel := i + 1
		for j, want := range row {
			lv := j + 1
			got := DispelChance(intel, lv)
			if want == dash {
				if got != 0 {
					t.Errorf("智力 %d vs 等級 %d:手冊是「—」,得 %d", intel, lv, got)
				}
				continue
			}
			if got != want {
				t.Errorf("智力 %d vs 等級 %d = %d,手冊 %d", intel, lv, got, want)
			}
		}
	}
}

// TestExpForLevelMatchesTheOriginal —— 經驗門檻是**公式**,不是手冊那張表。
//
//	門檻(n) = INT( (1.8^n + 2n) × 100 )      TOWN.EXE 0x10FF4(docs/re/223)
//
// ⚠ 兩條獨立證據都釘在這裡:
//  1. 原版實跑量到的四個點(訓練所的「你還需要 N 點經驗」)
//  2. 手冊 p.47 那張表 = 同一條公式**截到百位**,二十列一列不差
//
// 手冊的截尾在低等級差很多(第 1 級 380 → 300),照手冊實作會讓玩家
// 每一級都比原版早升,而畫面上沒有任何數字看起來是錯的。
func TestExpForLevelMatchesTheOriginal(t *testing.T) {
	// 原版實跑(2026-08-18,workplace/dosbox/shots/r3n*.png、r3o*.png)
	for lv, want := range map[int]int{1: 380, 2: 724, 3: 1183, 4: 1849} {
		if got := ExpForLevel(lv); got != want {
			t.Errorf("第 %d 級門檻 %d,原版實跑量到 %d", lv, got, want)
		}
	}
	// 手冊 p.47 的二十列 = 公式截到百位。
	manual := [...]int{
		1: 300, 2: 700, 3: 1_100, 4: 1_800, 5: 2_800,
		6: 4_600, 7: 7_500, 8: 12_600, 9: 21_600, 10: 37_700,
		11: 66_400, 12: 118_000, 13: 210_800, 14: 377_600, 15: 677_600,
		16: 1_217_500, 17: 2_189_300, 18: 3_938_200, 19: 7_086_100, 20: 12_752_200,
	}
	for lv := 1; lv <= MaxLevel; lv++ {
		if got := ExpForLevel(lv) / 100 * 100; got != manual[lv] {
			t.Errorf("第 %d 級公式值截到百位 = %d,手冊 p.47 是 %d", lv, got, manual[lv])
		}
	}
	// ⚠ 型別必須是 float32:float64 從第 11 級起會差 1–2 點。
	if ExpForLevel(11) != 66_468 {
		t.Errorf("第 11 級 = %d,MBF 單精度算出來是 66468 —— 是不是用了 float64?",
			ExpForLevel(11))
	}
}

func TestLevelUpUsesTheFormula(t *testing.T) {
	if CanLevelUp(1, 379) {
		t.Error("379 經驗不該能從 1 升 2(門檻 380)")
	}
	if !CanLevelUp(1, 380) {
		t.Error("380 經驗該能從 1 升 2")
	}
	// 頂級不再升
	if CanLevelUp(MaxLevel, 99_999_999) {
		t.Error("已達最高等級不該再升")
	}
}

// 升級免費(手冊 p.37)—— 這條測的是「介面上沒有金幣參數」這件事,
// 一旦有人加了價格,這個測試會編不過而不是靜靜地通過。
func TestLevelUpIsFree(t *testing.T) {
	var f func(int, int) bool = CanLevelUp
	if !f(5, 2_889) {
		t.Error("經驗到了就該能升,不看金幣")
	}
}

func TestRaceClassRestrictions(t *testing.T) {
	cases := []struct {
		r    Race
		c    Class
		want bool
	}{
		{Human, ClassHero, true}, {Human, ClassWizard, true},
		{Dwarf, ClassHero, true}, {Dwarf, ClassWizard, false},
		{Troll, ClassHero, true}, {Troll, ClassWizard, false},
		{Elf, ClassWizard, true}, {Elf, ClassHero, false},
		{Gnome, ClassWizard, true}, {Gnome, ClassHero, false},
	}
	for _, c := range cases {
		if got := AllowsClass(c.r, c.c); got != c.want {
			t.Errorf("%c/%c = %v,手冊 %v", c.r, c.c, got, c.want)
		}
	}
}

// 攻擊與施法成本相同,但只有施法結束回合 —— 不能用成本分辨動作。
func TestAttackDoesNotEndTurn(t *testing.T) {
	if ActAttack.Cost() != ActCast.Cost() {
		t.Fatal("前提變了:攻擊與施法的成本本來相同")
	}
	if ActAttack.EndsTurn() {
		t.Error("攻擊不該結束回合")
	}
	for _, a := range []Action{ActCast, ActUse, ActDispel} {
		if !a.EndsTurn() {
			t.Errorf("動作 %d 該結束回合(手冊 p.35)", a)
		}
	}
}
