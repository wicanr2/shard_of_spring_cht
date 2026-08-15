package main

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
	"shardofspring/internal/original"
	"shardofspring/internal/town"
	"shardofspring/internal/ui"
	"shardofspring/internal/world"
)

// 城鎮、商店與營地的畫面。docs/spec/11-town-camp-roster.md。
//
// ⚠ **清單放主視野**(61 欄),不放側欄(30 欄)——
// docs/spec/04 §5:那個「放不下」的未決項的前提是錯的。

type townMode int

const (
	townClosed    townMode = iota
	townBuildings          // 建築清單
	townShop               // 商店品項
	townCamp               // 營地
	townInn                // 旅店(位移 34 = −3)
	townHealer             // 治療所(−1)
	townTavern             // 酒館(−2)
	townTrainer            // 訓練所(−4)
)

type townState struct {
	mode  townMode
	name  string          // 城鎮名
	shops []original.Shop // 這個城鎮的商店
	shop  original.Shop   // 進去的那一間
	page  int             // 商品分頁(原版有 `+ for next page`,手冊 p.31)
	msg   string
}

// shopPageSize 是一頁的商品數。主視野高 612、行高 26 → 扣掉標題約 20 列。
const shopPageSize = 20

// enterTown 從世界地圖進城。回 false 表示這一格沒有城鎮。
//
// ⚠ 世界地圖只知道「這裡有城鎮」(值 30/31/32),**不知道是哪一個** ——
// 座標與 TOWNDATA 的對應未解。這裡按城鎮座標的排序取第 n 個,
// 是**佔位**,畫面上會標出來。
func (g *Game) enterTown(x, y int) bool {
	names := original.Towns(g.shops)
	if len(names) == 0 {
		return false
	}
	idx := g.townIndexAt(x, y)
	if idx < 0 || idx >= len(names) {
		return false
	}
	name := names[idx]
	ts := &townState{mode: townBuildings, name: name}
	for _, s := range g.shops {
		if s.Town == name {
			ts.shops = append(ts.shops, s)
		}
	}
	ts.msg = "⚠ 世界座標 → 城鎮的對應未解,這裡按座標排序取第 " +
		fmt.Sprint(idx+1) + " 個"
	g.town = ts
	return true
}

// townIndexAt 回傳這個座標是第幾個城鎮(依 y、x 排序)。
func (g *Game) townIndexAt(x, y int) int {
	type pt struct{ x, y int }
	var pts []pt
	for yy := 0; yy < world.H; yy++ {
		for xx := 0; xx < world.W; xx++ {
			if v := g.world.At(xx, yy); v >= 30 && v <= 32 {
				pts = append(pts, pt{xx, yy})
			}
		}
	}
	sort.Slice(pts, func(a, b int) bool {
		if pts[a].y != pts[b].y {
			return pts[a].y < pts[b].y
		}
		return pts[a].x < pts[b].x
	})
	for i, p := range pts {
		if p.x == x && p.y == y {
			return i
		}
	}
	return -1
}

// townKey 處理城鎮畫面的一次按鍵。
func (g *Game) townKey(k ebiten.Key) {
	ts := g.town
	switch ts.mode {
	case townBuildings:
		if i := int(k - ebiten.KeyA); i >= 0 && i < len(ts.shops) {
			ts.shop, ts.msg, ts.page = ts.shops[i], "", 0
			// 位移 34 的負值決定進哪個畫面(docs/re/138 §2)
			switch ts.shop.Kind {
			case original.ShopInn:
				ts.mode = townInn
			case original.ShopHealer:
				ts.mode = townHealer
			case original.ShopTavern:
				ts.mode = townTavern
			case original.ShopTrainer:
				ts.mode = townTrainer
			default:
				ts.mode = townShop
			}
		}
		// ⚠ 營地用 Z 不用 C —— 建築清單的字母是 A 起算,
		// 而 `C) Hamlet Hospital` 已經佔掉 C。同一個鍵兩個意思是
		// 「按下去做了別的事」,而畫面上看不出哪裡錯。
		if k == ebiten.KeyZ {
			ts.mode, ts.msg = townCamp, ""
		}
	case townInn, townHealer, townTavern, townTrainer:
		if k == ebiten.KeyEscape {
			ts.mode, ts.msg = townBuildings, ""
			return
		}
		switch ts.mode {
		case townInn:
			// 手冊 p.37:睡一晚回 2 HP、10 SP,而且供餐(不耗食糧)。
			if k == ebiten.KeyR {
				g.members = town.InnSleep(g.members)
				g.party.Clock.Tick()
				ts.msg = fmt.Sprintf("住了一晚,全隊回 %d 生命 %d 法力。⚠ 房價未解",
					town.InnHealHP, town.InnHealSP)
			}
		case townTavern:
			// ⚠ 食糧買在**酒館**,不是旅店(手冊 p.37,docs/re/140 §6)。
			if k == ebiten.KeyF {
				g.buyProvisions(1)
			}
		case townTrainer:
			// 選成員的編號升級。訓練免費(手冊 p.37)。
			if i := int(k - ebiten.Key1); i >= 0 && i < len(g.members) {
				g.trainMember(i, ts.shop.Extra)
			}
		}

	case townShop:
		// 分頁:原版就是 `+ for next page` / `- for last page`(手冊 p.31)
		switch k {
		case ebiten.KeyEqual, ebiten.KeyKPAdd:
			if (ts.page+1)*shopPageSize < len(g.itemList) {
				ts.page++
			}
			return
		case ebiten.KeyMinus, ebiten.KeyKPSubtract:
			if ts.page > 0 {
				ts.page--
			}
			return
		}
		if i := int(k - ebiten.KeyA); i >= 0 && i < shopPageSize {
			stock := ts.shopStock(g.itemList)
			if n := ts.page*shopPageSize + i; n < len(stock) {
				g.buyItem(stock[n])
			}
		}
		if k == ebiten.KeyEscape {
			ts.mode, ts.msg = townBuildings, ""
		}
	case townCamp:
		switch k {
		case ebiten.KeyR:
			g.members = town.Rest(g.members)
			g.party.Clock.Tick()
			ts.msg = "全隊休息了一會兒(這個動作原版沒有,恢復量是本引擎自訂的 " +
				fmt.Sprint(town.CampRestHeal) + " 點)"
		case ebiten.KeyS:
			// 手冊 p.38:每人耗 1 份食糧,回 1 HP、5 SP;沒得吃的人扣 1 HP。
			before := g.group.Provisions
			g.members, g.group.Provisions = town.CampSleep(g.members, before)
			for i := 0; i < town.CampSleepHours; i++ {
				g.party.Clock.Tick()
			}
			ts.msg = fmt.Sprintf("睡了一晚,吃掉 %d 份食糧(剩 %d)",
				before-g.group.Provisions, g.group.Provisions)
		case ebiten.KeyEscape:
			ts.mode, ts.msg = townBuildings, ""
		}
	}
}

