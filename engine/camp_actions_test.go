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
