package town

import "shardofspring/internal/original"

// 營地裡兩個每天一次的技能:H)unt 與 I)dentify。docs/re/166。
//
// 兩者共用 `CHARS.DAT` 位移 86 的「今天用過了」旗標,而且**閘門的順序**
// 與原版相同 —— 順序會改變玩家看到哪一句話。

// SkillGate 是技能用不成的原因。零值 = 可以用。
type SkillGate int

const (
	SkillOK        SkillGate = iota
	SkillNoSkill             // 'You don't have that skill.'
	SkillNotWizard           // 'That character is not a wizard.'
	SkillSpent               // 'You have used that skill today.'
	SkillDisabled            // 'That character is incapacitated.'
	SkillIndoors             // 'You're inside!'
)

func (g SkillGate) String() string {
	switch g {
	case SkillNoSkill:
		return "他沒有這項技能。"
	case SkillNotWizard:
		return "他不是法師。"
	case SkillSpent:
		return "他今天已經用過技能了。"
	case SkillDisabled:
		return "他現在動不了。"
	case SkillIndoors:
		return "你在室內!"
	}
	return ""
}

// 技能旗標的編號(1–10),與 CAMP.EXE 自己印的技能表一致(docs/re/166 §2/§3)。
const (
	SkillHunting    = 9 // 戰士表:位移 50
	SkillWeaponLore = 6 // 法師表:位移 47
	SkillPotionLore = 7 // 位移 48
	SkillItemLore   = 8 // 位移 49
)

// MaxActiveStatus 是還能用技能的最高狀態碼。
//
// 原版:服務呼叫操作碼 38 取狀態,`> 1` → `That character is incapacitated.`
// 所以正常(0)與中毒(1)可以用,束縛(2)以上不行(docs/re/166 §7)。
const MaxActiveStatus = 1

// hasSkill 讀第 n 個技能旗標(1-based)。
//
// ⚠ **同一格在兩張表裡是不同的技能** —— 職業由呼叫端先判。
func hasSkill(c original.Character, n int) bool {
	return n >= 1 && n <= len(c.Skills) && c.Skills[n-1] == '1'
}

// CanHunt 依原版的順序判 H)unt(docs/re/166 §2)。
//
// outdoors 由呼叫端給:原版是 `ds:3534 ≥ 99`,而那個變數的來源未解 ——
// 引擎用「不在迷宮也不在城鎮」。
func CanHunt(c original.Character, outdoors bool) SkillGate {
	if !outdoors {
		return SkillIndoors // ★ 這一關在選人之前
	}
	if c.Class != '1' || !hasSkill(c, SkillHunting) {
		return SkillNoSkill
	}
	if c.SkillUsed {
		return SkillSpent
	}
	if c.Status > MaxActiveStatus {
		return SkillDisabled
	}
	return SkillOK
}

// LoreFor 回傳辨識這個道具要哪一個 lore 技能;0 = 空格(原版直接回選單)。
//
// 分界是編號 `≤ 20` / `21–56` / 其餘,`99` = 空格(docs/re/166 §3)。
// 背包存的是 **0-based** 編號(docs/re/167 §3:商店的販售範圍最小值是 0,
// 而「藥水舖」賣 21–26 = ITEMS.DAT 第 22–27 列),所以:
//
//	0–20   ITEMS.DAT 第 1–21 列 = **全部 21 件武器與護甲**  → Weapon lore
//	21–56  第 22–57 列 = 藥水 + 任務道具                    → Potion lore
//	> 56   **沒有真實道具**                                  → Item lore
//
// ⚠ 第一段與手冊 p.39「WEAPON LORE 可以辨別武器和護甲」完全吻合,
// 但手冊另外兩句對不上:`Item lore` 那一段接不到任何真實道具,
// 對玩家而言是**一個永遠用不到的技能**(docs/re/167 §4)。**照程式走**,
// 因為反組譯是第 2 級證據、手冊是第 3 級。
func LoreFor(item int) int {
	switch {
	case item == original.NotEquipped:
		return 0
	case item <= 20:
		return SkillWeaponLore
	case item <= 56:
		return SkillPotionLore
	}
	return SkillItemLore
}

// CanIdentify 依原版的順序判 I)dentify(docs/re/166 §3)。
//
// ⚠ 順序與 CanHunt 不同:原版先判職業,再判「今天用過了」,最後判狀態。
func CanIdentify(c original.Character, item int) SkillGate {
	if c.Class != '2' {
		return SkillNotWizard
	}
	if c.SkillUsed {
		return SkillSpent
	}
	if c.Status > MaxActiveStatus {
		return SkillDisabled
	}
	if n := LoreFor(item); n != 0 && !hasSkill(c, n) {
		return SkillNoSkill // 'You are not trained in that lore!'
	}
	return SkillOK
}
