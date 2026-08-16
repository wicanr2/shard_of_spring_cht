package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/magic"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/ui"
)

// 補兩個流程缺口(docs/spec/19-coverage.md §2-1):
//
//  1. 戰鬥中的 `U)se an item` —— CMBT 有完整的字串鏈(索引 162–192),
//     引擎的按鍵處理完全沒有這個入口。
//  2. 藥劑「用在自己身上還是給別人」的分支 —— 營地與戰鬥各自有一句原文
//     (CAMP:92 的 `G)ive`、CMBT:165 的 `T)oss`,按鍵與措辭都不同,照抄不要統一),
//     兩邊都固定打自己,沒有問過。
//
// 走的路照 docs/spec/09-magic-items.md §5 既有的規則(編號 ≤ 26 不是魔法道具;
// `擲骰(100) ≤ 欄6` 才發動);套用效果沿用 internal/magic 既有的 Apply,
// 不另寫一套(docs/spec/16-camp-actions.md §5 的邊界)。

// potionPrompt 是「自己/給別人」子流程的狀態。營地與戰鬥各自一份
// (campPotion / combatPotion,見 Game 結構),不共用同一個實例 ——
// 兩邊的道具來源(original.Character 的背包)相同,但套用目標的型別不同
// (營地是 original.Character、戰鬥是 combat.Unit),硬併成一份會讓其中一邊
// 的欄位永遠用不到,不如各自輕量一份。
type potionPrompt struct {
	slot  int // 背包格
	stage int // 1 = 問 Y / G(或 Y / T);2 = 選目標
}

// useEntry 是道具選單裡的一格:背包格號 + 顯示名稱(未辨識的道具顯示成
// 未辨識,docs/spec/16 §3)。
type useEntry struct {
	slot int
	name string
}

// isCombatOnlyItem 猜「這件道具算不算戰鬥限定道具」——docs/spec/19-coverage.md
// §2-2:CAMP.EXE 會先判斷道具類別再擋(`That is a Combat Item!`,CAMP:100),
// 擋下之後不會問「自己/給別人」,直接回選單。但判斷式本身**沒有反組譯過**
// (另一個 agent 正在解),這裡先用一張具名的表頂著。
//
// ⚠ **這張表是猜的,不是讀到的判斷式。** 寫法照 docs/spec/16 §2
// (combatOnlySpell)的先例:魔法道具若觸發的法術屬於「戰鬥法術」
// (combatOnlySpell 判定的類別:群體/單體傷害、束縛、負威力的屬性增減),
// 就當作戰鬥限定道具擋在營地外 —— 沿用同一條分界,不是另外發明一條規則。
// 非魔法道具(編號 ≤ 26)沒有已知效果可以分類,一律當作不是戰鬥限定
// (原有的「不是魔法道具,什麼事也沒發生」訊息照樣會顯示)。
// ⛔ 之後 RE 有結果會來換掉這個函式,呼叫端(目前只有 campUseKey 一處)
// 不要散寫判斷式,只改這裡。
//
// ⚠ 戰鬥中的 `U)se an item` 沒有這道閘門 —— CMBT 的字串清冊裡找不到
// 對應的「這是戰鬥道具」訊息,推斷戰鬥限定道具本來就是設計成只能在戰鬥用,
// 不需要另外擋。
func (g *Game) isCombatOnlyItem(it original.Item) bool {
	if it.Index <= magic.MagicItemMin {
		return false
	}
	s, ok := g.spellByIndex(it.Col4)
	if !ok {
		return false
	}
	return combatOnlySpell(s)
}

// ── 營地:U)se an item 的「自己/給別人」子流程 ─────────────────────────

