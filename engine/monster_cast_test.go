package main

// 怪物施法(docs/re/226)的驗收。三條:投入點數、目標格、施完結束回合。

import (
	"strings"
	"testing"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// newCastingMonsterGame 建一場「一隻會施法的怪 vs 一個隊員」的戰鬥。
//
// 怪物固定放**系別 1**(第二回合 → FIRE STORM,法術 3),法力給滿,
// 所以 rules.MonsterCasts 在第 2 回合必定成立(法力 > 1 + 回合 == 2)。
func newCastingMonsterGame(t *testing.T, unitCost int) *Game {
	t.Helper()
	spells := make([]original.Spell, 20)
	for i := range spells {
		spells[i] = original.Spell{Index: i + 1, Name: "法術", School: 1,
			Effect: 2, Power: 3, UnitCost: unitCost} // 類別 2 = 單體傷害
	}
	// 法術 3 是 FIRE STORM:類別 1 = 群體傷害(rules.GroupDamageClass)。
	//
	// ⚠ 威力給 100 是為了讓**發動判定必過**(效力 = round(欄4 × 投入 ÷ 欄5),
	// 要 ≥ d100,docs/re/201)—— 否則這幾條驗收會被一次失敗的擲骰弄成紅的,
	// 而失敗的原因與它們要驗的東西無關。
	spells[2] = original.Spell{Index: 3, Name: "FIRE STORM", School: 1,
		Effect: rules.GroupDamageClass, Power: 100, UnitCost: unitCost}

	g := &Game{
		spells:  spells,
		items:   map[int]combat.Item{},
		rand:    combat.NewRand(1),
		noSrc:   map[int]bool{},
		members: []original.Character{{Party: '1', Name: "灰燼", ID: 1, Class: '1', Race: 'H', Speed: 10, Str: 10, ToHit: 10, MaxHP: 50, HP: 50, Level: 5, Weapon: original.NotEquipped, Armor: original.NotEquipped}},
	}
	g.field = combat.Build(g.members, []original.Monster{
		{Index: 1, Name: "Kobold", Speed: 6, HPDie: 20, ToHit: 7, Tier: 1},
	}, g.items, combat.NewRand(7))
	f := g.field
	m := &f.Units[combat.MonsterBase]
	m.SP = 40           // 法力給滿,不讓「不夠就全押」那一條混進來
	m.Action = 1        // 系別 1 → 第二回合 FIRE STORM
	m.X, m.Y = 10, 10   // 與隊員拉開,免得普攻先發生
	f.Units[combat.PartyBase].X, f.Units[combat.PartyBase].Y = 16, 16
	f.Round = rules.MonsterForcedCastRound // 第 2 回合必定施法
	g.points[combat.MonsterBase] = 12
	g.actor = combat.PartyBase
	return g
}

// 驗收 1:投入 = 單價 × 2,而且法力真的被扣掉。
func TestMonsterInvestsTwoLevels(t *testing.T) {
	const unitCost = 4
	g := newCastingMonsterGame(t, unitCost)
	before := g.field.Units[combat.MonsterBase].SP
	if !g.monsterCast(combat.MonsterBase) {
		t.Fatal("第 2 回合、法力 40 的怪物應該會施法")
	}
	spent := before - g.field.Units[combat.MonsterBase].SP
	if want := rules.MonsterInvestLevels * unitCost; spent != want {
		t.Errorf("投入 %d 點,應為單價 %d × %d = %d",
			spent, unitCost, rules.MonsterInvestLevels, want)
	}
	// 訊息列要看得到投入了幾點 —— 這是玩家唯一能察覺投入量的地方。
	joined := strings.Join(g.field.Log, "\n")
	if !strings.Contains(joined, "施放") {
		t.Errorf("訊息列沒有施法紀錄:%v", g.field.Log)
	}
}

// 驗收 2:法力不夠兩級時**全押**,不是不施法。
func TestMonsterInvestFallsBackToAllSP(t *testing.T) {
	for _, c := range []struct{ sp, cost, want int }{
		{sp: 40, cost: 4, want: 8},
		{sp: 7, cost: 4, want: 7},  // 不足兩級 → 全押
		{sp: 8, cost: 4, want: 8},  // 剛好兩級(原版寫 `> 2×單價 − 1`)
		{sp: 2, cost: 1, want: 2},
		{sp: 0, cost: 3, want: 0},
		{sp: -1, cost: 3, want: 0}, // 不可能出現,但不要回負數
	} {
		if got := rules.MonsterInvest(c.sp, c.cost); got != c.want {
			t.Errorf("法力 %d／單價 %d:投入 %d,應為 %d", c.sp, c.cost, got, c.want)
		}
	}
}

// 驗收 3:目標格 = 它鎖定的那個人 —— 群體傷害以那一格為中心的 5×5。
//
// ⚠ 這一條驗的是**格子**不是「有沒有打到」:把隊員放在遠處,
// 施法之後他要掉血;而如果目標格取成怪物自己那一格,5×5 就構不著他。
func TestMonsterCastCentersOnItsTarget(t *testing.T) {
	g := newCastingMonsterGame(t, 1)
	f := g.field
	p := &f.Units[combat.PartyBase]
	p.X, p.Y = 20, 20 // 離怪物 10 格以上,遠超過 5×5
	hp := p.HP
	if !g.monsterCast(combat.MonsterBase) {
		t.Fatal("應該會施法")
	}
	if f.Units[combat.PartyBase].HP >= hp {
		t.Errorf("隊員沒有掉血(%d → %d)—— 目標格沒有落在他身上",
			hp, f.Units[combat.PartyBase].HP)
	}
	// 施法結束這一隻的回合(手冊 p.35 / rules.ActCast.EndsTurn)
	if g.points[combat.MonsterBase] != 0 {
		t.Errorf("施完法還剩 %d 點行動點", g.points[combat.MonsterBase])
	}
}

// 驗收 4:法力 ≤ 1 的怪物一次都放不出來(rules.MonsterCasts 的硬門檻)。
func TestMonsterWithoutSPDoesNotCast(t *testing.T) {
	g := newCastingMonsterGame(t, 1)
	g.field.Units[combat.MonsterBase].SP = 1
	if g.monsterCast(combat.MonsterBase) {
		t.Error("法力 1 應該施不出來(門檻是 > 1)")
	}
}
