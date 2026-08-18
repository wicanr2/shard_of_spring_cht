package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/magic"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/town"
)

// 測試全部**不開視窗**:直接呼叫方法,不跑 ebiten.RunGame,
// 也不呼叫任何要畫圖的函式(campCastLines 等只組字串,沒有 GPU 依賴)。
// docs/spec/16 §6 的六條驗收,對應關係寫在每個測試前面。

func testPack() [original.PackSlots]int {
	var p [original.PackSlots]int
	for i := range p {
		p[i] = original.NotEquipped
	}
	return p
}

func testWizard(name string, skills string) original.Character {
	return original.Character{
		ID: 1, Name: name, Class: magic.WizardClass, Skills: skills,
		HP: 20, MaxHP: 20, SP: 20, MaxSP: 20, Level: 3, Str: 10,
		Pack: testPack(), Identified: strings.Repeat("0", original.PackSlots),
	}
}

func testHero(name string) original.Character {
	c := testWizard(name, "0000000000")
	c.Class = '1'
	return c
}

// ── combatOnlySpell(docs/spec/16 §2 的具名表)───────────────────────

func TestCombatOnlySpell(t *testing.T) {
	cases := []struct {
		name string
		s    original.Spell
		want bool
	}{
		{"群體傷害", original.Spell{Effect: magic.EffGroupDamage, Power: 10}, true},
		{"單體重擊", original.Spell{Effect: magic.EffSingleDamage, Power: 10}, true},
		{"束縛", original.Spell{Effect: magic.EffBind, Power: 1}, true},
		{"力量削弱(威力為負)", original.Spell{Effect: magic.EffStrength, Power: -1}, true},
		{"力量增益(威力為正)", original.Spell{Effect: magic.EffStrength, Power: 2}, false},
		{"生命治療(威力為正)", original.Spell{Effect: magic.EffHitPoints, Power: 3}, false},
		{"防護", original.Spell{Effect: magic.EffProtect, Power: 1}, false}, // 不算戰鬥法術,是另一個原因擋
		{"解毒", original.Spell{Effect: magic.EffCure}, false},
		{"復活", original.Spell{Effect: magic.EffRaise}, false},
		{"非戰鬥效用", original.Spell{Effect: magic.EffUtility}, false},
	}
	for _, c := range cases {
		if got := combatOnlySpell(c.s); got != c.want {
			t.Errorf("%s: combatOnlySpell=%v,want %v", c.name, got, c.want)
		}
	}
}

// EffProtect 因為 Character 沒有 ArmSkin 欄位而被擋,訊息要跟「戰鬥法術」不同句。
func TestCampSpellAllowed_ProtectHasDistinctReason(t *testing.T) {
	protect := original.Spell{Name: "護甲術", Effect: magic.EffProtect, Power: 1}
	combatSpell := original.Spell{Name: "烈焰猛擊術", Effect: magic.EffSingleDamage, Power: 20}

	protectMsg := campSpellAllowed(protect)
	combatMsg := campSpellAllowed(combatSpell)
	if protectMsg == "" || combatMsg == "" {
		t.Fatalf("兩者都該被擋:protect=%q combat=%q", protectMsg, combatMsg)
	}
	if protectMsg == combatMsg {
		t.Errorf("防護被擋的原因跟戰鬥法術不一樣,不該是同一句:%q", protectMsg)
	}
	if !strings.Contains(combatMsg, "戰鬥法術") {
		t.Errorf("戰鬥法術的訊息要提到「戰鬥法術」,got %q", combatMsg)
	}
}

// ── campCastCheck:docs/spec/16 §6 驗收 #1/#3 ─────────────────────────

func TestCampCastCheck_ThreeDistinctGateMessages(t *testing.T) {
	buff := original.Spell{Name: "強壯術", School: 2, Effect: magic.EffStrength,
		Power: 3, UnitCost: 1}
	combatSpell := original.Spell{Name: "冰雹暴風術", School: 4, Effect: magic.EffGroupDamage,
		Power: 8, UnitCost: 7}

	hero := testHero("戰士甲")
	wizardNoSkill := testWizard("巫師乙", "0000000000") // 系別 2 的旗標(索引1)是 '0'

	notWizard := campCastCheck(hero, buff, 1)
	noSkill := campCastCheck(wizardNoSkill, buff, 1)
	combatBlocked := campCastCheck(testWizard("巫師丙", "0100000000"), combatSpell, 1)

	if notWizard == "" || noSkill == "" || combatBlocked == "" {
		t.Fatalf("三種都該被擋:notWizard=%q noSkill=%q combat=%q",
			notWizard, noSkill, combatBlocked)
	}
	seen := map[string]bool{notWizard: true, noSkill: true, combatBlocked: true}
	if len(seen) != 3 {
		t.Errorf("三句訊息應該各不相同,got %q / %q / %q", notWizard, noSkill, combatBlocked)
	}
	if notWizard != notWizardMsg {
		t.Errorf("not-wizard 訊息 = %q,want %q", notWizard, notWizardMsg)
	}
}

// 投超過 SP、投不到一級 —— 各一個測試,訊息要對應 magic.Fail 既有的字串。
func TestCampCastCheck_InvestGates(t *testing.T) {
	// 每級單價 5,SP 只有 20。
	s := original.Spell{Name: "醫療術", School: 5, Effect: magic.EffHitPoints,
		Power: 3, UnitCost: 5}
	c := testWizard("巫師丁", "0000100000") // 系別 5 → 索引 4

	// 投超過 SP 走 CAMP:84 那一句(「你沒有那麼多!」),不是 magic 的通用句 ——
	// 原版兩句分開,引擎的閘門只有一個,依情境挑(campCastCheck 有說明)。
	if got := campCastCheck(c, s, 25); got != campNotThatMuch {
		t.Errorf("投超過 SP:got %q,want %q", got, campNotThatMuch)
	}
	if got := campCastCheck(c, s, 3); got != magic.FailBelowOneLevel.String() {
		t.Errorf("投不到一級(3 < 單價 5):got %q,want %q", got, magic.FailBelowOneLevel.String())
	}
	if got := campCastCheck(c, s, 10); got != "" {
		t.Errorf("投 10 點(2 級、10 ≤ SP)應該可以放,got %q", got)
	}
}

