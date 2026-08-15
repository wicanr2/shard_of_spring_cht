package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
	"shardofspring/internal/original"
	"shardofspring/internal/town"
	"shardofspring/internal/ui"
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
	// 營地的子畫面。原版選單有 11 個指令(docs/re/150 §5.2),
	// 這裡做了五個:角色卡、裝備、丟棄、調整隊形、傳遞。
	// 其餘的規則還沒解,選單上標「未實作」。
	campWho  int  // 選到第幾位成員
	campWho2 int  // 第二位(R)eorder / T)rade 用)
	campMode byte // 0 = 選單、'#' 角色卡、'W' 裝武器、'A' 穿防具、'D' 丟棄、'E' 裝備選類別、'R' 調隊形、'T' 傳遞
}

// shopPageSize 是一頁的商品數。主視野高 612、行高 26 → 扣掉標題約 20 列。
const shopPageSize = 20

// enterTown 從世界地圖進城。回 false 表示這一格沒有城鎮。
//
// 座標 → 城鎮的對應來自 **`TOWNDATA.BIN` 的座標表**(docs/re/53 §2、
// 兩軸依 docs/re/141 訂正),第 n 列對上 `TOWNDATA.DAT` 的第 n 個城鎮。
//
// ⚠ 這裡原本是「把地圖上的城鎮格按座標排序、取第 n 個」的佔位 ——
// 實跑走到 (24,12) 得到的是 **Arcania**(表的第 4 列),而排序給的是 Gleon。
// **那張表早就解出來了**(re/53 §2 三重驗證),只是實作端沒去用它。
func (g *Game) enterTown(x, y int) bool {
	names := original.Towns(g.shops)
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
	// 建築數要對得上座標表 —— 對不上就是兩份資料不同步,講出來不要吞掉。
	if idx < len(g.townSites) && g.townSites[idx].Shops != len(ts.shops) {
		ts.msg = fmt.Sprintf("⚠ 座標表說有 %d 間建築,實際載到 %d 間",
			g.townSites[idx].Shops, len(ts.shops))
	}
	g.town = ts
	return true
}

