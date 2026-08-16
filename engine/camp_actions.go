package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/magic"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/town"
)

// 營地三個未實作指令(P)rint / C)ast / U)se)+ 施法投入點數。
// docs/spec/16-camp-actions.md。
//
// 依 docs/spec/16 §5 的邊界:規則(internal/magic、internal/town)已經解好、
// 有測試,這裡只接線 —— 只新增,不改既有函式的簽章。

// ── E5:施法的投入點數(docs/spec/16 §1)──────────────────────────────
//
// 原版 CAMP.EXE 的提示是 `'Spell Pts ?'`。三條公式(docs/spec/09 §2、§4)
// 各吃不同的東西,已經在 internal/magic 裡:
//
//	等級     = INT(投入 ÷ 欄5)      magic.Level
//	威力     = 欄4 × 投入            magic.Power   (不是等級!)
//	狀態強度 = 欄5 ÷ 投入            magic.StatusMagnitude(投得多、值小)
//
// 這裡只負責「玩家怎麼把投入點數打進去」與「打完之後套哪一個閘門」。

// notWizardMsg 與 magic.Fail 的其他三句共用同一種「原版四句話各自獨立」
// 的寫法(docs/spec/16 §2)——但「不是巫師」本身不是 magic.Fail 的值
// (那邊的 Fail 需要先有一個 Spell 才能判),所以另外開一個常數,
// 兩處(選施法者時的早退、campCastCheck 的完整判斷)共用同一句,不會各寫一次。
const notWizardMsg = "這位角色不是巫師。" // CAMP:133

// combatOnlySpell 判斷這個法術算不算「戰鬥法術」——營地放不出來
// (原版 CAMP.EXE 的 `'That is a combat spell !'`,docs/spec/16 §2)。
//
// ⚠ **這張表是從效果類別推的,不是讀到的判斷式。** CAMP.EXE 怎麼判斷
// 「這是戰鬥法術」沒有反組譯過(docs/spec/16 §2:「沒有直接讀到」)。
// 推論依據兩條:
//
//   - 類別 1/2(群體/單體傷害)與類別 11(束縛)本質上要打敵人,
//     營地沒有戰場、沒有敵人可以選;
//   - 類別 3–6(屬性增減)欄4 可正可負,cast_scene.go 的 castAt() 已經有
//     「威力為負 = 攻擊/削弱」的判斷(docs/spec/09 §3、docs/re/171)——
//     這裡沿用同一條分界,不是另外發明一條規則。
//
// 不在這張表裡的類別(4/5/6 正威力的增益、7/8/9/10/12/13)當作「不是
// 戰鬥法術」。⛔ 這張表**不是** RE 結論,是實作決定,寫法上不要看起來像。
func combatOnlySpell(s original.Spell) bool {
	switch s.Effect {
	case magic.EffGroupDamage, magic.EffSingleDamage, magic.EffBind:
		return true
	case magic.EffToHit, magic.EffStrength, magic.EffHitPoints, magic.EffSpeed:
		return s.Power < 0
	}
	return false
}

// campSpellAllowed 判斷這個法術本身(先不管誰施、投多少)能不能在營地放。
// 回空字串 = 可以。
//
// ⚠ **`EffProtect`(防護類)不歸「戰鬥法術」,是另一個原因**:防護的效果
// 存在 `combat.Unit.ArmSkin`,而 `original.Character` 沒有這個欄位可以
// 持久化(build.go 每次開戰都從技能旗標重算 ArmSkin,不是從法術效果來)。
// 這是**本引擎的資料結構限制**,不是規則限制,也不是 RE 結論 ——
// 戰鬥中施放同一類法術完全沒問題,因為那時 Unit 活在那一場戰鬥裡。
// 用跟「戰鬥法術」不同的訊息講清楚,不要混成同一句。
func campSpellAllowed(s original.Spell) string {
	if combatOnlySpell(s) {
		return "那是戰鬥法術!" // CAMP:81
	}
	if s.Effect == magic.EffProtect {
		return "「" + s.Name + "」的防護效果沒辦法保存到下一場戰鬥,本引擎在營地不支援施放。"
	}
	return ""
}

