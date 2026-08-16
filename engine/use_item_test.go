package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/magic"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// 補兩個流程缺口(docs/spec/19-coverage.md §2-1)的測試:
//  1. 戰鬥中的 U)se an item(TestOpenUseItem_*、TestCombatUseItem_*、
//     TestCombatPotionKey_*)
//  2. 藥劑「自己/給別人」——營地(TestCampUseKey_Potion*)與戰鬥各一份
//
// 全部**不開視窗**:直接呼叫方法,不跑 ebiten.RunGame,沿用
// camp_actions_test.go / scripted_fight_test.go 已經有的 fixture 寫法。

// newCombatUseItemTestGame 建一個只給戰鬥「用道具」測試用的最小 Game。
// 兩位隊員速度都是 10(行動點數 = 速度,docs/spec/12 §2),
// 這樣第一位用完道具、回合結束後,nextActor 會換到第二位,
// 不會落到 endTurn() 去跑怪物回合 —— 測試不必管怪物,索性給空的怪物清單。
func newCombatUseItemTestGame(t *testing.T) *Game {
	t.Helper()
	c0 := original.Character{ID: 1, Name: "凱恩", Class: '1', Race: 'H',
		Speed: 10, Str: 10, ToHit: 10, MaxHP: 30, HP: 30, Level: 3,
		Weapon: original.NotEquipped, Armor: original.NotEquipped, Pack: testPack()}
	c0.Pack[0] = 30 // > 26 → 魔法道具
	c1 := original.Character{ID: 2, Name: "米菈", Class: '1', Race: 'E',
		Speed: 10, Str: 8, ToHit: 8, MaxHP: 25, HP: 25, Level: 3,
		Weapon: original.NotEquipped, Armor: original.NotEquipped, Pack: testPack()}

	// 道具:欄4=法術編號 0、欄5=投入 4、欄6=100(必定發動)。
	spell := original.Spell{Index: 0, Name: "劍術", School: 2, Effect: magic.EffStrength,
		Power: 1, UnitCost: 1}
	item := original.Item{Index: 30, Name: "神秘藥水", Col4: 0, Col5: 4, Col6: 100}

	g := &Game{
		members:  []original.Character{c0, c1},
		chars:    []original.Character{c0, c1},
		spells:   []original.Spell{spell},
		itemList: []original.Item{item},
		items:    map[int]combat.Item{},
		rand:     combat.NewRand(1),
	}
	g.field = combat.Build(g.members, nil, g.items, g.rand)
	g.field.Place()
	g.field.ResetPoints(&g.points)
	g.actor = g.firstActor()
	if g.actor < 0 {
		t.Fatalf("fixture 有問題:firstActor 找不到能動的人")
	}
	return g
}

// ── 戰鬥:openUseItem 的兩條失敗路徑不消耗點數(對稱 openCast)────────────

func TestOpenUseItem_EmptyPack_NoTurnConsumed(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	g.members[0].Pack[0] = original.NotEquipped
	before := g.points[g.actor]

	if g.openUseItem() {
		t.Fatalf("背包是空的,openUseItem 應該回 false")
	}
	if g.points[g.actor] != before {
		t.Errorf("開選單失敗不該扣點數,got %d want %d", g.points[g.actor], before)
	}
	last := g.field.Log[len(g.field.Log)-1]
	if !strings.Contains(last, "沒有可用的道具") { // CMBT:162
		t.Errorf("要有「沒有可用的道具」的訊息,得到 %q", last)
	}
}

func TestOpenUseItem_InsufficientPoints_NoTurnConsumed(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	g.points[g.actor] = rules.ActUse.Cost() - 1
	before := g.points[g.actor]

	if g.openUseItem() {
		t.Fatalf("點數不夠,openUseItem 應該回 false")
	}
	if g.points[g.actor] != before {
		t.Errorf("失敗不該扣點數,got %d want %d", g.points[g.actor], before)
	}
}

// ── 戰鬥:完整按鍵序列 U → A → Y(自己)/ T → 2(丟給第二位)──────────────

