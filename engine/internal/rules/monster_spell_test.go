package rules

import "testing"

// 第一回合:系別基底 + 擲骰,結果必須落在該系別的攻擊法術範圍內
// (docs/re/170 §2/§3)。落到隔壁系別就是基底或張數寫錯了。
func TestMonsterSpellStaysInsideItsFamily(t *testing.T) {
	// SPELLS.DAT 的五個攻擊系別:起始法術編號(1-based)
	first := map[int]int{1: 1, 2: 4, 3: 8, 4: 12, 5: 16}
	last := map[int]int{1: 3, 2: 7, 3: 11, 4: 15, 5: 18}
	for fam := 1; fam <= 5; fam++ {
		for roll := 1; roll <= SpellFamilyAttacks[fam]; roll++ {
			got := MonsterSpell(fam, 1, roll)
			if got < first[fam] || got > last[fam] {
				t.Errorf("系別 %d 擲 %d → 法術 %d,超出 %d–%d",
					fam, roll, got, first[fam], last[fam])
			}
		}
		// 最小的擲骰就是系別的第一個法術
		if got := MonsterSpell(fam, 1, 1); got != first[fam] {
			t.Errorf("系別 %d 擲 1 → %d,應為 %d", fam, got, first[fam])
		}
	}
}

// 系別 3 的張數是**讀到的常數 2** —— 它切在攻擊與增益的分界上。
// 寫成 4 會讓怪物放 WINGS(增益),而那在畫面上像是「怪物什麼都沒做」。
func TestFamilyThreeSkipsTheBuffs(t *testing.T) {
	if SpellFamilyAttacks[3] != 2 {
		t.Fatalf("系別 3 的張數是 %d,原版讀到的是常數 2", SpellFamilyAttacks[3])
	}
	seen := map[int]bool{}
	for roll := 1; roll <= 4; roll++ { // 故意擲超過,夾住之後仍不該碰到增益
		seen[MonsterSpell(3, 1, roll)] = true
	}
	if seen[10] || seen[11] { // WINGS OF VICTORY / WINGS
		t.Error("系別 3 不該選到兩個增益法術")
	}
}

// 第二回合起固定放暴風,而系別 2 / 5 沒有分支(回 0)。
func TestStormFromRoundTwo(t *testing.T) {
	for fam, want := range map[int]int{1: 3, 3: 8, 4: 12, 2: 0, 5: 0} {
		if got := MonsterSpell(fam, 2, 1); got != want {
			t.Errorf("系別 %d 第二回合 → %d,應為 %d", fam, got, want)
		}
	}
	// ⚠ 系別 2 / 5 回 0 是**讀到的缺口**,不是預設值:
	// 原版那兩個系別在第二回合起沒有分支(docs/re/170 §4)。
}