// campCastCheck 是 C)ast spell 在營地能不能放這個法術的完整判斷:
// 合併 campSpellAllowed(本引擎推的表)與 magic.CanCast 既有的三個閘門
// (不是巫師 / 沒有這一系符文技能 / 法力不足 / 投不到一級)。
// 回空字串 = 可以放。
//
// docs/spec/16 §6 驗收 #1/#2/#3 都靠這個函式的行為,所以它是測試的重點。
func campCastCheck(c original.Character, s original.Spell, invest int) string {
	if c.Class != magic.WizardClass {
		return notWizardMsg
	}
	if msg := campSpellAllowed(s); msg != "" {
		return msg
	}
	if f := magic.CanCast(c, s, invest); f != magic.OK {
		return f.String()
	}
	return ""
}

// campMaxInvestDigits 是投入點數輸入框允許的最大位數。法力點數全遊戲
// 沒有三位數的角色(手冊 p.48/49 的成長表封頂個位數乘級數),三位數綽綽有餘,
// 純粹是防止畫面被打爆。
const campMaxInvestDigits = 3

// campCastKey 處理 `C)ast spell` 的一次按鍵。四個階段:
//
//	0(campWho<0)   選施法者(原版 'Character # to cast spell')
//	1(castStage=1) 選法術(原版 'Cast what spell?')
//	2(castStage=2) 輸入投入點數(原版 'Spell Pts ?')
//	3(castStage=3) 選目標(原版 'Character # to cast on ?',
//	               三個地方各問一次,單一目標 —— docs/re/generated-string-refs.json)
func (g *Game) campCastKey(k ebiten.Key) {
	ts := g.town

	if ts.campWho < 0 {
		i := int(k - ebiten.Key1)
		if i < 0 || i >= len(g.members) {
			return
		}
		c := g.members[i]
		if c.Class != magic.WizardClass {
			ts.msg = c.Name + "：" + notWizardMsg
			ts.campMode, ts.campWho = 0, -1
			return
		}
		ts.campWho, ts.castStage = i, 1
		ts.msg = ""
		return
	}

	switch ts.castStage {
	case 1: // 選法術
		n := int(k - ebiten.KeyA)
		if n < 0 || n >= len(g.spells) || n >= 26 {
			return
		}
		s := g.spells[n]
		if msg := campSpellAllowed(s); msg != "" {
			ts.msg = s.Name + "：" + msg
			ts.campMode, ts.campWho, ts.castStage = 0, -1, 0
			return
		}
		ts.castSpell, ts.castInput, ts.castStage = s, "", 2

	case 2: // 輸入投入點數
		switch {
		case k >= ebiten.KeyDigit0 && k <= ebiten.KeyDigit9:
			if len(ts.castInput) < campMaxInvestDigits {
				ts.castInput += string(rune('0' + int(k-ebiten.KeyDigit0)))
			}
		case k == ebiten.KeyBackspace && ts.castInput != "":
			ts.castInput = ts.castInput[:len(ts.castInput)-1]
		case k == ebiten.KeyEnter, k == ebiten.KeyKPEnter:
			invest, _ := strconv.Atoi(ts.castInput) // 空字串 → 0,交給下面的閘門判
			c := g.members[ts.campWho]
			if msg := campCastCheck(c, ts.castSpell, invest); msg != "" {
				ts.msg = c.Name + "：" + msg
				ts.campMode, ts.campWho, ts.castStage = 0, -1, 0
				return
			}
			ts.campWho2, ts.castStage = -1, 3
		}

	case 3: // 選目標
		i := int(k - ebiten.Key1)
		if i < 0 || i >= len(g.members) {
			return
		}
		g.castInCamp(ts.campWho, i)
		ts.campMode, ts.campWho, ts.campWho2, ts.castStage = 0, -1, -1, 0
	}
}

