package maze

import "shardofspring/internal/original"

// 拉利斯之門(`DAZA REVELI`)的消費端。docs/re/205。
//
// 咒語在營地唸(docs/re/197):條件是**字串相符 且 人在第 5 座地城**,
// 效果是把 `GROUPS.DAT` 位移 65 設成 1 —— 那一篇當時沒有讀到「誰讀這個旗標」,
// 所以門開了在引擎裡仍然過不去。
//
// `MAZEMOVE` 0x1364B–0x13684 是消費端:同樣兩個條件成立時做一件事 ——
//
//	mov word ptr ds:707Eh, 10h
//
// `ds:707E` 落在迷宮格陣列裡(陣列區從 `ds:6822` 起,`INT 3F:4D` 一次清
// `0x2046` bytes = 4131 word = **81 × 51**),所以那是**編譯期就折算好的
// 常數索引**:`(0x707E − 0x6822) ÷ 2 = 1070`。
//
// ⚠ 這是本專案第一次遇到「寫入的位址本身就是答案」——
// `ds:707E` 在**全部十二支模組裡只出現這一次**(位元組層級掃過),
// 看起來像「寫了沒人讀」,實際上它是陣列的一格。
// **判準:一個只寫一次的位址,先問它落不落在某個陣列的範圍內。**
const (
	// GateFlagOffset 是隊伍記錄裡的旗標位移(docs/re/197 §3)。
	GateFlagOffset = 65
	// GateMaze 是拉利斯之塔的迷宮編號。
	GateMaze = 5
	// GateIndex 是原版陣列的線性索引。原版解碼器寫入的算式是
	// **`一列裡的第幾格 × 81 + 第幾列`**(`MAZEMOVE` 0x13FC6,docs/re/55 §1),
	// 所以 `1070 = 13 × 81 + 17` = **第 17 列的第 13 格**。
	GateIndex = 1070
	// GateTile 是開門之後填進去的圖塊值。阻擋區間是 5–10,所以
	// **9 → 16 就是「從擋路變成可通行」**(docs/formats/06 §2)。
	GateTile = 16
	// GateClosedTile 是門還關著時那一格的值,用來驗接線接對了格子。
	GateClosedTile = 9
)

// gateCell 把原版的線性索引換算成**引擎的** (Major, Minor)。
//
// 兩邊的線性排法不同,但**指到的格子相同**:
//
//	原版:index = 一列裡的第幾格 × 81 + 第幾列   (MAZEMOVE 0x13FC6)
//	引擎:Cells[第幾列 × 81 + 一列裡的第幾格]     (DecodeSQZ)
//
// 引擎的 `Major` = 第幾列(0–80)、`Minor` = 一列裡的第幾格(0–50)——
// 與 `MAZEDATA` 的值域一致(major 到 67、minor 到 49,docs/re/205 §4)。
// 所以只要把原版的索引拆開再**對調**就好。
func gateCell() (major, minor int) {
	pos, line := GateIndex/original.MazeRows, GateIndex%original.MazeRows
	return line, pos
}

// GateCell 回傳門那一格在引擎座標下的位置。
func GateCell() (major, minor int) { return gateCell() }

// OpenGate 把門那一格改成可通行。旗標沒開、或不是第 5 座地城就什麼都不做。
//
// 呼叫時機是**載入該層之後**,與原版一樣(原版每次進迷宮都重跑這一段,
// 所以旗標存在隊伍記錄裡就足夠,不必改迷宮檔)。
func OpenGate(m *original.Maze, mazeFile, flag int) bool {
	if m == nil || mazeFile != GateMaze || flag == 0 {
		return false
	}
	major, minor := gateCell()
	if !m.InBounds(major, minor) {
		return false
	}
	m.Cells[major*original.MazeRows+minor] = GateTile
	return true
}
