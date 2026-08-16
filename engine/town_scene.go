package main

import (
	"fmt"
	"strings"

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
	// 11 個全部做了:角色卡、裝備、丟棄、調整隊形、傳遞、睡覺、
	// 打獵、鑑定,加上 docs/spec/16 補上的施法、使用道具、列印。
	campWho  int  // 選到第幾位成員
	campWho2 int  // 第二位(R)eorder / T)rade 用)
	campMode byte // 0 = 選單、'#' 角色卡、'W' 裝武器、'A' 穿防具、'D' 丟棄、'E' 裝備選類別、'R' 調隊形、'T' 傳遞、'C' 施法、'U' 使用道具、'P' 列印

	// pendingBuy 非 nil = 商店裡已經選好道具,正在問「交給角色 #」
	// (docs/re/187:原版**會問**,而且問的是角色編號)。
	pendingBuy *original.Item

	// C)ast spell 的子階段(docs/spec/16 §1/§2)。campWho 借來當施法者;
	// 選目標另外用 campWho2 —— 與 R)eorder / T)rade 共用同一個欄位,
	// 但那兩個模式不會跟 'C' 同時開著,不會互相污染。
	castStage int            // 0=選施法者(用 campWho) 1=選法術 2=輸入投入點數 3=選目標(用 campWho2)
	castSpell original.Spell // 選定的法術
	castInput string         // 投入點數的數字輸入緩衝(docs/spec/16 §1 的 'Spell Pts ?')
	// utter 非 nil = 正在打咒語(docs/re/197 §5)。
	// ⚠ 原版的「施放哪個法術?」整格就是自由輸入,咒語打在那裡;
	// 引擎是字母選單,所以另開一個輸入 —— **那個按鍵是引擎的決定**。
	utter *string
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
			ts.mode, ts.msg = townCamp, "紮營中……" // WRLDMOVE:43 / MAZEMOVE:89
		}
	case townInn, townHealer, townTavern, townTrainer:
		if k == ebiten.KeyEscape {
			ts.mode, ts.msg = townBuildings, ""
			return
		}
		switch ts.mode {
		case townInn:
			// 手冊 p.37:睡一晚回 2 HP、10 SP,而且供餐(不耗食糧)。
			// 原版問住幾晚(TOWN:30/31),不是一次一晚。
			if k == ebiten.KeyR {
				g.townCount = 'R'
				ts.msg = innNightsPrompt(town.Price(town.TownInnPrice, ts.shop.PriceMult))
				return
			}
			if n := countKey(k); g.townCount == 'R' && n >= 0 {
				g.townCount = 0
				if n == 0 {
					ts.msg = ""
					return
				}
				g.innSleep(n)
			}
		case townTavern:
			// 原版的酒館選單是 `T)alk … B)uy food`(docs/re/142 §3)。
			// ⚠ 食糧買在**酒館**,不是旅店。
			if k == ebiten.KeyB {
				g.townCount = 'B'
				ts.msg = rationsPrompt(town.Price(town.TownFoodPrice, ts.shop.PriceMult))
				return
			}
			if n := countKey(k); g.townCount == 'B' && n >= 0 {
				g.townCount = 0
				if n == 0 {
					ts.msg = ""
					return
				}
				g.buyProvisions(n)
			}
		case townHealer:
			// 付款確認開著時只收 Y/N(TOWN:27/28)——⚠ 要放在選人之前,
			// 否則 `1`–`5` 會在確認畫面上把選到的人換掉。
			if g.healPay != nil {
				g.answerHealPay(k)
				return
			}
			// 1–5 選人,再按 W/P/B/R 選服務(docs/re/142 的四項)。
			if i := int(k - ebiten.Key1); i >= 0 && i < len(g.members) {
				ts.page = i // 借 page 當「選到第幾個人」
				ts.msg = g.members[i].Name + ":選 W 醫療 / P 解毒 / B 解除束縛 / R 復活"
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
			g.askHealPay(ts.page, kind)
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
		// 選好道具之後,原版問「Give to char #」(docs/re/187,實跑確認過:
		// 輸入編號 → 扣錢 → 東西進**那個人**的背包)。
		if ts.pendingBuy != nil {
			if k == ebiten.KeyEscape {
				ts.pendingBuy, ts.msg = nil, ""
				return
			}
			if i := int(k - ebiten.Key1); i >= 0 && i < len(g.members) {
				it := *ts.pendingBuy
				ts.pendingBuy = nil
				g.buyItem(it, i)
			}
			return
		}
		if i := int(k - ebiten.KeyA); i >= 0 && i < shopPageSize {
			stock := ts.shopStock(g.itemList)
			if n := ts.page*shopPageSize + i; n < len(stock) {
				it := stock[n]
				ts.pendingBuy = &it
				// 「價格為」「交給角色 #」都是原版的字(TOWN:14 / TOWN:15)。
				ts.msg = fmt.Sprintf("價格為 %d　交給角色 #(1–%d,(ESC離開))",
					town.Price(it.BasePrice, ts.shop.PriceMult), len(g.members))
			}
		}
		if k == ebiten.KeyEscape {
			ts.mode, ts.msg, ts.pendingBuy = townBuildings, "", nil
		}
	case townCamp:
		if ts.campMode != 0 {
			g.campSubKey(k)
			return
		}
		switch k {
		// ⚠ H)unt 與 I)dentify 原本沒列在這裡 —— campLetter() 早就把它們對到
		// 'H'/'I',hunt()/identify() 也早就寫好了,但因為這個 switch 沒把
		// KeyH/KeyI 分到這一支,兩個鍵一直落進下面「未實作」那支,訊息還是空的
		// (campUnimplemented 沒收這兩個鍵)。這是既有的接線漏洞,不是規則沒解——
		// 修 C/U/P 一定要動這個 switch(Go 不准同一個鍵在兩個 case 出現),
		// 順手把 H/I 接回去。
		case ebiten.KeyE, ebiten.KeyD, ebiten.KeyR, ebiten.KeyT, ebiten.KeyH, ebiten.KeyI,
			ebiten.KeyC, ebiten.KeyU, ebiten.KeyP:
			ts.campMode = campLetter(k)
			ts.msg = campSelectPrompt(ts.campMode)
			ts.campWho, ts.campWho2 = -1, -1
			ts.castStage, ts.castInput = 0, ""
		case ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5:
			// 原版是 `#)inspect char` —— 直接按編號。
			if i := int(k - ebiten.Key1); i < len(g.members) {
				ts.campMode, ts.campWho, ts.msg = '#', i, ""
			}
		case ebiten.KeyS:
			// 手冊 p.38:每人耗 1 份食糧,回 1 HP、5 SP;沒得吃的人扣 1 HP。
			// CAMP:55「You are not tired」—— 全隊生命與法力都滿就不用睡。
			// ⚠ 「累不累」的判準**沒有讀到**,這是引擎的定義;
			// 擋下來的理由是睡覺要吃食糧,滿血滿魔時睡是純損失。
			if partyRested(g.members) {
				ts.msg = "你們還不累"
				return
			}
			before := g.group.Provisions
			alive := aliveNames(g.members)
			g.members, g.group.Provisions = town.CampSleep(g.members, before)
			for i := 0; i < town.CampSleepHours; i++ {
				g.party.Clock.Tick()
			}
			// 原版是**先後兩句**:睡下去印 CAMP:56「You sleep...」,
			// 醒來印 CAMP:54「You have slept !」。引擎沒有中間的等待,
			// 兩句排在同一行。
			ts.msg = fmt.Sprintf("你們睡下了……你們睡了一覺!吃掉 %d 份食糧(剩 %d)",
				before-g.group.Provisions, g.group.Provisions)
			// CAMP:137 / TOWN:81「dies in the night.」—— 中毒或沒得吃的人
			// 可能撐不過去。⚠ 不講的話玩家隔天才會發現少了一個人。
			for _, n := range diedOvernight(alive, g.members) {
				ts.msg += "　" + n + " 在夜裡死去。"
			}
		case ebiten.KeyEscape:
			ts.mode, ts.msg = townBuildings, "拔營中……" // CAMP:19「Breaking Camp..」
		}
	}
}

