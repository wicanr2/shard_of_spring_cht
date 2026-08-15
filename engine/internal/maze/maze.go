// Package maze 是迷宮場景的狀態與規則。
//
// 規則出自 docs/spec/08-maze-scene.md;每一條在下面註明章節。
package maze

import "shardofspring/internal/original"

// VisibilityIsRadius:把 GROUPS.DAT 的能見度值當成視野半徑。
//
// ⚠ **實作決定,不是 RE 結論**(docs/spec/08 §3)。支持它的只有
// 「出貨值 3 / 2 的量級與 9×9 視野相容」。解出來時改這一個地方。
const VisibilityIsRadius = true

// DirZeroMatchesAny:事件表方向欄的 0 當成「任何朝向都觸發」。
//
// ⚠ 0 的語意未解(docs/re/60 §5)。選這個讀法的理由是它是唯一
// **不會讓那些事件永遠觸發不了**的讀法 —— 不是因為它比較可能。
const DirZeroMatchesAny = true

// Facing 沿用全專案的 1北 2東 3南 4西。
type Facing int

const (
	North Facing = 1
	East  Facing = 2
	South Facing = 3
	West  Facing = 4
)

// delta 回傳朝向在 (Major, Minor) 上的位移。
//
// ⚠ Major 是乘 81 的那個索引(docs/re/137)。哪個方向動 Major、
// 哪個動 Minor,**沒有被 RE 確認過** —— 這裡取「Major 是水平、Minor 是垂直」,
// 與世界地圖的 x/y 對齊,是實作決定。走起來若發現迷宮橫豎顛倒,改這裡。
func (f Facing) delta() (dMajor, dMinor int) {
	switch f {
	case North:
		return 0, -1
	case East:
		return 1, 0
	case South:
		return 0, 1
	case West:
		return -1, 0
	}
	return 0, 0
}

// State 是隊伍在迷宮裡的位置。
type State struct {
	Major, Minor int
	Facing       Facing
	Visibility   int // GROUPS.DAT 位移 59/61 的生效值
}

// Result 說明一次按鍵造成了什麼。與 world.Result 同一套語意。
type Result int

const (
	Turned Result = iota
	Moved
	Blocked
	// Left 是走出迷宮邊界 —— 原版在這一刻印 `Leaving maze ..` 並切回世界地圖
	// (docs/re/147:實跑從入口那一格往外走一步就離開)。
	Left
)

// Step 處理一次方向輸入。**朝向不同時只轉身,不位移** ——
// 與世界地圖同一條規則(docs/spec/05 §6)。
func (s *State) Step(dir Facing, m *original.Maze) Result {
	if s.Facing != dir {
		s.Facing = dir
		return Turned
	}
	dM, dm := dir.delta()
	nM, nm := s.Major+dM, s.Minor+dm
	// 走出邊界 = 離開迷宮(原版的 `Leaving maze ..`,docs/re/147)。
	//
	// ⚠ 這條**放在可通行性之前** —— 邊界外的格子讀出來是不可通行,
	// 若先判可通行就會變成「撞牆」,而玩家會被關在迷宮裡出不去。
	if !m.InBounds(nM, nm) {
		return Left
	}
	if !original.MazePassable(m.At(nM, nm)) {
		return Blocked
	}
	s.Major, s.Minor = nM, nm
	return Moved
}

// Trigger 是一次事件觸發的結果。
type Trigger struct {
	Kind   TriggerKind
	Text   string // KindText:DT 的敘述
	Number int    // KindText:文字編號;KindCrossLevel:未解的目標編號
	Major  int    // KindTeleport:目的地
	Minor  int
}

type TriggerKind int

const (
	KindNone       TriggerKind = iota
	KindText                   // 顯示房間敘述
	KindTeleport               // 同一層內傳送
	KindCrossLevel             // 跨關卡 —— 目標編號查不到文字
)

// Scan 掃事件表,回傳第一個命中的事件。
//
// ⚠ **掃全表是刻意的**(docs/spec/08 §4):原版就是 `FOR i = 0 TO 105`。
// 建索引要先假設「同一格只有一個事件」,而那件事沒有被驗證過。
func Scan(evs []original.Event, s State, text map[int]string) Trigger {
	for _, e := range evs {
		if e.Major != s.Major || e.Minor != s.Minor {
			continue
		}
		if e.Dir != 0 && Facing(e.Dir) != s.Facing {
			continue
		}
		if e.Dir == 0 && !DirZeroMatchesAny {
			continue
		}
		if e.Target < original.TextTargetMin {
			return Trigger{Kind: KindTeleport, Major: e.Target, Minor: e.DestMinor}
		}
		if t, ok := text[e.Target]; ok {
			return Trigger{Kind: KindText, Text: t, Number: e.Target}
		}
		// 目標 ≥ 100 但查不到文字 → 跨關卡(docs/spec/08 §5,4/4 零例外)
		return Trigger{Kind: KindCrossLevel, Number: e.Target}
	}
	return Trigger{}
}

// Visible 回傳這一格在目前的能見度下畫不畫。
// 切比雪夫距離 > 半徑 → 不畫(docs/spec/08 §3)。
func Visible(s State, major, minor int) bool {
	if !VisibilityIsRadius || s.Visibility <= 0 {
		return true
	}
	dM, dm := major-s.Major, minor-s.Minor
	if dM < 0 {
		dM = -dM
	}
	if dm < 0 {
		dm = -dm
	}
	if dM < dm {
		dM = dm
	}
	return dM <= s.Visibility
}
