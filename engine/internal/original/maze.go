package original

import (
	"bytes"
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
//
// 對應到 `.SQZ` 的形狀(docs/re/205 §4):
//
//	Major = **第幾列**(CR 分出來的第幾段,0–80)
//	Minor = **一列裡的第幾格**(0–50)
//
// ⚠ 原版陣列的線性排法**與這裡相反**(它乘的是「第幾格」,
// `MAZEMOVE` 0x13FC6)。兩邊指到的格子相同,只有把原版的**絕對索引**
// 搬進來時才需要換算 —— 目前只有拉利斯之門那一個常數。
//
// ⚠ 因此 `Minor` 實際只用到 0–50,51–80 恆為 0(每列最多 51 格)。
// 那不是資料缺損,是這個排法的空隙。
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
	// ⚠ **欄 0 是南北、欄 1 是東西** —— 與 docs/re/141 訂正過的兩軸一致。
	// 判別法是正對照:把兩欄直接當 (x,y) 去查地形,**11 個入口 0 個命中**;
	// 對調之後 **11/11 都落在入口圖塊(24/25/27/28)上**。
	//
	// ⚠ 這裡曾經寫反,而症狀是**地城完全進不去** ——
	// 沒有錯誤訊息,只是走到入口格什麼都不會發生。
	WorldX, WorldY int // 欄 1 / 欄 0
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
			WorldX: col(1, i), WorldY: col(0, i),
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

// 地城名的兩串 `DATA` 敘述在 `MENU.EXE` 的檔案位移(docs/re/123 §1)。
//
// ⚠ 名字**不在 `MAZEDATA.BIN` 裡** —— 那八欄沒有名稱欄(見 MazeEntry)。
// `MENU` 開機時把兩串 `DATA` 讀進 COMMON,`MAZEMOVE` 進地城時印在左上角。
const (
	menuDungeonList1 = 10030 // 6 個,對應入口 0–5
	menuDungeonList2 = 10103 // 7 個,對應入口 6–12
)

// DungeonNames 從 `MENU.EXE` 讀出 13 個地城名,索引 = `MAZEDATA` 的入口編號。
//
// 對應關係是**實跑量出來的**(docs/re/222):入口 0 `Old Man in Cave`、
// 2 `Black Fort`、5 `Ralith`、7 `Murthin`、11 `Vandiguard` ——
// 兩串各自的頭、中、尾都命中。
func DungeonNames(menuExe []byte) ([]string, error) {
	var out []string
	for _, spec := range []struct {
		off, want int
	}{{menuDungeonList1, 6}, {menuDungeonList2, 7}} {
		if spec.off >= len(menuExe) {
			return nil, fmt.Errorf("MENU.EXE 只有 %d bytes,讀不到位移 %d",
				len(menuExe), spec.off)
		}
		end := bytes.IndexByte(menuExe[spec.off:], 0)
		if end < 0 {
			return nil, fmt.Errorf("位移 %d 之後沒有 null 結尾", spec.off)
		}
		// ⚠ 每一串開頭有一個前導空白,那是 BASIC `DATA` 在逗號後保留空白的
		// 慣例,不是名稱的一部分(docs/re/123 §1)。末串的 `Ralith.` 也一樣 ——
		// 那個句點是敘述的結束,不是名字。
		var names []string
		for _, s := range strings.Split(string(menuExe[spec.off:spec.off+end]), ",") {
			names = append(names, strings.TrimSuffix(strings.TrimSpace(s), "."))
		}
		if len(names) != spec.want {
			return nil, fmt.Errorf("位移 %d 切出 %d 個名字,應為 %d:%q",
				spec.off, len(names), spec.want, names)
		}
		out = append(out, names...)
	}
	return out, nil
}
