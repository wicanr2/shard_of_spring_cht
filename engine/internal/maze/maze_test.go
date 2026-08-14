package maze

import (
	"testing"

	"shardofspring/internal/original"
)

func grid(cells map[[2]int]int) *original.Maze {
	m := &original.Maze{Majors: 20, Cells: make([]int, 20*original.MazeRows)}
	for k, v := range cells {
		m.Cells[k[0]*original.MazeRows+k[1]] = v
	}
	return m
}

// 朝向不同時只轉身 —— 與世界地圖同一條規則。
func TestTurnBeforeMove(t *testing.T) {
	m := grid(nil)
	s := &State{Major: 5, Minor: 5, Facing: East}
	if r := s.Step(North, m); r != Turned {
		t.Fatalf("朝東時按北應只轉身,得 %v", r)
	}
	if s.Major != 5 || s.Minor != 5 {
		t.Errorf("轉身不該位移,得 (%d,%d)", s.Major, s.Minor)
	}
	if r := s.Step(North, m); r != Moved {
		t.Fatalf("已朝北時按北應位移,得 %v", r)
	}
}

// 阻擋是 5–10 的區間。11 以上可以走 —— 這條在寫成「≥ 5 阻擋」時會失敗。
func TestBlockingIsAnInterval(t *testing.T) {
	for _, c := range []struct {
		tile int
		want Result
	}{{4, Moved}, {5, Blocked}, {10, Blocked}, {11, Moved}, {19, Moved}} {
		m := grid(map[[2]int]int{{5, 4}: c.tile})
		s := &State{Major: 5, Minor: 5, Facing: North}
		if r := s.Step(North, m); r != c.want {
			t.Errorf("格值 %d:得 %v,應為 %v", c.tile, r, c.want)
		}
	}
}

// 事件:文字、傳送、跨關卡三種。
func TestScanClassifiesEvents(t *testing.T) {
	text := map[int]string{101: "房間裡有一道門。"}
	evs := []original.Event{
		{Major: 3, Minor: 4, Dir: 0, Target: 101},
		{Major: 6, Minor: 7, Dir: 1, Target: 20, DestMinor: 30},
		{Major: 8, Minor: 9, Dir: 0, Target: 501},
	}
	for _, c := range []struct {
		s    State
		kind TriggerKind
	}{
		{State{Major: 3, Minor: 4, Facing: South}, KindText},
		{State{Major: 6, Minor: 7, Facing: North}, KindTeleport},
		{State{Major: 8, Minor: 9, Facing: West}, KindCrossLevel},
		{State{Major: 1, Minor: 1, Facing: North}, KindNone},
	} {
		if got := Scan(evs, c.s, text).Kind; got != c.kind {
			t.Errorf("(%d,%d) 朝向%d:得 %v,應為 %v",
				c.s.Major, c.s.Minor, c.s.Facing, got, c.kind)
		}
	}
	// 方向不合的事件不觸發
	if Scan(evs, State{Major: 6, Minor: 7, Facing: South}, text).Kind != KindNone {
		t.Error("方向 1 的事件在朝南時不該觸發")
	}
	// 傳送的目的地:Target 是 Major、欄4 是 Minor
	tr := Scan(evs, State{Major: 6, Minor: 7, Facing: North}, text)
	if tr.Major != 20 || tr.Minor != 30 {
		t.Errorf("傳送目的地 (%d,%d),應為 (20,30)", tr.Major, tr.Minor)
	}
}

// 視野半徑用切比雪夫距離(方形視野,不是圓形)。
func TestVisibilityIsChebyshev(t *testing.T) {
	s := State{Major: 10, Minor: 10, Visibility: 3}
	for _, c := range []struct {
		dM, dm int
		want   bool
	}{{0, 0, true}, {3, 3, true}, {3, 0, true}, {4, 0, false}, {0, 4, false}, {-3, 3, true}} {
		if got := Visible(s, 10+c.dM, 10+c.dm); got != c.want {
			t.Errorf("偏移 (%d,%d):可見=%v,應為 %v", c.dM, c.dm, got, c.want)
		}
	}
	// 半徑 0 或未設 → 全可見(不要讓沒設定的存檔變成全黑)
	if !Visible(State{Visibility: 0}, 50, 50) {
		t.Error("能見度 0 應視為未設定,全部可見")
	}
}
