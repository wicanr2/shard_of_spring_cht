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
// **這不是地球曆法。** 手冊 p.32 描述了這個世界的曆法,但只有「一月 34 天」
// 與計數器完全一致(docs/re/140 §4):
//
//	月  1–21   ← 手冊寫 22,**手冊錯**(進位指令是 `> 21 → 1`)
//	日  1–34   ← 相符
//	時  4–26   ← 重設值是 4,所以一天有 23 個時值;手冊的「26 小時」是上界
//
// ⚠ 兩個數字相同不代表量的是同一件事 —— 先問它是上界、長度還是索引。
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
//
// ⚠ 光源與能見度四欄也放在這裡,不是因為它們屬於世界地圖,而是因為
// **原版把它們與時鐘寫在同一支常式裡**(`USERLIB` 0x1043A–0x10577,
// docs/re/204)。拆開兩邊各推進一次,遲早會有一邊漏掉。
type State struct {
	X, Y      int    // 位移 35 / 37,世界座標
	Facing    Facing // 位移 41
	Clock     Clock
	Encounter int // 位移 25,下次遭遇檢查前的剩餘回合

	// MazeNum 是位移 83:**當前迷宮編號**,99 = 不在迷宮(docs/re/204 §1)。
	// ⚠ 這一欄先前被讀成「光源選擇」——名字錯了,機制沒錯:
	// 「在地面用天色、在迷宮用火把」的判斷條件就是它。
	MazeNum         int
	LightTurns      int // 位移 45,攜帶光源的剩餘回合
	VisLit, VisDark int // 位移 59 / 61,有光 / 無光時的能見度
	// Visibility 是**生效值**,每次推進重算(原版的 `ds:3050`)。
	// ⚠ 不進存檔 —— 記錄裡沒有它,讀檔後第一次推進就會補上。
	Visibility int
}

// NotInMaze 是位移 83 的哨兵:不在任何迷宮(docs/re/169 §4)。
const NotInMaze = 99

// 遭遇倒數的載入補值(`MENU` 0x119A2–0x119B2,docs/re/204 §3):
// 載入隊伍時如果剩餘 ≤ 2 就補回 25。
//
// ⚠ 這**不是**「每次遭遇之後填什麼」—— 那一個是 RollEncounter(docs/re/214)。
// 兩者是不同的量:這一條只保證「讀檔後不會立刻踩到遭遇」。
// ⚠ 而**兩邊的 25 是同一個數字**:`MENU` 用立即數、`CMBT` 走 DGROUP 常數。
const (
	EncounterFloor  = 2
	EncounterReload = 25
)

// 每打完一場之後重填的遭遇倒數(`CMBT 0x13295`–`0x132BA`,docs/re/214):
//
//	INT(RND × 10) + 25      → 值域 25…34
//
// ⚠ 下界 25 與 EncounterReload 是**同一個數字出現在兩支模組**
// (`MENU` 用立即數、`CMBT` 走 DGROUP 常數 `ds:9736`)—— 互相印證。
const (
	EncounterRollFaces = 10
	EncounterRollBase  = 25
)

// RollEncounter 擲下一次遭遇前的回合數。
//
// ⚠ 成語是 `INT(RND × N) + C`(配了 `INT 3D:03` 截尾,docs/re/185)——
// **不是** `Roll(N)`,那是 1…N;這裡是 0…N−1 再加 C。
func RollEncounter(r FloatRand) int {
	return int(float64(EncounterRollFaces)*r.Float01()) + EncounterRollBase
}

// FloatRand 是 RollEncounter 需要的擲骰來源。形狀與 combat.FloatRand 相同 ——
// ⛔ 這裡不 import combat:world 是下層,反過來會成環。
type FloatRand interface {
	Roll(faces int) int
	Float01() float64
}

