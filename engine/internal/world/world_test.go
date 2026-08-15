package world

import "testing"

func blankMap() *Map { return &Map{Cells: make([]int, W*H)} }

// docs/spec/05 §6 最重要的一條:朝向不同時**只轉身,不位移**。
// 這條錯了不會有任何錯誤訊息,只會讓每一步的遊戲時間消耗少一半。
func TestTurnBeforeMove(t *testing.T) {
	m := blankMap()
	s := &State{X: 50, Y: 50, Facing: East, Encounter: 10}

	if r := s.Step(North, m); r != Turned {
		t.Fatalf("朝東時按北,應只轉身,得 %v", r)
	}
	if s.X != 50 || s.Y != 50 {
		t.Errorf("轉身不該位移,座標變成 (%d,%d)", s.X, s.Y)
	}
	if s.Facing != North {
		t.Errorf("朝向應變成北,得 %v", s.Facing)
	}
	if s.Encounter != 10 {
		t.Errorf("純轉身不該遞減遭遇倒數,得 %d", s.Encounter)
	}
	if s.Clock != (Clock{}) {
		t.Errorf("純轉身不該推進時鐘,得 %+v", s.Clock)
	}

	if r := s.Step(North, m); r != Moved {
		t.Fatalf("已朝北時按北,應位移,得 %v", r)
	}
	if s.X != 50 || s.Y != 49 {
		t.Errorf("往北應 y−1,得 (%d,%d)", s.X, s.Y)
	}
	if s.Encounter != 9 {
		t.Errorf("位移應遞減遭遇倒數,得 %d", s.Encounter)
	}
}

// 四個方向的位移。1北 2東 3南 4西(docs/spec/05 §6)。
func TestDirections(t *testing.T) {
	m := blankMap()
	for _, c := range []struct {
		f      Facing
		dx, dy int
	}{{North, 0, -1}, {East, 1, 0}, {South, 0, 1}, {West, -1, 0}} {
		s := &State{X: 50, Y: 50, Facing: c.f}
		if r := s.Step(c.f, m); r != Moved {
			t.Fatalf("%v 應位移,得 %v", c.f, r)
		}
		if s.X != 50+c.dx || s.Y != 50+c.dy {
			t.Errorf("%v:得 (%d,%d),應為 (%d,%d)", c.f, s.X, s.Y, 50+c.dx, 50+c.dy)
		}
	}
}

func TestBlockedAtEdge(t *testing.T) {
	m := blankMap()
	s := &State{X: 0, Y: 0, Facing: West}
	if r := s.Step(West, m); r != Blocked {
		t.Errorf("在 x=0 往西應 Blocked,得 %v", r)
	}
	if s.X != 0 {
		t.Errorf("Blocked 不該改座標,得 x=%d", s.X)
	}
	s.Facing = North
	if r := s.Step(North, m); r != Blocked {
		t.Errorf("在 y=0 往北應 Blocked,得 %v", r)
	}
}

// 索引方向。y*W+x 而不是 x*H+y —— 寫反了地圖會轉 90 度,
// 而畫面上看起來仍像「地圖本來就長這樣」。
func TestMapIndexOrder(t *testing.T) {
	m := blankMap()
	m.Cells[7*H+3] = 42 // (x=7, y=3)
	if got := m.At(7, 3); got != 42 {
		t.Errorf("At(7,3) 得 %d,應為 42 —— 索引應為 x*H+y(docs/re/141)", got)
	}
	if got := m.At(3, 7); got == 42 {
		t.Error("At(3,7) 也回 42 —— x 與 y 寫反了")
	}
}

func TestOutOfBoundsIsBorder(t *testing.T) {
	m := blankMap()
	for _, c := range [][2]int{{-1, 0}, {0, -1}, {W, 0}, {0, H}} {
		if got := m.At(c[0], c[1]); got != 0 {
			t.Errorf("界外 (%d,%d) 應回 0(地圖邊界),得 %d", c[0], c[1], got)
		}
	}
}

// 四級進位。上限 10 / 26 / 34 / 21,**不是地球曆法**,
// 所以測試也不能寫成「24 小時進一天」。
func TestClockCarry(t *testing.T) {
	c := Clock{Sub: 10, Hour: 26, Day: 34, Month: 5}
	c.Tick()
	if c.Sub != 1 || c.Hour != 4 || c.Day != 1 || c.Month != 6 {
		t.Errorf("三級同時溢位後得 %+v,應為 {1 4 1 6}", c)
	}

	c = Clock{Sub: 9, Hour: 10, Day: 3, Month: 2}
	c.Tick()
	if c != (Clock{Sub: 10, Hour: 10, Day: 3, Month: 2}) {
		t.Errorf("最低階未溢位時只該動 Sub,得 %+v", c)
	}
	c.Tick()
	if c != (Clock{Sub: 1, Hour: 11, Day: 3, Month: 2}) {
		t.Errorf("Sub 溢位應回 1 並進位 Hour,得 %+v", c)
	}
}

// Hour 的下界是 4 不是 1 —— 這個怪數字正是「不要做單位換算」的理由。
func TestHourLowerBoundIsFour(t *testing.T) {
	c := Clock{Sub: 10, Hour: 26, Day: 1, Month: 1}
	c.Tick()
	if c.Hour != 4 {
		t.Errorf("Hour 溢位應回到下界 4(不是 0 也不是 1),得 %d", c.Hour)
	}
}

