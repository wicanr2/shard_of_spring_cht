package main

import (
	"fmt"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/town"
)

// 經驗值。**存在原版記錄裡**:`CHARS.DAT` 位移 90–93,MBF 單精度。
// docs/re/150、docs/formats/01。
//
// 五處程式碼、三支模組都讀這個位移,其中兩處緊接在載入 `Exp.  ` 標籤之後 ——
// 是**讀端**給出的答案,不是從資料側猜的。
//
// ⚠ 本引擎一度把經驗值放在自己的旁掛檔 `save/exp.json`,理由是「位移未解、
// 不要亂寫進原版記錄」。那個理由在位移解出來之後就不成立了 ——
// 留著旁掛檔會讓**同一個量有兩個真相**,而兩邊不同步時沒有任何一邊會報錯。

// charExp 回傳這個角色的經驗值(取整數,畫面只印整數)。
func (g *Game) charExp(c original.Character) int { return int(c.Exp) }

// awardExp 把一場戰鬥的經驗值分給有資格的成員。
//
// 資格由**戰場上的單位**決定,不是由名冊決定(docs/re/150 §2.1):
// 朝向 > 0(還在場上)且 狀態 < 5(未陣亡)。
//
// ⚠ **逃走的人拿不到。** 逃跑在原版是「把朝向清成 0」(docs/re/103),
// 而逃走的人**生命值仍然大於 0** —— 用生命值判斷會多發給他們,
// 而畫面上完全看不出來。
//
// 回傳分給每個人的數量與說明文字。
func (g *Game) awardExp(units []combat.Unit) (int, string) {
	total := combat.TotalExp(units)
	if total == 0 {
		return 0, ""
	}
	var idx []int // 有資格的成員在 g.members 裡的索引
	for u := combat.PartyBase; u < combat.PartyBase+combat.PartyMax; u++ {
		i := u - combat.PartyBase
		if u < len(units) && i < len(g.members) && units[u].EarnsExp() {
			idx = append(idx, i)
		}
	}
	share := combat.ExpShare(total, len(idx))
	for _, i := range idx {
		g.members[i].Exp += float64(share)
		g.syncMember(g.members[i])
	}
	// F3:照原版的結算措辭(CMBT:56/57 `Experience:` + ` (per character)`)。
	// ⚠ 原版**只印每人所得**,不印總額與人數 —— 這裡跟著只印每人所得,
	// 不多印玩家在原版看不到的數字。
	return share, fmt.Sprintf("經驗：%d(每位角色)", share)
}

// awardSpoils 發經驗與金幣。原版的結算畫面兩者一起印
// (`Experience:` 與 `Gold:`,docs/re/95 §3)。
//
// 金幣照 docs/re/207:每隻怪物 `INT(1.7^階級 + RND × 2.1^階級 + 1)`,加起來。
func (g *Game) awardSpoils(units []combat.Unit) string {
	_, msg := g.awardExp(units)
	gold := combat.TotalGold(units, combat.NewRand(g.spoilSeed()))
	if gold > 0 {
		// CMBT:58–63:原版**先問要不要撿**,不是直接進口袋。
		// 金幣先掛在 pendingGold,答 Y 才入帳。
		g.pendingGold = gold
		if msg != "" {
			msg += "；"
		}
		// CMBT:58「Gold:」+ 62/63「Do you take it?」「(Y/N)」。
		// ⚠ CMBT:60「 found 」**不在這一句** —— 那是道具那一條路的字
		// (docs/re/200 §1.1),先前接錯了。
		msg += fmt.Sprintf("金幣：%d　要撿嗎?(Y/N)", gold)
	}
	// 戰後掉落:編號跟著這場的金幣走(docs/re/200 §3.2)。
	// ⚠ 用**另一顆種子**,否則掉落會與金幣的擲骰互相糾纏。
	if gold > 0 {
		if m := g.offerLoot(gold); m != "" {
			if msg != "" {
				msg += "；"
			}
			msg += m
		}
	}
	return msg
}

// offerLoot 擲一件掉落道具並問「要撿嗎?」。回空字串表示沒有人收得下。
//
// 收件人是**由隊首往隊尾**第一個「狀態 < 2 且背包還有空格」的隊員
// (`CMBT 0x130EE` 起的兩層遞增迴圈)—— 與迷宮那一支的**由隊尾往隊首**
// 不同(docs/re/168 §1),兩邊各自照抄。
func (g *Game) offerLoot(gold int) string {
	item := combat.LootIndex(gold, combat.NewRand(g.spoilSeed()+1))
	for i := range g.members {
		if g.members[i].Status >= combat.StatusIncapacitated {
			continue
		}
		for slot := 0; slot < original.PackSlots; slot++ {
			if g.members[i].Pack[slot] != original.NotEquipped {
				continue
			}
			g.pendingLoot = &pendingLoot{who: i, slot: slot, item: item}
			// CMBT:61「finds」+ 62/63。⚠ 用**別名**(小寫名)——
			// 撿來的東西是未鑑定的(docs/re/200 §1.2)。
			return fmt.Sprintf("%s 找到了 %s。　要撿嗎?(Y/N)",
				g.members[i].Name, g.lootDisplayName(item))
		}
	}
	return "" // 全隊背包都滿了 —— 原版這時什麼都不問
}

// lootDisplayName 回傳未鑑定時該顯示的名字。
func (g *Game) lootDisplayName(index int) string {
	if it, ok := g.itemByIndex(index); ok {
		if it.Alias != "" {
			return it.Alias
		}
		return it.Name
	}
	return fmt.Sprintf("編號 %d 的東西", index)
}

// takeLoot 回答掉落的「要撿嗎?」。不答就是不撿,與金幣同一條規矩。
func (g *Game) takeLoot(yes bool) {
	p := g.pendingLoot
	g.pendingLoot = nil
	if p == nil || !yes {
		return
	}
	g.members[p.who].Pack[p.slot] = p.item
	// 撿來的東西是未鑑定的(docs/re/168 §2)。
	town.SetIdentified(&g.members[p.who], p.slot, false)
	g.syncMember(g.members[p.who])
	if g.field != nil {
		g.field.Log = append(g.field.Log,
			g.members[p.who].Name+" 收下了 "+g.lootDisplayName(p.item))
	}
}

// takeGold 回答戰後的「要撿嗎?」。
//
// ⚠ 不答就不入帳,**離開戰鬥時視為不撿** —— 靜靜地幫玩家撿起來會讓
// 那句問話變成假的。
func (g *Game) takeGold(yes bool) {
	if g.pendingGold <= 0 {
		return
	}
	if yes {
		g.group.Gold += float64(g.pendingGold)
		if g.field != nil {
			g.field.Log = append(g.field.Log,
				fmt.Sprintf("撿起了 %d 金幣", g.pendingGold))
		}
	}
	g.pendingGold = 0
}

// spoilSeed 讓同一場戰鬥的戰利品可重現 —— 用戰場的亂數來源會把
// 「同一顆種子跑兩次結果相同」這條驗收(docs/spec/07 §2)弄壞,
// 因為結算的擲骰次數取決於怪物數。
func (g *Game) spoilSeed() uint64 {
	return uint64(g.party.X)<<32 | uint64(g.party.Y)<<16 | uint64(g.party.Encounter)
}

// pendingLoot 是「擲出來了、還沒回答要不要撿」的那一件。
type pendingLoot struct {
	who  int // 收件人在 g.members 的索引
	slot int // 要放進背包第幾格
	item int // ITEMS.DAT 編號
}