// castInCamp 施展 ts.castSpell:施法者 g.members[casterIdx],
// 目標 g.members[targetIdx]。docs/spec/16 §2:「營地沒有戰場,
// 所以目標是隊員」。**沿用 internal/magic 既有的 Apply,不另寫一套** ——
// 用 campUnit/applyCampUnit 把 Character 包成一個用完即丟的 combat.Unit。
func (g *Game) castInCamp(casterIdx, targetIdx int) {
	ts := g.town
	s := ts.castSpell
	invest, _ := strconv.Atoi(ts.castInput)

	caster := &g.members[casterIdx]
	target := &g.members[targetIdx]

	casterUnit := campUnit(*caster)
	targetUnit := campUnit(*target)
	r := magic.Apply(s, invest, &casterUnit, []*combat.Unit{&targetUnit})

	caster.SP -= invest
	if caster.SP < 0 {
		caster.SP = 0
	}
	applyCampUnit(target, targetUnit)
	g.syncMember(*caster)
	g.syncMember(*target)

	verb := "對 " + target.Name + " "
	if casterIdx == targetIdx {
		verb = "對自己 "
	}
	ts.msg = fmt.Sprintf("%s %s施放「%s」(投入 %d)：%s",
		caster.Name, verb, s.Name, invest, r.Message)
}

// campUnit 把一個 Character 包成一個用完即丟的 combat.Unit,純粹是為了
// 讓既有的 magic.Apply 能在營地(沒有 combat.Field)也能跑。
//
// ⚠ **只搬「法術效果真的會動到」的欄位。** ArmSkin 沒有搬 ——
// Character 沒有這一欄(見 campSpellAllowed 的說明),所以 EffProtect
// 從一開始就被擋在營地施放清單外,這裡不需要處理它。
func campUnit(c original.Character) combat.Unit {
	return combat.Unit{
		Name: c.Name, HP: c.HP, Str: c.Str, ToHit: c.ToHit, Speed: c.Speed,
		Status: c.Status, StatMag: c.StatMag,
	}
}

// applyCampUnit 把 campUnit 施法後的結果寫回 Character。
//
// ⚠ **生命值夾在 [0, MaxHP]** —— 跟 internal/town/heal.go 的
// `c.HP = c.MaxHP` 同一個上限來源,magic.Apply 本身只夾下限(Unit 沒有
// MaxHP 可以夾上限)。
func applyCampUnit(c *original.Character, u combat.Unit) {
	c.HP = u.HP
	if c.HP < 0 {
		c.HP = 0
	}
	if c.MaxHP > 0 && c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	c.Str, c.ToHit, c.Speed = u.Str, u.ToHit, u.Speed
	c.Status, c.StatMag = u.Status, u.StatMag
}

// campCastLines 畫 C)ast spell 各階段的內容。
func (g *Game) campCastLines(ts *townState) []string {
	if ts.campWho < 0 {
		out := []string{campSelectPrompt('C')}
		for i, c := range g.members {
			out = append(out, fmt.Sprintf("%d) %s", i+1, c.Name))
		}
		return out
	}
	caster := g.members[ts.campWho]
	switch ts.castStage {
	case 1:
		out := []string{caster.Name + "：施放哪個法術?"} // CAMP:76
		for i, s := range g.spells {
			if i >= 26 {
				out = append(out, fmt.Sprintf("…還有 %d 個(未列出)", len(g.spells)-26))
				break
			}
			out = append(out, fmt.Sprintf("%c) %s（每級 %d 點法力）", 'A'+i, s.Name, s.UnitCost))
		}
		return out
	case 2:
		return []string{
			fmt.Sprintf("%s：花費幾點法力?（目前法力 %d／%d）", caster.Name, caster.SP, caster.MaxSP), // CAMP:83
			"輸入數字,Enter 確認、Backspace 修改：" + ts.castInput + "_",
		}
	case 3:
		out := []string{fmt.Sprintf("%s 對哪一位角色施放「%s」?", caster.Name, ts.castSpell.Name)}
		for i, c := range g.members {
			out = append(out, fmt.Sprintf("%d) %s", i+1, c.Name))
		}
		return out
	}
	return nil
}

