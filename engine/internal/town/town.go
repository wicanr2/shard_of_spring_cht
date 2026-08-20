// Package town 是城鎮、商店與營地的規則。
//
// 規則出自 docs/spec/11-town-camp-roster.md;每一條在下面註明章節。
package town

import (
	"math"

	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// Price 回傳某件道具在某間商店的售價。docs/spec/11 §3。
//
//	售價 = INT(基準價 × 該店倍率)
//
// ⚠ **先乘再取整**。倍率是 MBF 單精度,`1.1` 存進去是 1.100000023841858
// (docs/re/126 §3)—— 那個截斷誤差正是當初裁決這個欄位的證據,
// 所以計算順序不能改:先取整再乘會把誤差抹掉,而抹掉之後**價格看起來還是合理的**。
func Price(basePrice int, mult float64) int {
	if mult <= 0 {
		mult = 1 // 倍率 0 = 資料缺漏,不要讓東西變免費
	}
	return int(math.Floor(float64(basePrice) * mult))
}

// PackSlots 是背包格數。**10 格**(docs/re/144 §3),不是 15。
const PackSlots = original.PackSlots

// EmptyPackSlot 回傳第一個空的背包格號;滿了回 -1。
//
// ⚠ **空格的哨兵是 99**,與裝備欄的「未裝備」同一個值(docs/re/144 §3)。
// 先前這裡找的是值 `0` 的格子 —— 而每一格都是 99,所以它永遠找不到空位,
// **買東西一律回「背包已滿」**。那句話完全合理,所以那個 bug 不會被當成 bug。
func EmptyPackSlot(c original.Character) int {
	for i := 0; i < PackSlots; i++ {
		if c.Pack[i] == original.NotEquipped {
			return i
		}
	}
	return -1
}

// PackEmpty 回傳背包某一格是不是空的。
func PackEmpty(c original.Character, slot int) bool {
	return slot < 0 || slot >= PackSlots || c.Pack[slot] == original.NotEquipped
}

// BuyResult 說明一次購買的結果。
type BuyResult int

const (
	BuyOK BuyResult = iota
	BuyNoGold
	BuyPackFull
)

// docs/spec/19-module-text.md(F1):字面照 translations/module-text/TOWN.tsv——
// BuyNoGold 是 TOWN:76+77(「You don't have」+「enough gold!」),
// BuyPackFull 是 TOWN:19(「No more room!」)。
func (r BuyResult) String() string {
	switch r {
	case BuyNoGold:
		return "你沒有足夠的金幣!"
	case BuyPackFull:
		return "沒有空間了!"
	}
	return ""
}

// Buy 讓一位角色買一件道具。成功時扣金幣、放進背包第一個空格。
//
// ⚠ 金幣是 float64(MBF 單精度,docs/re/134),**不要在這裡轉成 int** ——
// 轉了之後大額金幣會溢位,而溢位後的值看起來仍然像個金額。
func Buy(gold *float64, c *original.Character, itemIndex, price int) BuyResult {
	if float64(price) > *gold {
		return BuyNoGold
	}
	slot := EmptyPackSlot(*c)
	if slot < 0 {
		return BuyPackFull
	}
	*gold -= float64(price)
	if *gold < 0 {
		*gold = 0 // 不讓金幣變負(docs/spec/11 §8 驗收 3)
	}
	c.Pack[slot] = itemIndex
	return BuyOK
}

// Equip 把背包第 slot 格的東西裝上。docs/spec/11 §4。
//
// ⚠ 裝備欄存的是**背包格號**,不是物品編號(docs/formats/01 位移 34/36)。
// 兩者都是小整數,寫錯不會報錯 —— 只會讓角色拿著背包裡另一件東西打人。
func Equip(c *original.Character, slot int, armor bool) bool {
	if slot != original.NotEquipped && (slot < 0 || slot >= PackSlots) {
		return false
	}
	if armor {
		c.Armor = slot
	} else {
		c.Weapon = slot
	}
	return true
}

// weaponSkill 是「裝這件武器要哪一項技能」的查表,索引就是**道具編號**。
//
// 表值出自 `CAMP.EXE` 的 `DATA` 敘述原文 `0,1,0,2,2,0,1,0,2,0,1,0`
// (docs/re/196 §1.1),原版讀進 `ds:6822 + 2i` 之後 `+ 42` 當記錄位移,
// 而技能 n 的位移是 `41 + n` —— 所以表值 = **技能編號 − 1**。
//
// ⚠ **編號 0(匕首)不在檢查範圍**:原版的閘門是 `編號 > 0`(§1),
// 所以誰都裝得上,巫師也是。留一格 0 只是為了索引對齊。
var weaponSkill = [12]int{0, 1, 0, 2, 2, 0, 1, 0, 2, 0, 1, 0}

// WeaponSkillOK 回傳這個角色能不能裝上這件武器。
//
//	巫師 → 一律不行(匕首除外)
//	戰士 → 要有對應的武器技能(劍 / 斧 / 錘)
//
// ⚠ 第二個回傳值是**這件東西要不要檢查**:防具與編號 0 的匕首都回 false,
// 呼叫端不該對它們印「沒有技能!」。
func WeaponSkillOK(c original.Character, itemIndex int) (ok, checked bool) {
	if itemIndex <= 0 || itemIndex >= len(weaponSkill) {
		return true, false // 匕首(0)、防具與其他編號都不走這一支
	}
	if rules.Class(c.Class) == rules.ClassWizard {
		return false, true
	}
	n := weaponSkill[itemIndex] + 1 // 技能編號(1 起算)
	return skillFlag(c, n) == '1', true
}

// skillFlag 取第 n 項技能的旗標字元(n 從 1 起算)。
func skillFlag(c original.Character, n int) byte {
	if n < 1 || n > len(c.Skills) {
		return '0'
	}
	return c.Skills[n-1]
}

// ---------------------------------------------------------------------------
// 休息與睡覺。恢復量出自手冊 p.37 / p.38(docs/re/140 §6);
// **價格仍然未解**,兩者分開看。
// ---------------------------------------------------------------------------

// 旅店睡一晚的恢復量(手冊 p.37:「每天可以恢復 2 點的生命和 10 點的法力」)。
const (
	InnHealHP = 2
	InnHealSP = 10
)

// 營地睡覺的恢復量與代價。**算式讀出來的**(docs/re/211)——
// 先前這一組是照手冊 p.38 湊的,四個數字裡有兩個不對。
const (
	// CampSleepHP / CampSleepSP:一次睡覺回 1 HP、5 SP。
	// 原版由參數挑「1/5」或「2/10」兩檔,而營地的呼叫端固定走前者
	// (`CAMP 0x10ED7` 傳 1)—— 後者八成是旅店那一支在用。
	CampSleepHP = 1
	CampSleepSP = 5
	// CampSleepFood 是**整晚**耗掉的食糧,不是每人一份
	// (`CAMP 0x10F00`:位移 23 減一次,在逐人迴圈**外面**)。
	CampSleepFood = 1
	// CampPoisonLoss 是中毒的人一晚扣的血:`缺糧 + 2`。
	// ⚠ **中毒那一支不回血** —— 原版是 if/else,不是「先回再扣」。
	CampPoisonLoss = 2
	// CampSleepHours 是睡覺推進的時鐘格數。原版直接把「時」設成 27
	// (`CAMP 0x10F46`),讓四級進位把它捲成隔天的 4 點 —— 也就是
	// **睡到天亮**,不是睡固定的小時數。
	CampSleepToHour = 27
)

// 城鎮服務的**基準價**。docs/re/142:在倍率恰好 1.0 的 Green Hamlet 讀到,
// 所以畫面上的數字就是基準價,不必反推(反推會踩到 MBF 的截斷誤差)。
//
// 實際收費一律是 `Price(基準價, 該店倍率)`。
const (
	TownInnPrice   = 25 // 住宿每晚
	TownFoodPrice  = 2  // 食糧每份 —— ⚠ 在**酒館**買,不在旅店
	HealPerHP      = 2  // 治療:每點生命。⚠ 單位是推的(見 docs/re/142 §結論)
	UnpoisonPrice  = 40 // 解毒,定價
	UnbindPerLv    = 94 // 解除束縛,依**角色等級**
	ResurrectPerLv = 100
)

// Unresolved 是要顯示在畫面上的未解項(docs/spec/11 §3、§4)。
//
// ⚠ **解掉的要從這裡刪掉。** 這張表印在營地畫面上,而一條過期的
// 「⚠ 某某未解」比沒有這張表更糟 —— 玩家會以為那個指令按了沒用,
// 於是不去按(2026-08-18 的 QA 對照抓到三條,docs/spec/14 §12-A)。
//
// 已經刪掉的:角色創造的骰法(re/156 + 178)、每場戰鬥的經驗(re/150)、
// 打獵(re/177)/ 鑑定(re/189)/ 傳遞 / 調整隊形(re/203)—— 全部解了也實作了。
// ⚠ **未解項要標在它所屬的畫面上。** 這兩條都是商店的事,卻曾經只畫在
// 營地 —— 在營地看到商店的註記,買家在商店反而什麼都看不到。
//
// ⚠ 這張表**只放「原版有而我們沒解」**。原版沒有的東西不屬於這裡:
// 「原版沒有賣出功能」曾經在這張表上,而本專案後來**加了**賣出
// (docs/spec/14 §13.7)—— 那是增補,標示走 town_sell.go 的 sellHint,
// 與「未解」是兩件事,混在一起讀起來都像藉口。
var ShopUnresolved = []string{
	"PERSUASIVENESS 技能怎麼降價未解,價格未套用它",
}

// CampUnresolved 是營地畫面的未解項。
var CampUnresolved = []string{
	"原版判斷「在野外」用 `ds:3534 ≥ 99`,那個變數的來源未解 ——" +
		"引擎改用「營地開在哪裡」(世界地圖 = 野外)",
}

// 這裡曾經有一個「休息一會兒」的自訂動作,綁在 `R` 上。
// 原版營地選單的 `R` 是 **R)eorder**(調整隊形順序,docs/re/150 §5.2),
// 而自訂動作佔著那個字母,會讓「原版按 R 會怎樣」永遠測不出來。
// **自己加的東西不要佔原版按鍵。**

// InnSleep 是旅店睡一晚。手冊 p.37:回 2 HP、10 SP,並且供餐(不耗食糧)。
func InnSleep(party []original.Character) []original.Character {
	return heal(party, InnHealHP, InnHealSP)
}

// CampSleep 是營地睡覺。算式出自 docs/re/211(`CAMP 0x12F6C` 的逐人迴圈)。
//
//	缺糧 = (睡前的食糧 ≤ 0)          ← **整隊共用一個旗標**,不是逐人扣
//	法力 += 5                        ← 中毒與否都回
//	中毒   → 生命 −= 缺糧 + 2        ← ⚠ **不回血**,是 if/else 不是「先回再扣」
//	沒中毒 → 生命 += 1 − 缺糧
//	生命 < 1 → 生命 = 0、**法力也歸零**
//	否則各自夾在最大值
//
// 回傳更新後的隊伍與剩下的食糧(整晚 1 份)。
//
// ⚠ 先前這裡照手冊 p.38 寫成「每人耗 1 份、吃不到的人扣 1 血、中毒的人
// 每小時扣 1 血 × 8 小時」——**三項都不對**:食糧是整晚一份、缺糧只是
// 抵銷回復、中毒一晚固定扣 2。手冊那一頁講的是感覺,不是算式。
func CampSleep(party []original.Character, provisions int) ([]original.Character, int) {
	starving := 0
	if provisions <= 0 {
		starving = 1
	}
	for i := range party {
		if !party[i].Occupied() {
			continue
		}
		party[i].SP += CampSleepSP
		if party[i].StatusName() == "中毒" {
			party[i].HP -= starving + CampPoisonLoss
		} else {
			party[i].HP += CampSleepHP - starving
		}
		if party[i].HP < 1 {
			party[i].HP, party[i].SP = 0, 0
			continue
		}
		clampVitals(&party[i])
	}
	if provisions >= CampSleepFood {
		provisions -= CampSleepFood
	}
	return party, provisions
}

func heal(party []original.Character, hp, sp int) []original.Character {
	for i := range party {
		if party[i].HP <= 0 {
			continue // 倒下的不會自己恢復
		}
		party[i].HP += hp
		party[i].SP += sp
		clampVitals(&party[i])
	}
	return party
}

func clampVitals(c *original.Character) {
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	if c.HP < 0 {
		c.HP = 0
	}
	if c.SP > c.MaxSP {
		c.SP = c.MaxSP
	}
	if c.SP < 0 {
		c.SP = 0
	}
}