// useItemOn 讓 g.members[who] 用背包第 slot 格的道具,效果套用在
// g.members[target] 身上(who == target 時就是「用在自己身上」)。
//
// docs/spec/09 §5 的三條規則不變,只是把「目標是誰」從寫死的呼叫者
// 改成參數:呼叫端(useItem 自己用 / campPotionKey 選完目標後)決定傳誰進來。
func (g *Game) useItemOn(who, slot, target int) {
	ts := g.town
	c := &g.members[who]
	tgt := &g.members[target]
	idx := c.Pack[slot]
	name := g.itemDisplayName(*c, slot)
	self := who == target

	// ⚠ 編號 ≤ 26 不是魔法道具,不走發動這條路(docs/spec/09 §5)——
	// 不是「查不到效果」,是原版這個機制本來就不管這些編號。
	if idx <= magic.MagicItemMin {
		if self {
			ts.msg = fmt.Sprintf("%s 用了%s,你真的沒辦法用這個!", c.Name, name)
		} else {
			ts.msg = fmt.Sprintf("%s 把%s交給 %s,你真的沒辦法用這個!", c.Name, name, tgt.Name)
		}
		return
	}
	it, ok := g.itemByIndex(idx)
	if !ok {
		ts.msg = fmt.Sprintf("%s：%s 的資料查不到,用不出效果。", c.Name, name)
		return
	}
	if !magic.ItemTriggers(idx, it.Col6, g.rand) {
		if self {
			ts.msg = fmt.Sprintf("%s 用了%s,法術失效!", c.Name, name)
		} else {
			ts.msg = fmt.Sprintf("%s 把%s交給 %s,法術失效!", c.Name, name, tgt.Name)
		}
		return
	}
	s, ok := g.spellByIndex(it.Col4)
	if !ok {
		ts.msg = fmt.Sprintf("%s 用了%s,發動了,但對應的法術（編號 %d）查不到。", c.Name, name, it.Col4)
		return
	}
	invest := it.Col5

	if self {
		cUnit := campUnit(*c)
		r := magic.Apply(s, invest, &cUnit, []*combat.Unit{&cUnit})
		applyCampUnit(c, cUnit)
		g.syncMember(*c)
		ts.msg = fmt.Sprintf("%s 用了%s,發動了「%s」：%s", c.Name, name, s.Name, r.Message)
		return
	}

	// ⚠ 給別人用 vs 自己用有沒有效果差別**未解**——這裡是「目標不同、
	// 效果相同」的假設,兩條路都走同一個 magic.Apply(docs/spec/09 §5)。
	casterUnit := campUnit(*c)
	targetUnit := campUnit(*tgt)
	r := magic.Apply(s, invest, &casterUnit, []*combat.Unit{&targetUnit})
	applyCampUnit(tgt, targetUnit)
	g.syncMember(*tgt)
	ts.msg = fmt.Sprintf("%s 把%s交給 %s,發動了「%s」：%s", c.Name, name, tgt.Name, s.Name, r.Message) // CAMP:92 的 G)ive
}

// campPotionKey 處理營地「自己/給別人」子流程的一次按鍵。
//
// 按鍵是 **Y / G**(CAMP:92 的 `G)ive`)——跟戰鬥的 `T)oss` 不同,照抄不要統一。
//
// ⚠ 這裡不處理 ESC:town_scene.go 的 campSubKey 在鍵盤事件到達這裡之前
// 就攔截了 ESC(直接把 campMode 重置回選單),campCastKey 的每個子階段
// 也是同一個前提,不是這裡漏寫。g.campPotion 的殘留由 campUseKey 在
// 下一輪重新選人時清掉(見那裡的註解)。
func (g *Game) campPotionKey(k ebiten.Key) {
	ts := g.town
	p := g.campPotion
	switch p.stage {
	case 1:
		switch k {
		case ebiten.KeyY:
			g.campPotion = nil
			g.useItemOn(ts.campWho, p.slot, ts.campWho)
			ts.campMode, ts.campWho = 0, -1
		case ebiten.KeyG:
			p.stage = 2
		}
	case 2:
		i := int(k - ebiten.Key1)
		if i < 0 || i >= len(g.members) {
			return
		}
		who := ts.campWho
		g.campPotion = nil
		g.useItemOn(who, p.slot, i)
		ts.campMode, ts.campWho = 0, -1
	}
}

// campPotionLines 畫營地「自己/給別人」子流程的內容。
func (g *Game) campPotionLines(ts *townState) []string {
	p := g.campPotion
	c := g.members[ts.campWho]
	name := g.itemDisplayName(c, p.slot)
	switch p.stage {
	case 1:
		// CAMP:92 的原句只說「the potion」,不說是哪一件 —— 這裡把剛選的
		// 那件的名字放在句子前面,問句本身照原版。
		// CAMP:92 + 93「 (ESC cancels)」
		return []string{name + "　你要把藥劑用在 Y)自己身上,還是 G)交給另一位角色?(ESC取消)"}
	case 2:
		out := []string{"要交給哪位角色?　角色 #："} // CAMP:95/96/97
		for i, m := range g.members {
			out = append(out, fmt.Sprintf("%d) %s", i+1, m.Name))
		}
		return out
	}
	return nil
}

// ── 戰鬥:U)se an item(docs/spec/12-combat-board.md §5.3、
// docs/spec/19-coverage.md §2-1)────────────────────────────────────

