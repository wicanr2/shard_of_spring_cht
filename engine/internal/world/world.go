// Package world 是世界地圖場景的狀態與規則。
//
// 全部規則出自 docs/spec/05-world-scene.md,每一條在下面註明章節。
// **規格改了才改這裡。**
package world

// 地圖尺寸。docs/spec/05 §1、docs/formats/05 §2。
//
// ⚠ **東西 121、南北 103**,而索引的跨距是 H(= 103):往東一格 = +103。
// 兩軸的名字被接反過一次(docs/re/141)—— 轉置過的地圖仍然畫得出來,
// 沒有任何錯誤訊息,是實跑走進城鎮才裁決出來的。
const (
	W = 121 // 東西 0–120
	H = 103 // 南北 0–102
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
// ⚠ 索引是 **x*H + y**(往東 +103),與迷宮的「欄 × 81 + 列」同一種形狀。
// 這一行寫成 y*W+x 過,而**那樣地圖照樣畫得出來,只是整張轉了 90 度**
// —— 上面那句警語當初就寫在這裡,擋不住,因為它警告的東西沒有症狀。
// 裁決它的是實跑(docs/re/141):城鎮在東邊還是南邊。
func (m *Map) At(x, y int) int {
	if x < 0 || x >= W || y < 0 || y >= H {
		return 0
	}
	return m.Cells[x*H+y]
}

// Clock 是四級計數器。docs/formats/02「時鐘的四級進位」。
//
// **這不是地球曆法。** 手冊 p.32 給了這個世界的曆法,與上限對得上
// (docs/re/140 §4):一天 26 小時 ✅、一月 34 天 ✅、一年 22 個月 ⚠。
//
// ⚠ 月數照 **21**,不照手冊的 22 —— 21 來自反組譯(證據第 2 級),
// 手冊是第 3 級,差 1 不足以推翻。改這個值之前要先回去讀進位那幾行指令。
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
	{1, 10}, // Sub
	{4, 26}, // Hour
	{1, 34}, // Day
	{1, 21}, // Month
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

// 可通行性的邊界常數。docs/re/131 §2 規則 1 把座標限制在 5–98,
// 而被限制的是**跨距 1 的那一條軸** —— 也就是**南北**(0–102,docs/re/141)。
// 南北兩端因此各有 5 排玩家永遠到不了。
// ⚠ 東西向沒有對應的檢查,原因未解。
const (
	MinY = 5
	MaxY = 98
)

func in(v, lo, hi int) bool { return v >= lo && v <= hi }

func oneOf(v int, set ...int) bool {
	for _, x := range set {
		if v == x {
			return true
		}
	}
	return false
}

// Passable 回傳「從 (fromX,fromY) 朝 dir 走到 (toX,toY)」是否允許。
//
// docs/re/131 §2 的八條規則,任一成立即不可通行。順序與原版一致。
func Passable(m *Map, fromX, fromY, toX, toY int, dir Facing) bool {
	// 1. 南北範圍
	if !in(toY, MinY, MaxY) {
		return false
	}
	cur, dst := m.At(fromX, fromY), m.At(toX, toY)

	// 2. 目標 10–12(11 = 海洋;10 與 12 語意未知但同樣阻擋)
	if in(dst, 10, 12) {
		return false
	}
	// 3. 目標 20/21 只能從地形 1–4 進入
	if oneOf(dst, 20, 21) && !in(cur, 1, 4) {
		return false
	}
	// 4. 從 20/21 不能走到海岸線 15–18
	if oneOf(cur, 20, 21) && in(dst, 15, 18) {
		return false
	}
	// 5–8. 海岸線的方向性阻擋。15/16/17/18 是海岸的四個轉角,
	// 各擋兩個相鄰方向(docs/re/131 §2)。
	switch dir {
	case North:
		if oneOf(cur, 15, 16) && oneOf(dst, 17, 18) {
			return false
		}
	case East:
		if oneOf(cur, 15, 18) && oneOf(dst, 16, 17) {
			return false
		}
	case South:
		if oneOf(cur, 17, 18) && oneOf(dst, 15, 16) {
			return false
		}
	case West:
		if oneOf(cur, 16, 17) && oneOf(dst, 15, 18) {
			return false
		}
	}
	return true
}

// Step 處理一次方向輸入。docs/spec/05 §6:
//
//	⚠ **朝向不同時只轉身,不位移。** 往北走的第一下若原本朝東,只會轉成朝北。
//	這一條會改變操作手感,也會改變每一步消耗的遊戲時間。
func (s *State) Step(dir Facing, m *Map) Result {
	if s.Facing != dir {
		s.Facing = dir
		return Turned
	}
	dx, dy := dir.delta()
	nx, ny := s.X+dx, s.Y+dy
	if nx < 0 || nx >= W || !Passable(m, s.X, s.Y, nx, ny, dir) {
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
