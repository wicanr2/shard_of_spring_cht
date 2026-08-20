package main

import (
	"shardofspring/internal/rules"
	"shardofspring/internal/town"
	"shardofspring/internal/world"
)

// 光源與能見度的接線(docs/re/204)。
//
// 規則本體在 `internal/world`(`State.light` / `Daylight`),因為原版把它與
// 時鐘寫在**同一支常式**裡(`USERLIB` 0x1043A–0x10577)。這一檔只負責
// 「引擎的哪個狀態對應原版的哪一欄」。

// syncMazeNum 把「現在在不在迷宮、是哪一座」同步到隊伍狀態,並立刻重算
// 生效能見度。**每一處改動 `g.level` 之後都要呼叫**。
//
// ⚠ 這一步不花動作,所以用 RefreshLight 而不是 Tick —— 進迷宮的那一刻
// 不該燒掉一回合火把。
func (g *Game) syncMazeNum() {
	g.party.MazeNum = g.mazeNumber()
	g.refreshNightVision()
	g.party.RefreshLight()
}

// 夜視的能見度。docs/re/229 §1。
//
//	沒有光的能見度(GROUPS.DAT 位移 61)= 2
//	隊上有**戰士**帶夜視 → 3
//
// ⚠ 原版把這一段放在 `MAZEMOVE` 每一輪都會跑的初始化裡
// (清逃跑旗標的後面),所以它會跟著隊伍變化重算。引擎在
// **進迷宮**與**讀檔**兩處呼叫 —— 隊伍在迷宮裡不會換人,兩者等價。
//
// ⚠ **職業一定要一起判。** 位移 46 在巫師表是「風誌」(法術系別 3),
// 只看旗標的話,帶風系的巫師會讓全隊在黑暗裡多看一圈。
func (g *Game) refreshNightVision() {
	v := world.VisDarkBase
	for _, c := range g.members {
		if rules.Class(c.Class) == rules.ClassHero &&
			town.HasSkill(c, town.SkillNightVision) {
			v = world.VisDarkNightVision
			break
		}
	}
	g.party.VisDark = v
	g.group.VisDark = v // 存檔要跟著走,否則讀回來又變成舊值
}

// loadLight 把 `GROUPS.DAT` 的四個欄位讀進隊伍狀態(位移 45/59/61/83)。
// 生效能見度不在記錄裡,由 RefreshLight 現算(docs/re/204 §2)。
//
// ⚠ **迷宮編號取自記錄(位移 83),不是取自 `g.level`。** 讀檔的當下
// 迷宮還沒重新載入(`resumeMaze` 在後面),用 `g.mazeNumber()` 會拿到
// 「不在迷宮」→ 火把當場被歸零 —— 在地城裡存檔再讀回來,燈就滅了。
// 原版做的是同一件事:`MENU` 的載入常式先讀位移 83,**再**決定要
// `CHAIN` 到 `WRLDMOVE` 還是 `MAZEMOVE`(docs/re/204 §1)。
func (g *Game) loadLight() {
	g.party.LightTurns = g.group.LightTurns
	g.party.VisLit, g.party.VisDark = g.group.VisLit, g.group.VisDark
	g.party.MazeNum = g.group.MazeNum
	// ⚠ 讀檔之後**重算一次夜視** —— 存檔裡那一格是上一次算出來的結果,
	// 而隊伍在城鎮換過人之後它就過期了。原版每進一次迷宮都重算。
	g.refreshNightVision()
	g.party.RefreshLight()
}