// useItemsFor 列出一位角色背包裡能選的道具。未辨識的也列出來,顯示成
// 未辨識的名稱(docs/spec/16 §3 的既有規則,戰鬥沿用同一套)。
func (g *Game) useItemsFor(c original.Character) []useEntry {
	var out []useEntry
	for i, v := range c.Pack {
		if v == original.NotEquipped {
			continue
		}
		out = append(out, useEntry{slot: i, name: g.itemDisplayName(c, i)})
	}
	return out
}

// openUseItem 打開戰鬥的 `U)se an item` 選單。回 false 表示這次沒開成
// (沒有人可以動、行動點數不足、或背包是空的)——這三種情況都**不消耗
// 行動點數也不結束回合**,跟 openCast 的失敗路徑對稱。
func (g *Game) openUseItem() bool {
	if g.field == nil {
		return false
	}
	i := g.actor
	if i < combat.PartyBase || i >= combat.PartyBase+combat.PartyMax {
		g.field.Log = append(g.field.Log, "現在沒有人可以行動")
		return false
	}
	if g.points[i] < rules.ActUse.Cost() {
		// CMBT:117「to use.」——「行動點數不足」那句的句尾(見 cast_scene.go)。
		g.field.Log = append(g.field.Log, g.field.Units[i].Name+useNoPoints)
		return false
	}
	idx := i - combat.PartyBase
	if idx >= len(g.members) {
		return false
	}
	list := g.useItemsFor(g.members[idx])
	if len(list) == 0 {
		g.field.Log = append(g.field.Log,
			g.field.Units[i].Name+"：這位角色沒有可用的道具!") // CMBT:162
		return false
	}
	g.useUnit, g.useList = i, list
	return true
}

// pickUseItem 選第 n 件道具。不是魔法道具的話直接結算(CMBT:164「不是有效
// 的魔法道具!」);是魔法道具的話先問自己/丟給別人(docs/spec/19-coverage.md
// §2-1)——跟營地一樣,先確認「這件東西真的會做什麼」才問目標。
func (g *Game) pickUseItem(n int) {
	if n < 0 || n >= len(g.useList) {
		return
	}
	slot := g.useList[n].slot
	g.useList = nil
	idx := g.useUnit - combat.PartyBase
	if idx < 0 || idx >= len(g.members) {
		return
	}
	itemIdx := g.members[idx].Pack[slot]
	if itemIdx <= magic.MagicItemMin {
		g.finishUseItem(slot, g.useUnit) // 反正沒有效果,目標是誰不影響結果
		return
	}
	g.combatPotion = &potionPrompt{slot: slot, stage: 1}
}

// combatPotionKey 處理戰鬥中「自己/丟給別人」子流程的一次按鍵。
// 回 true 表示這個鍵被吃掉了。
//
// ⚠ 按鍵是 **Y / T**(CMBT:165 的 `T)oss`),跟營地的 `G)ive` 不同,照抄不要統一。
func (g *Game) combatPotionKey(k ebiten.Key) bool {
	p := g.combatPotion
	if p == nil {
		return false
	}
	switch p.stage {
	case 1:
		switch k {
		case ebiten.KeyY:
			g.combatPotion = nil
			g.finishUseItem(p.slot, g.useUnit)
			return true
		case ebiten.KeyT:
			p.stage = 2
			return true
		}
		return false
	case 2:
		i := int(k - ebiten.Key1)
		if i < 0 || i >= len(g.members) {
			return false
		}
		target := combat.PartyBase + i
		g.combatPotion = nil
		g.finishUseItem(p.slot, target)
		return true
	}
	return false
}

