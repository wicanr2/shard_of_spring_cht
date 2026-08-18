package rules

import "math"

// 遭遇的怪物由「區域編號」挑。docs/re/169。
//
// 原版把區域編號(`ds:3656`)填在**送出戰鬥的那一支模組**裡,`CMBT` 才讀 ——
// 所以區域不是戰鬥的參數,是「你人在哪裡」的後果。

// ZoneTolerance 是接受一隻候選怪物的條件:`|難度階級 − 區域| ≤ 1`。
//
// 原版 `sub ax, ds:3656 / neg(若為負) / cmp ax,1 / jg 回頭`——
// **`neg` 是逐位元組讀到的**,所以判定是絕對值,不是單邊。
const ZoneTolerance = 1

// WorldZone 回傳世界地圖上這一格的遭遇區域(docs/re/169 §2)。
//
//	x > 70 且 y ≤ 60 → 3(東北)
//	y > 60           → 6(南方,Islandia)
//	其餘             → 1(西北,起始區)
//
// ⚠ 分界線 x = 70 上就是「ISLANDA」拱門那一格 —— 走過去等於換一個遭遇區。
func WorldZone(x, y int) int {
	switch {
	case x > 70 && y <= 60:
		return 3
	case y > 60:
		return 6
	}
	return 1
}

// MazeZone 回傳迷宮的遭遇區域(docs/re/169 §3)。
//
// mazeNo 是當前迷宮編號(原版 `ds:3534`;`99` = 不在迷宮)。
// mx / my 是樓層座標 —— **只有第 5 座用得到**。
func MazeZone(mazeNo, mx, my int) int {
	switch mazeNo {
	case 1, 2:
		return 1
	case 3, 6:
		return 3
	case 4:
		return 5
	case 5:
		// 塔的西北角比較淺,深處明顯更難。
		if mx < 25 && my < 25 {
			return 2
		}
		return 8
	case 12:
		return 9
	}
	// ⚠ 其餘(7–11、13)走預設 6 —— 那一行在原版的比較鏈最前面,
	// 之後的比較只覆寫命中的,所以這是讀到的行為不是漏掉。
	return 6
}

// NotInMaze 是「不在任何迷宮」的哨兵(原版 `ds:3534 = 99`,docs/re/169 §4)。
// 與背包空格是同一個值 —— 「只能在野外打獵」就是這個變數的直接後果。
const NotInMaze = 99

// ZoneAccepts 回傳這個難度階級能不能在這個區域出現。
func ZoneAccepts(zone, tier int) bool {
	d := tier - zone
	if d < 0 {
		d = -d
	}
	return d <= ZoneTolerance
}

// ---------------------------------------------------------------------------
// 一場遭遇有幾隻、是哪幾隻(docs/re/225 §2、§6)
// ---------------------------------------------------------------------------

// EncounterOffset 是隻數算式的常數項。原版是 `ds:9464`,
// 也就是擲骰式 `INT(RND×N)+1` 的那個 `+1`(docs/re/136 §3)。
const EncounterOffset = 1

// EncounterHalf 是算式裡的 `ds:9460 = 0.5`(docs/re/153 §6)。
const EncounterHalf = 0.5

// EncounterCount 回傳這一場有幾隻怪。`cap` 是遭遇表的欄 1。
//
//	隻數 = INT( cap × RND × 0.5 ) + cap × 0.5 + 1      (最後四捨五入成整數)
//	隻數 > cap → 隻數 = cap                             ★ 上限就是同一欄
//
// ⚠ **上限那一行不是保險絲,是規則**(`CMBT 0x11245` 的 `cmp` + `jl`)——
// 少了它,`cap = 7` 的表列會擲出 8 隻,而實測十二場的最大值就是 7。
//
// ⚠ 最後一步是**四捨五入**不是截尾(`INT 3F:77` 沒配 `INT 3D:03`,
// docs/re/185):`cap` 是奇數時兩者差 1。
func EncounterCount(capCol int, roll float64) int {
	if capCol <= 0 {
		return 1
	}
	n := float64(int(float64(capCol)*roll*EncounterHalf)) +
		float64(capCol)*EncounterHalf + EncounterOffset
	out := int(math.Floor(n + 0.5)) // roundHalfUp:負值也要往下取
	if out > capCol {
		out = capCol
	}
	if out < 1 {
		out = 1
	}
	return out
}

// EncounterRun 回傳「這一種怪連放幾隻」。`placed` 是已經放進清單的數量。
//
//	n = round( 隻數 × RND − 已放 + 1 )
//
// `CMBT 0x11294`–`0x112D1`:兩個 `RND` 分別決定**挑哪一欄**與**連放幾隻**,
// 所以一場遭遇會出現「四隻甲 ＋ 一隻乙」這種組合,而不是每一隻各擲一次。
//
// ⚠ 回傳值可能 ≤ 0 —— 原版的內層 `FOR` 在那種情況下一次都不跑,
// 外層迴圈重挑一欄。⛔ 不要把它夾到 1:那會改變組成的分佈。
func EncounterRun(total, placed int, roll float64) int {
	return int(math.Floor(float64(total)*roll - float64(placed) + EncounterOffset + 0.5))
}
