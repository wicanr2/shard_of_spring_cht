package original

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// readOpt 讀一個原版檔,回錯誤而不是中止 —— 用在「檔案存在與否本身是結論」
// 的測試裡(例如 DG4MAZE.SQZ 不存在,那是 docs/re/51 §3 的已知事實)。
func readOpt(name string) ([]byte, error) {
	for _, dir := range []string{"/game/sharspri", "../../../game/sharspri"} {
		if d, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return d, nil
		}
	}
	return nil, fmt.Errorf("讀不到 %s", name)
}

var mazeFiles = []string{
	"DG1MAZE.SQZ", "DG2MAZE.SQZ", "DG3MAZE.SQZ",
	"DG5MAZE.SQZ", "DG51MAZE.SQZ", "DG6MAZE.SQZ",
}

// docs/spec/08 §8 驗收 1:六個 .SQZ 全部解得開。
//
// ⚠ **列可以短於 81 格**,那是合法的(解碼器沒有欄數檢查,docs/formats/06 §1)。
// 這條測試刻意**不**斷言「每列都滿」——上一輪就是把那當成殘差追了很久。
func TestDecodeAllMazes(t *testing.T) {
	for _, n := range mazeFiles {
		m, err := DecodeSQZ(read(t, n))
		if err != nil {
			t.Errorf("%s:%v", n, err)
			continue
		}
		if m.Majors < 10 {
			t.Errorf("%s 只有 %d 欄,太少", n, m.Majors)
		}
		if len(m.Cells) != m.Majors*MazeRows {
			t.Errorf("%s:%d 格,應為 %d × %d", n, len(m.Cells), m.Majors, MazeRows)
		}
	}
}

// 驗收 2:索引是 `欄 × 81 + 列`。
//
// ⚠ 這條在寫成 `列 × ? + 欄` 時必須失敗 —— 世界地圖是另一個順序
// (docs/spec/08 §1),而兩者寫反都畫得出圖,只是轉了 90 度。
func TestMazeIndexIsColumnMajor(t *testing.T) {
	m := &Maze{Majors: 5, Cells: make([]int, 5*MazeRows)}
	m.Cells[3*MazeRows+7] = 42 // major 3、minor 7
	if got := m.At(3, 7); got != 42 {
		t.Errorf("At(3, 7) = %d,應為 42 —— 索引應為 major×81+minor", got)
	}
	if got := m.At(7, 3); got == 42 {
		t.Error("At(7,3) 也回 42 —— major 與 minor 寫反了")
	}
}

// 驗收 3 + 4:阻擋是**區間** 5–10;19 可通行且不繪製。
func TestMazeTileBehaviour(t *testing.T) {
	for v := 0; v <= 28; v++ {
		wantBlock := v >= 5 && v <= 10
		if MazePassable(v) == wantBlock {
			t.Errorf("格值 %d:可通行=%v,阻擋區間是 5–10", v, MazePassable(v))
		}
	}
	// 11 以上可以走 —— 「≥ 5 阻擋」的寫法會在這裡失敗
	if !MazePassable(11) {
		t.Error("格值 11 應可通行 —— 阻擋是區間不是下界")
	}
	for _, v := range []int{0, 18, 19} {
		if _, drawn := MazeDrawn(v); drawn {
			t.Errorf("格值 %d 應不繪製", v)
		}
	}
	if !MazePassable(TileTrigger) {
		t.Error("格值 19 是隱形觸發格,應可通行")
	}
	// 負值:取絕對值當繪製編號
	if id, drawn := MazeDrawn(-13); !drawn || id != 13 {
		t.Errorf("格值 −13 → (%d, %v),應為 (13, true)", id, drawn)
	}
}

