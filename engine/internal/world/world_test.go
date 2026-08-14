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
	m.Cells[3*W+7] = 42 // (x=7, y=3)
	if got := m.At(7, 3); got != 42 {
		t.Errorf("At(7,3) 得 %d,應為 42 —— 索引應為 y*W+x", got)
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