// ── E1 + E5:完整走一輪按鍵,驗證威力用的是投入而不是等級。docs/spec/16 §6 #1/#2 ──

func TestCampCastKey_FullFlow_PowerUsesInvestNotLevel(t *testing.T) {
	// 單價 2、每點威力 3:投 6 點 → 等級 3、正確威力 = 3×6 = 18。
	// 如果誤用「威力 × 等級」會變成 3×3 = 9 —— 兩個數字不同,測得出來。
	buff := original.Spell{Index: 0, Name: "強壯術", School: 1, Effect: magic.EffStrength,
		Power: 3, UnitCost: 2}

	caster := testWizard("施法者", "1000000000") // 系別 1 → 索引 0
	target := testWizard("目標", "0000000000")
	target.Str = 10

	g := &Game{
		members: []original.Character{caster, target},
		spells:  []original.Spell{buff},
		// ⚠ 營地施法有發動判定(docs/re/209:效力 = 欄4×投入÷欄5,擲 d100)——
		// **測發動之後的行為,就要先把發動判定拿掉**(docs/re/201 §3 的同一個坑)。
		rand:    alwaysLowRand{},
		chars:   make([]original.Character, 25),
	}
	for i, c := range g.members {
		c.ID = i + 1
		g.chars[i] = c
	}
	g.town = &townState{campMode: 'C', campWho: -1, campWho2: -1}

	// 階段 0:選施法者(1)
	g.campCastKey(ebiten.Key1)
	if g.town.campWho != 0 || g.town.castStage != 1 {
		t.Fatalf("選完施法者後 campWho=%d castStage=%d,想要 0/1", g.town.campWho, g.town.castStage)
	}
	// 階段 1:選法術(A = 第 0 個)
	g.campCastKey(ebiten.KeyA)
	if g.town.castStage != 2 {
		t.Fatalf("選完法術後 castStage=%d,想要 2", g.town.castStage)
	}
	// 階段 2:輸入 "6",Enter 確認
	g.campCastKey(ebiten.KeyDigit6)
	g.campCastKey(ebiten.KeyEnter)
	if g.town.castStage != 3 {
		t.Fatalf("投入 6 點應該過閘門進到選目標,castStage=%d msg=%q", g.town.castStage, g.town.msg)
	}
	// 階段 3:選目標(2 = 第二位)
	g.campCastKey(ebiten.Key2)

	if g.town.campMode != 0 {
		t.Fatalf("施法完應該回到營地選單,campMode=%q", g.town.campMode)
	}
	want := 10 + 3*6 // 力量 += 威力(3×投入6=18)
	if g.members[1].Str != want {
		t.Errorf("目標力量 = %d,want %d(3×投入6,不是 3×等級3=9)", g.members[1].Str, want)
	}
	if g.members[0].SP != 20-6 {
		t.Errorf("施法者法力 = %d,want %d", g.members[0].SP, 20-6)
	}
}

// ── E2:U)se an item(docs/spec/16 §6 #4)─────────────────────────────

func TestUseItem_UnidentifiedStillWorksButNameHidden(t *testing.T) {
	c := testWizard("使用者", "0000000000")
	c.Pack[0] = 30 // > 26 → 魔法道具
	c.Identified = "0000000000"

	// 道具:欄4=法術編號 0、欄5=投入 4、欄6=100(必定發動)。
	spell := original.Spell{Index: 0, Name: "劍術", School: 2, Effect: magic.EffStrength,
		Power: 1, UnitCost: 1}
	item := original.Item{Index: 30, Name: "神秘權杖", Col4: 0, Col5: 4, Col6: 100}

	g := &Game{
		members:  []original.Character{c},
		spells:   []original.Spell{spell},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c},
	}
	g.chars[0].ID = 1
	g.members[0].ID = 1
	g.town = &townState{campMode: 'U', campWho: -1}

	// 顯示名稱要是「未辨識」,不是「神秘權杖」。
	if name := g.itemDisplayName(g.members[0], 0); name != "未辨識的道具" {
		t.Errorf("未辨識的道具顯示成 %q,want 「未辨識的道具」", name)
	}

	g.useItem(0, 0)
	if !strings.Contains(g.town.msg, "未辨識的道具") {
		t.Errorf("使用訊息沒有標示未辨識:%q", g.town.msg)
	}
	if !strings.Contains(g.town.msg, "劍術") {
		t.Errorf("成功率 100 應該一定發動,訊息裡要看得到法術名:%q", g.town.msg)
	}
	if g.members[0].Str != 10+1*4 { // 威力 1 × 投入 4
		t.Errorf("力量 = %d,want %d", g.members[0].Str, 10+1*4)
	}
}

