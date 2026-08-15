package original

import (
	"fmt"
	"strconv"
	"strings"
)

// 迷宮。docs/formats/06-maze.md + docs/spec/08-maze-scene.md。

// .SQZ 的解碼常數,直接取自 MAZEMOVE 的解碼迴圈(docs/re/55)。
const (
	sqzBase   = 42 // ds:964Eh
	sqzThresh = 53 // ds:964Ch
	MazeRows  = 81 // mov di, 51h
)

// Maze 是一層迷宮。
//
// ⚠ **索引是 `Major × 81 + Minor`**,與世界地圖的 `y × W + x` 相反。
// 寫反了迷宮仍然畫得出來,只是轉了 90 度 —— 沒有任何錯誤訊息。
//
// ⚠ **刻意不叫「欄」與「列」。** docs/re/56 與 59 把乘 81 的那個索引叫「列」,
// docs/re/114 叫它「欄」—— 同一個東西兩個名字,而**算式才是真相**
// (docs/re/137)。程式碼用 Major/Minor,呼叫端就不必記哪個是哪個。
type Maze struct {
	Majors int // 有幾個 major(= 解碼出來的段數)
	Cells  []int
}

// InBounds 回傳座標在不在這一層裡面。
//
// ⚠ **界外不是「不可通行」而是「迷宮外面」**(docs/re/147):
// 從入口那一格往外走一步,原版就印 `Leaving maze ..` 切回世界地圖。
// `At` 對界外回 0(可通行),所以光看 `At` 分不出「空地」與「外面」。
func (m *Maze) InBounds(major, minor int) bool {
	return major >= 0 && major < m.Majors && minor >= 0 && minor < MazeRows
}

// At 回傳 (major, minor) 的格值。界外回 0(= 可通行但不繪製)。
func (m *Maze) At(major, minor int) int {
	if major < 0 || major >= m.Majors || minor < 0 || minor >= MazeRows {
		return 0
	}
	return m.Cells[major*MazeRows+minor]
}

// DecodeSQZ 解碼一個 DG*MAZE.SQZ。
//
// ⚠ **解碼器沒有欄數檢查** —— 有些列不足 81 格是合法的,不是殘差
// (docs/formats/06 §1)。所以這裡也不補齊、不報錯。
func DecodeSQZ(d []byte) (*Maze, error) {
	raw := d
	for len(raw) > 0 && raw[len(raw)-1] == 0x1A {
		raw = raw[:len(raw)-1]
	}
	rows := [][]int{{}}
	for i := 0; i < len(raw); {
		v := int(raw[i]) - sqzBase
		switch {
		case v <= -29: // CR → 換列
			if raw[i] == 13 {
				rows = append(rows, []int{})
			}
			i++
		case v >= sqzThresh: // 跑長
			if i+1 >= len(raw) {
				i = len(raw)
				continue
			}
			n := int(raw[i+1]) - sqzBase
			for k := 0; k < n; k++ {
				rows[len(rows)-1] = append(rows[len(rows)-1], v-sqzThresh)
			}
			i += 2
		default: // 字面
			rows[len(rows)-1] = append(rows[len(rows)-1], v)
			i++
		}
	}
	for len(rows) > 0 && len(rows[len(rows)-1]) == 0 {
		rows = rows[:len(rows)-1]
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf(".SQZ 解出 0 列")
	}

	// 解碼出來的每一段就是一個 major(段內依序是 minor)。
	m := &Maze{Majors: len(rows), Cells: make([]int, len(rows)*MazeRows)}
	for a, seg := range rows {
		for b, v := range seg {
			if b >= MazeRows {
				break
			}
			m.Cells[a*MazeRows+b] = v
		}
	}
	return m, nil
}

// 格值的行為。docs/spec/08 §2。
//
// ⚠ 「阻擋」與「不繪製」是**兩組獨立的規則**。19 是不畫但可走的隱形觸發格,
// 合併成一張表就沒了。
const (
	blockLo, blockHi = 5, 10
	TileTrigger      = 19 // 隱形觸發格
)

// MazePassable:5 ≤ 值 ≤ 10 阻擋,其餘可通行(docs/re/56 §1)。
// ⚠ 是**區間**不是「≥ 5」—— 11 以上可以走。
func MazePassable(v int) bool { return v < blockLo || v > blockHi }

