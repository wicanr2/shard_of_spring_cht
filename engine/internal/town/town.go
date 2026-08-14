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
// 未解的價格與恢復量。docs/spec/11 §3、§4。
// ---------------------------------------------------------------------------

// TownInnPrice 是旅店每晚的價格。**來源未解** —— 沒讀到這個數字從哪來。
// ⛔ 不要因為「看起來合理」就換一個值。
const TownInnPrice = 10

// TownFoodPrice 是每份食糧的價格。**來源未解**,同上。
const TownFoodPrice = 5

// CampRestHeal 是營地休息一次回復的生命值。**恢復量未解**。
const CampRestHeal = 1

// Unresolved 是要顯示在畫面上的未解項(docs/spec/11 §3、§4)。
var Unresolved = []string{
	"旅店房價與食糧單價的來源未解",
	"營地休息的恢復量未解",
	"賣出／訓練／治療／傳聞的規則未解,未列入指令",
}

// Rest 讓全隊休息一次。回復量是未解的常數。
func Rest(party []original.Character) []original.Character {
	for i := range party {
		if party[i].HP <= 0 {
			continue // 倒下的不會自己恢復
		}
		party[i].HP += CampRestHeal
		if party[i].HP > party[i].MaxHP {
			party[i].HP = party[i].MaxHP
		}
		party[i].SP += CampRestHeal
		if party[i].SP > party[i].MaxSP {
			party[i].SP = party[i].MaxSP
		}
	}
	return party
}