// equippedName 回傳裝備欄指到的道具名;沒裝就回 fallback
// (CAMP:42「No Weapon」/ 49「No Armor」)。
func (g *Game) equippedName(c original.Character, slot int, fallback string) string {
	if slot < 0 || slot >= town.PackSlots || c.Pack[slot] == original.NotEquipped {
		return fallback
	}
	if it, ok := g.itemByIndex(c.Pack[slot]); ok {
		return it.Name
	}
	return fallback
}

// partyRested 回傳全隊是不是生命與法力都滿(CAMP:55 的「不累」)。
// ⚠ 這是**引擎的定義**,原版怎麼判沒有讀到。
func partyRested(party []original.Character) bool {
	for _, c := range party {
		if !c.Occupied() {
			continue
		}
		if c.HP < c.MaxHP || c.SP < c.MaxSP {
			return false
		}
	}
	return true
}

// aliveNames / diedOvernight 用來認出「睡前活著、睡後倒下」的人。
//
// ⚠ 用**名字**而不是索引比對是刻意的:CampSleep 收的是切片、回的也是切片,
// 兩份在呼叫端是同一個底層陣列 —— 先存下來的是誰活著,不是他們的生命值。
func aliveNames(party []original.Character) map[string]bool {
	out := map[string]bool{}
	for _, c := range party {
		if c.Occupied() && c.HP > 0 {
			out[c.Name] = true
		}
	}
	return out
}

