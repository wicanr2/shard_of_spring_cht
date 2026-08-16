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
			got, _ := MonsterSpell(fam, 1, roll)
			if got < first[fam] || got > last[fam] {
				t.Errorf("系別 %d 擲 %d → 法術 %d,超出 %d–%d",
					fam, roll, got, first[fam], last[fam])
			}
		}
		// 最小的擲骰就是系別的第一個法術
		if got, _ := MonsterSpell(fam, 1, 1); got != first[fam] {
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
		sp, _ := MonsterSpell(3, 1, roll)
		seen[sp] = true
	}
	if seen[10] || seen[11] { // WINGS OF VICTORY / WINGS
		t.Error("系別 3 不該選到兩個增益法術")
	}
}

// 第二回合起固定放暴風,而系別 2 / 5 沒有分支(回 0)。
func TestStormFromRoundTwo(t *testing.T) {
	for fam, want := range map[int]int{1: 3, 3: 8, 4: 12, 2: 0, 5: 0} {
		if got, _ := MonsterSpell(fam, 2, 1); got != want {
			t.Errorf("系別 %d 第二回合 → %d,應為 %d", fam, got, want)
		}
	}
	// ⚠ 系別 2 / 5 回 0 是**讀到的缺口**,不是預設值:
	// 原版那兩個系別在第二回合起沒有分支(docs/re/170 §4)。
}

// 第一回合挑到群體傷害要重擲,而且**每個系別最多只有一招**是群體傷害
// (SPELLS.DAT 欄3 = 1 恰好只有那三招,docs/re/171 §2)。
func TestRoundOneRerollsGroupDamage(t *testing.T) {
	for fam, storm := range StormSpell {
		hits := 0
		for roll := 1; roll <= SpellFamilyAttacks[fam]; roll++ {
			sp, re := MonsterSpell(fam, 1, roll)
			if re {
				hits++
				if sp != storm {
					t.Errorf("系別 %d 重擲的是法術 %d,應為 %d", fam, sp, storm)
				}
			}
		}
		if hits != 1 {
			t.Errorf("系別 %d 有 %d 招要重擲,應為 1", fam, hits)
		}
	}
	// 沒有群體傷害的系別(2 / 5)一次都不重擲
	for _, fam := range []int{2, 5} {
		for roll := 1; roll <= SpellFamilyAttacks[fam]; roll++ {
			if _, re := MonsterSpell(fam, 1, roll); re {
				t.Errorf("系別 %d 不該有要重擲的招", fam)
			}
		}
	}
}

// ── 施法時機(docs/re/186 §3)────────────────────────────────────────────

func TestMonsterCastsNeedsSPAboveOne(t *testing.T) {
	// ⚠ 門檻是 **> 1**,不是 ≥ 1 —— 法力剛好 1 的怪物不施法。
	for _, sp := range []int{0, 1} {
		if MonsterCasts(sp, MonsterForcedCastRound, 0) {
			t.Errorf("法力 %d 不該施法(門檻是 > 1)", sp)
		}
	}
	if !MonsterCasts(2, MonsterForcedCastRound, 1) {
		t.Error("法力 2、第二回合應該施法")
	}
}

func TestMonsterCastsForcedOnRoundTwo(t *testing.T) {
	// 第二回合**必定**施法:擲骰再差也一樣(這是 re/170「第二回合起固定放
	// 暴風系」的來源)。
	if !MonsterCasts(10, 2, 0.99) {
		t.Error("第二回合應該不看擲骰就施法")
	}
	// ⚠ 第三回合又回到擲骰 —— 原版比的是 `== 2`,不是 `>= 2`。
	if MonsterCasts(10, 3, 0.99) {
		t.Error("第三回合擲骰沒過就不該施法")
	}
}

func TestMonsterCastsRollThreshold(t *testing.T) {
	if !MonsterCasts(10, 1, MonsterCastChance) {
		t.Errorf("擲骰剛好等於 %v 應該過(原版是 ≤)", MonsterCastChance)
	}
	if MonsterCasts(10, 1, MonsterCastChance+0.01) {
		t.Error("擲骰超過門檻不該施法")
	}
}