// buyItem 讓隊伍第一位成員買第 i 件道具。
//
// ⚠ **買給誰未解** —— 原版的商店介面沒有讀到選人的步驟。
// 這裡固定給第一位,畫面上標出來。
// shopStock 回傳這間店賣的道具。docs/re/138 §1:位移 34–36 是編號範圍。
//
// ⚠ **先前把 57 件全列進每一間店** —— 而那看起來完全正常,
// 玩家不會知道劍舖不該賣板甲。「多顯示一些」的錯誤沒有症狀。
func (ts *townState) shopStock(all []original.Item) []original.Item {
	if ts.shop.Kind != original.ShopGoods {
		return nil
	}
	var out []original.Item
	for _, it := range all {
		if it.Index >= ts.shop.First && it.Index <= ts.shop.Last {
			out = append(out, it)
		}
	}
	return out
}

func (g *Game) buyItem(it original.Item) {
	if len(g.members) == 0 {
		return
	}
	price := town.Price(it.BasePrice, g.town.shop.PriceMult)
	gold := g.group.Gold
	r := town.Buy(&gold, &g.members[0], it.Index, price)
	if r != town.BuyOK {
		g.town.msg = r.String()
		return
	}
	g.group.Gold = gold
	g.town.msg = fmt.Sprintf("%s 買下 %s(%d 金幣)", g.members[0].Name, it.Name, price)
}

// drawTown 畫城鎮畫面。清單在主視野(61 欄)。
func (g *Game) drawTown(dst *ebiten.Image) {
	ts, p := g.town, g.panel
	if ts == nil || ts.mode == townClosed || p == nil {
		return
	}
	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)

	switch ts.mode {
	case townBuildings:
		p.Draw(dst, ts.name, x, y)
		y += lh * 1.5
		for i, s := range ts.shops {
			p.Draw(dst, fmt.Sprintf("%c) %s", 'A'+i, s.Name), x, y)
			y += lh
		}
		y += lh * 0.5
		p.Draw(dst, "Z) 營地", x, y)

	case townShop:
		p.Draw(dst, fmt.Sprintf("%s　價格倍率 %.2f", ts.shop.Name, ts.shop.PriceMult), x, y)
		y += lh * 1.5
		stock := ts.shopStock(g.itemList)
		lo := ts.page * shopPageSize
		for i := 0; i < shopPageSize && lo+i < len(stock); i++ {
			it := stock[lo+i]
			p.Draw(dst, fmt.Sprintf("%c) %s", 'A'+i, ui.PadTo(it.Name, 20)), x, y)
			p.DrawRight(dst, fmt.Sprint(town.Price(it.BasePrice, ts.shop.PriceMult)),
				x+380, y)
			y += lh
		}
		pages := (len(stock) + shopPageSize - 1) / shopPageSize
		if pages < 1 {
			pages = 1
		}
		p.Draw(dst, fmt.Sprintf("第 %d／%d 頁　+ 下一頁　- 上一頁",
			ts.page+1, pages), x, y+lh*0.5)

	case townInn, townHealer, townTavern, townTrainer:
		p.Draw(dst, ts.shop.Name+"　"+ts.shop.Kind.String(), x, y)
		y += lh * 1.5
		for _, ln := range g.buildingLines(ts) {
			for _, w := range ui.Wrap(ln, 58) {
				p.Draw(dst, w, x, y)
				y += lh
			}
		}

	case townCamp:
		p.Draw(dst, "營地", x, y)
		y += lh * 1.5
		p.Draw(dst, fmt.Sprintf("S) 睡覺（每人耗 %d 份食糧，回 %d 生命 %d 法力；目前 %d 份）",
			town.CampSleepFood, town.CampSleepHP, town.CampSleepSP,
			g.group.Provisions), x, y)
		y += lh
		p.Draw(dst, "R) 休息一會兒（本引擎自訂，非原版動作）", x, y)
		y += lh * 2
		for _, u := range town.Unresolved {
			p.Draw(dst, "⚠ "+u, x, y)
			y += lh
		}
	}

	if ts.msg != "" {
		// 訊息面板 30 欄,折行不截斷(docs/spec/04 §5)
		my := float64(layout.Message.Y + ui.PanelPad)
		for _, ln := range ui.Wrap(ts.msg, 30) {
			p.Draw(dst, ln, float64(layout.Message.X+ui.PanelPad), my)
			my += lh
		}
	}
}