func diedOvernight(before map[string]bool, after []original.Character) []string {
	var out []string
	for _, c := range after {
		if before[c.Name] && c.HP <= 0 {
			out = append(out, c.Name)
		}
	}
	return out
}

// countKey 把數字鍵轉成 0–9;不是數字鍵回 −1。原版的「幾個」一律是
// 單鍵 1–9、0 離開(TOWN:31/59),沒有多位數輸入。
func countKey(k ebiten.Key) int {
	switch {
	case k >= ebiten.KeyDigit0 && k <= ebiten.KeyDigit9:
		return int(k - ebiten.KeyDigit0)
	case k >= ebiten.KeyKP0 && k <= ebiten.KeyKP9:
		return int(k - ebiten.KeyKP0)
	}
	return -1
}

// innNightsPrompt / rationsPrompt 是原版問「幾個」的兩句話。
// TOWN:30+31、TOWN:58+59。
func innNightsPrompt(cost int) string {
	return fmt.Sprintf("房間將花費 %d 金幣一晚。要住幾晚?(1–9,0離開)", cost)
}

func rationsPrompt(cost int) string {
	return fmt.Sprintf("隊伍一天的食糧花費 %d 金幣。要買幾份口糧?(1–9,0離開)", cost)
}

// innSleep 在旅店住 n 晚。⚠ **錢不夠就一晚都不住** ——
// 住一半再說「錢不夠」會讓玩家搞不清楚自己到底住了幾晚。
func (g *Game) innSleep(n int) {
	ts := g.town
	each := town.Price(town.TownInnPrice, ts.shop.PriceMult)
	cost := each * n
	if float64(cost) > g.group.Gold {
		ts.msg = "你沒有足夠的金幣!" // TOWN:76+77
		return
	}
	g.group.Gold -= float64(cost)
	for i := 0; i < n; i++ {
		g.members = town.InnSleep(g.members)
		g.party.Clock.Tick()
	}
	// TOWN:32「You sleep...」
	ts.msg = fmt.Sprintf("你們睡了一覺……住 %d 晚,花 %d 金幣,全隊回 %d 生命 %d 法力",
		n, cost, town.InnHealHP*n, town.InnHealSP*n)
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

// buyItem 讓第 who 位成員(0 起算)買下一件道具。
//
// 「買給誰」是**原版問的**(docs/re/187 實跑:`Give to char #`,輸入編號之後
// 金幣才扣、東西才進那個人的背包)。⚠ 先前固定給第一位,是靜態沒讀到那句話 ——
// 而那句字串一直都在清冊裡(TOWN:15),只是搜尋時找的是別的字眼。
func (g *Game) buyItem(it original.Item, who int) {
	if who < 0 || who >= len(g.members) {
		return
	}
	price := town.Price(it.BasePrice, g.town.shop.PriceMult)
	gold := g.group.Gold
	r := town.Buy(&gold, &g.members[who], it.Index, price)
	if r != town.BuyOK {
		g.town.msg = r.String()
		return
	}
	g.group.Gold = gold
	g.syncMember(g.members[who]) // 背包改了要寫回名冊,否則存檔會蓋掉
	// TOWN:17「 ok!」
	g.town.msg = fmt.Sprintf("好!%s 買下 %s(%d 金幣)", g.members[who].Name, it.Name, price)
}

// drawTown 畫城鎮畫面。清單在主視野(61 欄)。
// pages 回傳這間店的商品要分成幾頁(至少 1)。提示列與畫面共用同一個算法,
// 免得「畫面說第 1／1 頁、提示列卻叫你翻頁」。
func (ts *townState) pages(items []original.Item) int {
	n := (len(ts.shopStock(items)) + shopPageSize - 1) / shopPageSize
	if n < 1 {
		return 1
	}
	return n
}

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
		p.Draw(dst, "歡迎來到……"+ts.name, x, y) // WRLDMOVE:40
		y += lh * 1.5
		for i, s := range ts.shops {
			p.Draw(dst, fmt.Sprintf("%c) %s", 'A'+i, s.Name), x, y)
			y += lh
		}
		y += lh * 0.5
		p.Draw(dst, "Z) 營地", x, y)

	case townShop:
		// WRLDMOVE:42「Entering ...」—— 原版走進一間店時印的那一句。
		p.Draw(dst, fmt.Sprintf("正在進入……%s　價格倍率 %.2f",
			ts.shop.Name, ts.shop.PriceMult), x, y)
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
		pageLine := fmt.Sprintf("第 %d／%d 頁　+ 下一頁　- 上一頁",
			ts.page+1, ts.pages(g.itemList))
		if ts.page+1 < ts.pages(g.itemList) {
			pageLine += "　【更多】" // TOWN:11「[MORE]」
		}
		p.Draw(dst, pageLine, x, y+lh*0.5)

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
		p.Draw(dst, "營地：", x, y) // CAMP:6
		y += lh * 1.5
		// 指令列的字面照 CAMP.tsv 第 7–18 列(F3)——**十一個指令 + ESC**,
		// 順序與原版選單相同。⚠ 括號後面**不留空格**:原版是 `P)rint char(s)`,
		// 字母與詞是連著的,那個字母就是要按的鍵。
		for _, ln := range []string{
			" S)睡覺　　#)查看角色　　P)列印角色卡",
			" C)施放法術　　R)調整隊形　　T)交易",
			" D)丟棄　　E)裝備　　H)打獵　　I)鑑定　　U)使用道具",
			" ESC離開",
		} {
			p.Draw(dst, ln, x, y)
			y += lh
		}
		// 睡覺的代價與收益原版沒有印,但那幾個數字是規則的一部分
		// (手冊 p.38),放在指令列下面當註腳。
		p.Draw(dst, fmt.Sprintf("（睡覺每人耗 %d 份食糧,回 %d 生命 %d 法力；目前 %d 份）",
			town.CampSleepFood, town.CampSleepHP, town.CampSleepSP,
			g.group.Provisions), x, y)
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

	g.drawMessage(dst, ts.msg) // message.go:折行不截斷(docs/spec/04 §5)
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
	case ebiten.KeyC:
		return 'C'
	case ebiten.KeyU:
		return 'U'
	case ebiten.KeyP:
		return 'P'
	}
	return 0
}

