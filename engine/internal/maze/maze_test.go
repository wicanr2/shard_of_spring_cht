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

// 五個機關的觸發點是**目標編號**,不是座標(docs/re/161 §3)。
// 它們同時也有 DT 文字,所以 Text 必須照樣帶出來。
func TestScanClassifiesSpecialTargets(t *testing.T) {
	for _, c := range []struct {
		target int
		kind   TriggerKind
	}{
		{TargetPool, KindPool},
		{TargetGem, KindGem},
		{TargetRiddle, KindRiddle},
		{TargetPriest, KindScript},
		{TargetFinalBoss, KindScript},
		{519, KindText}, // 相鄰的普通編號不該被誤判
	} {
		text := map[int]string{c.target: "敘述"}
		evs := []original.Event{{Major: 1, Minor: 2, Dir: 0, Target: c.target}}
		tr := Scan(evs, State{Major: 1, Minor: 2, Facing: North}, text)
		if tr.Kind != c.kind {
			t.Errorf("目標 %d:得 %v,應為 %v", c.target, tr.Kind, c.kind)
		}
		if tr.Text != "敘述" {
			t.Errorf("目標 %d:文字被吃掉了", c.target)
		}
	}
}

// 氏族謎題:順序不拘,而且原版的漏洞要照抄(docs/re/162 §3)。
func TestClanSolved(t *testing.T) {
	for _, c := range []struct {
		name    string
		answers [4]string
		want    bool
	}{
		{"照順序", [4]string{"MURTHIN", "CERCION", "LOTHIAN", "VANDIGUARD"}, true},
		{"打亂", [4]string{"VANDIGUARD", "LOTHIAN", "MURTHIN", "CERCION"}, true},
		{"少一個", [4]string{"MURTHIN", "CERCION", "LOTHIAN", "ELDRON"}, false},
		{"全空", [4]string{}, false},
		// ⚠ 這一條**不是**期望的遊戲行為,是原版判定的漏洞:
		// 它只數對得上幾組,沒有記哪個名字用過。
		{"同一個名字四次也會過", [4]string{"MURTHIN", "MURTHIN", "MURTHIN", "MURTHIN"}, true},
	} {
		if got := ClanSolved(c.answers); got != c.want {
			t.Errorf("%s:得 %v,應為 %v", c.name, got, c.want)
		}
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

// 走出邊界 = 離開迷宮(docs/re/147),不是撞牆。
//
// ⚠ 這條的順序很重要:界外的格值讀出來是 0(可通行),
// 而**如果先判可通行再判邊界**,玩家會被關在迷宮裡出不去 ——
// 而畫面上只會看到「走不動」,像是牆。
func TestSteppingOutOfBoundsLeavesTheMaze(t *testing.T) {
	m := &original.Maze{Majors: 3, Cells: make([]int, 3*original.MazeRows)}
	tried := 0
	for _, dir := range []Facing{North, East, South, West} {
		dM, dm := dir.delta()
		if dM >= 0 && dm >= 0 {
			continue // 這個方向從 (0,0) 走不出邊界
		}
		tried++
		s := &State{Major: 0, Minor: 0, Facing: dir}
		if got := s.Step(dir, m); got != Left {
			t.Errorf("朝向 %v 從 (0,0) 往外走應為 Left,得 %v", dir, got)
		}
	}
	if tried == 0 {
		t.Fatal("沒有測到任何方向 —— delta() 的正負號變了,這條測試失效了")
	}
}

// 寶石謎題:答案是累積字串 BBRG(docs/re/155 §1)。
func TestGemPuzzle(t *testing.T) {
	if !GemSolved(GemAnswer) {
		t.Error("正確答案應該算對")
	}
	// ⚠ 前綴不算對 —— 原版收滿四個字元才比一次
	for _, s := range []string{"", "B", "BB", "BBR", "BBRGG", "bbrg", "BGRB"} {
		if GemSolved(s) {
			t.Errorf("%q 不該算對", s)
		}
	}
}

// 治療池:11 次上限、狀態 > 2 治不了、回血夾在滿血(docs/re/155 §2)。
func TestHealingPool(t *testing.T) {
	if !PoolAvailable(10) || PoolAvailable(11) || PoolAvailable(12) {
		t.Error("上限是「已使用次數 < 11」")
	}
	for st := 0; st <= 5; st++ {
		want := st <= 2 // 正常 / 中毒 / 束縛 可以;凝滯 / 冰封 / 死亡 不行
		if got := PoolCanHeal(st); got != want {
			t.Errorf("狀態 %d → %v,應為 %v", st, got, want)
		}
	}
	// 夾住:差 3 血時擲出 10 只回 3
	if got := PoolHeal(7, 10, 10); got != 3 {
		t.Errorf("7/10 擲 10 應回 3,得 %d", got)
	}
	if got := PoolHeal(7, 10, 2); got != 2 {
		t.Errorf("7/10 擲 2 應回 2,得 %d", got)
	}
	// 滿血的人回 0,不會變成負數
	if got := PoolHeal(10, 10, 5); got != 0 {
		t.Errorf("滿血應回 0,得 %d", got)
	}
}

// 事件作廢:只在打完仗之後,而且同一個目標的所有列一起消失(docs/re/165 §3)。
func TestDisableTargetBlanksEveryRow(t *testing.T) {
	evs := []original.Event{
		{Major: 57, Minor: 14, Dir: 1, Target: 204},
		{Major: 57, Minor: 14, Dir: 2, Target: 204}, // 同一格的另一個入口方向
		{Major: 3, Minor: 4, Dir: 0, Target: 101},   // 房間敘述,不該被動到
	}
	if n := DisableTarget(evs, 204); n != 2 {
		t.Errorf("作廢了 %d 列,應為 2", n)
	}
	for i := 0; i < 2; i++ {
		e := evs[i]
		if e.Major != DisabledCoord || e.Minor != DisabledCoord || e.Dir != DisabledCoord {
			t.Errorf("第 %d 列沒作廢:%+v", i, e)
		}
	}
	if evs[2].Major != 3 {
		t.Error("目標不同的列不該被動到")
	}
	// 作廢之後掃不到
	text := map[int]string{204: "巨人"}
	if got := Scan(evs, State{Major: 57, Minor: 14, Facing: North}, text).Kind; got != KindNone {
		t.Errorf("作廢後仍掃得到:%v", got)
	}
}
