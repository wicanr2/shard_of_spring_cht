package rules

// 怪物的選招。docs/re/170。
//
// ⚠ 這一份只管**放什麼**。「往哪走、打誰」仍未解
// (docs/re/158 §2),⛔ 不要拿這裡的東西去補那一半。

// SpellFamilyBase 是五個攻擊系別在 SPELLS.DAT 裡的基底(0-based 偏移),
// 索引是屬性 14(1–5)。把欄 2 攤開,五個系別剛好從法術 1/4/8/12/16 起。
var SpellFamilyBase = map[int]int{1: 0, 2: 3, 3: 7, 4: 11, 5: 15}

// SpellFamilyAttacks 是每個系別**擲骰的面數**(docs/re/226 §1,逐條讀出來)。
//
//	系別 1  INT(RND × 2) + 1      → 法術 1–2
//	系別 2  INT(RND × 3) + 1      → 法術 4–6      (ds:94BE = 3)
//	系別 4  INT(RND × 4) + 1      → 法術 12–15
//	系別 5  INT(RND × 3) + 1      → 法術 16–18
//
// 面數來自 `INT 3F:AD` 的內嵌位元組:那條 API 把**浮點指數加上**該值,
// 也就是 `FAC × 2^n` —— 內嵌 01 = ×2、02 = ×4(`BRUN30 0x1AC66` 的
// `lodsb` + `add es:1Dh, al`)。系別 2/5 不走它,走 `ds:94BE` 的乘法。
//
// ⚠ **系別 3 不在這張表裡** —— 它不擲骰,見 SpellFamilyFixed。
var SpellFamilyAttacks = map[int]int{1: 2, 2: 3, 4: 4, 5: 3}

// SpellFamilyFixed 是**不擲骰**的系別:直接寫死系別內的第幾張。
//
// 系別 3 在 `CMBT 0x1563E` 是 `mov ds:9B14h, 2` —— 常數 2,配基底 7
// 得到法術 **9**(STILL AIR)。⚠ 先前當成「1…2 的擲骰」,那會讓
// 系別 3 的怪物有一半機率放法術 8(TEMPEST),而 8 是第二回合才放的那一張。
var SpellFamilyFixed = map[int]int{3: 2}

// StormSpell 是第二回合起固定放的那一招,索引是屬性 14(docs/re/170 §4)。
//
// 三招正好是 FIRESTRM / WINDSTRM / HAILSTRM.BIN 三張**從不載入**的圖
// (docs/re/61)—— 那三張圖是為這三招畫的,DOS 版把呈現拿掉了。
//
// ⚠ 系別 2 與 5 **沒有**對應的分支 —— 它們第二回合起**落回隨機挑**
// (docs/re/231:`0x155AA` 的 `jmp` 落在第一回合那個迴圈的入口)。
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
//
// roll 是 1…SpellFamilyAttacks[family] 的擲骰結果,由呼叫端給 ——
// 與 maze.PoolHeal 同一條原則:擲骰交給呼叫端,這裡只做規則。
//
// ⚠ **系別 2 / 5 在第二回合起走的是與第一回合相同的隨機挑選**
// (docs/re/231):`CMBT 0x155AA` 那一條 `jmp` 落在 **`0x155B0`** ——
// 正是第一回合的擲骰迴圈,不是「不施法」。
// 先前這裡回 0,那些怪從第二回合起就再也不施法了,而**畫面上看不出來**:
// 一隻不施法的怪就是走過來砍人,看起來完全正常。
func MonsterSpell(family, round, roll int) (spell int, reroll bool) {
	// 第二回合起,系別 1 / 3 / 4 固定放自己那一招暴風;
	// 沒有那一招的系別**落回隨機挑**(不是不放)。
	if round > 1 {
		if storm, ok := StormSpell[family]; ok {
			return storm, false
		}
	}
	base, ok := SpellFamilyBase[family]
	if !ok {
		return 0, false
	}
	off, fixed := SpellFamilyFixed[family]
	if !fixed {
		off = roll
		n := SpellFamilyAttacks[family]
		if off < 1 {
			off = 1
		}
		if off > n {
			off = n
		}
	}
	got := base + off
	// 挑到群體傷害(`SPELLS.DAT` 欄3 = 1)就重選。
	// 每個系別最多只有一招是群體傷害,就是 StormSpell 那一張。
	//
	// ⚠ **這道過濾只在第一回合成立** —— 原版的閘門是
	// 「欄3 == 1 **且** 回合 == 1」(`CMBT 0x156DE`/`0x156ED`,docs/re/171 §2)。
	// 對系別 2 / 5 而言差別是零(它們沒有群體傷害的招),
	// 所以這一行擋的是**別的系別哪天落到這條路**時不要被無限重擲。
	if round == 1 && got == StormSpell[family] {
		return got, true
	}
	return got, false
}

// ── 施法時機(docs/re/186 §3)────────────────────────────────────────────

// MonsterCastChance 是「這一回合要不要施法」的擲骰門檻:**0.3**。
// `CMBT 0x12491` 的 `mov di, 941Eh` → `ds:941E` 的初值是 MBF 0.3
// (`9a 99 19 7f`),讀出來的。
const MonsterCastChance = 0.3

// MonsterCastMinSP 是施法的硬門檻:法力**大於** 1(`CMBT 0x12481` 的
// `cmp [屬性7], 1` 配 `jle`)。⚠ 是 > 1 不是 ≥ 1。
const MonsterCastMinSP = 1

// MonsterForcedCastRound 是必定施法的那一回合。原版寫死比較 `== 2`
// (`CMBT 0x1249F`),不是「第二回合起」——第三回合又回到擲骰。
const MonsterForcedCastRound = 2

// MonsterCasts 回答「這隻怪物這一回合施不施法」。
//
//	法力 > 1  且  ( 回合 == 2  或  擲骰 ≤ 0.3 )
//
// docs/re/186 §3 讀到的形狀:`(回合==2 OR 擲骰) AND 法力足夠`。
// ⚠ 法力是**硬門檻**,在 OR 的外面 —— 第二回合也不能無中生有。
//
// 這與 MonsterSpell 的分工:這裡決定「施不施」,那裡決定「施哪一招」。
func MonsterCasts(sp, round int, roll float64) bool {
	if sp <= MonsterCastMinSP {
		return false
	}
	return round == MonsterForcedCastRound || roll <= MonsterCastChance
}

// ── 投入點數(docs/re/226 §2)──────────────────────────────────────────

// MonsterInvestLevels 是怪物投入的級數:**兩級**。原版把單價 `shl dx, 1`
// 之後拿去比法力(`CMBT 0x15755`),所以是固定的兩倍,不是擲骰。
const MonsterInvestLevels = 2

// MonsterInvest 回傳怪物這一次施法要投入幾點。
//
//	法力 ≥ 單價 × 2 → 投入 單價 × 2
//	否則            → 投入 剩下的全部法力
//
// 原版的判斷寫成 `法力 > 單價×2 − 1`(`CMBT 0x15759` 的 `dec dx` 配 `jg`),
// 與 `≥` 等價 —— ⚠ 兩者在**整數**上才等價,不要改寫成浮點比較。
func MonsterInvest(sp, unitCost int) int {
	want := MonsterInvestLevels * unitCost
	if sp >= want {
		return want
	}
	if sp < 0 {
		return 0
	}
	return sp
}