// campSelectPrompt 回傳按下營地指令鍵之後、選人畫面要顯示的提示語,
// 逐句對應 translations/module-text/CAMP.tsv 的 `Character # to … ?` 系列
// (docs/spec/19-module-text.md)。
//
// ⚠ R)eorder 與 P)rint 在清冊裡沒有對應到這一句原文(P)rint 的原文屬於
// `na-printer`,不在本輪翻譯範圍;R)eorder 找不到對應的「選人」原句)——
// 兩者沿用既有的通用措辭,不要硬湊。
func campSelectPrompt(mode byte) string {
	switch mode {
	case 'E':
		return "哪位角色要裝備?(ESC離開)" // CAMP:39
	case 'D':
		return "哪位角色要丟棄道具?(ESC離開)" // CAMP:51
	case 'T':
		return "從哪位角色交易?(ESC離開)" // CAMP:34
	case 'H':
		return "哪位角色要打獵?(ESC離開)" // CAMP:64
	case 'I':
		return "哪位角色要鑑定?(ESC離開)" // CAMP:58
	case 'C':
		return "哪位角色要施法?(ESC離開)" // CAMP:75
	case 'U':
		return "哪位角色要使用道具?(ESC離開)" // CAMP:88
	}
	return "按編號選人"
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
	// C)ast spell / U)se an item / P)rint 各有自己的多階段流程
	// (docs/spec/16),獨立寫在 camp_actions.go,不擠進下面這條
	// 「先選人、再選背包格」的共用路徑。
	switch ts.campMode {
	case 'C':
		g.campCastKey(k)
		return
	case 'U':
		g.campUseKey(k)
		return
	case 'P':
		g.campPrintKey(k)
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
	// CAMP:41/48「 skips.」—— 武器/防具那一步按 Enter 直接略過。
	if (ts.campMode == 'W' || ts.campMode == 'A') &&
		(k == ebiten.KeyEnter || k == ebiten.KeyKPEnter) {
		ts.msg = g.members[ts.campWho].Name + "：略過。"
		ts.campMode, ts.campWho = 0, -1
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
		// 武器技能閘門(docs/re/196 §1):巫師或沒有對應技能就裝不上。
		// ⚠ 只擋武器,而且**編號 0 的匕首不檢查** —— 那是原版 `編號 > 0`
		// 那一行的直接後果,不是特例補丁。
		if !armor {
			if ok, checked := town.WeaponSkillOK(*c, c.Pack[slot]); checked && !ok {
				ts.msg = campNoSkill
				return
			}
		}
		town.Equip(c, slot, armor)
		kind := "武器"
		if armor {
			kind = "防具"
		}
		// CAMP:50「OK !」—— 原版裝備成功只回一聲「好!」。
		ts.msg = fmt.Sprintf("好!%s 的%s換成 %s", c.Name, kind, it.Name)
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
		// CAMP:53「Dropped ! 」
		ts.msg = fmt.Sprintf("已丟棄!%s 的 %s(拿不回來)", c.Name, it.Name)
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
		g.town.msg = "你沒有足夠的金幣!" // TOWN:76+77
		return
	}
	g.group.Gold -= float64(price)
	g.group.Provisions += n
	// TOWN:60「Done!」
	g.town.msg = fmt.Sprintf("完成!買了 %d 份食糧,花 %d 金幣(⚠ 單價未解)", n, price)
}

// healMember 在治療所替第 i 位成員做一項服務。
// healPayState 是治療所的付款確認(TOWN:27+28)。
//
// ⚠ 原版**先報價再收錢**;引擎先前直接扣 —— 那不只是少一句話,
// 玩家在復活這種昂貴服務上沒有反悔的機會。
type healPayState struct {
	who  int
	kind town.HealKind
	cost int
}

// askHealPay 報價並等 Y/N。價格 0(不需要這項服務)就直接走原本的路,
// 那一支會說「不需要」。
func (g *Game) askHealPay(i int, k town.HealKind) {
	if i < 0 || i >= len(g.members) {
		return
	}
	cost := town.HealCost(g.members[i], k, g.town.shop.PriceMult)
	if cost == 0 {
		g.healMember(i, k)
		return
	}
	g.healPay = &healPayState{who: i, kind: k, cost: cost}
	g.town.msg = fmt.Sprintf("這將花費 %d 金幣,付款嗎?(Y/N)", cost)
}

// answerHealPay 收 Y/N。其他鍵一律忽略 —— 問句還在畫面上。
func (g *Game) answerHealPay(k ebiten.Key) {
	p := g.healPay
	switch k {
	case ebiten.KeyY:
		g.healPay = nil
		g.healMember(p.who, p.kind)
	case ebiten.KeyN, ebiten.KeyEscape:
		g.healPay = nil
		g.town.msg = ""
	}
}

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
			g.town.msg = fmt.Sprintf("你沒有足夠的金幣!(需要 %d)", cost) // TOWN:76+77
		}
		return
	}
	g.group.Gold = gold
	if c.ID >= 1 && c.ID <= len(g.chars) {
		g.chars[c.ID-1] = *c
	}
	// TOWN:20「Done!」
	g.town.msg = fmt.Sprintf("完成!%s 接受「%s」,花 %d 金幣", c.Name, k.String(), cost)
}

