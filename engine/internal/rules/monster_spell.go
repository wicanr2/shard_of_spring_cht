package rules

// 怪物的選招。docs/re/170。
//
// ⚠ 這一份只管**放什麼**。「往哪走、打誰」仍未解
// (docs/re/158 §2),⛔ 不要拿這裡的東西去補那一半。

// SpellFamilyBase 是五個攻擊系別在 SPELLS.DAT 裡的基底(0-based 偏移),
// 索引是屬性 14(1–5)。把欄 2 攤開,五個系別剛好從法術 1/4/8/12/16 起。
var SpellFamilyBase = map[int]int{1: 0, 2: 3, 3: 7, 4: 11, 5: 15}

// SpellFamilyAttacks 是每個系別**只挑攻擊法術**的張數(docs/re/170 §3)。
//
// 系別 3 是讀到的常數 2(TEMPEST / STILL AIR,跳過兩個增益);
// 系別 2 / 5 走 ds:94BE = 3(docs/re/160 §1.1)。
//
// ⚠ 系別 1 與 4 的張數**沒有讀到** —— 它們走 `INT 3F:AD` 加一個內嵌位元組,
// 那條 API 未解。這裡填 3 與 4 是**由「只挑攻擊法術」推的**:
// 系別 1 的 1–3、系別 4 的 12–15 全是攻擊法術。
var SpellFamilyAttacks = map[int]int{1: 3, 2: 3, 3: 2, 4: 4, 5: 3}

// StormSpell 是第二回合起固定放的那一招,索引是屬性 14(docs/re/170 §4)。
//
// 三招正好是 FIRESTRM / WINDSTRM / HAILSTRM.BIN 三張**從不載入**的圖
// (docs/re/61)—— 那三張圖是為這三招畫的,DOS 版把呈現拿掉了。
//
// ⚠ 系別 2 與 5 **沒有**對應的分支,它們第二回合起做什麼未讀。
var StormSpell = map[int]int{
	1: 3,  // FIRE STORM
	3: 8,  // TEMPEST
	4: 12, // HAIL STORM
}

// GroupDamageClass 是「群體傷害」的效果類別(SPELLS.DAT 欄3 = 1)。
//
// 原版第一回合選完之後查 `ds:7340[法術編號]`,是 **1** 就重選 ——
// 而 `ds:7340` 就是欄3(docs/re/171 §1)。所以規則是
// **第一回合不放群體傷害,第二回合起只放它**。
//
// 欄3 = 1 的恰好只有 FIRE STORM / TEMPEST / HAIL STORM 三招,
// 也就是 StormSpell 那三個 —— 兩邊是同一件事的兩半。
const GroupDamageClass = 1

// MonsterSpell 回傳這一回合怪物要放的法術編號(**1-based**)。
//
// `reroll` 為真表示**這一擲要重來**(第一回合挑到群體傷害,docs/re/171 §2)。
// `spell` 為 0 表示這個系別在這一回合**沒有東西可放** ——
// ⚠ **兩者是不同的情況**,不要用同一個 0 表示:
// 「重擲」是原版的迴圈,「沒東西可放」是原版沒有那一條分支(系別 2 / 5)。
//
// roll 是 1…SpellFamilyAttacks[family] 的擲骰結果,由呼叫端給 ——
// 與 maze.PoolHeal 同一條原則:擲骰交給呼叫端,這裡只做規則。
func MonsterSpell(family, round, roll int) (spell int, reroll bool) {
	if round > 1 {
		return StormSpell[family], false // 系別 2 / 5 沒有分支 → 0
	}
	base, ok := SpellFamilyBase[family]
	if !ok {
		return 0, false
	}
	n := SpellFamilyAttacks[family]
	if roll < 1 {
		roll = 1
	}
	if roll > n {
		roll = n
	}
	got := base + roll
	// 第一回合挑到群體傷害(SPELLS.DAT 欄3 = 1)就重選。
	// 每個系別最多只有一招是群體傷害,就是 StormSpell 那一張。
	if got == StormSpell[family] {
		return got, true
	}
	return got, false
}
