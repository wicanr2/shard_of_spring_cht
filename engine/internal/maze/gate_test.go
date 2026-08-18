package maze

import (
	"os"
	"path/filepath"
	"testing"

	"shardofspring/internal/original"
)

// 拉利斯之門(docs/re/205)的驗收。
//
// ⚠ 這一組測試的價值不在「OpenGate 有沒有把值改掉」(那是一行賦值),
// 而在**它改的是不是那一格**:原版寫的是一個編譯期折算好的位址,
// 換算成引擎座標中間隔了兩層(陣列基底、以及兩邊 Major/Minor 定義相反)。
// 任何一層改了,下面的斷言就會失敗。

func dg5(t *testing.T) *original.Maze {
	t.Helper()
	var dir string
	for _, d := range []string{"/game/sharspri", "../../../../game/sharspri"} {
		if _, err := os.Stat(d); err == nil {
			dir = d
			break
		}
	}
	if dir == "" {
		t.Skip("找不到原版 game/sharspri —— 這一項沒有被驗證")
	}
	b, err := os.ReadFile(filepath.Join(dir, "DG5MAZE.SQZ"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := original.DecodeSQZ(b)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// 驗收 1:門那一格在原版資料裡就是一個**擋路**的格子。
//
// 這是「換算對不對」的獨立佐證 —— 如果 GateIndex 換算錯了,
// 落點幾乎一定是 0(空地)或 7(牆),而不會剛好是 9。
func TestGateCellIsTheClosedGate(t *testing.T) {
	m := dg5(t)
	major, minor := GateCell()
	if got := m.At(major, minor); got != GateClosedTile {
		t.Fatalf("門那一格(major %d, minor %d)是 %d,應為 %d —— "+
			"換算或迷宮座標約定改過了,回去看 docs/re/205 §2",
			major, minor, got, GateClosedTile)
	}
	if original.MazePassable(GateClosedTile) {
		t.Error("圖塊 9 應該擋路,否則這道門一開始就是開的")
	}
	if !original.MazePassable(GateTile) {
		t.Errorf("圖塊 %d 應該可通行,否則開門沒有意義", GateTile)
	}
}

// 驗收 2:門是**一條走廊上唯一的阻擋**。
//
// 前後各兩格可通行、門本身擋路 —— 這個形狀才讓「開門」有意義。
// ⚠ 走廊的方向是 minor 軸(原版的「列」),與 GateCell 的換算一致。
func TestGateSitsInACorridor(t *testing.T) {
	m := dg5(t)
	major, minor := GateCell()
	for _, d := range []int{-2, -1, 1, 2} {
		if v := m.At(major+d, minor); !original.MazePassable(v) {
			t.Errorf("門的第 %+d 格是 %d(擋路)—— 那就不是一條被門封住的走廊", d, v)
		}
	}
}

// 驗收 3:旗標沒開、或不是第 5 座地城,就不准動任何一格。
func TestOpenGateOnlyWhenFlagAndMaze(t *testing.T) {
	major, minor := GateCell()
	for _, tc := range []struct {
		name       string
		mazeFile   int
		flag       int
		wantOpened bool
	}{
		{"旗標沒開", GateMaze, 0, false},
		{"別座地城", 2, 1, false},
		{"兩個都成立", GateMaze, 1, true},
	} {
		g := dg5(t)
		if got := OpenGate(g, tc.mazeFile, tc.flag); got != tc.wantOpened {
			t.Errorf("%s:OpenGate 回 %v,應為 %v", tc.name, got, tc.wantOpened)
		}
		want := GateClosedTile
		if tc.wantOpened {
			want = GateTile
		}
		if got := g.At(major, minor); got != want {
			t.Errorf("%s:那一格是 %d,應為 %d", tc.name, got, want)
		}
	}
}

// 驗收 4:開門之後那一格可以走 —— 用 Step 走一次,不是只看格值。
func TestPartyCanWalkThroughOpenedGate(t *testing.T) {
	m := dg5(t)
	major, minor := GateCell()

	// 走廊沿 major 軸(TestGateSitsInACorridor 驗的就是這一軸),
	// 而 major 是**南北**軸(docs/re/224)—— 從門的南邊那一格朝北走一步就撞到門。
	blocked := State{Major: major + 1, Minor: minor, Facing: North}
	if r := blocked.Step(North, m); r == Moved {
		t.Fatal("門還關著就走得過去 —— 那這道門從頭到尾沒有作用")
	}

	OpenGate(m, GateMaze, 1)
	open := State{Major: major + 1, Minor: minor, Facing: North}
	if r := open.Step(North, m); r != Moved {
		t.Errorf("門開了卻走不過去(Step 回 %v)", r)
	}
}