// buyProvisions 在酒館買食糧。手冊 p.37(docs/re/140 §6)。
//
// ⚠ 食糧是**隊伍層級**的(GROUPS.DAT 位移 23),不是每個人各自帶。
func (g *Game) buyProvisions(n int) {
	price := town.Price(town.TownFoodPrice*n, g.town.shop.PriceMult)
	if float64(price) > g.group.Gold {
		g.town.msg = "金幣不足"
		return
	}
	g.group.Gold -= float64(price)
	g.group.Provisions += n
	g.town.msg = fmt.Sprintf("買了 %d 份食糧,花 %d 金幣(⚠ 單價未解)", n, price)
}

// trainMember 讓第 i 位成員在訓練所升級。
func (g *Game) trainMember(i, guildExtra int) {
	c := &g.members[i]
	exp := g.charExp(*c)
	r := town.Train(c, exp, guildExtra)
	if r != town.TrainOK {
		g.town.msg = c.Name + ":" + r.String()
		return
	}
	// 名冊裡的那一份也要跟著改,否則存檔寫回去的是舊的
	if c.ID >= 1 && c.ID <= len(g.chars) {
		g.chars[c.ID-1] = *c
	}
	g.town.msg = fmt.Sprintf("%s 升到第 %d 級(生命 %d／法力 %d)",
		c.Name, c.Level, c.MaxHP, c.MaxSP)
}

// buildingLines 回傳特殊建築畫面的內容。docs/re/138。
func (g *Game) buildingLines(ts *townState) []string {
	switch ts.shop.Kind {
	case original.ShopInn:
		return []string{
			fmt.Sprintf("價格倍率 %.2f", ts.shop.PriceMult),
			"R) 住一晚",
			fmt.Sprintf("睡一晚回 %d 生命、%d 法力，並供餐（手冊 p.37）。",
				town.InnHealHP, town.InnHealSP),
			"⚠ 住宿的**基準價**未解 —— 倍率讀得出來，乘的那個數字還沒定位。",
		}
	case original.ShopHealer:
		return []string{
			fmt.Sprintf("價格倍率 %.2f", ts.shop.PriceMult),
			"治癒傷勢的費用依傷害程度、解毒是定價、解束縛與復活依等級（手冊 p.37）。",
			"⚠ 這四項的**金額**都未解，本畫面只顯示不施作。",
		}
	case original.ShopTavern:
		lines := []string{
			fmt.Sprintf("F) 買 1 份食糧　目前 %d 份（⚠ 單價未解）", g.group.Provisions),
			"",
		}
		if r, ok := g.rumors[ts.shop.Extra]; ok {
			return append(lines, r)
		}
		return append(lines, fmt.Sprintf(
			"⚠ 第 %d 段傳聞未定位(docs/re/138 §4:找到 10 段、索引有 11 個)。"+
				"⛔ 這裡不拿別段頂替。", ts.shop.Extra))
	case original.ShopTrainer:
		art, teaches := "武術", "戰士"
		if ts.shop.Extra == 1 {
			art, teaches = "魔法", "法師"
		}
		lines := []string{
			"專精:" + art + "(位移 36 = " + fmt.Sprint(ts.shop.Extra) + ")，只收" + teaches,
			"訓練免費，只看經驗夠不夠（手冊 p.37）。按編號選人：",
			"",
		}
		for i, c := range g.members {
			need := town.NeedExp(c.Level)
			state := fmt.Sprintf("%d／%d", g.charExp(c), need)
			if need == 0 {
				state = "已達最高等級"
			}
			lines = append(lines, fmt.Sprintf("%d) %s　%s　第 %d 級　經驗 %s",
				i+1, c.Name, c.ClassName(), c.Level, state))
		}
		return append(lines, "", town.GrowthAssumption)
	}
	return nil
}
