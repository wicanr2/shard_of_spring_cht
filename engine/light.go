package main

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
	g.party.RefreshLight()
}

// loadLight 把 `GROUPS.DAT` 的三個光源欄位讀進隊伍狀態(位移 45/59/61)。
// 生效能見度不在記錄裡,由 RefreshLight 現算(docs/re/204 §2)。
func (g *Game) loadLight() {
	g.party.LightTurns = g.group.LightTurns
	g.party.VisLit, g.party.VisDark = g.group.VisLit, g.group.VisDark
	g.syncMazeNum()
}
