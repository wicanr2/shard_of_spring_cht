package main

import (
	"fmt"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/magic"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/ui"
)

// 施法介面。docs/spec/09-magic-items.md。
//
// 流程:戰鬥中按 C → 列出當前施法者可施的法術 → 按字母選 →
// **玩家輸入投入點數**(`'Spell Pts ?'`,docs/spec/02 §1)→ 移動游標 →
// PgDn 施放。
//
// ⚠ 投入量會改變威力與狀態強度(docs/spec/09 §2 的三條公式),
// 所以它不是介面細節。

// 施法流程的字面,照 `translations/module-text/CMBT.tsv`(F3)。
//
// ⚠ 原版問的是**法術名稱**,所以它需要「沒有這個法術」「你不會那個法術」
// 這些錯誤訊息;引擎用字母選單,但**清單列的是全部法術**而不是只列施得出來的,
// 那些檢查才有機會發生 —— 少了它們,玩家永遠不知道自己為什麼施不出某個法術。
const (
	castMenuHead   = "要施放哪個法術?(ENTER離開)" // 88+89+90
	castNoSuchSpell = "沒有這個 法術!"            // 91+92
	castNotCombat  = "那不是戰鬥法術!"           // 93+94
	castNoSkill    = "你不會那個法術!"            // 95+96
	castNotWizard  = " 不是巫師,無法施放法術。"    // 84+85+86+87
	castSPPrompt   = " 花費幾點法力? "            // 101
	castNotThatMuch = "你沒有 那麼多!"            // 102+103
	castNoTarget   = "沒有選定目標!"             // 108+109
	// 97+98+99+100:群體傷害法術在第 1 回合放不出來(docs/re/195 §1)。
	castNotPrepared = "你將還沒準備好那個法術,要到下一回合才行。"
	// 124+125:游標周圍 5×5 一個單位都沒有(docs/re/195 §2)。
	castNoOneInArea = "目標區域內沒有人!"
	// 113「Hit PgDn key」—— ⚠ 這一句是**游標階段的確認鍵**,
	// 不是法術清單的翻頁提示。先前接在翻頁上,而那個位置原版沒有這句話。
	castPageHint   = "按 PgDn 鍵施放"            // 113
	castWhere      = "你想施放到哪裡?"           // 118+119
	castEscExit    = "(ESC離開)"                 // 114
	// 117/120(`to use.` / `to cast.`)是行動點數不足那兩句的句尾。
	// ⚠ 前半段沒有單獨的字串可對,**這個對應是推的**,不是讀到的。
	castNoPoints  = "：行動點數不足,無法施放。"
	useNoPoints   = "：行動點數不足,無法使用。"
)

// castRequires 是「那個法術至少需要 N 點法力。」(CMBT:104+105+106)。
func castRequires(n int) string {
	return fmt.Sprintf("那個法術至少需要 %d 點法力。", n)
}

// openCast 開施法選單。**施法的是目前輪到的那個人**(docs/spec/12 §2)——
// 先前是「掃過全隊,找第一個會法術的」,那在戰場上是錯的:
// 施法要扣**那個人**的行動點數,而且做完結束**他的**回合。
//
// 回 false 表示這一次開不起來(訊息已經寫進 Log)。
func (g *Game) openCast() bool {
	if g.field == nil {
		return false
	}
	i := g.actor
	if i < combat.PartyBase || i >= combat.PartyBase+combat.PartyMax {
		g.field.Log = append(g.field.Log, "現在沒有人可以行動")
		return false
	}
	if g.points[i] < rules.ActCast.Cost() {
		g.field.Log = append(g.field.Log, g.field.Units[i].Name+castNoPoints)
		return false
	}
	if i-combat.PartyBase >= len(g.members) {
		return false
	}
	// 不是巫師就到此為止 —— 原版連法術清單都不會出現(CMBT:84–87)。
	if c := g.members[i-combat.PartyBase]; c.Class != magic.WizardClass {
		g.field.Log = append(g.field.Log, c.Name+castNotWizard)
		return false
	}
	if len(g.spells) == 0 {
		return false
	}
	g.castUnit, g.castList, g.castPage = i, g.spells, 0
	return true
}

// castPageSize 是法術清單一頁幾個。原版用 `Hit PgDn key`(CMBT:113)翻頁,
// 這裡沿用同一個鍵。
const castPageSize = 20

