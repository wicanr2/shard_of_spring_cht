package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/original"
	"shardofspring/internal/town"
	"shardofspring/internal/ui"
)

// 商店賣出的介面。docs/spec/14 §13.7。
//
// ⛔ **原版沒有這個功能**(docs/re/142 §4)。這一整檔是重製版的增補,
// 所以畫面上要**明說**它不是原版的東西 —— 玩家在商店看到「賣出」
// 不會知道那是後加的,就像聽到背景音樂不會知道那不是 1986 年的曲子。
//
// 流程是**買的鏡像**:原版買東西是「選品項 → 問交給誰」,
// 賣就是「選誰 → 選他背包裡哪一件」。

// 賣出畫面的字面。⚠ 這些**沒有原版對應**,是自己寫的中性說明句 ——
// 不模仿原版語氣(docs/spec/14 §13.7)。
const (
	sellBanner   = "賣出（重製版增補，原版沒有這個功能）"
	sellHint     = "F2）賣出東西"
	sellNothing  = "他身上沒有這間店收的東西。"
	sellPackHead = "　背包："
)

func sellPickWho(n int) string {
	return fmt.Sprintf("%s：賣給誰的東西？選角色 #（1–%d，(ESC離開)）", sellBanner, n)
}

// sellKey 處理賣出模式下的按鍵。
func (g *Game) sellKey(k ebiten.Key) {
	ts := g.town
	if k == ebiten.KeyEscape {
		// ⚠ 兩段各自退一層 —— 選好人之後按 ESC 應該退回選人,
		// 而不是一路退出賣出模式。與施法游標的 ESC 同一個道理。
		if ts.sellWho >= 0 {
			ts.sellWho, ts.msg = -1, sellPickWho(len(g.members))
			return
		}
		ts.selling, ts.msg = false, ""
		return
	}
	if ts.sellWho < 0 {
		if i := int(k - ebiten.Key1); i >= 0 && i < len(g.members) {
			ts.sellWho = i
			ts.msg = g.sellPrompt(i)
		}
		return
	}
	if i := int(k - ebiten.KeyA); i >= 0 && i < town.PackSlots {
		g.sellItem(ts.sellWho, i)
	}
}

// sellPrompt 是選好人之後那一句。
func (g *Game) sellPrompt(who int) string {
	if len(g.sellable(who)) == 0 {
		return sellNothing + "　(ESC 換人)"
	}
	return fmt.Sprintf("%s：選一格賣出（A–%c，(ESC離開)）",
		g.members[who].Name, 'A'+town.PackSlots-1)
}

// sellLine 是背包某一格在賣出清單上的那一行;第二個回傳值 = 賣得掉嗎。
func (g *Game) sellLine(who, slot int) (string, bool) {
	c := g.members[who]
	item := c.Pack[slot]
	if item == original.NotEquipped {
		return "", false
	}
	name := g.lootDisplayName(item)
	switch {
	case c.Weapon == slot || c.Armor == slot:
		return fmt.Sprintf("%c) %s（裝備中）", 'A'+slot, ui.PadTo(name, 20)), false
	case !town.ShopBuys(g.town.shop, item):
		return fmt.Sprintf("%c) %s（這間店不收）", 'A'+slot, ui.PadTo(name, 20)), false
	}
	price := g.sellPriceOf(item)
	return fmt.Sprintf("%c) %s　%d 金幣", 'A'+slot, ui.PadTo(name, 20), price), true
}

// sellable 回傳這位角色身上這間店收得下的格號。
func (g *Game) sellable(who int) []int {
	if who < 0 || who >= len(g.members) {
		return nil
	}
	var out []int
	for slot := 0; slot < town.PackSlots; slot++ {
		if _, ok := g.sellLine(who, slot); ok {
			out = append(out, slot)
		}
	}
	return out
}

// sellPriceOf 是這間店收購某個編號的價格。
//
// ⚠ 查不到道具定義時回 0 —— **不要猜一個基準價**。0 金幣的賣出
// 玩家看得出不對勁,而一個編出來的價格看不出來。
func (g *Game) sellPriceOf(item int) int {
	for _, it := range g.itemList {
		if it.Index == item {
			return town.SellPrice(it.BasePrice, g.town.shop.PriceMult)
		}
	}
	return 0
}

// sellItem 賣掉第 who 位成員背包第 slot 格的東西。
func (g *Game) sellItem(who, slot int) {
	ts := g.town
	if who < 0 || who >= len(g.members) {
		return
	}
	r, item := town.Sell(&g.group.Gold, &g.members[who], ts.shop, slot)
	if r != town.SellOK {
		ts.msg = r.String()
		return
	}
	name := g.lootDisplayName(item)
	price := g.sellPriceOf(item)
	gold := g.group.Gold
	town.TakeSale(&gold, &g.members[who], slot, price)
	g.group.Gold = gold
	g.syncMember(g.members[who]) // 背包改了要寫回名冊,否則存檔會蓋掉
	ts.msg = fmt.Sprintf("賣掉 %s，得到 %d 金幣。", name, price)
}