func TestCombatUseItem_SelfTarget_AppliesEffectAndEndsTurn(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	actor := g.actor

	if !g.openUseItem() {
		t.Fatalf("openUseItem 應該成功:%v", g.field.Log)
	}
	if len(g.useList) != 1 || g.useList[0].slot != 0 {
		t.Fatalf("道具清單應該只有背包第 0 格,got %+v", g.useList)
	}

	g.pickUseItem(0)
	if g.combatPotion == nil || g.combatPotion.stage != 1 {
		t.Fatalf("魔法道具應該先問自己/丟給別人,got %+v", g.combatPotion)
	}

	if !g.combatPotionKey(ebiten.KeyY) {
		t.Fatalf("Y 應該被吃掉")
	}
	if g.combatPotion != nil {
		t.Errorf("套用完應該清掉 combatPotion")
	}
	if g.field.Units[actor].Str != 10+1*4 { // 威力 1 × 投入 4
		t.Errorf("Str = %d, want %d", g.field.Units[actor].Str, 10+4)
	}
	if g.points[actor] != 0 {
		t.Errorf("用完道具應該直接結束回合(docs/spec/12 §2),points = %d, want 0", g.points[actor])
	}
	if g.actor == actor {
		t.Errorf("回合應該換下一個人,g.actor 還是原本那位")
	}
	// ⚠ 找**整份 Log** 不是只看最後一行 —— 道具可能在效果之後壞掉
	// (docs/re/190),那一句會排在後面。
	if !logHas(g.field.Log, "發動了「劍術」") {
		t.Errorf("訊息要看得到法術名,got %v", g.field.Log)
	}
}

