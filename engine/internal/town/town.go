// Package town 是城鎮、商店與營地的規則。
//
// 規則出自 docs/spec/11-town-camp-roster.md;每一條在下面註明章節。
package town

import (
	"math"

	"shardofspring/internal/original"
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

// PackSlots 是背包格數(CHARS.DAT 位移 54 + 2i,i = 0…14)。
const PackSlots = 15

// EmptyPackSlot 回傳第一個空的背包格號;滿了回 -1。
//
// ⚠ 空格的表示是 **0**。裝備欄的「未裝備」哨兵是 99,兩者不同 ——
// 混用會讓「卸下裝備」變成「背包第 99 格」。
func EmptyPackSlot(c original.Character) int {
	for i := 0; i < PackSlots; i++ {
		if c.Pack[i] == 0 {
			return i
		}
	}
	return -1
}

// BuyResult 說明一次購買的結果。
type BuyResult int

const (
	BuyOK BuyResult = iota
	BuyNoGold
	BuyPackFull
)

func (r BuyResult) String() string {
	switch r {
	case BuyNoGold:
		return "金幣不足"
	case BuyPackFull:
		return "背包已滿"
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

// ---------------------------------------------------------------------------
// 休息與睡覺。恢復量出自手冊 p.37 / p.38(docs/re/140 §6);
// **價格仍然未解**,兩者分開看。
// ---------------------------------------------------------------------------

// 旅店睡一晚的恢復量(手冊 p.37:「每天可以恢復 2 點的生命和 10 點的法力」)。
const (
	InnHealHP = 2
	InnHealSP = 10
)

// 營地睡覺的恢復量與代價(手冊 p.38)。
const (
	CampSleepHP       = 1 // 回 1 HP
	CampSleepSP       = 5 // 回 5 SP
	CampSleepFood     = 1 // 每人耗 1 份食糧
	CampStarveDamage  = 1 // 沒食糧吃的人扣 1 HP
	CampPoisonPerHour = 1 // 中毒的人每小時扣 1 HP
	CampSleepHours    = 8 // 睡滿八小時
)

// TownInnPrice 是旅店每晚的價格。**來源未解** —— 沒讀到這個數字從哪來。
// ⛔ 不要因為「看起來合理」就換一個值。
const TownInnPrice = 10

// TownFoodPrice 是每份食糧的價格。**來源未解**,同上。
//
// ⚠ 食糧在**酒館**買,不在旅店(手冊 p.37,docs/re/140 §6)。
const TownFoodPrice = 5

// CampRestHeal 保留給「不睡覺、只休息一會兒」。原版沒有這個動作 ——
// 它是本引擎為了讓玩家能推進時鐘而加的,**恢復量因此是自訂的,不是未解的原版值**。
const CampRestHeal = 1

// Unresolved 是要顯示在畫面上的未解項(docs/spec/11 §3、§4)。
var Unresolved = []string{
	"旅店房價、食糧單價、治療費用的金額未解(恢復量已知)",
	"賣出價與角色創造的骰法未解,未列入指令",
	"經驗值存在存檔的哪個位移未解,升級用引擎自己的計數",
}

// Rest 讓全隊休息一次(引擎自訂的動作,見 CampRestHeal)。
func Rest(party []original.Character) []original.Character {
	return heal(party, CampRestHeal, CampRestHeal)
}

// InnSleep 是旅店睡一晚。手冊 p.37:回 2 HP、10 SP,並且供餐(不耗食糧)。
func InnSleep(party []original.Character) []original.Character {
	return heal(party, InnHealHP, InnHealSP)
}

// CampSleep 是營地睡覺。手冊 p.38。
//
// 回傳更新後的隊伍與剩下的食糧。每個人耗 1 份;輪到誰沒得吃,誰就**扣 1 HP** ——
// ⚠ 是「吃不到的那個人扣血」,不是「食糧不夠時全隊都扣」。
//
// ⚠ 中毒的人睡覺期間每小時扣 1 HP(手冊 p.38 明講「睡了八個小時後將會發生意外」),
// 所以中毒的人睡覺是**淨損失**,不要在介面上把睡覺說成一律有益。
func CampSleep(party []original.Character, provisions int) ([]original.Character, int) {
	for i := range party {
		if !party[i].Occupied() {
			continue
		}
		if provisions >= CampSleepFood {
			provisions -= CampSleepFood
			party[i].HP += CampSleepHP
			party[i].SP += CampSleepSP
		} else {
			party[i].HP -= CampStarveDamage
		}
		if party[i].StatusName() == "中毒" {
			party[i].HP -= CampPoisonPerHour * CampSleepHours
		}
		clampVitals(&party[i])
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
