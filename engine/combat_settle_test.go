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
	"shardofspring/internal/town"
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
	// CMBT:60–63:金幣先掛著等玩家答「要撿嗎?」,答 Y 才入帳。
	if g.group.Gold != gold0 {
		t.Errorf("還沒答「要撿嗎?」就先入帳了:%.0f → %.0f", gold0, g.group.Gold)
	}
	if g.pendingGold > 0 {
		g.takeGold(true)
		if g.group.Gold <= gold0 {
			t.Errorf("答 Y 之後金幣沒有進來:%.0f → %.0f", gold0, g.group.Gold)
		}
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
	// 撿一次就沒了 —— 再答一次不該再給一份。
	g.takeGold(true)
	after := g.group.Gold
	g.takeGold(true)
	if g.group.Gold != after {
		t.Errorf("「要撿嗎?」答了兩次拿到兩份:%.0f → %.0f", after, g.group.Gold)
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

// ── D4:TACTICS 顯示(docs/re/186 §2、手冊 p.35)────────────────────────

func TestTacticsLineOnlyWithTheSkill(t *testing.T) {
	g := newKillingBlowGame(t)
	mon := &g.field.Units[combat.MonsterBase]
	mon.Target = combat.PartyBase // 鎖定第一位隊員

	g.field.Tactics = false
	if got := g.tacticsLine(*mon); got != "" {
		t.Errorf("沒有策略技能時不該顯示任何東西,得到 %q", got)
	}

	g.field.Tactics = true
	got := g.tacticsLine(*mon)
	if !strings.Contains(got, g.field.Units[combat.PartyBase].Name) {
		t.Errorf("有策略技能時應該顯示鎖定對象的名字,得到 %q", got)
	}
}

// TestTacticsLineNeedsARealTarget:還沒鎖定就不顯示 ——
// ⚠ 顯示「未知」會洩漏「牠有目標」這件事,而那本身就是資訊。
func TestTacticsLineNeedsARealTarget(t *testing.T) {
	g := newKillingBlowGame(t)
	g.field.Tactics = true
	mon := g.field.Units[combat.MonsterBase]
	mon.Target = 0 // 還沒鎖定(0 不在隊員範圍 9…13)
	if got := g.tacticsLine(mon); got != "" {
		t.Errorf("沒有鎖定對象時不該顯示,得到 %q", got)
	}
}

// ── 戰後掉落(docs/re/200)────────────────────────────────────────────

// TestLootIndexRange:編號永遠落在 2–46。
func TestLootIndexRange(t *testing.T) {
	for gold := 0; gold <= 2000; gold += 37 {
		for seed := 1; seed <= 5; seed++ {
			v := combat.LootIndex(gold, combat.NewRand(uint64(seed)))
			if v < combat.LootMin || v > combat.LootMax {
				t.Fatalf("金幣 %d 種子 %d 擲出 %d,超出 %d–%d",
					gold, seed, v, combat.LootMin, combat.LootMax)
			}
		}
	}
}

// TestLootIndexIsTwoWhenNoGold:金幣 0 → G 是 0 → 兩個 RND 項都是 0 → 編號 2。
func TestLootIndexIsTwoWhenNoGold(t *testing.T) {
	if v := combat.LootIndex(0, combat.NewRand(9)); v != combat.LootMin {
		t.Errorf("金幣 0 應該固定擲出 %d,得到 %d", combat.LootMin, v)
	}
}

// TestLootIndexUsesTwoRolls:兩個獨立 RND 相加是三角分佈 ——
// 若被簡化成一次 RND,分佈會變平。這裡用「靠近中位數的比例」當訊號。
//
// ⚠ 這條測的是**分佈不是值**:同樣的值域下,兩次相加會把結果拉向中間。
func TestLootIndexUsesTwoRolls(t *testing.T) {
	// ⚠ 金幣要挑到**不會觸發重擲**的量,否則截斷會把分佈整個扭掉:
	// G = round(70 × 0.575) = 40 → 上界 40×0.5×2 + 2 = 42 < 46。
	const gold, n = 70, 4000
	mid, total := 0, 0
	r := combat.NewRand(1234)
	for i := 0; i < n; i++ {
		v := combat.LootIndex(gold, r)
		total++
		if v >= 17 && v <= 27 { // 峰值 22 附近的四分之一寬
			mid++
		}
	}
	// 兩個均勻分佈相加 → P(|S−1| ≤ 0.25) = 0.4375;單次 RND 只有 0.25。
	if got := float64(mid) / float64(total); got < 0.35 {
		t.Errorf("中段比例 %.2f 太低 —— 分佈看起來像單次 RND,不像兩次相加", got)
	}
}

// TestSpoilsOffersLootAndAnswerSticks:結算會問「要撿嗎?」,
// 答 Y 東西進背包而且是**未鑑定**的,答 N 什麼都不留。
func TestSpoilsOffersLootAndAnswerSticks(t *testing.T) {
	for _, yes := range []bool{true, false} {
		g := newPlayingGame(t)
		for i := range g.members {
			for s := range g.members[i].Pack {
				g.members[i].Pack[s] = original.NotEquipped
			}
			g.members[i].Status = 0
		}
		g.pendingGold, g.pendingLoot = 0, nil
		msg := g.awardSpoils(deadMonsters())
		if g.pendingLoot == nil {
			t.Fatalf("結算應該問一次掉落,訊息 %q", msg)
		}
		if !strings.Contains(msg, "要撿嗎?(Y/N)") {
			t.Errorf("要問「要撿嗎?(Y/N)」,得到 %q", msg)
		}
		p := *g.pendingLoot
		g.takeLoot(yes)
		got := g.members[p.who].Pack[p.slot]
		if yes && got != p.item {
			t.Errorf("答 Y 應該把 %d 放進第 %d 格,得到 %d", p.item, p.slot, got)
		}
		if !yes && got != original.NotEquipped {
			t.Errorf("答 N 不該放東西,第 %d 格變成 %d", p.slot, got)
		}
		if yes && town.IsIdentified(g.members[p.who], p.slot) {
			t.Error("撿來的東西應該是未鑑定的(docs/re/168 §2)")
		}
		if g.pendingLoot != nil {
			t.Error("答完之後 pendingLoot 要清掉")
		}
	}
}

// deadMonsters 擺一組被打倒的怪物,讓 TotalGold 一定 > 0。
func deadMonsters() []combat.Unit {
	var us [combat.Slots]combat.Unit
	for i := 0; i < 3; i++ {
		us[combat.MonsterBase+i] = combat.Unit{Name: "地精", IsMonster: true,
			HP: 0, Tier: 5, Exp: 20}
	}
	return us[:]
}