// logHas 回傳訊息紀錄裡有沒有哪一行含 sub。
func logHas(log []string, sub string) bool {
	for _, s := range log {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func TestCombatUseItem_TossToAnother_AppliesEffectToTargetNotCaster(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	caster := g.actor
	target := combat.PartyBase + 1
	casterStrBefore := g.field.Units[caster].Str

	if !g.openUseItem() {
		t.Fatalf("openUseItem 應該成功")
	}
	g.pickUseItem(0)
	if g.combatPotion == nil || g.combatPotion.stage != 1 {
		t.Fatalf("應該進入自己/丟給別人子流程")
	}

	if !g.combatPotionKey(ebiten.KeyT) { // CMBT:165 的 T)oss(不是 G)ive)
		t.Fatalf("T 應該被吃掉")
	}
	if g.combatPotion == nil || g.combatPotion.stage != 2 {
		t.Fatalf("按 T 之後應該進入選目標,got %+v", g.combatPotion)
	}

	if !g.combatPotionKey(ebiten.Key2) { // 選第 2 位(米菈)
		t.Fatalf("選第 2 位應該被吃掉")
	}
	if g.combatPotion != nil {
		t.Errorf("選完目標應該清掉 combatPotion")
	}
	if g.field.Units[target].Str != 8+1*4 {
		t.Errorf("目標 Str = %d, want %d", g.field.Units[target].Str, 8+4)
	}
	if g.field.Units[caster].Str != casterStrBefore {
		t.Errorf("施放者自己的 Str 不該被動到,got %d want %d", g.field.Units[caster].Str, casterStrBefore)
	}
	if g.points[caster] != 0 {
		t.Errorf("丟給別人一樣要結束施放者的回合,points = %d", g.points[caster])
	}
	if !logHas(g.field.Log, "丟給") || !logHas(g.field.Log, "米菈") {
		t.Errorf("訊息要講清楚丟給誰,got %v", g.field.Log)
	}
}

func TestCombatUseItem_NotMagicItem_StillEndsTurn(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	g.members[0].Pack[0] = 5 // ≤ 26,非魔法道具
	actor := g.actor
	strBefore := g.field.Units[actor].Str

	if !g.openUseItem() {
		t.Fatalf("openUseItem 應該成功")
	}
	g.pickUseItem(0)
	if g.combatPotion != nil {
		t.Errorf("非魔法道具不該問自己/丟給別人")
	}
	if g.field.Units[actor].Str != strBefore {
		t.Errorf("非魔法道具不該改任何屬性,got %d want %d", g.field.Units[actor].Str, strBefore)
	}
	if g.points[actor] != 0 {
		t.Errorf("非魔法道具一樣結束回合(手冊 p.35),points = %d, want 0", g.points[actor])
	}
	last := g.field.Log[len(g.field.Log)-1]
	if !strings.Contains(last, "不是有效的魔法道具") { // CMBT:164
		t.Errorf("訊息要照 CMBT:164 的措辭,got %q", last)
	}
}

// combatPotionKey 的按鍵是 Y/T,不是營地的 Y/G —— 按錯模組的鍵不該被吃掉。
func TestCombatPotionKey_GIsNotRecognized(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	g.openUseItem()
	g.pickUseItem(0)
	if g.combatPotion == nil {
		t.Fatalf("前提不成立:應該已經在問自己/丟給別人")
	}
	if handled := g.combatPotionKey(ebiten.KeyG); handled {
		t.Errorf("戰鬥用的是 T)oss(CMBT:165)不是 G)ive,G 不該被吃掉")
	}
	if g.combatPotion.stage != 1 {
		t.Errorf("按了不認得的鍵不該推進階段,stage = %d", g.combatPotion.stage)
	}
}

// ── isCombatOnlyItem(具名的猜測表,docs/spec/19-coverage.md §2-2)──────

func TestIsCombatOnlyItem(t *testing.T) {
	dmgSpell := original.Spell{Index: 1, Name: "烈焰風暴", Effect: magic.EffGroupDamage}
	healSpell := original.Spell{Index: 2, Name: "治療術", Effect: magic.EffHitPoints, Power: 3}
	g := &Game{spells: []original.Spell{dmgSpell, healSpell}}

	cases := []struct {
		name string
		it   original.Item
		want bool
	}{
		{"魔法道具接群體傷害法術=戰鬥道具", original.Item{Index: 30, Col4: 1}, true},
		{"魔法道具接正威力增益法術=不是戰鬥道具", original.Item{Index: 31, Col4: 2}, false},
		{"非魔法道具(編號≤26)一律不是戰鬥道具", original.Item{Index: 20, Col4: 1}, false},
		{"魔法道具但接的法術查不到=不是戰鬥道具", original.Item{Index: 32, Col4: 999}, false},
	}
	for _, c := range cases {
		if got := g.isCombatOnlyItem(c.it); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

// ── 營地:U)se an item 的「自己/給別人」子流程(docs/spec/19-coverage.md §2-1)

func TestCampUseKey_PotionSelf_FullFlow(t *testing.T) {
	c := testWizard("藥師", "0000000000")
	c.Pack[0] = 30
	c.Identified = "1000000000"

	spell := original.Spell{Index: 0, Name: "劍術", School: 2, Effect: magic.EffStrength,
		Power: 1, UnitCost: 1}
	item := original.Item{Index: 30, Name: "神秘藥水", Col4: 0, Col5: 4, Col6: 100}

	g := &Game{
		members:  []original.Character{c},
		spells:   []original.Spell{spell},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c},
	}
	g.town = &townState{campMode: 'U', campWho: 0}

	g.campUseKey(ebiten.KeyA) // 選背包第 0 格
	if g.campPotion == nil || g.campPotion.stage != 1 {
		t.Fatalf("魔法道具應該先問自己/給別人,got %+v", g.campPotion)
	}

	g.campUseKey(ebiten.KeyY) // Y)ourself(CAMP:92)
	if g.campPotion != nil {
		t.Errorf("套用完應該清掉 campPotion")
	}
	if g.members[0].Str != 10+1*4 {
		t.Errorf("Str = %d, want %d", g.members[0].Str, 10+4)
	}
	if !strings.Contains(g.town.msg, "發動了「劍術」") {
		t.Errorf("訊息要看得到法術名:%q", g.town.msg)
	}
	if g.town.campMode != 0 || g.town.campWho != -1 {
		t.Errorf("流程結束應該回營地選單,campMode=%v campWho=%d", g.town.campMode, g.town.campWho)
	}
}

func TestCampUseKey_PotionGive_FullFlow_TargetsOtherCharacter(t *testing.T) {
	c0 := testWizard("藥師", "0000000000")
	c0.ID = 1
	c0.Pack[0] = 30
	c0.Identified = "1000000000"
	c1 := testWizard("同伴", "0000000000")
	c1.ID = 2

	spell := original.Spell{Index: 0, Name: "劍術", School: 2, Effect: magic.EffStrength,
		Power: 1, UnitCost: 1}
	item := original.Item{Index: 30, Name: "神秘藥水", Col4: 0, Col5: 4, Col6: 100}

	g := &Game{
		members:  []original.Character{c0, c1},
		spells:   []original.Spell{spell},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c0, c1},
	}
	g.town = &townState{campMode: 'U', campWho: 0}

	g.campUseKey(ebiten.KeyA) // 選背包第 0 格
	g.campUseKey(ebiten.KeyG) // G)ive it to another character(CAMP:92)
	if g.campPotion == nil || g.campPotion.stage != 2 {
		t.Fatalf("按 G 之後應該進入選目標,got %+v", g.campPotion)
	}

	g.campUseKey(ebiten.Key2) // 要交給哪位(CAMP:95):選第 2 位
	if g.campPotion != nil {
		t.Errorf("選完目標應該清掉 campPotion")
	}
	if g.members[1].Str != 10+1*4 {
		t.Errorf("目標 Str = %d, want %d", g.members[1].Str, 10+4)
	}
	if g.members[0].Str != 10 {
		t.Errorf("施放者自己的 Str 不該被動到,got %d", g.members[0].Str)
	}
	if !strings.Contains(g.town.msg, "交給") || !strings.Contains(g.town.msg, "同伴") {
		t.Errorf("訊息要講清楚交給誰,got %q", g.town.msg)
	}
}

// campPotionKey 的按鍵是 Y/G,不是戰鬥的 Y/T —— 按錯模組的鍵不該推進階段。
func TestCampUseKey_PotionKey_TIsNotRecognized(t *testing.T) {
	c := testWizard("藥師", "0000000000")
	c.Pack[0] = 30
	c.Identified = "1000000000"
	spell := original.Spell{Index: 0, Name: "劍術", School: 2, Effect: magic.EffStrength,
		Power: 1, UnitCost: 1}
	item := original.Item{Index: 30, Name: "神秘藥水", Col4: 0, Col5: 4, Col6: 100}

	g := &Game{
		members:  []original.Character{c},
		spells:   []original.Spell{spell},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c},
	}
	g.town = &townState{campMode: 'U', campWho: 0}
	g.campUseKey(ebiten.KeyA)
	if g.campPotion == nil {
		t.Fatalf("前提不成立:應該已經在問自己/給別人")
	}

	g.campUseKey(ebiten.KeyT) // 營地不認得 T,戰鬥才是 T)oss
	if g.campPotion == nil || g.campPotion.stage != 1 {
		t.Errorf("按了不認得的鍵不該推進階段,got %+v", g.campPotion)
	}
}

// ── 營地:「那是戰鬥用道具!」閘門(docs/spec/19-coverage.md §2-2)────────

func TestCampUseKey_CombatItemGate_BlocksBeforeAskingTarget(t *testing.T) {
	c := testWizard("藥師", "0000000000")
	c.Pack[0] = 30
	c.Identified = "1000000000"

	// 群體傷害 → combatOnlySpell 判定為戰鬥法術,連帶讓道具變成戰鬥道具。
	dmgSpell := original.Spell{Index: 0, Name: "烈焰風暴", School: 1, Effect: magic.EffGroupDamage,
		Power: 5, UnitCost: 1}
	item := original.Item{Index: 30, Name: "戰鬥法杖", Col4: 0, Col5: 4, Col6: 100}

	g := &Game{
		members:  []original.Character{c},
		spells:   []original.Spell{dmgSpell},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c},
	}
	g.town = &townState{campMode: 'U', campWho: 0}

	strBefore := g.members[0].Str
	g.campUseKey(ebiten.KeyA)

	if g.campPotion != nil {
		t.Errorf("戰鬥道具不該問自己/給別人,應該直接擋下")
	}
	if !strings.Contains(g.town.msg, "戰鬥用道具") {
		t.Errorf("訊息要講「那是戰鬥用道具!」(CAMP:100),got %q", g.town.msg)
	}
	if g.members[0].Str != strBefore {
		t.Errorf("被擋下的道具不該有任何效果變化,got %d want %d", g.members[0].Str, strBefore)
	}
	if g.town.campMode != 0 || g.town.campWho != -1 {
		t.Errorf("擋下之後應該回營地選單,campMode=%v campWho=%d", g.town.campMode, g.town.campWho)
	}
}

