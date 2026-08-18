package main

import (
	"shardofspring/internal/combat"
	"shardofspring/internal/rules"
)

// 怪物施法。docs/re/226(投入點數與目標格)＋ 170(選招)＋ 186 §3(時機)。
//
// 三段規則各有出處,合起來才是一次施法:
//
//	施不施 → rules.MonsterCasts  法力 > 1 且(回合 == 2 或 30%)
//	放哪招 → rules.MonsterSpell  系別內隨機;第二回合起固定放風暴
//	投多少 → rules.MonsterInvest 單價 × 2,不夠就全押
//	放哪格 → **它鎖定的那個人所在的格**(re/226 §3)
//
// ⚠ 原版不另外挑格子:它把游標捲到屬性 15 那個單位身上,
// 接下來走的是**與玩家完全相同**的那一段程式(`CMBT 0x14F20`)。
// 所以這裡直接呼叫 castAt —— 引擎的玩家施法走的也是它。

// monsterCast 讓第 i 隻怪物嘗試施法。
//
// 回傳 true = 這一隻這一輪已經用掉了(施法結束回合,手冊 p.35);
// false = 沒施法,呼叫端接著走 MonsterTurn。
func (g *Game) monsterCast(i int) bool {
	f := g.field
	if f == nil || len(g.spells) == 0 {
		return false
	}
	u := f.Units[i]
	if !rules.MonsterCasts(u.SP, f.Round, f.Rand.Float01()) {
		return false
	}
	n := g.pickMonsterSpell(u.Action, f.Round)
	if n < 1 || n > len(g.spells) {
		// 系別 2 / 5 在第二回合起沒有分支(re/170 §4 未解)——
		// 挑不到就當作不施法。⚠ 這是**實作決定**,不是原版行為。
		return false
	}
	s := g.spells[n-1]
	invest := rules.MonsterInvest(u.SP, s.UnitCost)
	if invest < 1 {
		return false
	}
	// 目標格 = 它鎖定的那個人(re/226 §3)。鎖定的人不在場就重挑一個,
	// 這正是 Retarget 在做的事(re/186 §2:死了或離場才換)。
	j := f.Retarget(i)
	if j < 0 {
		return false
	}
	t := f.Units[j]
	// 扣法力:原版在**選目標之前**就扣掉(`CMBT 0x15791`,docs/re/226 §2),
	// 玩家那一支也是(confirmCastSP)。castAt 兩個呼叫端都先扣好。
	f.Units[i].SP -= invest
	if f.Units[i].SP < 0 {
		f.Units[i].SP = 0
	}
	// castAt 用 g.castUnit 當施法者 —— 換成這隻怪,而且**不會**碰
	// g.actor 那一段(castUnit != actor),所以不影響玩家的回合。
	g.castUnit = i
	ok := g.castAt(s, invest, t.X, t.Y)
	g.castUnit = combat.PartyBase
	if ok {
		// 施法結束這一隻的回合(rules.ActCast.EndsTurn)。
		g.points[i] = 0
	}
	return ok
}

// pickMonsterSpell 擲出這一回合要放的法術編號(1-based),0 = 挑不到。
//
// ⚠ 第一回合挑到群體傷害要**重擲**(re/171 §2),不是換一招 ——
// 重擲的次數原版沒有上限,這裡設 8 次上限只是防無窮迴圈:
// 每個系別最多只有一招是群體傷害,擲不到的機率隨次數指數下降。
func (g *Game) pickMonsterSpell(family, round int) int {
	n := rules.SpellFamilyAttacks[family]
	if n < 1 {
		n = 1
	}
	for try := 0; try < 8; try++ {
		spell, reroll := rules.MonsterSpell(family, round, g.field.Rand.Roll(n))
		if !reroll {
			return spell
		}
	}
	return 0
}