// 編號 ≤ 26 不是魔法道具,不走發動這條路(docs/spec/09 §5)。
func TestUseItem_NotMagicItemDoesNothing(t *testing.T) {
	c := testWizard("使用者", "0000000000")
	c.Pack[0] = 5 // ≤ 26
	c.Identified = "1000000000"
	item := original.Item{Index: 5, Name: "短劍"}

	g := &Game{
		members:  []original.Character{c},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c},
	}
	g.members[0].ID, g.chars[0].ID = 1, 1
	g.town = &townState{campMode: 'U', campWho: -1}

	before := g.members[0].Str
	g.useItem(0, 0)
	if g.members[0].Str != before {
		t.Errorf("編號 ≤ 26 的道具不該改動任何屬性,Str 從 %d 變成 %d", before, g.members[0].Str)
	}
	// 措辭照 CAMP:91「You really can't make use of this!」(F3)。
	if !strings.Contains(g.town.msg, "你真的沒辦法用這個!") {
		t.Errorf("訊息要講清楚這件東西用不出效果:%q", g.town.msg)
	}
}

// 狀態強度 = 欄5 ÷ 投入(投得多、值小)—— 用一件觸發束縛效果的魔法道具測,
// 因為營地施法把束縛擋在外面(combatOnlySpell),這條公式只有道具這條路
// 走得到 magic.Apply 的 EffBind 分支。
func TestUseItem_StatusMagnitudeIsUnitCostOverInvestNotReversed(t *testing.T) {
	c := testWizard("使用者", "0000000000")
	c.Pack[0] = 40
	c.Identified = "1000000000"

	// 單價(欄5 概念,這裡是 spell.UnitCost)= 12,投入 = 3
	// 正確:強度 = 12 ÷ 3 = 4。寫反(投入 ÷ 單價)會得到 0。
	bind := original.Spell{Index: 7, Name: "空氣凝結術", School: 3, Effect: magic.EffBind,
		Power: 1, UnitCost: 12, HitMsg: "凝結了!"}
	item := original.Item{Index: 40, Name: "凝結權杖", Col4: 7, Col5: 3, Col6: 100}

	g := &Game{
		members:  []original.Character{c},
		spells:   []original.Spell{bind},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c},
	}
	g.members[0].ID, g.chars[0].ID = 1, 1
	g.town = &townState{campMode: 'U', campWho: -1}

	g.useItem(0, 0)
	if g.members[0].Status != bind.School {
		t.Fatalf("狀態應該變成系別 %d,got %d(訊息:%q)", bind.School, g.members[0].Status, g.town.msg)
	}
	if g.members[0].StatMag != 4 {
		t.Errorf("狀態強度 = %d,want 4(12÷3,不是 3÷12=0)", g.members[0].StatMag)
	}
}

// ── E3:P)rint(docs/spec/16 §6 #5)────────────────────────────────────

func TestCampPrintKey_9SelectsWholeParty(t *testing.T) {
	members := []original.Character{testWizard("甲", "0000000000"), testWizard("乙", "0000000000")}
	g := &Game{members: members, chars: append([]original.Character{}, members...)}
	g.town = &townState{campMode: 'P', campWho: -1}

	g.campPrintKey(ebiten.Key9)
	if g.town.campWho != printWholeParty {
		t.Errorf("按 9 應該選到「整隊」(%d),got %d", printWholeParty, g.town.campWho)
	}
	if g.town.campWho == 8 {
		t.Errorf("9 不能被當成「第 9 個人」的 0-based 索引 8")
	}
}

func TestCampPrintLines_NoPrinterMentioned(t *testing.T) {
	members := []original.Character{testWizard("甲", "1000000000")}
	members[0].Level, members[0].HP, members[0].MaxHP = 4, 15, 20
	g := &Game{members: members, slot: 1, chars: append([]original.Character{}, members...)}
	g.town = &townState{campMode: 'P', campWho: 0}

	lines := g.campPrintLines(g.town)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "印表機") || strings.Contains(joined, "printer") ||
		strings.Contains(joined, "Printer") {
		t.Errorf("不該提印表機(CLAUDE.md §1.2 邊界):%q", joined)
	}
	if !strings.Contains(joined, "甲") {
		t.Errorf("單人列印要看得到那個人的名字:%q", joined)
	}

	g.town.campWho = printWholeParty
	whole := strings.Join(g.campPrintLines(g.town), "\n")
	if !strings.Contains(whole, "甲") {
		t.Errorf("整隊列印要看得到隊員名字:%q", whole)
	}
}

// campCastCheck 額外檢查:回空字串代表可以放(正向案例,配合上面幾個負向案例)。
func TestCampCastCheck_OK(t *testing.T) {
	s := original.Spell{Name: "醫療術", School: 5, Effect: magic.EffHitPoints,
		Power: 3, UnitCost: 1}
	c := testWizard("巫師", "0000100000")
	if got := campCastCheck(c, s, 1); got != "" {
		t.Errorf("合法的施法不該被擋,got %q", got)
	}
}

// applyCampUnit 的生命值要夾在 [0, MaxHP],不能靠 magic.Apply 自己夾上限
// (Unit 沒有 MaxHP 這個概念,docs/spec/16)。
func TestApplyCampUnit_ClampsHP(t *testing.T) {
	c := original.Character{HP: 5, MaxHP: 20}
	applyCampUnit(&c, combat.Unit{HP: 999})
	if c.HP != 20 {
		t.Errorf("HP 應該夾在 MaxHP=20,got %d", c.HP)
	}
	applyCampUnit(&c, combat.Unit{HP: -5})
	if c.HP != 0 {
		t.Errorf("HP 應該夾在 0,got %d", c.HP)
	}
}