// campPotion 的殘留在 ESC 之後不該污染下一輪選人(campUseKey 開頭的防呆)。
func TestCampUseKey_StalePotionClearedOnReselect(t *testing.T) {
	c := testWizard("藥師", "0000000000")
	c.Pack[0] = 30
	c.Identified = "1000000000"
	item := original.Item{Index: 30, Name: "神秘藥水", Col4: 0, Col5: 4, Col6: 100}
	spell := original.Spell{Index: 0, Name: "劍術", School: 2, Effect: magic.EffStrength,
		Power: 1, UnitCost: 1}

	g := &Game{
		members:  []original.Character{c},
		spells:   []original.Spell{spell},
		itemList: []original.Item{item},
		rand:     combat.NewRand(1),
		chars:    []original.Character{c},
	}
	g.town = &townState{campMode: 'U', campWho: 0}
	g.campUseKey(ebiten.KeyA)
	if g.campPotion == nil {
		t.Fatalf("前提不成立:應該已經在問自己/給別人")
	}

	// 模擬 town_scene.go 的 campSubKey:ESC 直接把 campMode/campWho 重置,
	// 不會經過 campUseKey 的收尾(見 use_item.go 的說明)。
	g.town.campMode, g.town.campWho = 0, -1

	// 重新開 U)se an item,先選人。
	g.campUseKey(ebiten.Key1)
	if g.campPotion != nil {
		t.Errorf("重新選人應該清掉上一輪的殘留,got %+v", g.campPotion)
	}
}