// MazeDrawn:0 / 18 / 19 不繪製(docs/re/56 §2)。負值取絕對值後當繪製編號。
func MazeDrawn(v int) (int, bool) {
	if v == 0 || v == 18 || v == TileTrigger {
		return 0, false
	}
	if v < 0 {
		return -v, true
	}
	return v, true
}

// ---------------------------------------------------------------------------
// MAZEDATA.BIN — 13 筆 × 8 欄
// ---------------------------------------------------------------------------

// MazeEntry 是一個關卡。
//
// ⚠ 起始位置的兩欄是 **(Major, Minor)**,與事件表和 GROUPS.DAT **同一個順序**
// (docs/re/137:12/12 對 10/12)。
type MazeEntry struct {
	WorldX, WorldY int // 欄 0 / 1
	StartMajor     int // 欄 2 —— 乘 81 的那一個
	StartMinor     int // 欄 3
	Facing         int // 欄 4(值域只有 {1, 3},原因未解)
	TextFile       int // 欄 5 → DT<n>TEXT.DAT
	MazeFile       int // 欄 6 → DG<n>MAZE.SQZ
	TextCount      int // 欄 7 = 文字記錄數 − 1
}

const mazeDataCols = 8

// ParseMazeData 解析 MAZEDATA.BIN(BSAVE 容器,word 陣列,**欄主序**)。
func ParseMazeData(d []byte) ([]MazeEntry, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	w := bodyWords(b.Body)
	n := len(w) / mazeDataCols
	if n == 0 {
		return nil, fmt.Errorf("MAZEDATA.BIN 只有 %d 個 word", len(w))
	}
	out := make([]MazeEntry, n)
	col := func(c, i int) int { return int(int16(w[c*n+i])) }
	for i := range out {
		out[i] = MazeEntry{
			WorldX: col(0, i), WorldY: col(1, i),
			StartMajor: col(2, i), StartMinor: col(3, i),
			Facing:   col(4, i),
			TextFile: col(5, i), MazeFile: col(6, i), TextCount: col(7, i),
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// DE*EFF.BIN — 106 × 5 的事件表
// ---------------------------------------------------------------------------

// EventRows 是事件表的列數。原版的查表迴圈是 FOR i = 0 TO 105
// (docs/re/60 §1),欄距也是 106。
const EventRows = 106

// Event 是一筆事件。docs/re/59 §1 + docs/re/137 的順序訂正。
type Event struct {
	Major, Minor int // 欄 0 / 1:觸發位置(Major 乘 81)
	Dir          int // 欄 2:方向 0–4。⚠ 0 的語意未解(docs/spec/08 §4)
	Target       int // 欄 3:≥ 100 → DT 文字編號;< 100 → 目的地的 Major
	DestMinor    int // 欄 4:目的地的 Minor
}

// TextTargetMin 是「目標是文字編號」的門檻(docs/re/59 §1)。
const TextTargetMin = 100

// ParseEvents 解析一個 DE*EFF.BIN。
func ParseEvents(d []byte) ([]Event, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	w := bodyWords(b.Body)
	if len(w) < EventRows*5 {
		return nil, fmt.Errorf("DE*EFF.BIN 只有 %d 個 word,不足 %d",
			len(w), EventRows*5)
	}
	out := make([]Event, EventRows)
	col := func(c, i int) int { return int(int16(w[c*EventRows+i])) }
	for i := range out {
		out[i] = Event{
			Major: col(0, i), Minor: col(1, i), Dir: col(2, i),
			Target: col(3, i), DestMinor: col(4, i),
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// DT*TEXT.DAT — 房間文字
// ---------------------------------------------------------------------------

// ParseDungeonText 解析 DT<n>TEXT.DAT:每列是「3 位數編號 + 敘述」。
func ParseDungeonText(d []byte) map[int]string {
	out := map[int]string{}
	s := strings.ReplaceAll(string(d), "\x1a", "")
	for _, ln := range strings.Split(s, "\r\n") {
		if len(ln) < 3 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(ln[:3]))
		if err != nil {
			continue
		}
		out[n] = strings.TrimSpace(ln[3:])
	}
	return out
}