// 小工具測試:投入點數的數字緩衝要能正確累積與退格。
func TestCastInputDigitsAndBackspace(t *testing.T) {
	g := &Game{
		members: []original.Character{testWizard("甲", "1000000000")},
		spells: []original.Spell{{Index: 0, School: 1, Effect: magic.EffStrength,
			Power: 1, UnitCost: 1}},
		rand:  combat.NewRand(1),
		chars: []original.Character{testWizard("甲", "1000000000")},
	}
	g.members[0].ID, g.chars[0].ID = 1, 1
	g.town = &townState{campMode: 'C', campWho: -1, campWho2: -1}

	g.campCastKey(ebiten.Key1) // 選施法者
	g.campCastKey(ebiten.KeyA) // 選法術
	g.campCastKey(ebiten.KeyDigit1)
	g.campCastKey(ebiten.KeyDigit2)
	if g.town.castInput != "12" {
		t.Fatalf("輸入 1、2 後緩衝應該是 %q,got %q", "12", g.town.castInput)
	}
	g.campCastKey(ebiten.KeyBackspace)
	if g.town.castInput != "1" {
		t.Fatalf("退格後應該是 %q,got %q", "1", g.town.castInput)
	}
	if n, err := strconv.Atoi(g.town.castInput); err != nil || n != 1 {
		t.Fatalf("輸入緩衝應該可以解析成 1,got %q", g.town.castInput)
	}
}

// ── 風行術(docs/re/193)────────────────────────────────────────────────

// TestWindWalkTeleportsAndLeaves:營地放風行術 → 全隊到 (8,8),地城與營地都收掉。
//
// ⚠ 這一條同時釘住「離開地城」——原版是轉交 WRLDMOVE 模組,
// 只把座標改掉而留在地城裡,玩家會站在原地看著一段狂風敘述什麼都沒發生。
func TestWindWalkTeleportsAndLeaves(t *testing.T) {
	g := newPlayingGame(t)
	// ⚠ 營地施法有發動判定(docs/re/209),而測試法術的效力多半擲不過 ——
	// **測發動之後的行為,就要先把發動判定拿掉**(docs/re/201 §3 的同一個坑)。
	g.rand = alwaysLowRand{}
	g.level = &mazeLevel{}
	g.town = &townState{mode: townCamp}
	g.party.X, g.party.Y = 40, 40

	wind, ok := g.spellByIndex(magic.SpellWindWalk)
	if !ok {
		t.Fatalf("資產裡找不到索引 %d 的法術", magic.SpellWindWalk)
	}
	if wind.Effect != magic.EffUtility {
		t.Fatalf("索引 %d 應該是效果類別 %d,得到 %d(資產與 docs/re/193 對不上)",
			magic.SpellWindWalk, magic.EffUtility, wind.Effect)
	}

	g.town.castSpell, g.town.castInput = wind, "10"
	g.castInCamp(0, 0)

	if g.party.X != magic.WindWalkX || g.party.Y != magic.WindWalkY {
		t.Errorf("應該傳送到 (%d,%d),得到 (%d,%d)",
			magic.WindWalkX, magic.WindWalkY, g.party.X, g.party.Y)
	}
	if g.level != nil {
		t.Error("風行術之後不該還在地城裡")
	}
	if g.town != nil {
		t.Error("風行術之後不該還在營地裡")
	}
	if !strings.Contains(g.overlay, "狂風") {
		t.Errorf("要印 CAMP:109 的狂風敘述,得到 %q", g.overlay)
	}
}

// TestOtherUtilitySpellsDoNotTeleport:類別 12 的另外兩個(照明)不會搬人。
// ⚠ 它們的規則沒讀出來(docs/re/193 §5),但「不傳送」是讀到的 ——
// 那個判斷式比的是列號 22,不是效果類別。
func TestOtherUtilitySpellsDoNotTeleport(t *testing.T) {
	g := newPlayingGame(t)
	g.town = &townState{mode: townCamp}
	g.party.X, g.party.Y = 40, 40
	for _, s := range g.spells {
		if s.Effect != magic.EffUtility || s.Index == magic.SpellWindWalk {
			continue
		}
		g.town.castSpell, g.town.castInput = s, "10"
		g.castInCamp(0, 0)
		if g.town == nil {
			t.Fatalf("「%s」(索引 %d)不該把隊伍搬走", s.Name, s.Index)
		}
		if g.party.X != 40 || g.party.Y != 40 {
			t.Fatalf("「%s」動到了座標:(%d,%d)", s.Name, g.party.X, g.party.Y)
		}
	}
}

// ── 群體傷害:第一回合的閘門與 5×5 作用範圍(docs/re/195)──────────────

// groupSpellGame 擺一場戰鬥,施法者是隊員,場上有一隻怪。
func groupSpellGame(t *testing.T, round, mx, my int) (*Game, original.Spell) {
	t.Helper()
	g := newPlayingGame(t)
	g.rand = alwaysLowRand{} // 發動判定固定成功,見 castInCamp 的說明
	var group original.Spell
	for _, s := range g.spells {
		if s.Effect == magic.EffGroupDamage {
			group = s
			break
		}
	}
	if group.Name == "" {
		t.Fatal("資產裡找不到類別 1 的法術")
	}
	f := &combat.Field{Rand: combat.NewRand(7), Round: round}
	f.Units[combat.MonsterBase] = combat.Unit{Name: "地精", HP: 20,
		Facing: combat.South, IsMonster: true, X: mx, Y: my}
	f.Units[combat.PartyBase] = combat.Unit{Name: "凱恩", HP: 20, SP: 50,
		Facing: combat.North, X: 13, Y: 13}
	g.field = f
	g.castUnit = combat.PartyBase
	return g, group
}

