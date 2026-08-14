// Package world 是世界地圖場景的狀態與規則。
//
// 全部規則出自 docs/spec/05-world-scene.md,每一條在下面註明章節。
// **規格改了才改這裡。**
package world

// 地圖尺寸。docs/spec/05 §1。
const (
	W = 103
	H = 121
)

// 朝向。docs/spec/05 §6,與 GROUPS.DAT 位移 41 相同。
type Facing int

const (
	North Facing = 1
	East  Facing = 2
	South Facing = 3
	West  Facing = 4
)

// delta 回傳該朝向的位移。
func (f Facing) delta() (dx, dy int) {
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

// Map 是 103×121 的地形值陣列。
type Map struct {
	Cells []int // len == W*H,索引 = y*W + x(docs/spec/05 §1)
}

// At 回傳 (x,y) 的地形值。界外回傳 0(= 地圖邊界,docs/spec/05 §2.1)。
//
// ⚠ 索引是 y*W + x。迷宮是「欄 × 81 + 列」,**不同的順序**
// (docs/spec/00-index.md 實作前必讀第 4 條)。寫反了地圖仍然畫得出來,
// 只是轉了 90 度 —— 不會有任何錯誤訊息。
func (m *Map) At(x, y int) int {
	if x < 0 || x >= W || y < 0 || y >= H {
		return 0
	}
	return m.Cells[y*W+x]
}

// Clock 是四級計數器。docs/formats/02「時鐘的四級進位」。
//
// ⚠ **上限 10 / 26 / 34 / 21 不是地球曆法。**「一天幾小時」未解且不可從這些
// 數字推 —— 這裡只照抄計數器,不做任何單位換算
// (docs/spec/00-index.md 實作前必讀第 1 條)。
type Clock struct {
	Sub   int // 位移 33,範圍 1–10,時以下的計數(不顯示)
	Hour  int // 位移 31,範圍 4–26
	Day   int // 位移 29,範圍 1–34
	Month int // 位移 27,範圍 1–21
}

// clockLevel 描述一級計數器的上下界。
type clockLevel struct {
	lo, hi int
}

var clockLevels = [4]clockLevel{
	{1, 10},  // Sub
	{4, 26},  // Hour
	{1, 34},  // Day
	{1, 21},  // Month
}

// Tick 推進一格。低階溢位時進位到高階。
func (c *Clock) Tick() {
	fields := [4]*int{&c.Sub, &c.Hour, &c.Day, &c.Month}
	for i, f := range fields {
		*f++
		if *f <= clockLevels[i].hi {
			return
		}
		*f = clockLevels[i].lo
		// 繼續進位到下一級
	}
	// 最高階也溢位就回捲 —— 原版行為未解(沒有讀到年的計數器)。
	// 這裡回捲而不是停住,是為了不讓遊戲在第 21 個月當掉。
}

// State 是世界場景需要的隊伍狀態。欄位對應 GROUPS.DAT(docs/formats/02)。
type State struct {
	X, Y      int    // 位移 35 / 37,世界座標
	Facing    Facing // 位移 41
	Clock     Clock
	Encounter int // 位移 25,下次遭遇檢查前的剩餘回合
}

// Result 說明一次按鍵造成了什麼。呼叫端用它決定要不要推進時鐘。
type Result int

const (
	Turned  Result = iota // 只轉身,沒有位移
	Moved                 // 實際位移了
	Blocked               // 撞到地圖邊界
)

// Step 處理一次方向輸入。docs/spec/05 §6:
//
//	⚠ **朝向不同時只轉身,不位移。** 往北走的第一下若原本朝東,只會轉成朝北。
//	這一條會改變操作手感,也會改變每一步消耗的遊戲時間。
//
// ⚠ **沒有可通行性檢查。** docs/spec/05 §5:世界地圖的可通行性規則
// **沒有被 RE 過**,而「海洋當然不能走」是常識不是證據
// (本作有 WINGS / WIND WALK 這類移動法術,可能是有條件的)。
// 依 engine-plan §4「不要猜一個值填進去」,這裡只擋地圖邊界。
func (s *State) Step(dir Facing, m *Map) Result {
	if s.Facing != dir {
		s.Facing = dir
		return Turned
	}
	dx, dy := dir.delta()
	nx, ny := s.X+dx, s.Y+dy
	if nx < 0 || nx >= W || ny < 0 || ny >= H {
		return Blocked
	}
	s.X, s.Y = nx, ny

	// docs/spec/05 §7:每次**實際位移**推進時鐘一格,並遞減遭遇倒數。
	// 純轉身不推進。
	s.Clock.Tick()
	if s.Encounter > 0 {
		s.Encounter--
	}
	return Moved
}
