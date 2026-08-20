package town

import (
	"testing"

	"shardofspring/internal/original"
)

func sellShop() original.Shop {
	// 賣編號 0–6 的店(翠綠村「精選兵器」的範圍),倍率 1.1。
	return original.Shop{Kind: original.ShopGoods, First: 0, Last: 6, PriceMult: 1.1}
}

// 收購價 = **售價**的 65%,不是基準價的 65% × 倍率。
//
// ⚠ 兩種算法只差一兩枚金幣,而差在哪一邊玩家看不出來 ——
// 所以要釘住「賣價是這間店售價的六成五」這句話本身(docs/spec/14 §13.7)。
func TestSellPriceIsSixtyFivePercentOfTheShopPrice(t *testing.T) {
	for _, c := range []struct {
		base int
		mult float64
	}{
		{100, 1.0}, {100, 1.1}, {35, 0.9}, {1, 1.0}, {0, 1.0},
	} {
		shopPrice := Price(c.base, c.mult)
		want := int(float64(shopPrice) * SellRate)
		if got := SellPrice(c.base, c.mult); got != want {
			t.Errorf("基準價 %d 倍率 %g:售價 %d → 收購 %d,應為 %d",
				c.base, c.mult, shopPrice, got, want)
		}
	}
	// 比例本身:65%,不是 50% 也不是 70%。
	if SellRate != 0.65 {
		t.Errorf("收購比例 %g,專案負責人指定的是 0.65", SellRate)
	}
}

// 賣價一定不高於買價 —— 否則在同一間店買進賣出就能無限賺錢。
func TestSellNeverExceedsBuy(t *testing.T) {
	for base := 0; base <= 500; base++ {
		for _, mult := range []float64{0.9, 1.0, 1.1, 1.25} {
			if SellPrice(base, mult) >= Price(base, mult) && Price(base, mult) > 0 {
				t.Fatalf("基準價 %d 倍率 %g:收購 %d ≥ 售價 %d —— 買進賣出就能刷錢",
					base, mult, SellPrice(base, mult), Price(base, mult))
			}
		}
	}
}

// 只收自己賣的範圍;服務類建築一律不收。
func TestShopBuysOnlyItsOwnRange(t *testing.T) {
	s := sellShop()
	for _, c := range []struct {
		item int
		want bool
	}{{0, true}, {6, true}, {7, false}, {-1, false}, {56, false}} {
		if got := ShopBuys(s, c.item); got != c.want {
			t.Errorf("編號 %d → %v,應為 %v", c.item, got, c.want)
		}
	}
	for _, k := range []original.ShopKind{
		original.ShopHealer, original.ShopTavern, original.ShopInn, original.ShopTrainer,
	} {
		if ShopBuys(original.Shop{Kind: k, First: 0, Last: 6}, 3) {
			t.Errorf("%v 是服務不是商店,不該收東西", k)
		}
	}
}

// 裝備中的東西不能賣,而且判斷用的是**背包格號**不是物品編號。
//
// ⚠ 這條分得出兩種寫法:Weapon 存的是格號 1,而格 1 裡的物品編號是 5。
// 若誤用物品編號去比,格 1 會被判成沒裝備 → 賣掉玩家手上的劍。
func TestEquippedItemCannotBeSold(t *testing.T) {
	c := original.Character{}
	for i := range c.Pack {
		c.Pack[i] = original.NotEquipped
	}
	c.Pack[1] = 5
	c.Pack[5] = 3
	c.Weapon, c.Armor = 1, original.NotEquipped
	gold := 0.0
	if r, _ := Sell(&gold, &c, sellShop(), 1); r != SellEquipped {
		t.Errorf("格 1 裝備中,應回 SellEquipped,得 %v", r)
	}
	// 格 5 的物品編號是 3,而 Weapon == 1 —— 用編號比的話這裡會誤判成裝備中。
	if r, _ := Sell(&gold, &c, sellShop(), 5); r != SellOK {
		t.Errorf("格 5 沒有裝備,應可賣,得 %v", r)
	}
}

// 空格、店不收:兩條各自有自己的訊息,而且都不動金幣。
func TestSellRefusals(t *testing.T) {
	c := original.Character{Weapon: original.NotEquipped, Armor: original.NotEquipped}
	for i := range c.Pack {
		c.Pack[i] = original.NotEquipped
	}
	c.Pack[0] = 40 // 不在 0–6 的範圍內
	gold := 100.0
	if r, _ := Sell(&gold, &c, sellShop(), 3); r != SellEmpty {
		t.Errorf("空格應回 SellEmpty,得 %v", r)
	}
	if r, _ := Sell(&gold, &c, sellShop(), 0); r != SellNotStocked {
		t.Errorf("店不收應回 SellNotStocked,得 %v", r)
	}
	if gold != 100 {
		t.Errorf("賣不成不該動金幣,金幣變成 %g", gold)
	}
	// 每一種拒絕都要有話講 —— 空字串會讓玩家以為按鍵沒反應。
	for _, r := range []SellResult{SellEmpty, SellEquipped, SellNotStocked} {
		if r.String() == "" {
			t.Errorf("%v 沒有訊息", r)
		}
	}
}

// TakeSale 清格子 + 入帳,兩件事都要做。
func TestTakeSaleClearsSlotAndPaysGold(t *testing.T) {
	c := original.Character{}
	for i := range c.Pack {
		c.Pack[i] = original.NotEquipped
	}
	c.Pack[2] = 4
	gold := 10.0
	TakeSale(&gold, &c, 2, 33)
	if c.Pack[2] != original.NotEquipped {
		t.Errorf("賣掉之後那一格應該是空的,得 %d", c.Pack[2])
	}
	if gold != 43 {
		t.Errorf("金幣 %g,應為 43", gold)
	}
}