// TestGroupSpellBlockedOnRoundOne:第 1 回合放不出來,而且**不扣法力**。
func TestGroupSpellBlockedOnRoundOne(t *testing.T) {
	g, s := groupSpellGame(t, 1, 13, 11)
	sp0 := g.field.Units[combat.PartyBase].SP
	if g.castAt(s, 10, 13, 11) {
		t.Error("第 1 回合不該施放成功")
	}
	if g.field.Units[combat.PartyBase].SP != sp0 {
		t.Errorf("被擋下來不該扣法力:%d → %d", sp0, g.field.Units[combat.PartyBase].SP)
	}
	if !logHas(g.field.Log, "下一回合") {
		t.Errorf("要說 CMBT:97–100 那一句,得到 %v", g.field.Log)
	}
}

// TestGroupSpellHitsOnlyInArea:5×5 之外的怪打不到,而且會說「區域內沒有人」。
func TestGroupSpellHitsOnlyInArea(t *testing.T) {
	// 怪在 (13,13),游標放到 (13,20) —— 差 7 格,遠超過 ±2。
	g, s := groupSpellGame(t, 2, 13, 13)
	if g.castAt(s, 10, 13, 20) {
		t.Error("範圍內沒有怪,不該施放成功")
	}
	if !logHas(g.field.Log, "目標區域內沒有人") {
		t.Errorf("要說 CMBT:124/125,得到 %v", g.field.Log)
	}
	if g.field.Units[combat.MonsterBase].HP != 20 {
		t.Error("範圍外的怪不該掉血")
	}
}

// TestGroupSpellAreaEdge:剛好在半徑上(±2)算在範圍內。
func TestGroupSpellAreaEdge(t *testing.T) {
	g, s := groupSpellGame(t, 2, 15, 11) // 與游標 (13,12) 差 (2,-1)
	if !g.castAt(s, 10, 13, 12) {
		t.Fatalf("±2 應該算在範圍內,卻沒施放成功:%v", g.field.Log)
	}
	if g.field.Units[combat.MonsterBase].HP >= 20 {
		t.Error("範圍內的怪應該吃到傷害")
	}
}

// TestUnitsInAreaIsFiveByFive:半徑就是 2,別的數字會讓上面兩條同時通過。
func TestUnitsInAreaIsFiveByFive(t *testing.T) {
	if combat.AreaRadius != 2 {
		t.Fatalf("docs/re/195 §2 讀到的是 ±2(5×5),得到 ±%d", combat.AreaRadius)
	}
}

// TestGroupSpellIgnoresOccupantOrder:游標壓在某個單位身上時,群體法術仍然
// 打**整塊 5×5**,不是只打那一個。
//
// ⚠ 這條擋的是一個排序缺陷:類別 1 的分支若排在「游標那一格上的單位」之後,
// 群體法術會在游標壓到人時退化成單體 —— 而傷害照樣出現,畫面上看不出差別。
func TestGroupSpellIgnoresOccupantOrder(t *testing.T) {
	g, s := groupSpellGame(t, 2, 14, 12)
	// 再放一隻,兩隻都在游標 (13,12) 的 5×5 內。
	g.field.Units[combat.MonsterBase+1] = combat.Unit{Name: "狗頭人", HP: 20,
		Facing: combat.South, IsMonster: true, X: 13, Y: 12}
	if !g.castAt(s, 10, 13, 12) { // 游標正壓在第二隻身上
		t.Fatalf("應該施放成功:%v", g.field.Log)
	}
	for _, i := range []int{combat.MonsterBase, combat.MonsterBase + 1} {
		if g.field.Units[i].HP >= 20 {
			t.Errorf("單位 %d 在範圍內卻沒吃到傷害(群體退化成單體)", i)
		}
	}
}

// ── 照明法術與武器技能閘門(docs/re/196)──────────────────────────────

// TestLightSpellSetsTurnsAndVisibility:回合數 = INT(欄4×投入÷欄5)×5+20、
// 能見度 = 欄4。
func TestLightSpellSetsTurnsAndVisibility(t *testing.T) {
	g := newPlayingGame(t)
	// ⚠ 營地施法有發動判定(docs/re/209),而測試法術的效力多半擲不過 ——
	// **測發動之後的行為,就要先把發動判定拿掉**(docs/re/201 §3 的同一個坑)。
	g.rand = alwaysLowRand{}
	g.town = &townState{mode: townCamp}
	torch, ok := g.spellByIndex(magic.SpellMagicTorch)
	if !ok {
		t.Fatalf("找不到索引 %d 的法術", magic.SpellMagicTorch)
	}
	// ⚠ 寫的是**隊伍狀態**不是記錄(docs/re/204:g.party 是遊玩中的唯一真相)。
	g.party.LightTurns, g.party.VisLit = 0, 0
	g.town.castSpell, g.town.castInput = torch, "4"
	g.castInCamp(0, 0)

	wantTurns := torch.Power*4/torch.UnitCost*magic.LightTurnFactor + magic.LightTurnBase
	if g.party.LightTurns != wantTurns {
		t.Errorf("光源回合數應該是 %d,得到 %d", wantTurns, g.party.LightTurns)
	}
	if g.party.VisLit != torch.Power {
		t.Errorf("能見度應該等於欄4(%d),得到 %d", torch.Power, g.party.VisLit)
	}
	if !strings.Contains(g.town.msg, "回合的照明") {
		t.Errorf("要說 CAMP:134/135,得到 %q", g.town.msg)
	}
}

// TestCrystalightIsBrighter:水晶燈術的能見度比魔法火炬高一格 ——
// 欄4 同時決定亮度與持續回合(docs/re/196 §3.1)。
func TestCrystalightIsBrighter(t *testing.T) {
	g := newPlayingGame(t)
	torch, ok1 := g.spellByIndex(magic.SpellMagicTorch)
	crys, ok2 := g.spellByIndex(magic.SpellCrystalight)
	if !ok1 || !ok2 {
		t.Fatal("資產裡缺照明法術")
	}
	_, v1, _ := magic.LightEffect(torch, 4)
	_, v2, _ := magic.LightEffect(crys, 4)
	if v2 <= v1 {
		t.Errorf("水晶燈術(%d)應該比魔法火炬(%d)亮", v2, v1)
	}
}

