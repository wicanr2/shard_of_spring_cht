package rules

import "testing"

// 第一回合:系別基底 + 擲骰,結果必須落在該系別的攻擊法術範圍內
// (docs/re/170 §2/§3)。落到隔壁系別就是基底或張數寫錯了。
func TestMonsterSpellStaysInsideItsFamily(t *testing.T) {
	// SPELLS.DAT 的五個攻擊系別:起始法術編號(1-based)
	first := map[int]int{1: 1, 2: 4, 3: 8, 4: 12, 5: 16}
	last := map[int]int{1: 3, 2: 7, 3: 11, 4: 15, 5: 18}
	for fam := 1; fam <= 5; fam++ {
		n, rolls := SpellFamilyAttacks[fam], true
		if _, fixed := SpellFamilyFixed[fam]; fixed {
			n, rolls = 1, false // 系別 3 不擲骰(docs/re/226 §1)
		}
		for roll := 1; roll <= n; roll++ {
			got, _ := MonsterSpell(fam, 1, roll)
			if got < first[fam] || got > last[fam] {
				t.Errorf("系別 %d 擲 %d → 法術 %d,超出 %d–%d",
					fam, roll, got, first[fam], last[fam])
			}
		}
		// 會擲骰的系別:最小的擲骰就是它的第一個法術
		if got, _ := MonsterSpell(fam, 1, 1); rolls && got != first[fam] {
			t.Errorf("系別 %d 擲 1 → %d,應為 %d", fam, got, first[fam])
		}
	}
}

// 系別 3 **不擲骰**:`CMBT 0x1563E` 是 `mov ds:9B14h, 2`,配基底 7 → 法術 9。
//
// ⚠ 先前當成「1…2 的擲骰」,那會讓它有一半機率在第一回合放法術 8
// (TEMPEST)—— 而 8 是第二回合才放的那一張。
func TestFamilyThreeIsFixed(t *testing.T) {
	if SpellFamilyFixed[3] != 2 {
		t.Fatalf("系別 3 的固定值是 %d,原版讀到的是常數 2", SpellFamilyFixed[3])
	}
	for roll := 1; roll <= 4; roll++ { // 擲什麼都一樣
		if sp, re := MonsterSpell(3, 1, roll); sp != 9 || re {
			t.Errorf("系別 3 擲 %d → 法術 %d(重擲 %v),應恆為 9", roll, sp, re)
		}
	}
}

// 第二回合起:系別 1 / 3 / 4 固定放暴風;**系別 2 / 5 落回隨機挑**。
//
// ⚠ 「沒有暴風分支」不等於「不施法」。原版 `0x155AA` 的 `jmp` 落在
// **`0x155B0`** —— 第一回合那個擲骰迴圈的入口(docs/re/231)。
// 先前這裡期望 0,而那讓系別 2 / 5 的怪從第二回合起再也不施法,
// **畫面上看不出來**:一隻不施法的怪就是走過來砍人。
func TestStormFromRoundTwo(t *testing.T) {
	for fam, want := range map[int]int{1: 3, 3: 8, 4: 12} {
		if got, _ := MonsterSpell(fam, 2, 1); got != want {
			t.Errorf("系別 %d 第二回合 → %d,應為暴風 %d", fam, got, want)
		}
	}
	// 系別 2 / 5:第二回合擲什麼,就與第一回合擲同一個數字得到的一樣。
	for _, fam := range []int{2, 5} {
		for roll := 1; roll <= SpellFamilyAttacks[fam]; roll++ {
			first, _ := MonsterSpell(fam, 1, roll)
			later, _ := MonsterSpell(fam, 2, roll)
			if later != first || later == 0 {
				t.Errorf("系別 %d 擲 %d:第一回合 %d、第二回合 %d —— "+
					"兩者應該相同且不為 0", fam, roll, first, later)
			}
		}
	}
}

// 第一回合挑到群體傷害要重擲(docs/re/171 §2)。
//
// ⚠ **只有系別 4 真的擲得到** —— 面數讀出來之後(docs/re/226 §1),
// 系別 1 的擲骰只到法術 2、系別 3 固定法術 9,兩者的暴風(3 / 8)
// 在第一回合**構造上就選不到**。重擲那一段仍然存在,但只對系別 4 生效。
func TestRoundOneRerollsGroupDamage(t *testing.T) {
	reroll := map[int]int{}
	for fam, storm := range StormSpell {
		n, ok := SpellFamilyAttacks[fam]
		if !ok {
			n = 1 // 固定的系別:擲什麼都一樣
		}
		for roll := 1; roll <= n; roll++ {
			sp, re := MonsterSpell(fam, 1, roll)
			if re {
				reroll[fam]++
				if sp != storm {
					t.Errorf("系別 %d 重擲的是法術 %d,應為 %d", fam, sp, storm)
				}
			}
		}
	}
	if reroll[4] != 1 {
		t.Errorf("系別 4 應該有 1 招要重擲(法術 12),得到 %d", reroll[4])
	}
	for _, fam := range []int{1, 3} {
		if reroll[fam] != 0 {
			t.Errorf("系別 %d 第一回合構造上就選不到暴風,不該有重擲", fam)
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