// ── 欄6 是損壞率不是發動率(docs/re/190)────────────────────────────────

// TestMagicItemBreaksInsteadOfFailing 釘住這個方向。
//
// ⚠ 讀反了**沒有症狀**:戒指有時有效有時沒效,在這類遊戲裡看起來完全正常。
// 這一條驗的是「法術照樣發動」+「壞掉的道具離開背包」兩件事一起成立。
func TestMagicItemBreaksInsteadOfFailing(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	actor := g.actor
	// 欄6 = 100 → 必壞(火把那一類,docs/re/190 §3)
	for i := range g.itemList {
		if g.itemList[i].Index == g.members[0].Pack[0] {
			g.itemList[i].Col6 = 100
		}
	}
	before := g.field.Units[actor].Str

	g.openUseItem()
	g.pickUseItem(0)
	g.combatPotionKey(ebiten.KeyY)

	if g.field.Units[actor].Str == before {
		t.Error("欄6 = 100 是「一定壞」不是「一定不發動」—— 法術該照樣生效")
	}
	if !logHas(g.field.Log, "道具損壞了!") {
		t.Errorf("壞掉要說「道具損壞了!」,got %v", g.field.Log)
	}
	if g.members[0].Pack[0] != original.NotEquipped {
		t.Errorf("壞掉的道具該離開背包,第 0 格還是 %d", g.members[0].Pack[0])
	}
}

// TestMagicItemWithZeroBreakChanceSurvives:欄6 = 0(鑰匙那一類)永遠不壞。
func TestMagicItemWithZeroBreakChanceSurvives(t *testing.T) {
	g := newCombatUseItemTestGame(t)
	idx := g.members[0].Pack[0]
	for i := range g.itemList {
		if g.itemList[i].Index == idx {
			g.itemList[i].Col6 = 0
		}
	}
	g.openUseItem()
	g.pickUseItem(0)
	g.combatPotionKey(ebiten.KeyY)

	if logHas(g.field.Log, "道具損壞了!") {
		t.Errorf("欄6 = 0 不該壞,got %v", g.field.Log)
	}
	if g.members[0].Pack[0] != idx {
		t.Errorf("欄6 = 0 的道具該留在背包,第 0 格變成 %d", g.members[0].Pack[0])
	}
}