// TestWeaponSkillGate:巫師裝不上武器、戰士要有對應技能,而**匕首誰都能裝**。
func TestWeaponSkillGate(t *testing.T) {
	swordsman := original.Character{Class: '1', Skills: "1000000000"} // 技能 1 = 劍
	axeman := original.Character{Class: '1', Skills: "0100000000"}    // 技能 2 = 斧
	wizard := original.Character{Class: '2', Skills: "1111111111"}

	for _, c := range []struct {
		who          original.Character
		item         int
		wantOK       bool
		wantChecked  bool
		desc         string
	}{
		{swordsman, 2, true, true, "會劍的裝短劍"},
		{swordsman, 1, false, true, "會劍的裝小斧"},
		{axeman, 1, true, true, "會斧的裝小斧"},
		{axeman, 3, false, true, "會斧的裝釘錘"},
		{wizard, 2, false, true, "巫師裝短劍"},
		{wizard, 0, true, false, "巫師裝匕首 —— 編號 0 不檢查"},
		{swordsman, 12, true, false, "防具不走這一支"},
	} {
		ok, checked := town.WeaponSkillOK(c.who, c.item)
		if ok != c.wantOK || checked != c.wantChecked {
			t.Errorf("%s:得到 (ok=%v, checked=%v),期望 (%v, %v)",
				c.desc, ok, checked, c.wantOK, c.wantChecked)
		}
	}
}

// ── DAZA REVELI 大門(docs/re/197)──────────────────────────────────────

// utterGame 擺一個站在指定地城編號裡、正在唸咒語的隊伍。
func utterGame(t *testing.T, mazeFile int) *Game {
	t.Helper()
	g := newPlayingGame(t)
	if mazeFile != NotInMaze {
		g.level = &mazeLevel{entry: original.MazeEntry{MazeFile: mazeFile}}
	}
	g.town = &townState{mode: townCamp, campMode: 'C', campWho: 0, castStage: 1}
	empty := ""
	g.town.utter = &empty
	return g
}

func say(g *Game, phrase string) {
	g.utterRunes([]rune(phrase))
	g.utterKey(ebiten.KeyEnter)
}

// TestGateOpensInMazeFive:在第 5 座地城唸對 → 位移 65 的旗標立起來。
func TestGateOpensInMazeFive(t *testing.T) {
	g := utterGame(t, gateMazeNumber)
	g.group.GateOpen = 0
	say(g, gatePhrase)
	if g.group.GateOpen != 1 {
		t.Errorf("大門旗標應該立起來,得到 %d", g.group.GateOpen)
	}
	if g.town.msg != gateOpensMsg {
		t.Errorf("要說「%s」,得到 %q", gateOpensMsg, g.town.msg)
	}
}

// TestGatePhraseElsewhereMumbles:同一句咒語在別的地城沒有用 ——
// 條件是**兩個**(字串相符 + 在第 5 座),不是一個。
func TestGatePhraseElsewhereMumbles(t *testing.T) {
	for _, n := range []int{1, 6, NotInMaze} {
		g := utterGame(t, n)
		g.group.GateOpen = 0
		say(g, gatePhrase)
		if g.group.GateOpen != 0 {
			t.Errorf("地城 %d 不該開得了門", n)
		}
		if g.town.msg != gateMumble {
			t.Errorf("地城 %d 應該喃喃自語,得到 %q", n, g.town.msg)
		}
	}
}

// TestUtterNonsenseMumbles:亂打 → CAMP:80。
func TestUtterNonsenseMumbles(t *testing.T) {
	g := utterGame(t, gateMazeNumber)
	say(g, "XYZZY")
	if g.town.msg != gateMumble {
		t.Errorf("要說 CAMP:80,得到 %q", g.town.msg)
	}
}

// TestUtterOnlyTakesASCII:中文字不進緩衝(咒語是 ASCII,而 IME 不可靠)。
func TestUtterOnlyTakesASCII(t *testing.T) {
	g := utterGame(t, gateMazeNumber)
	g.utterRunes([]rune("風行術AB"))
	if got := *g.town.utter; got != "AB" {
		t.Errorf("只該收 ASCII,得到 %q", got)
	}
}

// TestUtterEscapeCancels:ESC 取消,不留下訊息也不動旗標。
func TestUtterEscapeCancels(t *testing.T) {
	g := utterGame(t, gateMazeNumber)
	g.utterRunes([]rune(gatePhrase))
	g.utterKey(ebiten.KeyEscape)
	if g.town.utter != nil {
		t.Error("ESC 之後緩衝要收掉")
	}
	if g.group.GateOpen != 0 {
		t.Error("ESC 不該開門")
	}
}

// ── 發動判定(docs/re/201)──────────────────────────────────────────────

// TestEffectLevelRoundsHalfUp:發動判定用的效力是 `round(欄4 × 投入 ÷ 欄5)`,
// **不是** `Power()` —— 後者根本不除單價。兩個量各有各的用途,別混用。
func TestEffectLevelRoundsHalfUp(t *testing.T) {
	s := original.Spell{Power: 3, UnitCost: 2}
	if got := magic.EffectLevel(s, 5); got != 8 { // round(3×5/2) = round(7.5) = 8
		t.Errorf("效力應該是 8,得到 %d", got)
	}
	if got := magic.Power(s, 5); got != 15 { // 3 × 5,沒有除單價
		t.Errorf("Power 應該是 15,得到 %d", got)
	}
	// 四捨五入不是截尾(docs/re/185):7.5 → 8。
	if got := magic.EffectLevel(original.Spell{Power: 1, UnitCost: 2}, 1); got != 1 {
		t.Errorf("0.5 應該進位成 1,得到 %d", got)
	}
}