// townIndexAt 用 TOWNDATA.BIN 的座標表查這一格是第幾個城鎮;不是城鎮回 -1。
func (g *Game) townIndexAt(x, y int) int {
	for i, s := range g.townSites {
		if s.X == x && s.Y == y {
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
				cost := town.Price(town.TownInnPrice, ts.shop.PriceMult)
				if float64(cost) > g.group.Gold {
					ts.msg = "金幣不足"
					return
				}
				g.group.Gold -= float64(cost)
				g.members = town.InnSleep(g.members)
				g.party.Clock.Tick()
				ts.msg = fmt.Sprintf("住了一晚,花 %d 金幣,全隊回 %d 生命 %d 法力",
					cost, town.InnHealHP, town.InnHealSP)
			}
		case townTavern:
			// 原版的酒館選單是 `T)alk … B)uy food`(docs/re/142 §3)。
			// ⚠ 食糧買在**酒館**,不是旅店。
			if k == ebiten.KeyB {
				g.buyProvisions(1)
			}
		case townHealer:
			// 1–5 選人,再按 W/P/B/R 選服務(docs/re/142 的四項)。
			if i := int(k - ebiten.Key1); i >= 0 && i < len(g.members) {
				ts.page = i // 借 page 當「選到第幾個人」
				ts.msg = g.members[i].Name + ":選 W 治療 / P 解毒 / B 解束縛 / R 復活"
				return
			}
			var kind town.HealKind
			switch k {
			case ebiten.KeyW:
				kind = town.HealWounds
			case ebiten.KeyP:
				kind = town.HealPoison
			case ebiten.KeyB:
				kind = town.HealBind
			case ebiten.KeyR:
				kind = town.HealDeath
			default:
				return
			}
			g.healMember(ts.page, kind)
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
		if ts.campMode != 0 {
			g.campSubKey(k)
			return
		}
		switch k {
		case ebiten.KeyE, ebiten.KeyD, ebiten.KeyR, ebiten.KeyT:
			ts.campMode = campLetter(k)
			ts.msg = "按編號選人"
			ts.campWho, ts.campWho2 = -1, -1
		case ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5:
			// 原版是 `#)inspect char` —— 直接按編號。
			if i := int(k - ebiten.Key1); i < len(g.members) {
				ts.campMode, ts.campWho, ts.msg = '#', i, ""
			}
		case ebiten.KeyP, ebiten.KeyH, ebiten.KeyI, ebiten.KeyC, ebiten.KeyU:
			ts.msg = campUnimplemented[k] + ":原版有這個指令,但規則還沒解,本引擎未實作"
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
		p.Draw(dst, "1–5) 檢視角色　E) 裝備　D) 丟棄　R) 調整隊形　T) 傳遞", x, y)
		y += lh
		p.Draw(dst, "P) 列印角色表　H) 打獵　I) 鑑定　C) 施法　U) 使用道具", x, y)
		y += lh
		p.Draw(dst, "　　（上面這五個原版有，規則未解，本引擎未實作）", x, y)
		y += lh
		if ts.campMode != 0 {
			y += lh * 0.5
			for _, ln := range g.campLines(ts) {
				p.Draw(dst, ln, x, y)
				y += lh
			}
		}
		y += lh
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

// campUnimplemented 是原版選單上有、但規則還沒解的指令。
//
// ⚠ **列出來比隱藏好。** 少一個指令的選單看起來完全正常,
// 沒有人會發現漏了什麼 —— 而 `R)eorder` 會直接影響戰鬥站位。
var campUnimplemented = map[ebiten.Key]string{
	// P)rint 是驅動 1986 年的並列埠印表機。⚠ 依 CLAUDE.md §1.2 的邊界,
	// 它**不影響 remake 的行為**,所以不解也不做 —— 但選單上要看得到。
	ebiten.KeyP: "P) 列印角色表(原版輸出到印表機,本引擎不做)",
	ebiten.KeyC: "C) 施法",
	ebiten.KeyU: "U) 使用道具",
}

// campLetter 把按鍵換成營地子畫面的代號。
func campLetter(k ebiten.Key) byte {
	switch k {
	case ebiten.KeyE:
		return 'E'
	case ebiten.KeyD:
		return 'D'
	case ebiten.KeyR:
		return 'R'
	case ebiten.KeyT:
		return 'T'
	case ebiten.KeyH:
		return 'H'
	case ebiten.KeyI:
		return 'I'
	}
	return 0
}

// campSubKey 處理營地子畫面(裝備 / 檢視 / 丟棄)的按鍵。
//
// 流程:先按編號選人,再按背包格的字母。
// ⚠ 裝備欄存的是**背包格號**不是物品編號(docs/formats/01 位移 34/36)——
// 兩者都是小整數,寫錯不會報錯,只會讓角色拿著背包裡另一件東西打人。
func (g *Game) campSubKey(k ebiten.Key) {
	ts := g.town
	if k == ebiten.KeyEscape {
		ts.campMode, ts.msg = 0, ""
		return
	}
	if ts.campWho < 0 {
		if i := int(k - ebiten.Key1); i >= 0 && i < len(g.members) {
			ts.campWho = i
			ts.msg = ""
			if ts.campMode == 'R' || ts.campMode == 'T' {
				ts.msg = "再按一個編號選對方"
			}
		}
		return
	}
	// 角色卡只是看,沒有下一步。
	if ts.campMode == '#' {
		return
	}
	// H)unt:選完人就結束,不必選道具(docs/re/166 §2)。
	if ts.campMode == 'H' {
		g.hunt(ts.campWho)
		ts.campMode, ts.campWho = 0, -1
		return
	}
	// R)eorder 與 T)rade 都要選第二個人
	if ts.campMode == 'R' || ts.campMode == 'T' {
		if ts.campWho2 < 0 {
			if i := int(k - ebiten.Key1); i >= 0 && i < len(g.members) {
				ts.campWho2 = i
			}
			return
		}
		if ts.campMode == 'R' {
			if town.Reorder(&g.group, g.members, ts.campWho+1, ts.campWho2+1) {
				ts.msg = "調換了隊形順序（戰場站位跟著改）"
			} else {
				ts.msg = "那兩個位置換不了"
			}
			ts.campMode, ts.campWho, ts.campWho2 = 0, -1, -1
			return
		}
		// T)rade:選完兩個人再選背包格
		slot := int(k - ebiten.KeyA)
		if slot < 0 || slot >= town.PackSlots {
			return
		}
		r := town.Trade(&g.members[ts.campWho], &g.members[ts.campWho2], slot)
		ts.msg = g.members[ts.campWho].Name + "：" + r.String()
		if r == town.TradeOK {
			g.syncMember(g.members[ts.campWho])
			g.syncMember(g.members[ts.campWho2])
		}
		return
	}
	// `E)quip` 選完人再選武器還是防具 —— **分類不在 `ITEMS.DAT` 裡,在呼叫端**
	// (docs/formats/04)。用編號範圍去猜會在資料外的編號上默默猜錯。
	if ts.campMode == 'E' {
		switch k {
		case ebiten.KeyW:
			ts.campMode = 'W'
		case ebiten.KeyA:
			ts.campMode = 'A'
		}
		return
	}
	slot := int(k - ebiten.KeyA)
	if slot < 0 || slot >= town.PackSlots {
		return
	}
	c := &g.members[ts.campWho]
	if town.PackEmpty(*c, slot) {
		ts.msg = "那一格是空的"
		return
	}
	switch ts.campMode {
	case 'I':
		g.identify(ts.campWho, slot)
		ts.campMode, ts.campWho = 0, -1
		return
	case 'W', 'A':
		it, ok := g.itemByIndex(c.Pack[slot])
		if !ok {
			ts.msg = "這件東西的資料查不到"
			return
		}
		armor := ts.campMode == 'A'
		town.Equip(c, slot, armor)
		kind := "當武器"
		if armor {
			kind = "當防具"
		}
		ts.msg = fmt.Sprintf("%s 把 %s %s", c.Name, it.Name, kind)
	case 'D':
		it, _ := g.itemByIndex(c.Pack[slot])
		// 丟掉的東西拿不回來(手冊 p.39)。卸下同一格的裝備,
		// 否則裝備欄會指向一個空格 —— 那個錯誤不會報錯。
		if c.Weapon == slot {
			c.Weapon = original.NotEquipped
		}
		if c.Armor == slot {
			c.Armor = original.NotEquipped
		}
		// ⚠ 空格填 99 不是 0(docs/re/144 §3)—— 填 0 會讓那一格看起來裝著第 0 號道具
		c.Pack[slot] = original.NotEquipped
		ts.msg = fmt.Sprintf("%s 丟掉了 %s(拿不回來)", c.Name, it.Name)
	}
	if c.ID >= 1 && c.ID <= len(g.chars) {
		g.chars[c.ID-1] = *c
	}
}

// itemByIndex 查一件道具。
func (g *Game) itemByIndex(n int) (original.Item, bool) {
	for _, it := range g.itemList {
		if it.Index == n {
			return it, true
		}
	}
	return original.Item{}, false
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

// healMember 在治療所替第 i 位成員做一項服務。
func (g *Game) healMember(i int, k town.HealKind) {
	if i < 0 || i >= len(g.members) {
		return
	}
	c := &g.members[i]
	gold := g.group.Gold
	cost := town.HealCost(*c, k, g.town.shop.PriceMult)
	if !town.Heal(&gold, c, k, g.town.shop.PriceMult) {
		if cost == 0 {
			g.town.msg = c.Name + " 不需要「" + k.String() + "」"
		} else {
			g.town.msg = fmt.Sprintf("金幣不足(需要 %d)", cost)
		}
		return
	}
	g.group.Gold = gold
	if c.ID >= 1 && c.ID <= len(g.chars) {
		g.chars[c.ID-1] = *c
	}
	g.town.msg = fmt.Sprintf("%s 接受「%s」,花 %d 金幣", c.Name, k.String(), cost)
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

// charCard 是原版 `#)inspect char` 的角色卡(docs/re/150 §1.1)。
//
// 版面照原版:左邊五個基本屬性、右邊等級與衍生值。
// ⚠ 屬性名用**手冊的譯名**(速度／力量／智能／體能／技巧,
// translations/glossary.md)—— 原版畫面上的英文標籤與手冊同一組詞,
// 所以譯名不必另外決定。
func (g *Game) charCard(c original.Character) []string {
	sp := "—"
	if c.MaxSP > 0 {
		sp = fmt.Sprintf("%d／%d", c.SP, c.MaxSP)
	}
	st := c.StatusName()
	if st == "" {
		st = "正常"
	}
	return []string{
		fmt.Sprintf("%s　%s %s　等級 %d", c.Name, c.RaceName(), c.ClassName(), c.Level),
		"",
		fmt.Sprintf("速度 %2d　　生命 %d／%d", c.Speed, c.HP, c.MaxHP),
		fmt.Sprintf("力量 %2d　　法力 %s", c.Str, sp),
		fmt.Sprintf("智能 %2d　　經驗 %d", c.Int, g.charExp(c)),
		fmt.Sprintf("體能 %2d　　狀態 %s", c.End, st),
		fmt.Sprintf("技巧 %2d", c.ToHit),
	}
}

// campLines 回傳營地子畫面的內容。
func (g *Game) campLines(ts *townState) []string {
	if ts.campMode == '#' && ts.campWho >= 0 && ts.campWho < len(g.members) {
		return g.charCard(g.members[ts.campWho])
	}
	title := map[byte]string{'E': "裝備", 'W': "拿武器", 'A': "穿防具", 'D': "丟棄",
		'R': "調整隊形", 'T': "傳遞"}[ts.campMode]
	if ts.campWho < 0 {
		out := []string{title + "：按編號選人"}
		for i, c := range g.members {
			out = append(out, fmt.Sprintf("%d) %s", i+1, c.Name))
		}
		return out
	}
	c := g.members[ts.campWho]
	if ts.campMode == 'R' {
		return []string{c.Name + "：再按一個編號,兩人交換隊形順序",
			"（順序不只是顯示 —— 戰場站位直接用它算,docs/re/160）"}
	}
	if ts.campMode == 'T' {
		if ts.campWho2 < 0 {
			return []string{c.Name + "：再按一個編號選對方"}
		}
		out := []string{fmt.Sprintf("%s → %s（按字母選要給的那一格）",
			c.Name, g.members[ts.campWho2].Name)}
		return append(out, g.packLines(c)...)
	}
	if ts.campMode == 'E' {
		return []string{c.Name + "：W) 當武器　A) 當防具",
			"（`ITEMS.DAT` 沒有「武器還是防具」這個欄位，分類在呼叫端）"}
	}
	out := []string{fmt.Sprintf("%s 的背包（%s，按字母選格）", c.Name, title)}
	return append(out, g.packLines(c)...)
}

// packLines 列出一個角色的背包。
func (g *Game) packLines(c original.Character) []string {
	var out []string
	for i, n := range c.Pack {
		if n == original.NotEquipped {
			continue
		}
		name := "（編號 " + fmt.Sprint(n) + "，查不到）"
		if it, ok := g.itemByIndex(n); ok {
			name = it.Name
		}
		mark := ""
		if c.Weapon == i {
			mark = "　←武器"
		}
		if c.Armor == i {
			mark = "　←防具"
		}
		out = append(out, fmt.Sprintf("%c) %s%s", 'A'+i, name, mark))
	}
	// ⚠ 這裡是 0 不是 1 —— 抽成獨立函式之後 out 不再帶標題那一列。
	// 寫成 1 的話空背包會**什麼都不顯示**,而畫面上看起來像正常的空清單。
	if len(out) == 0 {
		out = append(out, "（背包是空的）")
	}
	return out
}

// buildingLines 回傳特殊建築畫面的內容。docs/re/138。
func (g *Game) buildingLines(ts *townState) []string {
	switch ts.shop.Kind {
	case original.ShopInn:
		return []string{
			fmt.Sprintf("價格倍率 %.2f", ts.shop.PriceMult),
			fmt.Sprintf("R) 住一晚　%d 金幣",
				town.Price(town.TownInnPrice, ts.shop.PriceMult)),
			fmt.Sprintf("睡一晚回 %d 生命、%d 法力，並供餐（手冊 p.37）。",
				town.InnHealHP, town.InnHealSP),
		}
	case original.ShopHealer:
		lines := []string{
			fmt.Sprintf("價格倍率 %.2f　（治療所不受說服技能影響）", ts.shop.PriceMult),
			"",
			fmt.Sprintf("治療傷勢　每點生命 %d　　解毒　%d",
				town.Price(town.HealPerHP, ts.shop.PriceMult),
				town.Price(town.UnpoisonPrice, ts.shop.PriceMult)),
			fmt.Sprintf("解除束縛　每級 %d　　復活　每級 %d",
				town.Price(town.UnbindPerLv, ts.shop.PriceMult),
				town.Price(town.ResurrectPerLv, ts.shop.PriceMult)),
			"",
			"按編號選人，再按 W 治療 / P 解毒 / B 解束縛 / R 復活：",
		}
		for i, c := range g.members {
			lines = append(lines, fmt.Sprintf("%d) %s　生命 %d／%d　%s　治療費 %d",
				i+1, c.Name, c.HP, c.MaxHP, c.StatusName(),
				town.HealCost(c, town.HealWounds, ts.shop.PriceMult)))
		}
		return append(lines, "", town.ResurrectAssumption)
	case original.ShopTavern:
		lines := []string{
			fmt.Sprintf("B) 買 1 份食糧　每份 %d 金幣　目前 %d 份",
				town.Price(town.TownFoodPrice, ts.shop.PriceMult), g.group.Provisions),
			"",
		}
		if r, ok := g.rumors[ts.shop.Extra]; ok {
			return append(lines, r)
		}
		// 11 個索引現在都有文字(docs/re/142 §5:第 11 段是續作預告)。
		// ⛔ 真的查不到時仍然不拿別段頂替。
		return append(lines, fmt.Sprintf(
			"⚠ 第 %d 段傳聞的譯文載不進來,這裡不拿別段頂替。", ts.shop.Extra))
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

// hunt 與 identify 是營地裡兩個「每天一次」的技能。docs/re/166。
//
// ⚠ 兩者的**結果**都還沒讀完:打獵的擲骰(`INT 3D:33` 的參數、加項、上限)
// 與鑑定的 `Failed` 分支都未解(docs/re/166 §6)。
// 所以這裡只做**閘門**——閘門是逐條讀出來的,結果不是。
// ⛔ 不要為了「有東西發生」自己編一個成功率。

func (g *Game) hunt(who int) {
	ts := g.town
	c := &g.members[who]
	// 原版的「在野外」是 `ds:3534 ≥ 99`,而那個變數的來源未解 ——
	// 引擎用「不在迷宮、也不在城鎮」。營地開在城鎮裡時就是室內。
	outdoors := g.level == nil && g.town != nil && g.town.mode == townCamp
	if gate := town.CanHunt(*c, outdoors); gate != town.SkillOK {
		ts.msg = c.Name + "：" + gate.String()
		return
	}
	c.SkillUsed = true
	g.syncMember(*c)
	ts.msg = c.Name + " 出去打獵了。⚠ 收穫多少未解（docs/re/166 §2）"
}

func (g *Game) identify(who, slot int) {
	ts := g.town
	c := &g.members[who]
	item := c.Pack[slot]
	if gate := town.CanIdentify(*c, item); gate != town.SkillOK {
		ts.msg = c.Name + "：" + gate.String()
		return
	}
	c.SkillUsed = true
	g.syncMember(*c)
	name := "那件東西"
	if it, ok := g.itemByIndex(item); ok {
		name = it.Name
	}
	ts.msg = c.Name + " 辨識了 " + name + "。⚠ 成功率未解（docs/re/166 §3）"
}
