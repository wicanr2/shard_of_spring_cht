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

// ⚠ **補給品沒有上限**。原版夾在 `ds:6F10`,而那個變數沒有編譯期初值
// (docs/re/177 §6),靜態讀不到。⛔ 不要自己編一個上限 ——
// 編一個就等於發明了一條原版沒有的規則,而玩家會撞到它。
// 要解就是在原版裡把補給品堆到上限看它停在哪。

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
func IsIdentified(c original.Character, slot int) bool {
	return slot >= 0 && slot < len(c.Identified) && c.Identified[slot] == '1'
}

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