// pickSpell 用字母選**目前這一頁**的第 n 個法術,選完進「投入幾點法力」那一步。
func (g *Game) pickSpell(n int) {
	page := g.castPageSpells()
	if n < 0 || n >= len(page) {
		// CMBT:91+92「 There is no such spell!」—— 原版問的是法術名稱,
		// 打錯就回這一句;引擎的對應是**按到清單以外的字母**。
		g.field.Log = append(g.field.Log, castNoSuchSpell)
		return
	}
	s := page[n]
	c := g.members[g.castUnit-combat.PartyBase]
	// 檢查的順序照原版字串在表裡的順序:先問是不是戰鬥法術,再問會不會。
	if !combatOnlySpell(s) {
		g.field.Log = append(g.field.Log, castNotCombat)
		return
	}
	if magic.CanCast(c, s, s.UnitCost) == magic.FailNoSkill {
		g.field.Log = append(g.field.Log, castNoSkill)
		return
	}
	g.castSP = &castSPState{spell: s}
	g.castList = nil
}

// castPageSpells 回傳目前這一頁要列出來的法術。
func (g *Game) castPageSpells() []original.Spell {
	lo := g.castPage * castPageSize
	if lo >= len(g.castList) {
		return nil
	}
	hi := lo + castPageSize
	if hi > len(g.castList) {
		hi = len(g.castList)
	}
	return g.castList[lo:hi]
}

// castPages 回傳法術清單共幾頁(至少 1)。
func (g *Game) castPages() int {
	n := (len(g.castList) + castPageSize - 1) / castPageSize
	if n < 1 {
		return 1
	}
	return n
}

// castSPState 是「投入幾點法力」那一步(CMBT:101)。
type castSPState struct {
	spell original.Spell
	input string
}

// castSPKey 處理投入點數那一步的按鍵。回 true 表示這一格被吃掉了。
func (g *Game) castSPKey(k ebiten.Key) bool {
	st := g.castSP
	if st == nil {
		return false
	}
	switch {
	case k == ebiten.KeyEscape:
		g.castSP = nil
		return true
	case k == ebiten.KeyBackspace:
		if st.input != "" {
			st.input = st.input[:len(st.input)-1]
		}
		return true
	case k == ebiten.KeyEnter || k == ebiten.KeyKPEnter:
		g.confirmCastSP()
		return true
	case k >= ebiten.KeyDigit0 && k <= ebiten.KeyDigit9:
		if len(st.input) < 3 {
			st.input += string(rune('0' + (k - ebiten.KeyDigit0)))
		}
		return true
	case k >= ebiten.KeyKP0 && k <= ebiten.KeyKP9:
		if len(st.input) < 3 {
			st.input += string(rune('0' + (k - ebiten.KeyKP0)))
		}
		return true
	}
	return false
}

// confirmCastSP 收下投入點數,過關就進選格階段。
func (g *Game) confirmCastSP() {
	st := g.castSP
	invest, err := strconv.Atoi(st.input)
	if err != nil || invest < 1 {
		return // 還沒輸入,不要當成取消
	}
	c := g.members[g.castUnit-combat.PartyBase]
	switch magic.CanCast(c, st.spell, invest) {
	case magic.FailNoPoints:
		g.field.Log = append(g.field.Log, castNotThatMuch) // CMBT:102+103
		st.input = ""
		return
	case magic.FailBelowOneLevel:
		unit := st.spell.UnitCost
		if unit < 1 {
			unit = 1
		}
		g.field.Log = append(g.field.Log, castRequires(unit)) // CMBT:104–106
		st.input = ""
		return
	}
	// 原版:選完法術之後「螢幕會出現一個游標…再按下 SPACE BAR 施行」(手冊 p.34)。
	u := g.field.Units[g.castUnit]
	g.cursor = &castCursor{spell: st.spell, invest: invest, x: u.X, y: u.Y}
	g.castSP = nil
}

// castSPLines 是投入點數那一步要顯示的兩行。
func (g *Game) castSPLines() []string {
	st := g.castSP
	if st == nil {
		return nil
	}
	c := g.field.Units[g.castUnit]
	return []string{
		fmt.Sprintf("%s／%s%s(目前法力 %d)", c.Name, st.spell.Name, castSPPrompt, c.SP),
		"輸入數字,Enter 確認、Backspace 修改、ESC 取消：" + st.input + "_",
	}
}

// castCursor 是選格子的游標。**存法術本身不存索引** ——
// 索引要配上「當時那份清單」才有意義,而清單在游標階段已經關掉了。
type castCursor struct {
	spell  original.Spell
	invest int // 這次投入幾點法力(CMBT:101 那一步收下來的)
	x, y   int
}