func TestEncounterNeverNegative(t *testing.T) {
	m := blankMap()
	s := &State{X: 50, Y: 50, Facing: North, Encounter: 1}
	s.Step(North, m)
	s.Step(North, m)
	if s.Encounter != 0 {
		t.Errorf("遭遇倒數不該變負,得 %d", s.Encounter)
	}
}

// --------------------------------------------------------------------------
// 可通行性。docs/re/131 §2 的八條規則,每條一個測試。
// --------------------------------------------------------------------------

func mapWith(cells map[[2]int]int) *Map {
	m := blankMap()
	for k, v := range cells {
		m.Cells[k[0]*H+k[1]] = v // 索引 x*H+y(docs/re/141)
	}
	return m
}

// 規則 1 限制的是**南北**座標(跨距 1 的那一條軸,docs/re/141)。
func TestPassableNorthSouthRange(t *testing.T) {
	m := blankMap()
	for _, c := range []struct {
		y  int
		ok bool
	}{{4, false}, {5, true}, {98, true}, {99, false}} {
		if got := Passable(m, 50, c.y+1, 50, c.y, North); got != c.ok {
			t.Errorf("南北=%d 可通行=%v,應為 %v(規則 1:5–98)", c.y, got, c.ok)
		}
	}
}

// 規則 2 的範圍是 10–12,**不是只有 11**。寫成「海洋不能走」會漏掉 10 和 12。
func TestPassableBlockedTiles(t *testing.T) {
	for v := 9; v <= 13; v++ {
		m := mapWith(map[[2]int]int{{50, 49}: v})
		want := v < 10 || v > 12
		if got := Passable(m, 50, 50, 50, 49, North); got != want {
			t.Errorf("目標地形 %d 可通行=%v,應為 %v(規則 2:10–12 阻擋)", v, got, want)
		}
	}
}

func TestPassableTile20Entry(t *testing.T) {
	for _, cur := range []int{1, 4, 5, 11} {
		m := mapWith(map[[2]int]int{{50, 50}: cur, {50, 49}: 20})
		want := cur >= 1 && cur <= 4
		if got := Passable(m, 50, 50, 50, 49, North); got != want {
			t.Errorf("從地形 %d 進 20:可通行=%v,應為 %v(規則 3)", cur, got, want)
		}
	}
}

func TestPassableFromTile20ToCoast(t *testing.T) {
	m := mapWith(map[[2]int]int{{50, 50}: 20, {50, 49}: 16})
	if Passable(m, 50, 50, 50, 49, North) {
		t.Error("從 20 走到海岸線 16 應被擋(規則 4)")
	}
}

// 規則 5–8:海岸線的方向性阻擋。逐一列舉 4 方向 × 4 起點 × 4 終點 = 64 種,
// 與 docs/re/131 §2 的表比對。
//
// ⚠ 第一版測試寫「反方向不該被擋」——**那個假設是錯的**。
// 規則是成對的(北擋 15/16→17/18、南擋 17/18→15/16),那是**雙向的牆**。
// 測試裡的假設也要有出處,不能想當然。
func TestPassableCoastDirectional(t *testing.T) {
	// docs/re/131 §2:每個方向的 (從, 到) 阻擋集合
	rules := map[Facing][2][]int{
		North: {{15, 16}, {17, 18}},
		East:  {{15, 18}, {16, 17}},
		South: {{17, 18}, {15, 16}},
		West:  {{16, 17}, {15, 18}},
	}
	coast := []int{15, 16, 17, 18}
	for _, dir := range []Facing{North, East, South, West} {
		r := rules[dir]
		for _, cur := range coast {
			for _, dst := range coast {
				want := !(oneOf(cur, r[0]...) && oneOf(dst, r[1]...))
				m := mapWith(map[[2]int]int{{50, 50}: cur, {50, 49}: dst})
				if got := Passable(m, 50, 50, 50, 49, dir); got != want {
					t.Errorf("朝向%d %d→%d:可通行=%v,規格說 %v", dir, cur, dst, got, want)
				}
			}
		}
	}
}

// 海岸的四個值各擋兩個相鄰方向 —— 那是四個轉角。
// 這條把 §2 的推論(而不只是規則本身)也鎖住。
func TestCoastTileBlocksTwoDirections(t *testing.T) {
	// docs/re/131 §2:15=東北角、16=西北角、17=西南角、18=東南角
	want := map[int][]Facing{
		15: {North, East}, 16: {North, West},
		17: {South, West}, 18: {East, South},
	}
	for tile, dirs := range want {
		var blocked []Facing
		for _, dir := range []Facing{North, East, South, West} {
			// 拿另外三個海岸值當目標,只要有一個被擋就算這個方向被擋
			hit := false
			for _, dst := range []int{15, 16, 17, 18} {
				if dst == tile {
					continue
				}
				m := mapWith(map[[2]int]int{{50, 50}: tile, {50, 49}: dst})
				if !Passable(m, 50, 50, 50, 49, dir) {
					hit = true
				}
			}
			if hit {
				blocked = append(blocked, dir)
			}
		}
		if len(blocked) != len(dirs) {
			t.Errorf("地形 %d 擋住 %v,應為 %v(各擋兩個相鄰方向)", tile, blocked, dirs)
			continue
		}
		for i := range dirs {
			if blocked[i] != dirs[i] {
				t.Errorf("地形 %d 擋住 %v,應為 %v", tile, blocked, dirs)
				break
			}
		}
	}
}