// TestFizzlesThreshold:效力 ≥ 擲骰才成功。
func TestFizzlesThreshold(t *testing.T) {
	for _, c := range []struct {
		level, roll int
		want        bool
	}{
		{100, 100, false}, // 剛好夠
		{99, 100, true},   // 差一點
		{1, 1, false},
		{0, 1, true},
	} {
		if got := magic.Fizzles(c.level, fixedRoll(c.roll)); got != c.want {
			t.Errorf("效力 %d 對擲骰 %d:得到 %v,期望 %v", c.level, c.roll, got, c.want)
		}
	}
}

// fixedRoll 讓 Roll 永遠回同一個值。
type fixedRoll int

func (f fixedRoll) Roll(int) int { return int(f) }

// TestCombatSpellCanFizzle:戰鬥施法失敗時說 CMBT:141,而且**法力照樣扣**。
func TestCombatSpellCanFizzle(t *testing.T) {
	g, s := groupSpellGame(t, 2, 14, 12)
	g.field.Rand = alwaysHighRand{}
	sp0 := g.field.Units[combat.PartyBase].SP
	if !g.castAt(s, 1, 13, 12) { // 投入 1 → 效力很低 → 必定失敗
		t.Fatal("失敗也算「這一次施過了」,castAt 應該回 true")
	}
	if !logHas(g.field.Log, magic.MsgSpellFails) {
		t.Errorf("要說「%s」,得到 %v", magic.MsgSpellFails, g.field.Log)
	}
	if g.field.Units[combat.PartyBase].SP != sp0-1 {
		t.Errorf("失敗也扣法力:%d → %d", sp0, g.field.Units[combat.PartyBase].SP)
	}
	if g.field.Units[combat.MonsterBase].HP != 20 {
		t.Error("失敗不該造成傷害")
	}
}

// alwaysHighRand 讓每一次擲骰都回最大值。
type alwaysHighRand struct{}

func (alwaysHighRand) Roll(n int) int      { return n }
func (alwaysHighRand) Float01() float64    { return 0.999999 }

// ── 迷宮定點道具(docs/re/202)──────────────────────────────────────────

// TestMazeLootGoesToLastMember:由**隊尾往隊首**找空格(docs/re/168 §1)——
// 與戰鬥掉落的由前往後相反,兩支各自照抄。
func TestMazeLootGoesToLastMember(t *testing.T) {
	g := newPlayingGame(t)
	if len(g.members) < 2 {
		t.Skip("這個 fixture 只有一位隊員,分不出方向")
	}
	for i := range g.members {
		for s := range g.members[i].Pack {
			g.members[i].Pack[s] = original.NotEquipped
		}
	}
	last := len(g.members) - 1
	msg := g.giveMazeItem(49)
	if g.members[last].Pack[0] != 49 {
		t.Errorf("應該給隊尾那一位,%s 的第 0 格是 %d", g.members[last].Name, g.members[last].Pack[0])
	}
	if g.members[0].Pack[0] != original.NotEquipped {
		t.Error("隊首不該拿到")
	}
	if town.IsIdentified(g.members[last], 0) {
		t.Error("撿來的東西要是未鑑定的(docs/re/168 §2)")
	}
	if !strings.Contains(msg, mazeFoundItem) {
		t.Errorf("要說 MAZEMOVE:84,得到 %q", msg)
	}
}

// TestMazeLootPackFull:全隊背包都滿 → MAZEMOVE:85,而且不覆蓋任何東西。
func TestMazeLootPackFull(t *testing.T) {
	g := newPlayingGame(t)
	for i := range g.members {
		for s := range g.members[i].Pack {
			g.members[i].Pack[s] = 3
		}
	}
	if msg := g.giveMazeItem(49); msg != mazePackFull {
		t.Errorf("要說 MAZEMOVE:85,得到 %q", msg)
	}
	for i := range g.members {
		for s := range g.members[i].Pack {
			if g.members[i].Pack[s] != 3 {
				t.Fatalf("背包滿的時候不該動任何一格")
			}
		}
	}
}

// TestLootEventTable:六件定點道具的對照表(docs/re/202 §3)。
// ⚠ 705 **不在表裡** —— 它是謎題的獎賞,不是踩到就給。
func TestLootEventTable(t *testing.T) {
	want := map[int]int{202: 49, 303: 48, 304: 50, 305: 6, 314: 52}
	if len(maze.LootEvents) != len(want) {
		t.Fatalf("表應該有 %d 筆,得到 %d", len(want), len(maze.LootEvents))
	}
	for ev, item := range want {
		if got := maze.LootEvents[ev]; got != item {
			t.Errorf("事件 %d 應該給道具 %d,得到 %d", ev, item, got)
		}
	}
	if _, in := maze.LootEvents[maze.TargetRiddle]; in {
		t.Error("705 不該在踩到就給的那張表裡")
	}
	if maze.RiddleReward != 29 {
		t.Errorf("謎題獎賞應該是風暴戒(29),得到 %d", maze.RiddleReward)
	}
}