// cursorKey 處理游標階段的按鍵。
//
// **方向鍵與 I/J/K/M 都收。** 手冊 p.34 寫的是 I/J/K/M(Apple II 的菱形配置),
// 但 DOS 版自己的字串是 `Use arrow keys to position cursor.`(CMBT:110/111)——
// ⚠ 手冊講的是**另一個平台**(CLAUDE.md §6:套手冊之前先問這一頁講哪個平台),
// 而模組裡的字串是 DOS 版自己說的話,證據等級較高。兩組都收,不必二選一。
func (g *Game) cursorKey(k ebiten.Key) bool {
	cu := g.cursor
	if cu == nil {
		return false
	}
	switch k {
	case ebiten.KeyI, ebiten.KeyUp:
		cu.y--
	case ebiten.KeyM, ebiten.KeyDown:
		cu.y++
	case ebiten.KeyJ, ebiten.KeyLeft:
		cu.x--
	case ebiten.KeyK, ebiten.KeyRight:
		cu.x++
	// ⚠ 確認鍵是 **PgDn**,不是空白鍵。原版游標階段的畫面下方就寫著
	// `Hit PgDn key to cast.`(CMBT:113,2026-08-18 實跑 `q3b-d3.png`)——
	// 手冊 p.34 的 SPACE BAR 講的是 **Apple II 版**
	// (CLAUDE.md §6:套手冊之前先問這一頁講哪個平台)。
	// 空白鍵一併留著:重製版的配置擋不到別的東西。
	case ebiten.KeyPageDown, ebiten.KeyKPDivide, ebiten.KeySpace:
		// 沒打中任何東西時**游標留著**,讓玩家改選一格 ——
		// 關掉游標等於白白吃掉一次施法。
		if g.castAt(cu.spell, cu.invest, cu.x, cu.y) {
			g.cursor, g.castList = nil, nil
		}
		return true
	case ebiten.KeyEscape:
		g.cursor, g.castList = nil, nil
		return true
	default:
		return false
	}
	if cu.x < 0 {
		cu.x = 0
	}
	if cu.y < 0 {
		cu.y = 0
	}
	if cu.x >= combat.BoardW {
		cu.x = combat.BoardW - 1
	}
	if cu.y >= combat.BoardH {
		cu.y = combat.BoardH - 1
	}
	return true
}

// castAt 對某一格施放第 n 個法術。
//
// ⚠ **法術的作用範圍未解**([`spec/09`])—— 原版有全場的風暴類法術
// (`FIRE STORM` 的圖檔是整片的),也有單體的。這裡把目標取成
// **游標那一格上的單位**,而增益類仍然套全隊 ——
// 範圍解出來之後只要改這個函式。
func (g *Game) castAt(s original.Spell, invest, cx, cy int) bool {
	if invest < 1 {
		invest = 1
	}
	// 群體傷害在第 1 回合施放不出來(docs/re/195 §1)。
	// ⚠ 擋在**扣法力之前** —— 原版那一段也是先擋再扣。
	if s.Effect == magic.EffGroupDamage && g.field.Round == 1 {
		g.field.Log = append(g.field.Log, castNotPrepared)
		return false
	}
	caster := &g.field.Units[g.castUnit]

	// 目標:游標那一格上的單位。
	//
	// ⚠ **類別 1(群體傷害)不選目標** —— `CMBT 0x14F26` 對類別 1 直接跳過
	// 選目標那一段(docs/re/172 §3),所以它可以放在空地上,打的是敵方全部。
	// ⚠ 其餘類別放在空地上就是**沒有選定目標**(CMBT:108+109),原版有這句話;
	// 先前引擎在這裡落回「敵方全部」,那是**實作決定**,而且會讓玩家
	// 以為自己打中了。
	var targets []*combat.Unit
	if s.Effect == magic.EffGroupDamage {
		// 類別 1 **完全跳過選目標**(docs/re/172 §3),游標決定的是
		// 以它為中心的 5×5 範圍(docs/re/195 §2)。
		//
		// ⚠ 順序重要:這一支要排在「游標那一格上的單位」**之前**。
		// 排在後面的話,游標剛好壓在某個單位身上時就只打那一個,
		// 而畫面上看起來完全正常 —— 群體法術變成單體。
		//
		// ⚠ **隊員也會吃到**(docs/re/208):原版的掃描迴圈跑滿
		// 14 個單位槽(0–13),沒有敵我判斷 —— 站進自己的火裡是玩家的事。
		// ⛔ 不要「順手」加一個 IsMonster 過濾:那會讓群體法術變成安全的,
		// 而畫面上完全看不出規則被改過。
		for _, i := range g.field.UnitsInArea(cx, cy) {
			targets = append(targets, &g.field.Units[i])
		}
		if len(targets) == 0 {
			g.field.Log = append(g.field.Log, castNoOneInArea)
			return false
		}
	} else if j := g.field.Occupant(cx, cy); j >= 0 {
		targets = append(targets, &g.field.Units[j])
	} else {
		g.field.Log = append(g.field.Log, castNoTarget)
		return false
	}
	// 增益類對自己人。⚠ **原版怎麼選目標未解** —— 這裡是實作決定。
	//
	// ⚠ 類別 3–6 的**欄4 帶正負**(docs/formats/04):正 = 增益、負 = 傷害。
	// `COLUMN OF FIRE` 是類別 5 且威力 −3,那是**傷害法術**,不是治療 ——
	// 只看類別會把它送去打自己人,而畫面上只顯示「XXX:-3」,看不出打錯了。
	friendly := false
	switch s.Effect {
	case magic.EffRaise, magic.EffCure, magic.EffUnbind, magic.EffProtect:
		friendly = true
	case magic.EffToHit, magic.EffStrength, magic.EffHitPoints, magic.EffSpeed:
		friendly = s.Power > 0
	}
	if friendly {
		targets = nil
		for i := combat.PartyBase; i < combat.PartyBase+combat.PartyMax; i++ {
			if g.field.Units[i].Name != "" {
				targets = append(targets, &g.field.Units[i])
			}
		}
	}

	caster.SP -= invest
	if caster.SP < 0 {
		caster.SP = 0
	}
	// 發動判定:效力 = round(欄4 × 投入 ÷ 欄5),要 ≥ d100 才成功
	// (docs/re/201)。⚠ **法力照樣扣掉** —— 原版扣點的位置沒讀到,
	// 這裡沿用引擎既有的順序,是具名的實作決定。
	if magic.Fizzles(magic.EffectLevel(s, invest), g.field.Rand) {
		g.field.Log = append(g.field.Log,
			fmt.Sprintf("%s 施放 %s(投入 %d)", caster.Name, s.Name, invest))
		g.field.Log = append(g.field.Log, magic.MsgSpellFails)
		if g.castUnit == g.actor {
			g.points[g.actor] = 0
			g.nextActor()
		}
		g.castList = nil
		return true
	}
	r := magic.Apply(s, invest, caster, targets)
	// 施法扣 3 點,而且**做完直接結束這個人的回合**(手冊 p.35,docs/spec/12 §2)。
	// ⚠ 攻擊的成本也是 3 但**不會**結束回合 —— 不能用成本判斷。
	if g.castUnit == g.actor {
		g.points[g.actor] = 0
		g.nextActor()
	}
	g.field.Log = append(g.field.Log,
		fmt.Sprintf("%s 施放 %s(投入 %d)", caster.Name, s.Name, invest))
	g.field.Log = append(g.field.Log, r.Message)
	g.castList = nil
	return true
}