// finishUseItem 是道具流程的終點:對 field.Units[target] 套用
// g.useUnit 背包第 slot 格那件道具的效果,扣點數、結束 g.useUnit 的回合。
//
// ⚠ 手冊 p.35:施法、用物品、降魔三條都寫「會取消行動能力」,而且是
// **做了這個動作就結束**,不是「效果真的發動才結束」——不管道具是不是
// 魔法道具、擲骰有沒有過,回合都在這裡結束(docs/spec/12 §2)。
func (g *Game) finishUseItem(slot, target int) {
	f := g.field
	casterIdx := g.useUnit
	idx := casterIdx - combat.PartyBase
	if idx < 0 || idx >= len(g.members) {
		return
	}
	c := g.members[idx]
	itemIdx := c.Pack[slot]
	name := g.itemDisplayName(c, slot)
	caster := &f.Units[casterIdx]
	self := target == casterIdx

	switch {
	case itemIdx <= magic.MagicItemMin:
		// docs/spec/09 §5:編號 ≤ 26 不走發動這條路。
		if self {
			f.Log = append(f.Log, fmt.Sprintf("%s 用了%s,不是有效的魔法道具!", caster.Name, name)) // CMBT:164
		} else {
			f.Log = append(f.Log, fmt.Sprintf("%s 把%s丟給 %s,不是有效的魔法道具!",
				caster.Name, name, f.Units[target].Name))
		}
	default:
		it, ok := g.itemByIndex(itemIdx)
		switch {
		case !ok:
			f.Log = append(f.Log, fmt.Sprintf("%s：%s 的資料查不到,用不出效果。", caster.Name, name))
		case !magic.ItemTriggers(itemIdx, it.Col6, g.rand):
			if self {
				f.Log = append(f.Log, fmt.Sprintf("%s 用了%s,法術失效!", caster.Name, name))
			} else {
				f.Log = append(f.Log, fmt.Sprintf("%s 把%s丟給 %s,法術失效!",
					caster.Name, name, f.Units[target].Name))
			}
		default:
			s, ok := g.spellByIndex(it.Col4)
			if !ok {
				f.Log = append(f.Log, fmt.Sprintf("%s 用了%s,發動了,但對應的法術（編號 %d）查不到。",
					caster.Name, name, it.Col4))
			} else {
				// ⚠ 給別人用 vs 自己用有沒有效果差別**未解**——這裡是「目標
				// 不同、效果相同」的假設,兩條路都走同一個 magic.Apply。
				r := magic.Apply(s, it.Col5, caster, []*combat.Unit{&f.Units[target]})
				if self {
					f.Log = append(f.Log, fmt.Sprintf("%s 用了%s,發動了「%s」：%s", caster.Name, name, s.Name, r.Message))
				} else {
					f.Log = append(f.Log, fmt.Sprintf("%s 把%s丟給 %s,發動了「%s」：%s", // CMBT:165 的 T)oss
						caster.Name, name, f.Units[target].Name, s.Name, r.Message))
				}
			}
		}
	}

	// 用物品的成本跟施法一樣是 3 點,而且做完直接結束回合(docs/spec/12 §2)。
	if casterIdx == g.actor {
		g.points[g.actor] = 0
		g.nextActor()
	}
}

// drawUseMenu 畫戰鬥中 U)se an item 的選單,以及「自己/丟給別人」子流程。
// 版面照 cast_scene.go 的 drawCastMenu(蓋掉訊息列同一塊)。
func (g *Game) drawUseMenu(dst *ebiten.Image) {
	if g.panel == nil || g.field == nil {
		return
	}
	if len(g.useList) == 0 && g.combatPotion == nil {
		return
	}
	clearMessage(dst) // message.go
	p := g.panel
	lh := p.LineHeight()
	x := float64(layout.Message.X + ui.PanelPad)
	y := float64(layout.Message.Y + ui.PanelPad)

	if g.combatPotion != nil {
		idx := g.useUnit - combat.PartyBase
		if idx < 0 || idx >= len(g.members) {
			return
		}
		name := g.itemDisplayName(g.members[idx], g.combatPotion.slot)
		switch g.combatPotion.stage {
		case 1:
			// CMBT:165 + 167「 (ESC cancels)」
			p.Draw(dst, name+"　你要把藥劑用在 Y)自己身上,還是 T)丟給另一位角色?(ESC取消)",
				x, y)
		case 2:
			// CMBT:168/169/170 + 115/116「Who do you want to use it on?」
			p.Draw(dst, "你想對誰使用?　要丟給哪位角色?　角色 #：", x, y)
			y += lh
			for i, m := range g.members {
				// ⚠ 不要把 "%d) %s" 這種格式化字串直接寫在 Draw() 呼叫裡 ——
				// tools/check_ui_language.py 掃的是「傳給繪製函式的字串常數」,
				// 先組成變數再畫,跟 combat_scene.go 的 unitLine() 同一種寫法。
				line := fmt.Sprintf("%d) %s", i+1, m.Name)
				p.Draw(dst, line, x, y)
				y += lh
			}
		}
		return
	}

	// CMBT:163「Use:  (ESC exits)」
	p.Draw(dst, g.field.Units[g.useUnit].Name+"　使用：(ESC離開)", x, y)
	y += lh
	for i, e := range g.useList {
		if y > float64(layout.Message.Y+layout.Message.H)-lh*2 {
			p.Draw(dst, fmt.Sprintf("…還有 %d 個(未分頁)", len(g.useList)-i), x, y)
			break
		}
		p.Draw(dst, fmt.Sprintf("%c) %s", 'A'+i, e.name), x, y)
		y += lh
	}
}