// trainMember 讓第 i 位成員在訓練所升級。
func (g *Game) trainMember(i, guildExtra int) {
	c := &g.members[i]
	// TOWN:39「This character is incapacitated!」——⚠ 與營地那一句
	// (CAMP:131,句號結尾)是**兩個地方**,原版分成兩份字串。
	// 門檻沿用同一個(`> 1`,docs/re/166 §7)——那是**讀到的**。
	if c.Status > town.MaxActiveStatus {
		g.town.msg = c.Name + "：這位角色行動不能!"
		return
	}
	exp := g.charExp(*c)
	before := town.AttrSnapshot(*c)
	hpBefore, spBefore := c.MaxHP, c.MaxSP
	r := town.Train(c, exp, guildExtra, g.rand)
	if r == town.TrainNotEnoughExp {
		// TOWN:40+41「The Guild decides you need N experience before
		// gaining a level.」——經驗差額由這裡算,TrainResult 本身沒有這個數字。
		g.town.msg = fmt.Sprintf("%s：公會判定你還需要 %d 點經驗才能升級。",
			c.Name, town.NeedExp(c.Level)-exp)
		return
	}
	if r != town.TrainOK {
		g.town.msg = c.Name + ":" + r.String()
		return
	}
	// 名冊裡的那一份也要跟著改,否則存檔寫回去的是舊的
	if c.ID >= 1 && c.ID <= len(g.chars) {
		g.chars[c.ID-1] = *c
	}
	// 措辭照 TOWN.tsv 第 42–50 列(F3):原版升級印的是
	//
	//	You made a level!  You gain N hit points.  You also gain M Spell Points!
	//	Stats are up by: 1 pt of X * 1 pt of Y
	//	You have K points left.
	//
	// ⚠ 屬性列**一點一筆**,不是「X+2」——原版擲三次、每次五選一 +1,
	// 同一項可以被選中兩次(docs/re/183 §5),而它照樣印兩筆 `1 pt of X`。
	// 併成「+2」會把「擲三次」這條規則從畫面上抹掉。
	var ups []string
	for k, d := range town.AttrGrowth(before, *c) {
		for j := 0; j < d; j++ {
			ups = append(ups, "1 點"+town.AttrNames[k])
		}
	}
	msg := fmt.Sprintf("%s 你升級了!你獲得 %d 點生命點數。",
		c.Name, c.MaxHP-hpBefore)
	if d := c.MaxSP - spBefore; d > 0 {
		msg += fmt.Sprintf("你還獲得 %d 點法力點數!", d)
	}
	if len(ups) > 0 {
		msg += "屬性提升：" + strings.Join(ups, "* ")
	}
	// 技能點會累積(docs/re/183 §6),所以印的是總數不是這次發的。
	msg += fmt.Sprintf("你還剩 %d 點可分配。", c.SkillPts)
	g.town.msg = msg
	// docs/spec/20-skill-allocation.md:升級發的點數要有地方花——接技能點
	// 分配畫面(蓋掉主視野,上面這句訊息留在訊息列繼續顯示)。onDone 是
	// nil——這裡要接續的事(升級訊息)已經在打開畫面之前設定好了。
	g.openSkillAlloc(c.ID, i, nil)
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
	// C)ast / U)se / P)rint 各自的畫面在 camp_actions.go。
	switch ts.campMode {
	case 'C':
		return g.campCastLines(ts)
	case 'U':
		return g.campUseLines(ts)
	case 'P':
		return g.campPrintLines(ts)
	}
	if ts.campMode == '#' && ts.campWho >= 0 && ts.campWho < len(g.members) {
		return g.charCard(g.members[ts.campWho])
	}
	// 背包標題的字面照 CAMP.tsv(F3):`Item to drop ?`(52)、`Item to ID ?`(60)、
	// `Trade which ?`(35)、`Weapon? `(40)、`Armor?  `(47)。
	title := map[byte]string{'E': "裝備", 'W': "武器?", 'A': "防具?", 'D': "要丟棄哪件?",
		'R': "調整隊形", 'T': "交易哪一件?", 'H': "打獵", 'I': "要鑑定哪件?"}[ts.campMode]
	if ts.campWho < 0 {
		out := []string{campSelectPrompt(ts.campMode)}
		for i, c := range g.members {
			out = append(out, fmt.Sprintf("%d) %s", i+1, c.Name))
		}
		return out
	}
	c := g.members[ts.campWho]
	if ts.campMode == 'R' {
		// CAMP:68/69「Enter new 」+「position for:」
		return []string{"輸入新的位置給：" + c.Name + "（再按一個編號,兩人交換）",
			"（順序不只是顯示 —— 戰場站位直接用它算,docs/re/160）"}
	}
	if ts.campMode == 'T' {
		if ts.campWho2 < 0 {
			return []string{c.Name + "：交給誰?（再按一個編號）"} // CAMP:36
		}
		out := []string{fmt.Sprintf("%s → %s（按字母選要給的那一格）",
			c.Name, g.members[ts.campWho2].Name)}
		return append(out, g.packLines(c)...)
	}
	if ts.campMode == 'E' {
		// CAMP:40/47 是兩段獨立的提問(`Weapon? ` / `Armor?  `),
		// 原版先問武器再問防具;引擎併成一步讓玩家自己選要裝哪一格。
		// CAMP:42/49「No Weapon」/「No Armor」—— 現在裝什麼要看得見,
		// 否則玩家得自己記得剛才裝了哪一格。
		return []string{c.Name + "：W)武器?　A)防具?",
			"目前　" + g.equippedName(c, c.Weapon, "無武器") +
				"　" + g.equippedName(c, c.Armor, "無防具"),
			"（`ITEMS.DAT` 沒有「武器還是防具」這個欄位,分類在呼叫端）"}
	}
	// CAMP:70「Letter?」—— 原版問完「哪一件」之後才問字母。
	// CAMP:41/48「 skips.」—— 原版在武器/防具那一步可以直接略過。
	head := fmt.Sprintf("%s 的背包（%s 字母?", c.Name, title)
	if ts.campMode == 'W' || ts.campMode == 'A' {
		head += "，Enter 略過。"
	}
	out := []string{head + "）"}
	return append(out, g.packLines(c)...)
}

