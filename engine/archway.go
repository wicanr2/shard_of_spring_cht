package main

import "shardofspring/internal/world"

// 「ISLANDA」拱門的敘述(WRLDMOVE:18)。規則出自 docs/re/198。
//
// ⚠ **兩個條件**:站在 (70, 57) **而且**朝南。原版 `WRLDMOVE 0x10C86`–`0x10CAA`
// 是三個比較 AND 起來的 —— 少了朝向那一項,玩家從南邊走上來也會看到
// 「透過拱門可以看見……」,而那時他背對著拱門。
const (
	archwayX      = 70
	archwayY      = 57
	archwayFacing = world.South
)

// archwayText 是 WRLDMOVE:18。原文一整段,引擎照樣一整段。
const archwayText = "你面前是一座嵌在石門裡的拱門。透過拱門可以看見一大片島嶼," +
	"以及蜿蜒至地平線的陸地小徑。拱門上方以哥德體大寫字母刻著：「ISLANDA」"

// archwayCheck 在每次移動之後判一次。條件不成立時**不清掉現有的訊息** ——
// 那會把別的提示吃掉。
func (g *Game) archwayCheck() {
	if g.party.X == archwayX && g.party.Y == archwayY &&
		g.party.Facing == archwayFacing {
		g.overlay = archwayText
	}
}
