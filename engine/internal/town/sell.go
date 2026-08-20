package town

import (
	"math"

	"shardofspring/internal/original"
)

// 商店賣出。**這一層不是原版的東西。**
//
// 原版的商店只有買,沒有任何賣出的字串或分支(docs/re/142 §4)。
// 這是專案負責人裁定的**重製版增補**(2026-08-20,docs/spec/14 §13.7),
// 與場景配樂(docs/spec/13 §7)同一個地位 —— 規矩因此也一樣:
// **要讓玩家知道這不是原版的東西**,不能讓它混進「還原」裡。
//
// 三條設計約束:
//
//  1. **收購價由原版的售價推出來**,不另立一張表 —— 售價已經含了
//     `TOWNDATA` 的價格倍率,所以物價高的城鎮賣得多一點,
//     與買東西的貴賤是同一個機制。
//  2. **每間店只收自己賣的範圍。** 那張範圍表本來就寫著每間店做哪一行生意
//     (docs/re/138 §1),忽略它等於自己發明一套規則。
//  3. **不發明原版沒有的細則** —— 沒有「未鑑定折價」、沒有殺價、
//     沒有隨機浮動。增補要小,越小越不會與原版的規則打架。

// SellRate 是收購價佔售價的比例。專案負責人指定 **65%**。
const SellRate = 0.65

// SellPrice 是這間店收購一件道具的價格。
//
// ⚠ **從 Price() 的結果再乘**,不是從 BasePrice 乘兩個係數 ——
// 原版的售價本身就是 `INT(基準價 × 倍率)`,先取整過。
// 兩種算法差一兩枚金幣,而**差在哪一邊看不出來**,
// 所以要定死「賣價是售價的六成五」這句話本身。
func SellPrice(basePrice int, mult float64) int {
	return int(math.Floor(float64(Price(basePrice, mult)) * SellRate))
}

// SellResult 說明一次賣出的結果。
type SellResult int

const (
	SellOK         SellResult = iota
	SellEmpty                 // 那一格沒有東西
	SellEquipped              // 裝備中,要先卸下
	SellNotStocked            // 這間店不做這一行的生意
)

// ⚠ 這幾句**沒有原版對應**(原版沒有這個功能),所以措辭是自己寫的。
// 寫成中性的說明句,不模仿原版的語氣 —— 玩家分不出來的話,
// 他會以為原版就有賣出,而 §13.7 要避免的正是這件事。
func (r SellResult) String() string {
	switch r {
	case SellEmpty:
		return "那一格沒有東西。"
	case SellEquipped:
		return "裝備中的東西不能賣,要先卸下。"
	case SellNotStocked:
		return "這間店不收這種東西。"
	}
	return ""
}

// ShopBuys 回答這間店收不收某個編號的道具。
//
// 只有有販售範圍的店會收(治療所 / 旅店 / 酒館 / 訓練所是服務),
// 而且**只收自己賣的那個範圍**。
//
// ⚠ 副作用是**任務道具賣不掉** —— 它們不在任何一間店的範圍內。
// 那是想要的性質,不是漏洞:賣掉拉利斯之門的鑰匙會讓遊戲卡死。
func ShopBuys(s original.Shop, itemIndex int) bool {
	return s.Kind == original.ShopGoods &&
		itemIndex >= s.First && itemIndex <= s.Last
}

// Sell 把一位角色背包第 slot 格的東西賣給這間店。
//
// ⚠ 金幣是 float64(MBF 單精度,docs/re/134),與 Buy 同一個理由:
// **不要在這裡轉成 int**。
func Sell(gold *float64, c *original.Character, s original.Shop, slot int) (SellResult, int) {
	if slot < 0 || slot >= PackSlots || c.Pack[slot] == original.NotEquipped {
		return SellEmpty, 0
	}
	// ⚠ 裝備欄存的是**背包格號**不是物品編號(docs/formats/01 位移 34/36)——
	// 拿物品編號來比會在「編號剛好等於格號」時誤判,而那不是罕見情況:
	// 兩者都是 0–56 的小整數。
	if c.Weapon == slot || c.Armor == slot {
		return SellEquipped, 0
	}
	if !ShopBuys(s, c.Pack[slot]) {
		return SellNotStocked, 0
	}
	return SellOK, c.Pack[slot]
}

// TakeSale 完成一次賣出:清掉背包那一格、金幣入帳。
//
// ⚠ 與 Sell 分開是因為**價格要由呼叫端算**(它才知道道具的基準價)。
// 合成一個的話,規則層就得認識道具表,而它現在不必。
func TakeSale(gold *float64, c *original.Character, slot, price int) {
	c.Pack[slot] = original.NotEquipped
	*gold += float64(price)
}