// drawCastMenu 在訊息列上方畫法術清單。
//
// ⚠ 列的是**全部法術**不是只列施得出來的 —— 見上面 castMenuHead 那一段。
func (g *Game) drawCastMenu(dst *ebiten.Image) {
	if len(g.castList) == 0 || g.panel == nil {
		return
	}
	// 蓋掉底下的訊息 —— 兩層字疊在一起是「看得到但讀不出來」,
	// 而那比缺字更難察覺。
	clearMessage(dst) // message.go
	p := g.panel
	lh := p.LineHeight()
	x := float64(layout.Message.X + ui.PanelPad)
	y := float64(layout.Message.Y + ui.PanelPad)
	p.Draw(dst, g.field.Units[g.castUnit].Name+"　"+castMenuHead, x, y)
	y += lh
	page := g.castPageSpells()
	for i, s := range page {
		if y > float64(layout.Message.Y+layout.Message.H)-lh*2 {
			// ⛔ 不可以默默截斷:少列幾個法術在畫面上完全沒有症狀。
			p.Draw(dst, fmt.Sprintf("…這一頁還有 %d 個放不下", len(page)-i), x, y)
			return
		}
		invest := s.UnitCost
		if invest < 1 {
			invest = 1
		}
		p.Draw(dst, fmt.Sprintf("%c) %s  每級 %d 點", 'A'+i, s.Name, invest), x, y)
		y += lh
	}
	// ⚠ 翻頁的提示**不要借 CMBT:113** —— 那一句是游標階段的
	// 「按 PgDn 鍵施放」,原版在法術清單這裡沒有對應的字串。
	if n := g.castPages(); n > 1 {
		p.Draw(dst, fmt.Sprintf("第 %d／%d 頁　+／- 翻頁", g.castPage+1, n), x, y)
	}
}

// drawCastSP 畫「投入幾點法力」那一步(CMBT:101)。
func (g *Game) drawCastSP(dst *ebiten.Image) {
	if g.castSP == nil || g.panel == nil {
		return
	}
	clearMessage(dst)
	p := g.panel
	lh := p.LineHeight()
	x := float64(layout.Message.X + ui.PanelPad)
	y := float64(layout.Message.Y + ui.PanelPad)
	for _, ln := range g.castSPLines() {
		p.Draw(dst, ln, x, y)
		y += lh
	}
}
