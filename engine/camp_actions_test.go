package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/magic"
	"shardofspring/internal/original"
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
		rand:    combat.NewRand(1),
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
