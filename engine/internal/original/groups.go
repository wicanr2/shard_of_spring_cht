package original

import (
	"encoding/binary"
	"fmt"
	"math"
)

// GROUPS.DAT — 隊伍狀態與存檔。docs/formats/02-groups-dat.md
// + docs/re/134(載入常式的逐欄對照)。
//
// 5 筆 × 90 bytes。整數欄位全在**奇數**位移(與 CHARS.DAT 相反,docs/re/99)。

const (
	GroupRecLen = 90
	GroupSlots  = 5

	// MemberSlots 是成員槽的數量:位移 2i−1,i = 1…9(docs/re/135 §3)。
	//
	// **九個槽就是 3×3 的陣型格**(docs/re/203):營地的 `R)eorder` 把它畫成
	// `A B C / D E F / G H I`,玩家用字母選位置。人數上限 5 而格子有 9,
	// 是因為那是**站位**不是名額 —— 九格裡永遠有空位可以搬進去。
	MemberSlots = 9
	// 成員編號的有效範圍。原版是 `1 ≤ 值 ≤ 32`;不在範圍內(出貨檔用 99)= 空槽。
	// ⚠ 上限 32 與名冊的 25 槽對不上,同樣未解。
	memberMin, memberMax = 1, 32

	// 位移。名稱與 docs/re/134 §2 的表一一對應。
	offGold       = 19 // 4 bytes,**MBF 單精度**,不是整數
	offProvisions = 23
	offEncounter  = 25
	offMonth      = 27
	offDay        = 29
	offHour       = 31
	offSub        = 33
	offWorldX     = 35
	offWorldY     = 37
	offFacing     = 41
	offLightTurns = 45
	offVisLit     = 59
	offVisDark    = 61
	// offPoolUses 是迷宮治療池的已使用次數(docs/re/155 §2.1)。
	// `< 11` 才能用,用完原版印 `This pool is empty!`。
	// ⚠ **跨迷宮共用一個計數器** —— 它存在隊伍記錄裡,不在迷宮資料裡。
	offPoolUses  = 63
	// offGateOpen 是 `DAZA REVELI` 大門已開啟的旗標(docs/re/197 §3)。
	// ⚠ 存在**隊伍記錄**裡 → 跨存檔留著,不是只在這一趟有效。
	offGateOpen  = 65
	offMazeX     = 79
	offMazeY     = 81
	offMazeNum   = 83
	offFled      = 85
)

// NotInMaze 是位移 83 的哨兵值:**不在任何迷宮**(docs/re/169 §4、re/204 §1)。
//
// ⚠ 這一欄先前叫 `LightPick`(「沒有攜帶光源」)。名字錯了、機制沒錯:
// 「在地面用天色、在迷宮用火把」的判斷條件就是它,而它同時決定
// `MENU` 要 `CHAIN` 到 `WRLDMOVE` 還是 `MAZEMOVE`,也決定遭遇區域。
// 99 與 CHARS.DAT 的「未裝備」共用同一個哨兵約定,但語意無關。
const NotInMaze = 99

// Group 是一筆隊伍記錄。
type Group struct {
	// Members 是九個成員槽的原始值(位移 1,3,…,17)。
	// **保留原始值而不是過濾後的清單** —— 槽號是有意義的,而且存檔要寫回。
	Members [MemberSlots]int

	Gold                  float64 // 位移 19–22,MBF 單精度 —— **不是 int16**
	Provisions            int
	Encounter             int
	Month, Day, Hour, Sub int
	WorldX, WorldY        int
	Facing                int
	LightTurns            int
	VisLit, VisDark       int
	PoolUses              int // 位移 63:治療池已使用次數(docs/re/155)
	GateOpen              int // 位移 65:`DAZA REVELI` 大門已開啟(docs/re/197)
	MazeX, MazeY          int // 1-based
	MazeNum               int // 位移 83:當前迷宮編號,99 = 不在迷宮
	Fled                  int

	// Raw 是原始 90 bytes。**未解的欄位(39、43、47–57、67–77、87–90)
	// 只存在於這裡** —— 存檔往返靠它逐位元組保留(docs/spec/06 §3)。
	Raw []byte
}

// Blank 回傳這筆記錄是否整份空白(0x20)= **從未被使用過的隊伍**。
//
// 記錄由空白字串建立、再逐欄覆寫,所以 `CVI(" ")` = 0x2020 = 8224
// 等於「這個欄位還沒用過」。不判這一條的話,新遊戲會從座標 (8224, 8224)
// 開始,而那**不會拋任何例外**(docs/spec/06 §4)。
//
// ⚠ 出貨檔的第 1–4 筆是空白,**第 5 筆是一份真實存檔**
// (手冊寫的 PARTY #5,docs/re/135)。
func (g Group) Blank() bool {
	for _, b := range g.Raw {
		if b != ' ' {
			return false
		}
	}
	return true
}