// Daylight 是自然光的能見度,由**時**決定(USERLIB 0x104A2–0x104FC)。
//
//	預設 4;時 ≤ 5 或 ≥ 14 → 3;時 ≤ 2 或 ≥ 17 → 2;時 ≥ 19 → 1
//
// 四條由上而下覆蓋,不是互斥的分支。時的值域是 4–26,所以「≤ 2」那一條
// 在遊戲裡到不了 —— **照原版寫著**,不要因為「用不到」就刪掉:
// 刪掉之後就沒有人記得它存在過([`CLAUDE.md`](../../../CLAUDE.md) 的柵欄原則)。
func Daylight(hour int) int {
	v := 4
	if hour <= 5 || hour >= 14 {
		v = 3
	}
	if hour <= 2 || hour >= 17 {
		v = 2
	}
	if hour >= 19 {
		v = 1
	}
	return v
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
//
// ⚠ **轉身也要推進時鐘**(docs/re/149:實跑量出來的)。
// 先前只在實際位移時推進 —— 那樣時鐘會走得比原版慢,
// 而**慢的時鐘沒有症狀**:天色、遭遇、食糧消耗全部一起變慢,
// 玩起來只覺得「這遊戲比較寬鬆」。
func (s *State) Step(dir Facing, m *Map) Result {
	if s.Facing != dir {
		s.Facing = dir
		s.tick()
		return Turned
	}
	dx, dy := dir.delta()
	nx, ny := s.X+dx, s.Y+dy
	if nx < 0 || nx >= W || !Passable(m, s.X, s.Y, nx, ny, dir) {
		return Blocked
	}
	cost := MoveCost(m, s.X, s.Y, nx, ny)
	s.X, s.Y = nx, ny
	for i := 0; i < cost; i++ {
		s.tick()
	}
	return Moved
}

// Mountain 回傳這個地形值是不是山地(圖塊 7/8/9,docs/spec/05 §2)。
func Mountain(v int) bool { return v >= 7 && v <= 9 }

// MoveCost 是一步花掉的時鐘格數。
//
//	起點或終點是山地 → 2,否則 1
//
// 手冊 p.32「在丘陵上行走時間會過得比原來快兩倍」的實際形狀
// (docs/re/151,四次量測)。⚠ **山→山也是 2,不是 3** ——
// 「起點 + 終點各加一」那個版本對前三次量測同樣成立,
// 是第四次(山→山)把它排除的。
//
// 轉身與存檔恆為 1,不受地形影響。
func MoveCost(m *Map, fx, fy, tx, ty int) int {
	if Mountain(m.At(fx, fy)) || Mountain(m.At(tx, ty)) {
		return 2
	}
	return 1
}

// Tick 推進時鐘一格並遞減遭遇倒數 —— 給場景層在「非移動的動作」上呼叫
// (docs/re/149:存檔也算一個動作)。
func (s *State) Tick() { s.tick() }

// tick 推進時鐘一格並遞減遭遇倒數。
//
// 原版的 `dec 位移25` 就接在時鐘進位那一段的最後(docs/re/107 §1),
// 所以**兩者一定同步** —— 不要在別處單獨動其中一個。
func (s *State) tick() {
	s.Clock.Tick()
	if s.Encounter > 0 {
		s.Encounter--
	}
	s.light()
}

// light 是每次推進都要跑的光源與能見度(USERLIB 0x10494–0x10577,docs/re/204 §2)。
//
//	在地面(位移 83 == 99):能見度 = 自然光(時),**光源回合歸零**
//	在迷宮且還有光:      能見度 = 位移 59,然後回合數 −1
//	在迷宮且沒有光:      能見度 = 位移 61,回合數夾在 0
//
// ⚠ **先取能見度再遞減** —— 回合數從 1 走到 0 的那一次仍算「有光」。
// ⚠ **走出迷宮火把就熄了**,不是留著下次用(`WRLDMOVE` 0x10F6B 把兩欄
// 一起寫:光源回合 0、迷宮編號 99)。這是原版的行為,不是簡化。
// RefreshLight 只重算生效能見度,**不遞減**回合數 —— 給「進出迷宮的那一刻」
// 用:場景換了但沒有花掉一個動作。走出迷宮同樣把火把熄掉(與 light 一致)。
func (s *State) RefreshLight() {
	n := s.LightTurns
	s.light()
	if s.MazeNum != NotInMaze {
		s.LightTurns = n
	}
}

func (s *State) light() {
	if s.MazeNum == NotInMaze {
		s.Visibility = Daylight(s.Clock.Hour)
		s.LightTurns = 0
		return
	}
	if s.LightTurns > 0 {
		s.Visibility = s.VisLit
		s.LightTurns--
		return
	}
	s.Visibility = s.VisDark
	s.LightTurns = 0
}

// 沒有光時的能見度。docs/re/229 §1:原版進迷宮時把 `GROUPS.DAT` 位移 61
// 設成 2,隊上有**戰士**帶夜視就設成 3。
//
// ⚠ 獨立印證:出貨的 PARTY #5 那一格就是 **2**,而那支隊伍沒有夜視。
const (
	VisDarkBase        = 2
	VisDarkNightVision = 3
)