// TestGroupSpellHitsPartyToo:範圍內的**隊員**也會吃到(docs/re/208)。
//
// ⚠ 這一條擋的是「順手加一個 IsMonster 過濾」—— 加了之後群體法術變成安全的,
// 而畫面上完全看不出規則被改過:玩家只會覺得這個法術很好用。
// 原版的掃描迴圈跑滿 14 個單位槽(`cmp ax, 0Dh`),沒有敵我判斷。
func TestGroupSpellHitsPartyToo(t *testing.T) {
	g, s := groupSpellGame(t, 2, 14, 12)
	// 再擺一位隊員在範圍內(施法者本人留在遠處的 (13,13),不受影響)。
	g.field.Units[combat.PartyBase+1] = combat.Unit{Name: "灰燼", HP: 20,
		Facing: combat.North, X: 12, Y: 12}
	if !g.castAt(s, 10, 13, 12) {
		t.Fatalf("應該施放成功:%v", g.field.Log)
	}
	if g.field.Units[combat.MonsterBase].HP >= 20 {
		t.Error("範圍內的怪應該吃到傷害")
	}
	if g.field.Units[combat.PartyBase+1].HP >= 20 {
		t.Error("範圍內的**隊員**也應該吃到傷害(docs/re/208 §2:迴圈上界 13)")
	}
}

// TestGroupSpellAreaCountsPartyAsPresent:區域裡只有隊員時,法術照樣放得出去。
// 「目標區域內沒有人」是**掃不到任何單位**,不是「掃不到怪」。
func TestGroupSpellAreaCountsPartyAsPresent(t *testing.T) {
	g, s := groupSpellGame(t, 2, 30, 30) // 怪擺得很遠
	g.field.Units[combat.PartyBase+1] = combat.Unit{Name: "灰燼", HP: 20,
		Facing: combat.North, X: 12, Y: 12}
	if !g.castAt(s, 10, 13, 12) {
		t.Fatalf("區域裡有隊員就不算「沒有人」:%v", g.field.Log)
	}
	if logHas(g.field.Log, "目標區域內沒有人") {
		t.Errorf("不該說沒有人:%v", g.field.Log)
	}
}

// ── 營地施法的發動判定(docs/re/209)────────────────────────────────────

// TestCampCastCanFizzle:效力低就放不出來,而且**法力照樣扣**。
//
// ⚠ 這一條與戰鬥那一支是**同一條規則**(原版在營地另有一份程式碼,
// 算式一字不差)。擋的是「營地不判發動」——那會讓低投入的法術在營地
// 百發百中,而畫面上只是看起來比較好用。
func TestCampCastFizzlesAndStillCostsPoints(t *testing.T) {
	buff := original.Spell{Index: 0, Name: "強壯術", School: 1, Effect: magic.EffStrength,
		Power: 1, UnitCost: 10} // 效力 = 1×2÷10 → 四捨五入 0,必定失敗
	caster := testWizard("施法者", "1000000000")
	target := testWizard("目標", "0000000000")
	target.Str = 10

	g := &Game{
		members: []original.Character{caster, target},
		spells:  []original.Spell{buff},
		rand:    combat.NewRand(1), // 真亂數:效力 0 時擲什麼都失敗
		chars:   make([]original.Character, 25),
	}
	for i, c := range g.members {
		c.ID = i + 1
		g.chars[i] = c
	}
	g.town = &townState{campMode: 'C', campWho: 0, campWho2: -1, castStage: 3,
		castSpell: buff, castInput: "10"}
	sp0 := g.members[0].SP

	g.castInCamp(0, 1)

	if g.members[1].Str != 10 {
		t.Errorf("發動失敗不該有效果,目標力量 = %d", g.members[1].Str)
	}
	if !strings.Contains(g.town.msg, magic.MsgSpellFails) {
		t.Errorf("要說 CAMP:128「%s」,得到 %q", magic.MsgSpellFails, g.town.msg)
	}
	if g.members[0].SP != sp0-10 {
		t.Errorf("失敗照樣扣法力:%d → %d,想要 %d", sp0, g.members[0].SP, sp0-10)
	}
}

// 世界地圖按 C 紮營,而且算**野外**(打得到獵)。
//
// 原版:`Making Camp..` 在 WRLDMOVE.EXE 與 MAZEMOVE.EXE 各一次、
// TOWN.EXE 零次(docs/spec/14 §12-B)。營地的 11 個指令在原版是
// 野外隨時可用的 —— 只能在城鎮紮營會改變遠征地城的節奏。
func TestCampFromWorldIsWild(t *testing.T) {
	g := newPlayingGame(t)
	press(t, g, ebiten.KeyC)
	if g.town == nil {
		t.Fatal("世界地圖按 C 沒有紮營")
	}
	if g.town.mode != townCamp {
		t.Errorf("紮營之後的模式是 %v,要 townCamp", g.town.mode)
	}
	if !g.town.wild {
		t.Error("世界地圖紮營要算野外(打得到獵)")
	}
	// 拔營回地圖,不是回建築清單 —— 野外沒有建築清單。
	press(t, g, ebiten.KeyEscape)
	if g.town != nil {
		t.Errorf("野外拔營應該回地圖,卻停在 %v", g.town.mode)
	}
}

// 城鎮裡紮營算**室內** —— 先前的判定是「不在迷宮 + 模式是營地」,
// 那在城鎮裡也成立,於是城裡打得到獵,而畫面上完全看不出哪裡不對。
func TestCampInTownIsIndoors(t *testing.T) {
	g := newPlayingGame(t)
	if len(g.townSites) == 0 {
		t.Skip("沒有城鎮座標表")
	}
	s := g.townSites[0]
	if !g.enterTown(s.X, s.Y) {
		t.Fatal("進不了城鎮")
	}
	press(t, g, ebiten.KeyZ)
	if g.town.mode != townCamp {
		t.Fatalf("城鎮按 Z 沒進營地,模式是 %v", g.town.mode)
	}
	if g.town.wild {
		t.Error("城鎮裡紮營要算室內(打不到獵)")
	}
}
