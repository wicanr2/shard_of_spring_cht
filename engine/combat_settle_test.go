package main

// 戰鬥結算的回歸測試(combat_scene.go 的 g.settle())。
//
// 這一條釘的是 T1 整合測試翻出來的缺口:**勝負在最後一隻怪物倒下的那一刻
// 就定了**,而結算原本只寫在 endTurn() 裡 —— 隊員砍死最後一隻怪之後
// 如果還有行動點數(或還有別的隊員沒動),endTurn() 不會被呼叫,
// 於是打贏了卻沒發經驗、沒撿到金幣。
//
// ⚠ 這種缺口沒有症狀:畫面照樣顯示戰鬥結束、ESC 照樣回世界地圖,
// 只有經驗值那一欄默默沒動。整合測試靠種子碰到它,這裡把它釘死。

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// newKillingBlowGame 擺一個「一擊必殺、而且殺完還有點數」的場面:
//
//   - 隊員赤手(武器 ≥ 60 → 傷害面數 = 力量),力量 12 → 傷害至少 1
//   - 命中能力 30 → 門檻 120,d100 擲不出不中(docs/re/185:值域 1–101)
//   - 速度 30 → 行動點數 30,攻擊只花 3(docs/spec/12 §2)
//   - 怪物生命骰 1 → 生命 1,站在隊員面前
func newKillingBlowGame(t *testing.T) *Game {
	t.Helper()
	c := original.Character{ID: 1, Name: "凱恩", Class: '1', Race: 'H',
		Speed: 30, Str: 12, ToHit: 30, MaxHP: 30, HP: 30, Level: 3,
		Weapon: original.NotEquipped, Armor: original.NotEquipped}
	m := original.Monster{Index: 0, Name: "地精", Speed: 1, Str: 1, ToHit: 1,
		HPDie: 1, Weapon: 0, Class: 1, Armor: 0, Exp: 40, Tier: 1, SP: 0}

	g := &Game{
		members: []original.Character{c},
		chars:   []original.Character{c},
		items:   map[int]combat.Item{},
		rand:    combat.NewRand(7),
	}
	g.field = combat.Build(g.members, []original.Monster{m}, g.items, g.rand)
	g.field.Place()
	g.field.ResetPoints(&g.points)
	g.settled = false
	g.actor = g.firstActor()
	if g.actor < 0 {
		t.Fatal("fixture 有問題:firstActor 找不到能動的人")
	}
	// 把怪物挪到隊員正前方(北)—— Place() 的佈陣是佔位,測試要的是相鄰。
	u := &g.field.Units[g.actor]
	u.Facing = combat.North
	mon := &g.field.Units[combat.MonsterBase]
	mon.X, mon.Y = u.X, u.Y-1
	mon.HP = 1
	return g
}

func TestPlayerKillingBlowAwardsSpoils(t *testing.T) {
	g := newKillingBlowGame(t)
	actor := g.actor
	exp0, gold0 := g.members[0].Exp, g.group.Gold

	if !g.boardKey(ebiten.KeyA) {
		t.Fatal("攻擊鍵應該被戰場吃掉")
	}
	if o := g.field.Outcome(); o != combat.MonstersDead {
		t.Fatalf("一擊應該打死生命 1 的怪物,得到 %v;log:%v", o, g.field.Log)
	}
	// 這一條是缺口成立的前提:殺完還有點數 → 原本不會走到 endTurn()。
	if g.points[actor] <= 0 {
		t.Fatalf("fixture 失效:殺完點數就用光了(%d),測不到這個缺口", g.points[actor])
	}
	if g.members[0].Exp <= exp0 {
		t.Errorf("最後一擊由玩家打出時沒有發經驗:%v → %v", exp0, g.members[0].Exp)
	}
	if g.group.Gold < gold0 {
		t.Errorf("金幣反而變少了:%.0f → %.0f", gold0, g.group.Gold)
	}
	if !hasLine(g.field.Log, "戰鬥結束") {
		t.Errorf("紀錄裡應該有「戰鬥結束」,得到 %v", g.field.Log)
	}
	if g.actor != -1 {
		t.Errorf("結算後不該還有人能動,g.actor = %d", g.actor)
	}
}

func TestSettleIsIdempotent(t *testing.T) {
	g := newKillingBlowGame(t)
	g.boardKey(ebiten.KeyA)
	exp1 := g.members[0].Exp
	gold1 := g.group.Gold

	// 三條入口再各走一次 —— 冪等旗標擋不住的話經驗會翻倍。
	g.settle()
	g.endTurn()
	g.stepCombat()

	if g.members[0].Exp != exp1 {
		t.Errorf("重複結算把經驗發了第二次:%v → %v", exp1, g.members[0].Exp)
	}
	if g.group.Gold != gold1 {
		t.Errorf("重複結算把金幣發了第二次:%.0f → %.0f", gold1, g.group.Gold)
	}
	if n := countLines(g.field.Log, "戰鬥結束"); n != 1 {
		t.Errorf("「戰鬥結束」出現 %d 次,應該只有 1 次;log:%v", n, g.field.Log)
	}
}

// TestSettleKeepsPriestBlessing 確認搬進 settle() 之後,祭司事件的後續文字
// 還在(docs/spec/17 §3)—— 那段判斷原本寫在 endTurn() 裡。
func TestSettleKeepsPriestBlessing(t *testing.T) {
	g := newKillingBlowGame(t)
	// startScriptedCombat 對目標 204 會把這個記號放在 log[0],settle() 靠它判斷。
	g.field.Log = append([]string{rules.PriestEncounterMark}, g.field.Log...)

	g.boardKey(ebiten.KeyA)

	if !hasLine(g.field.Log, rules.PriestBlessing) {
		t.Errorf("祭司事件打贏後應該顯示祝福文字;log:%v", g.field.Log)
	}
}

func hasLine(log []string, sub string) bool { return countLines(log, sub) > 0 }

func countLines(log []string, sub string) int {
	n := 0
	for _, s := range log {
		if strings.Contains(s, sub) {
			n++
		}
	}
	return n
}