// 驗收 5:MAZEDATA 的起始位置落在可通行格。
//
// 12/12(第 13 筆是 (0,0) 的佔位)。
// 這條鎖住的是**順序**:欄 2 是乘 81 的那一個,反過來只有 10/12(docs/re/137)。
func TestMazeDataStartsArePassable(t *testing.T) {
	entries, err := ParseMazeData(read(t, "MAZEDATA.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 13 {
		t.Fatalf("MAZEDATA 有 %d 筆,應為 13", len(entries))
	}
	mazes := map[int]*Maze{}
	ok, bad := 0, 0
	for _, e := range entries {
		if e.WorldX == 0 && e.WorldY == 0 {
			continue // 第 13 筆是佔位
		}
		m, seen := mazes[e.MazeFile]
		if !seen {
			d, err := readOpt(fmt.Sprintf("DG%dMAZE.SQZ", e.MazeFile))
			if err != nil {
				t.Errorf("關卡指向 DG%dMAZE.SQZ,讀不到:%v", e.MazeFile, err)
				continue
			}
			if m, err = DecodeSQZ(d); err != nil {
				t.Errorf("DG%d:%v", e.MazeFile, err)
				continue
			}
			mazes[e.MazeFile] = m
		}
		if MazePassable(m.At(e.StartMajor, e.StartMinor)) {
			ok++
		} else {
			bad++
			t.Errorf("DG%d 的起點(major %d, minor %d)格值 %d 不可通行 —— "+
				"欄 2/3 的順序可能反了",
				e.MazeFile, e.StartMajor, e.StartMinor,
				m.At(e.StartMajor, e.StartMinor))
		}
	}
	if ok != 12 {
		t.Errorf("可通行的起點 %d 個,應為 12(docs/re/137)", ok)
	}
}

// 驗收 6:事件表 106 × 5,方向欄的值域 ⊆ {0,1,2,3,4}。
func TestParseEvents(t *testing.T) {
	for _, n := range []string{"DE1EFF.BIN", "DE5EFF.BIN", "DE51EFF.BIN"} {
		d, err := readOpt(n)
		if err != nil {
			t.Errorf("%s:%v", n, err)
			continue
		}
		evs, err := ParseEvents(d)
		if err != nil {
			t.Errorf("%s:%v", n, err)
			continue
		}
		if len(evs) != EventRows {
			t.Errorf("%s 解出 %d 筆,應為 %d", n, len(evs), EventRows)
		}
		for i, e := range evs {
			if e.Dir < 0 || e.Dir > 4 {
				t.Errorf("%s 第 %d 筆方向 %d,值域應為 0–4(docs/re/59 §1c)",
					n, i, e.Dir)
			}
		}
	}
}

// DT 文字:編號 → 敘述。
func TestParseDungeonText(t *testing.T) {
	d, err := readOpt("DT5TEXT.DAT")
	if err != nil {
		t.Skip(err)
	}
	m := ParseDungeonText(d)
	if len(m) < 5 {
		t.Errorf("DT5TEXT.DAT 只解出 %d 條", len(m))
	}
	for n, s := range m {
		if n < 0 || s == "" {
			t.Errorf("編號 %d 的敘述是空的", n)
		}
	}
}

// 驗收 7:負格值的位置,事件目標都是「查不到文字」的編號(跨關卡)。
//
// docs/re/60 §3:全六個迷宮只有 4 個負值格,一對一對上 4 個沒有文字的事件。
// 這條同時驗證了「目標 ≥ 100 但查不到文字 = 跨關卡」這個判別法
// (docs/spec/08 §5)—— 若哪天多出一個負值格卻有文字,判別法就不成立了。
func TestNegativeTilesAreCrossLevel(t *testing.T) {
	entries, err := ParseMazeData(read(t, "MAZEDATA.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	neg, crossLevel := 0, 0
	seen := map[int]bool{}
	for _, e := range entries {
		if seen[e.MazeFile] {
			continue
		}
		seen[e.MazeFile] = true
		md, err := readOpt(fmt.Sprintf("DG%dMAZE.SQZ", e.MazeFile))
		if err != nil {
			continue
		}
		m, err := DecodeSQZ(md)
		if err != nil {
			t.Errorf("DG%d:%v", e.MazeFile, err)
			continue
		}
		ed, err1 := readOpt(fmt.Sprintf("DE%dEFF.BIN", e.TextFile))
		td, err2 := readOpt(fmt.Sprintf("DT%dTEXT.DAT", e.TextFile))
		if err1 != nil || err2 != nil {
			continue
		}
		evs, err := ParseEvents(ed)
		if err != nil {
			t.Errorf("DE%d:%v", e.TextFile, err)
			continue
		}
		text := ParseDungeonText(td)

		for a := 0; a < m.Majors; a++ {
			for b := 0; b < MazeRows; b++ {
				if m.At(a, b) >= 0 {
					continue
				}
				neg++
				for _, ev := range evs {
					if ev.Major != a || ev.Minor != b {
						continue
					}
					if ev.Target < TextTargetMin {
						t.Errorf("DG%d 的負值格 (%d,%d) 事件目標 %d < %d,不是轉移",
							e.MazeFile, a, b, ev.Target, TextTargetMin)
						continue
					}
					if _, ok := text[ev.Target]; ok {
						t.Errorf("DG%d 的負值格 (%d,%d) 目標 %d **有**文字 —— "+
							"「查不到文字 = 跨關卡」這個判別法要重看(docs/spec/08 §5)",
							e.MazeFile, a, b, ev.Target)
						continue
					}
					crossLevel++
				}
			}
		}
	}
	if neg != 4 {
		t.Errorf("負值格 %d 個,docs/re/60 §3 記 4", neg)
	}
	if crossLevel != 4 {
		t.Errorf("判成跨關卡的 %d 個,應為 4", crossLevel)
	}
}

// 地城入口的座標必須落在入口圖塊(24/25/27/28)上。
//
// ⚠ 這條是**正對照**:兩欄直接當 (x,y) 讀是 **0/11 命中**、對調是 **11/11**。
// 而寫反了的症狀是**地城完全進不去** —— 沒有錯誤訊息,
// 走到入口格什麼都不會發生,看起來就像「那一格本來就沒東西」。
func TestMazeEntrancesLandOnEntranceTiles(t *testing.T) {
	ents, err := ParseMazeData(read(t, "MAZEDATA.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	cells, err := DecodeWorldMap(read(t, "WRLDMAP.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	at := func(x, y int) int {
		if x < 0 || x >= WorldW || y < 0 || y >= WorldH {
			return -1
		}
		return int(cells[x*WorldH+y])
	}
	isEntrance := map[int]bool{24: true, 25: true, 27: true, 28: true}
	hit, total := 0, 0
	for _, e := range ents {
		if e.WorldX == 0 && e.WorldY == 0 {
			continue // 第 0 列是全零的哨兵
		}
		total++
		if isEntrance[at(e.WorldX, e.WorldY)] {
			hit++
		}
	}
	// ⚠ 12 筆非零裡有 1 筆落在別的圖塊上 —— docs/re/51 §2 早就標出那一筆特殊。
	// 這裡要求的是「絕大多數命中」,而**對調之後會掉到 0**。
	if hit < total-1 {
		t.Errorf("%d/%d 個入口落在入口圖塊上 —— 兩欄可能又接反了", hit, total)
	}
}
