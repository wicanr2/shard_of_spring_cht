package town

import (
	"shardofspring/internal/combat"
	"shardofspring/internal/original"
)

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

// docs/spec/19-module-text.md(F1):字面照 translations/module-text/CAMP.tsv。
// ⚠ SkillNoSkill 這裡用的是 H)unt 情境的原文(CAMP:130「You don't have that
// skill.」);I)dentify 情境原文不同(CAMP:61「You are not trained in that
// lore!」),那句在呼叫端(town_scene.go 的 identify())另外覆寫,不在這裡合併。
func (g SkillGate) String() string {
	switch g {
	case SkillNoSkill:
		return "你沒有那項技能。"
	case SkillNotWizard:
		return "這位角色不是巫師。"
	case SkillSpent:
		return "你今天已經用過那項技能了。"
	case SkillDisabled:
		// CAMP:131「That character is incapacitated.」
		// ⚠ TOWN:39 是同一句但結尾是 `!` —— 那是**城鎮**(治療所/訓練所)的閘門,
		// 引擎沒有那一道,所以這裡只照 CAMP 那一份。
		return "這位角色行動不能。"
	case SkillIndoors:
		return "你在室內!"
	}
	return ""
}

// 技能旗標的編號(1–10),與 CAMP.EXE 自己印的技能表一致(docs/re/166 §2/§3)。
const (
	SkillHunting    = 9 // 戰士表:位移 50
	SkillWeaponLore = 6 // 巫師表:位移 47
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
// HasSkill 是 hasSkill 的匯出版本 —— 引擎層也要判技能旗標
// (docs/re/229:夜視)。⚠ **旗標本身沒有語意**,同一格在兩張技能表
// 是不同技能,呼叫端一定要連職業一起判。
func HasSkill(c original.Character, n int) bool { return hasSkill(c, n) }

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

// HuntFaces / HuntOffset 是打獵收穫的兩個常數,**都是讀出來的**
// (docs/re/177 §4):
//
//	16  `INT 3F:AD, 04` —— 那個 API 把內嵌位元組加到浮點累加器的指數,
//	    所以 04 是「乘 2⁴」,不是「加 4」
//	−6  DGROUP `ds:731E` 的初值(MBF `00 00 c0 83`)
const (
	HuntFaces  = 16
	HuntOffset = -6
)

// HuntYield 回傳一次打獵的收穫份數;**0 就是失敗**。
//
//	原版:ds:731C = INT(RND × 16) + (−6);若 < 0 夾成 0
//
// ⚠ **只夾負數,`0` 不夾** —— 收穫 0 走 `The hunt was not successful.`,
// 不是「成功但拿 0 份」。所以失敗率是 7/16 而不是 6/16。
//
// r.Roll(n) 回 1…n,而原版的 `INT(RND × 16)` 是 0…15,所以要減 1。
//
// ⚠ 28 個實跑樣本回檢過(docs/re/177 §4.1,χ² ≈ 9.3 / 9 df),
// 但那是**回檢不是推導** —— 兩個常數都來自檔案。
func HuntYield(r combat.Rand) int {
	n := r.Roll(HuntFaces) - 1 + HuntOffset
	if n < 0 {
		return 0
	}
	return n
}

// ProvisionCap 是打獵之後補給品的上限(docs/re/218 §1)。
//
// `CAMP 0x10306` 執行期寫 `ds:6F10 = 0FFh`,`0x1134D` 用它夾住打獵的結算:
//
//	補給品 ← min(補給品 + 收穫, 255)
//
// ⚠ **只夾打獵這一條路徑。** `TOWN` 一次都沒有碰 `ds:6F10`
// (掃描分母 4,419 條指令),所以在酒館買補給品沒有經過這個夾子 ——
// 照讀到的做,不要推廣到買賣。
//
// ⚠ 255 **不是**欄位寬度的自然上限:`GROUPS.DAT` 位移 23 是 2-byte
// (docs/formats/02),放得下 32767。這是遊戲自己訂的數字。
const ProvisionCap = 255

// CapProvisions 夾住打獵之後的補給品。
func CapProvisions(n int) int {
	if n > ProvisionCap {
		return ProvisionCap
	}
	return n
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

// ── 撿道具(docs/re/168)────────────────────────────────────────────

// SetIdentified 設定第 slot 格的「已辨識」旗標(CHARS.DAT 位移 74 + slot)。
//
// ⚠ **原版有四個寫入端,其中一個差一格**:撿道具那一個寫 `75 + slot`
// (docs/re/168 §3)。本引擎一律寫 `74 + slot` —— **這一條刻意不照抄**,
// 因為它寫壞的是另一個欄位(位移 84 = 狀態效果強度),
// 照抄會讓我們產出的存檔把破壞帶回原版(docs/re/168 §5)。
func SetIdentified(c *original.Character, slot int, v bool) {
	if slot < 0 || slot >= PackSlots {
		return
	}
	f := []byte(c.Identified)
	for len(f) < PackSlots {
		f = append(f, '0')
	}
	if v {
		f[slot] = '1'
	} else {
		f[slot] = '0'
	}
	c.Identified = string(f[:PackSlots])
}

// IsIdentified 回傳第 slot 格的道具辨識過沒有。
// 規則本體在 original —— 戰鬥訊息也要問同一件事(docs/re/192 §4),
// 而 combat 不能倒過來 import town。
func IsIdentified(c original.Character, slot int) bool { return c.IsIdentified(slot) }

// PickUp 把一件道具塞進隊伍的背包,回傳拿到的人與格號;拿不下回 (-1, -1)。
//
// ⚠ **由隊尾往隊首找**(原版的外層迴圈是遞減,docs/re/168 §1)——
// 與 T)rade / E)quip 的方向相反,而這件事在畫面上分得出來:
// 東西會先進最後一個隊員的背包。
//
// ⚠ 撿到的東西標成**未辨識**,所以要花 I)dentify 才知道是什麼。
func PickUp(party []original.Character, item int) (who, slot int) {
	for i := len(party) - 1; i >= 0; i-- {
		for s := 0; s < PackSlots; s++ {
			if party[i].Pack[s] != original.NotEquipped {
				continue
			}
			party[i].Pack[s] = item
			SetIdentified(&party[i], s, false)
			return i, s
		}
	}
	return -1, -1
}

// ── 鑑定的成功率(docs/re/189)────────────────────────────────────────────

// IdentifyFactor 是鑑定成功門檻的乘數:`ds:72D2` 的 DGROUP 初值 = MBF **4.5**。
const IdentifyFactor = 4.5

// IdentifyThreshold 是鑑定的成功門檻(d100 ≤ 門檻 → 成功)。
//
//	門檻 = 智能 × 4.5
//
// ⚠ **與道具本身無關** —— 算式裡沒有道具的任何欄位(docs/re/189)。
// 智能 20 約九成、智能 10 約四成五。
func IdentifyThreshold(intellect int) float64 {
	return float64(intellect) * IdentifyFactor
}

// IdentifySucceeds 擲一次 d100 判鑑定成不成功。
//
// ⚠ 擲骰是 `INT(RND × 100) + 1`(配了 `INT 3D:03` 截尾),值域 1…100 ——
// 與命中擲骰的 `round(RND×100+1)` 不是同一個成語(docs/re/185)。
func IdentifySucceeds(c original.Character, r Roller) bool {
	return float64(r.Roll(100)) <= IdentifyThreshold(c.Int)
}