// ParseGroups 解析整個 GROUPS.DAT,固定回 5 筆(含空白的)。
func ParseGroups(d []byte) ([]Group, error) {
	if len(d) != GroupRecLen*GroupSlots {
		return nil, fmt.Errorf("GROUPS.DAT 長度 %d,應為 %d(%d × %d)",
			len(d), GroupRecLen*GroupSlots, GroupSlots, GroupRecLen)
	}
	out := make([]Group, GroupSlots)
	for i := range out {
		r := d[i*GroupRecLen : (i+1)*GroupRecLen]
		g := Group{
			Gold:       MBF(r[offGold-1 : offGold+3]),
			Provisions: u16(r, offProvisions),
			Encounter:  u16(r, offEncounter),
			Month:      u16(r, offMonth),
			Day:        u16(r, offDay),
			Hour:       u16(r, offHour),
			Sub:        u16(r, offSub),
			WorldX:     u16(r, offWorldX),
			WorldY:     u16(r, offWorldY),
			Facing:     u16(r, offFacing),
			LightTurns: u16(r, offLightTurns),
			VisLit:     u16(r, offVisLit),
			VisDark:    u16(r, offVisDark),
			PoolUses:   u16(r, offPoolUses),
			GateOpen:   u16(r, offGateOpen),
			MazeX:      u16(r, offMazeX),
			MazeY:      u16(r, offMazeY),
			MazeNum:    u16(r, offMazeNum),
			Fled:       u16(r, offFled),
			Raw:        append([]byte(nil), r...),
		}
		for k := range g.Members {
			g.Members[k] = u16(r, 1+2*k) // 位移 2i−1,i 從 1 起
		}
		out[i] = g
	}
	return out, nil
}

// ValidMemberID 回傳這個成員槽的值是不是一個真的角色編號。
// 空槽在出貨資料裡是 `0x2020`(兩個空白)或 99 這類哨兵。
func ValidMemberID(v int) bool { return v >= memberMin && v <= memberMax }

// MemberSlotNumbers 回傳實際在隊的成員各佔**第幾個槽**(1–9,與 MemberIDs 同序)。
//
// ⚠ 槽號不是「隊伍裡的第幾個人」:搬到有間隔的槽時兩者不同,
// 而**戰場站位看的是槽號**(docs/re/210)。
func (g Group) MemberSlotNumbers() []int {
	var out []int
	for i, v := range g.Members {
		if ValidMemberID(v) {
			out = append(out, i+1)
		}
	}
	return out
}

// MemberIDs 回傳實際在隊的角色編號,依槽序。
// 值不在 1–32 的槽(出貨檔用 99)略過 —— 與原版 0x114CA 的判斷相同。
func (g Group) MemberIDs() []int {
	var out []int
	for _, v := range g.Members {
		if v >= memberMin && v <= memberMax {
			out = append(out, v)
		}
	}
	return out
}

// Bytes 把 Group 寫回 90 bytes。
//
// **從 Raw 出發,只覆寫已知的欄位** —— 位移 1–18 與其他未解位置原樣保留。
// 若把未解的位置清成 0,存檔在原版裡就打不開了,而**測試不會發現**
// (我們的解析器讀不到那些欄位,往返測試也就看不見它們)。
func (g Group) Bytes() []byte {
	r := make([]byte, GroupRecLen)
	if len(g.Raw) == GroupRecLen {
		copy(r, g.Raw)
	} else {
		for i := range r {
			r[i] = ' '
		}
	}
	for k, v := range g.Members {
		binary.LittleEndian.PutUint16(r[2*k:2*k+2], uint16(int16(v)))
	}
	copy(r[offGold-1:offGold+3], PutMBF(g.Gold))
	for _, f := range []struct {
		pos, v int
	}{
		{offProvisions, g.Provisions}, {offEncounter, g.Encounter},
		{offMonth, g.Month}, {offDay, g.Day}, {offHour, g.Hour}, {offSub, g.Sub},
		{offWorldX, g.WorldX}, {offWorldY, g.WorldY}, {offFacing, g.Facing},
		{offLightTurns, g.LightTurns}, {offVisLit, g.VisLit}, {offVisDark, g.VisDark},
		{offPoolUses, g.PoolUses}, {offGateOpen, g.GateOpen},
		{offMazeX, g.MazeX}, {offMazeY, g.MazeY},
		{offMazeNum, g.MazeNum}, {offFled, g.Fled},
	} {
		binary.LittleEndian.PutUint16(r[f.pos-1:f.pos+1], uint16(int16(f.v)))
	}
	return r
}

// PutMBF 把 float64 編成 4-byte MBF 單精度(MKS$ 的反向)。
//
// 格式與 MBF() 對稱:b[3] = 指數 + 129、b[2] 的最高位是符號、
// 尾數 23 bits 隱含前導 1。**指數 0 代表數值 0**(沒有非正規數)。
func PutMBF(v float64) []byte {
	out := make([]byte, 4)
	if v == 0 || math.IsNaN(v) {
		return out // 指數 0 = 零
	}
	neg := v < 0
	v = math.Abs(v)
	e := 129
	for v >= 2 {
		v /= 2
		e++
	}
	for v < 1 {
		v *= 2
		e--
	}
	frac := uint32(math.Round((v - 1) * (1 << 23)))
	if frac >= 1<<23 { // 進位溢位:1.9999… 進成 2.0
		frac = 0
		e++
	}
	if e < 1 {
		return out // 下溢 → 0
	}
	if e > 255 { // 上溢 —— 夾在最大值,不要回 0(那會讓金幣憑空消失)
		frac, e = 1<<23-1, 255
	}
	out[0] = byte(frac)
	out[1] = byte(frac >> 8)
	out[2] = byte(frac>>16) & 0x7F
	if neg {
		out[2] |= 0x80
	}
	out[3] = byte(e)
	return out
}