// packLines 列出一個角色的背包。
func (g *Game) packLines(c original.Character) []string {
	var out []string
	for i, n := range c.Pack {
		if n == original.NotEquipped {
			continue
		}
		name := "（編號 " + fmt.Sprint(n) + ",查不到）"
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
			fmt.Sprintf("睡一晚回 %d 生命、%d 法力,並供餐（手冊 p.37）。",
				town.InnHealHP, town.InnHealSP),
		}
	case original.ShopHealer:
		lines := []string{
			fmt.Sprintf("價格倍率 %.2f　（治療所不受說服技能影響）", ts.shop.PriceMult),
			"",
			fmt.Sprintf("醫療　每點生命 %d　　解毒　%d",
				town.Price(town.HealPerHP, ts.shop.PriceMult),
				town.Price(town.UnpoisonPrice, ts.shop.PriceMult)),
			fmt.Sprintf("解除束縛　%d/等級　　復活　%d/等級", // TOWN:25「/lvl」
				town.Price(town.UnbindPerLv, ts.shop.PriceMult),
				town.Price(town.ResurrectPerLv, ts.shop.PriceMult)),
			"",
			// TOWN:26「Enter character # to be healed, (ESC exits)」
			"輸入要醫療的角色編號,(ESC離開)　再按 W 醫療 / P 解毒 / B 解除束縛 / R 復活",
		}
		for i, c := range g.members {
			lines = append(lines, fmt.Sprintf("%d) %s　生命 %d／%d　%s　治療費 %d",
				i+1, c.Name, c.HP, c.MaxHP, c.StatusName(),
				town.HealCost(c, town.HealWounds, ts.shop.PriceMult)))
		}
		return append(lines, "", town.ResurrectAssumption)
	case original.ShopTavern:
		lines := []string{
			// TOWN:55 的原句把兩個指令寫在一行(T)alk / B)uy food)。
			// 引擎沒有 T)alk 那一步 —— 傳聞直接印在下面,所以只留 B)。
			"你想要：T)與其他冒險者交談,B)購買食糧?(ESC離開)",
			// TOWN:58「One day's food for the party costs N gold.」
			fmt.Sprintf("隊伍一天的食糧花費 %d 金幣。目前 %d 份",
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
		// 措辭照 TOWN.tsv 第 34–37 列(F3):原版的招呼語是
		// 「Welcome to the Fighter's Guild.」/「… Wizard's Guild.」,
		// 選人那句是「Which character # seeks advancement in his art (ESC exits) ?」。
		guild, teaches := "戰士公會。", "戰士"
		if ts.shop.Extra == 1 {
			guild, teaches = "巫師公會。", "巫師"
		}
		lines := []string{
			"歡迎來到" + guild + "(位移 36 = " + fmt.Sprint(ts.shop.Extra) + ",只收" + teaches + ")",
			"哪位角色要精進技藝?(ESC離開)",
			"訓練免費,只看經驗夠不夠（手冊 p.37）。",
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
		return append(lines, "", town.GrowthNote)
	}
	return nil
}

// hunt 與 identify 是營地裡兩個「每天一次」的技能。docs/re/166。
//
// 打獵的擲骰讀出來了(docs/re/177 §4):`max(0, INT(RND × 16) − 6)`。
// ⚠ 補給品的**上限**仍未解(`ds:6F10` 沒有編譯期初值),所以這裡不夾。
//
// 鑑定的成功率解出來了(docs/re/189):`d100 ≤ 智能 × 4.5`,與道具無關。

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
	// ⚠ 收穫 0 是**失敗**,不是「成功但拿 0 份」——原版分成兩句話
	// (CAMP:65「The hunt was」+ CAMP:66/67「not successful.」/「successful!」)。
	if n := town.HuntYield(g.rand); n > 0 {
		g.group.Provisions += n
		ts.msg = fmt.Sprintf("%s：這次打獵有收穫!食糧 +%d（共 %d）", c.Name, n, g.group.Provisions)
	} else {
		ts.msg = c.Name + "：這次打獵沒有收穫。"
	}
}

func (g *Game) identify(who, slot int) {
	ts := g.town
	c := &g.members[who]
	item := c.Pack[slot]
	if gate := town.CanIdentify(*c, item); gate != town.SkillOK {
		// I)dentify 的「沒有技能」原文與 H)unt 不同(CAMP:61 vs CAMP:130),
		// 兩者共用同一個 SkillGate 值,這裡照呼叫端覆寫成 I)dentify 那句。
		msg := gate.String()
		if gate == town.SkillNoSkill {
			msg = "你沒有受過這方面的知識訓練!"
		}
		ts.msg = c.Name + "：" + msg
		return
	}
	c.SkillUsed = true
	g.syncMember(*c)
	// 成功率 = `d100 ≤ 智能 × 4.5`(docs/re/189)。⚠ 與道具本身無關。
	if !town.IdentifySucceeds(*c, g.rand) {
		ts.msg = c.Name + "：失敗" // CAMP:62「Failed」
		return
	}
	name := "那件東西"
	if it, ok := g.itemByIndex(item); ok {
		name = it.Name
	}
	ts.msg = c.Name + " 辨識了 " + name
}