// ── E2:U)se an item(docs/spec/16 §3)────────────────────────────────
//
// 選人 → 列出背包 → 選一件 → 套用。⚠ 未辨識的道具照樣可以用,
// 但顯示成未辨識的名稱(docs/spec/11 §4 的四個寫入端之一)。
// 魔法道具的發動走 docs/spec/09 §5:編號 ≤ 26 不走這條路;
// `擲骰(100) ≤ 欄6`(magic.ItemTriggers,已經有測試)。

func (g *Game) campUseKey(k ebiten.Key) {
	ts := g.town
	if ts.campWho < 0 {
		// 防呆:town_scene.go 的 campSubKey 在按到 ESC 時直接把 campMode
		// 重置回選單,不會經過下面「自己/給別人」子流程的收尾 —— 新一輪
		// 選人之前先清掉上一輪可能留下的殘留(use_item.go 的 campPotion)。
		g.campPotion = nil
		i := int(k - ebiten.Key1)
		if i < 0 || i >= len(g.members) {
			return
		}
		ts.campWho, ts.msg = i, ""
		return
	}
	if g.campPotion != nil {
		g.campPotionKey(k) // use_item.go:「自己/給別人」子流程
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

	// 「那是戰鬥用道具!」閘門(docs/spec/19-coverage.md §2-2,CAMP:100)。
	// use_item.go 的 isCombatOnlyItem 是**猜的**,不是讀到的判斷式。
	if it, ok := g.itemByIndex(c.Pack[slot]); ok && g.isCombatOnlyItem(it) {
		ts.msg = c.Name + "：那是戰鬥用道具!"
		ts.campMode, ts.campWho = 0, -1
		return
	}

	if c.Pack[slot] <= magic.MagicItemMin {
		// 不是魔法道具,不會有「給誰用」的差別,維持原本的單階段流程。
		g.useItem(ts.campWho, slot)
		ts.campMode, ts.campWho = 0, -1
		return
	}

	// 魔法道具:先問「自己/給別人」(docs/spec/19-coverage.md §2-1,CAMP:92)。
	g.campPotion = &potionPrompt{slot: slot, stage: 1}
}

// itemDisplayName 回傳一件道具在畫面上該顯示的名字 ——
// 未辨識就顯示成「未辨識的道具」,不管它實際是什麼(docs/spec/16 §3)。
func (g *Game) itemDisplayName(c original.Character, slot int) string {
	n := c.Pack[slot]
	if n == original.NotEquipped {
		return ""
	}
	if !town.IsIdentified(c, slot) {
		return "未辨識的道具"
	}
	if it, ok := g.itemByIndex(n); ok {
		return it.Name
	}
	return fmt.Sprintf("（編號 %d,查不到）", n)
}

// spellByIndex 依 SPELLS.DAT 的編號查一個法術。魔法道具的欄4 存的就是這個
// 編號(docs/spec/09 §5)。
func (g *Game) spellByIndex(n int) (original.Spell, bool) {
	for _, s := range g.spells {
		if s.Index == n {
			return s, true
		}
	}
	return original.Spell{}, false
}

// useItem 讓 g.members[who] 用背包第 slot 格的道具在自己身上。
//
// ⚠ docs/spec/19-coverage.md §2-1 補上「給別人用」之前,這是唯一的路徑;
// 現在是 useItemOn(who, slot, who)(use_item.go)的簡寫,保留給
// 「非魔法道具」的單階段流程與既有測試用 —— 行為與訊息完全不變。
func (g *Game) useItem(who, slot int) {
	g.useItemOn(who, slot, who)
}

// campUseLines 畫 U)se an item 的內容。
func (g *Game) campUseLines(ts *townState) []string {
	if ts.campWho < 0 {
		out := []string{campSelectPrompt('U')}
		for i, c := range g.members {
			out = append(out, fmt.Sprintf("%d) %s", i+1, c.Name))
		}
		return out
	}
	if g.campPotion != nil {
		return g.campPotionLines(ts) // use_item.go
	}
	c := g.members[ts.campWho]
	out := []string{c.Name + " 的背包（按字母選要用的道具）"}
	n := len(out)
	for i, v := range c.Pack {
		if v == original.NotEquipped {
			continue
		}
		out = append(out, fmt.Sprintf("%c) %s", 'A'+i, g.itemDisplayName(c, i)))
	}
	if len(out) == n {
		out = append(out, "這位角色沒有可用的道具!") // CAMP:90
	}
	return out
}

// ── E3:P)rint char(s)(docs/spec/16 §4,docs/re/166 §4)───────────────
//
// 原版驅動並列埠印表機;CLAUDE.md §1.2 的邊界是**不做印表機**,
// 改成把同一份內容顯示在畫面上。版面照原版字串:
//
//	Party #      Location:   Wilderness
//	Level:       Hit Pts.    Spell Pts.   Experience   Status
//	Skills:
//	Items:
//
// ⚠ `9` = 整隊,不是第 9 個人(隊伍最多 5 人,不會撞號)。

// printWholeParty 是 campWho 的一個哨兵值,代表玩家按了 `9`(整隊),
// 要跟「還沒選」(-1)與合法的成員索引(0–4)分開。
const printWholeParty = -2

func (g *Game) campPrintKey(k ebiten.Key) {
	ts := g.town
	if ts.campWho == -1 {
		if k == ebiten.Key9 {
			ts.campWho = printWholeParty
			return
		}
		i := int(k - ebiten.Key1)
		if i < 0 || i >= len(g.members) {
			return
		}
		ts.campWho = i
		return
	}
	// 結果已經顯示 —— 任意鍵回營地選單(不是印表機,看完就結束)。
	ts.campMode, ts.campWho = 0, -1
}

// campPrintLines 畫 P)rint 的內容。
func (g *Game) campPrintLines(ts *townState) []string {
	if ts.campWho == -1 {
		return []string{
			"輸入要列印的角色編號。",
			"（ESC 離開,或按 9 印整隊）",
		}
	}
	var out []string
	if ts.campWho == printWholeParty {
		for i, c := range g.members {
			if i > 0 {
				out = append(out, "")
			}
			out = append(out, g.printSheet(c)...)
		}
	} else {
		out = g.printSheet(g.members[ts.campWho])
	}
	return append(out, "", "（按任意鍵返回）")
}

// printSheet 是原版 `P)rint` 版面的中文化(docs/re/166 §4)。
//
// ⚠ 「Location: Wilderness」是**照字面搬的常數字串** —— 程式碼那一段
// 沒有讀過(docs/re/166 §4:「程式碼沒有讀,本節只是字串盤點」),
// 不知道它是不是會依地點變化;沒有相反證據前,不自己加一個沒讀過的判斷式。
func (g *Game) printSheet(c original.Character) []string {
	sp := "—"
	if c.MaxSP > 0 {
		sp = fmt.Sprintf("%d／%d", c.SP, c.MaxSP)
	}
	st := c.StatusName()
	if st == "" {
		st = "正常"
	}
	out := []string{
		fmt.Sprintf("第 %d 隊　　位置：野外", g.slot),
		fmt.Sprintf("%s　　等級：%d", c.Name, c.Level),
		fmt.Sprintf("生命 %d／%d　　法力 %s　　經驗 %d　　狀態 %s",
			c.HP, c.MaxHP, sp, g.charExp(c), st),
		"技能：" + strings.Join(charSkillNames(c), "、"),
		"道具：",
	}
	return append(out, g.packLines(c)...)
}

// charSkillNames 列出一個角色目前擁有的技能名稱(docs/spec/11 §7)。
func charSkillNames(c original.Character) []string {
	var out []string
	for i := 1; i <= len(c.Skills) && i <= 10; i++ {
		if c.Skills[i-1] != '1' {
			continue
		}
		if name, ok := town.SkillName(rules.Class(c.Class), i); ok {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		out = append(out, "（無）")
	}
	return out
}
