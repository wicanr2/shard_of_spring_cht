package original

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// CHARS.DAT — 角色名冊。docs/formats/01-chars-dat.md。
//
// 25 筆 × 94 bytes,定長隨機存取。位移一律 **1-based**(BASIC MID$ 慣例),
// 整數欄位全在**偶數**位移(這是它與 GROUPS.DAT 的判別法,docs/re/99)。

const (
	CharRecLen = 94
	CharSlots  = 25
	// PartySlots 是每一隊的人數上限(手冊 + ds:34F8,docs/re/133 §2)。
	// ⚠ 與 CharSlots 沒有關係 —— 25 = 5×5 是巧合,原版沒有「每隊剛好 5 人」。
	PartySlots = 5
	// NoParty 是位移 1 在「有角色但不屬於任何隊伍」時的值。
	// EmptySlot 是「這一槽從來沒有角色」。
	//
	// ⚠ 兩者**曾經被當成同一件事**(docs/re/133 §3):出貨資料裡只看得到 `'*'`,
	// 所以分不開。讓原版自己造一個角色就分開了 —— 新角色的位移 1 是 `'0'`
	// (docs/re/144 §5)。
	NoParty   = '0'
	EmptySlot = '*'
	// NotEquipped 是裝備格號的哨兵值(位移 34/36),
	// **同時也是背包空格的哨兵**(位移 54–72,docs/re/144 §3)。
	NotEquipped = 99
	// PackSlots 是背包格數。**10 格,不是 15**(docs/re/144 §3):
	// 位移 74 起讀成 word 是 12336 = 0x3030 = 兩個 ASCII `'0'`,
	// 那是另一串字串旗標,不是背包的後五格。手冊 p.39 也寫「只能帶 10 件」。
	PackSlots = 10
	// offExp 是經驗值的 1-based 位移,4 bytes MBF 單精度。
	// 五處程式碼、三支模組都讀這裡(docs/re/150 §1),
	// 其中兩處緊接在載入 `Exp.  ` 標籤之後 —— 標籤與欄位相鄰。
	offExp = 90
	// offSkillUsed 是「今天用過技能」的旗標位移(docs/re/166 §1)。
	// `H)unt` 與 `I)dentify` 共用。
	offSkillUsed = 86
)

// 狀態碼(位移 38)。= 法術系別編號(docs/formats/03),名稱取自 `MENU.EXE`
// 的 `OK,Poisoned,Bound,Still Air,Frozen,D E A D`。
//
// ⚠ **`StatusDead` 不等於「HP 為 0」。** 原版的戰後結算同時檢查兩者
// (docs/re/150 §2),所以兩個判斷不能互相替代 ——
// 出貨資料上死人的 HP 也是 0,差別要到「HP 0 但狀態不是 5」才看得出來。
const (
	StatusOK       = 0
	StatusPoisoned = 1
	StatusBound    = 2
	StatusStill    = 3
	StatusFrozen   = 4
	StatusDead     = 5
)

// Character 是一筆角色記錄。欄位順序與 docs/formats/01 的表相同。
type Character struct {
	Party   byte   // 位移 1:所屬隊伍 '1'–'5';'*' = 無(docs/re/133)
	Name    string // 位移 2–11
	ID      int    // 位移 12,1–25
	Race    byte   // 位移 14:H/T/D/E/G
	Class   byte   // 位移 15:'1' = Hero、'2' = Wizard
	Speed   int    // 位移 16
	Str     int    // 位移 18
	Int     int    // 位移 20
	End     int    // 位移 22
	ToHit   int    // 位移 24
	MaxHP   int    // 位移 26
	HP      int    // 位移 28
	MaxSP   int    // 位移 30
	SP      int    // 位移 32
	Weapon  int    // 位移 34:背包格號,99 = 未裝備
	Armor   int    // 位移 36:同上
	Status  int    // 位移 38:= 法術系別編號(docs/formats/03)
	Level   int    // 位移 40
	Skills  string // 位移 42–51:十個 '0'/'1',**表由職業決定**
	Pack    [PackSlots]int
	Flags2  string // 位移 74–83:第二串十個旗標,**語意未解**(docs/re/144 §6)
	StatMag int    // 位移 84:狀態效果強度
	// SkillUsed:今天已經用過技能(docs/re/166 §1)。原版是 `MID$(記錄, 86, 1) = "1"`,
	// `H)unt` 與 `I)dentify` 共用同一個旗標。
	// ⚠ 出貨存檔這一格是**空白**不是 `'0'` —— 判定要寫成「等於 `'1'` 才算用過」。
	SkillUsed bool
	SkillPts  int     // 位移 89:剩餘技能點數(docs/re/144 §4)
	Exp       float64 // 位移 90–93:經驗值,MBF 單精度(docs/re/150)
	Raw       []byte  // 原始 94 bytes —— 存檔往返要逐位元組保留未解欄位
}

// Occupied 回傳這一槽是否有角色。
//
// ⚠ **判定用名稱,不用位移 1**(docs/spec/06 §2)。
// 位移 1 的 '*' 究竟是「空槽」還是「無隊伍」未解,而名稱空白在兩種讀法下
// 都是空槽 —— 這條判定因此不依賴那個未解項。
func (c Character) Occupied() bool { return strings.TrimSpace(c.Name) != "" }

// InParty 回傳這個角色是否編在某一隊,以及隊號(1–5)。
func (c Character) InParty() (int, bool) {
	if !c.Occupied() || c.Party < '1' || c.Party > '5' {
		return 0, false
	}
	return int(c.Party - '0'), true
}

// ParseChars 解析整個 CHARS.DAT。回傳固定 25 筆,含空槽 ——
// **不要在這裡過濾**,槽號就是角色編號,濾掉會讓索引位移。
func ParseChars(d []byte) ([]Character, error) {
	if len(d) != CharRecLen*CharSlots {
		return nil, fmt.Errorf("CHARS.DAT 長度 %d,應為 %d(%d × %d)",
			len(d), CharRecLen*CharSlots, CharSlots, CharRecLen)
	}
	out := make([]Character, CharSlots)
	for i := range out {
		r := d[i*CharRecLen : (i+1)*CharRecLen]
		c := Character{
			Party: r[0], Name: str(r, 2, 10), ID: u16(r, 12),
			Race: r[13], Class: r[14],
			Speed: u16(r, 16), Str: u16(r, 18), Int: u16(r, 20),
			End: u16(r, 22), ToHit: u16(r, 24),
			MaxHP: u16(r, 26), HP: u16(r, 28),
			MaxSP: u16(r, 30), SP: u16(r, 32),
			Weapon: u16(r, 34), Armor: u16(r, 36),
			Status: u16(r, 38), Level: u16(r, 40),
			Skills: string(r[41:51]), Flags2: string(r[73:83]),
			StatMag: u16(r, 84), SkillPts: int(r[88]),
			SkillUsed: r[offSkillUsed-1] == '1',
			Exp:       MBF(r[offExp-1 : offExp+3]),
			Raw:       append([]byte(nil), r...),
		}
		// 背包:位移 54 + 2i(docs/formats/01)
		for k := range c.Pack {
			c.Pack[k] = u16(r, 54+2*k)
		}
		out[i] = c
	}
	return out, nil
}

// Party 回傳第 n 隊(1–5)的成員,依名冊順序。
func Party(chars []Character, n int) []Character {
	var out []Character
	for _, c := range chars {
		if p, ok := c.InParty(); ok && p == n {
			out = append(out, c)
		}
	}
	return out
}

// 種族與職業的顯示名。docs/formats/01 + translations/glossary.md。
var (
	raceName = map[byte]string{
		'H': "人類", 'T': "巨魔", 'D': "矮人", 'E': "精靈", 'G': "地精",
	}
	className = map[byte]string{'1': "戰士", '2': "法師"}
	// 狀態。translations/glossary.md「狀態」表,0 起算。
	// ⚠ 0(OK)在狀態欄顯示**空白**,不顯示「正常」——
	// 五個人都正常時整欄空著,異常才跳出來(docs/spec/06 §5)。
	statusName = [...]string{"", "中毒", "束縛", "凝滯", "冰封", "死亡"}
)

func (c Character) RaceName() string  { return raceName[c.Race] }
func (c Character) ClassName() string { return className[c.Class] }

// StatusName 回傳狀態的中文兩字;正常回空字串。
func (c Character) StatusName() string {
	if c.Status < 0 || c.Status >= len(statusName) {
		return "?"
	}
	return statusName[c.Status]
}

// Bytes 把角色寫回 94 bytes。
//
// ⚠ 與 Group.Bytes 同一條原則:**從 Raw 出發,只覆寫已解的欄位**
// (docs/spec/06 §3)。位移 1、12–51、84 之外的位置原樣保留 ——
// 清成 0 會讓存檔在原版裡打不開,而我們的往返測試看不見。
func (c Character) Bytes() []byte {
	r := make([]byte, CharRecLen)
	if len(c.Raw) == CharRecLen {
		copy(r, c.Raw)
	} else {
		for i := range r {
			r[i] = ' '
		}
	}
	r[0] = c.Party
	name := []byte(c.Name)
	for i := 0; i < 10; i++ {
		if i < len(name) {
			r[1+i] = name[i]
		} else {
			r[1+i] = ' '
		}
	}
	put := func(pos1, v int) {
		binary.LittleEndian.PutUint16(r[pos1-1:pos1+1], uint16(int16(v)))
	}
	put(12, c.ID)
	r[13], r[14] = c.Race, c.Class
	put(16, c.Speed)
	put(18, c.Str)
	put(20, c.Int)
	put(22, c.End)
	put(24, c.ToHit)
	put(26, c.MaxHP)
	put(28, c.HP)
	put(30, c.MaxSP)
	put(32, c.SP)
	put(34, c.Weapon)
	put(36, c.Armor)
	put(38, c.Status)
	put(40, c.Level)
	for i := 0; i < 10; i++ {
		if i < len(c.Skills) {
			r[41+i] = c.Skills[i]
		}
	}
	for k, v := range c.Pack {
		put(54+2*k, v)
	}
	put(84, c.StatMag)
	// 位移 86:⚠ 只在旗標為真時寫 '1' —— 出貨存檔的「沒用過」是**空白**,
	// 寫成 '0' 會動到一個原版沒動過的位元組(docs/re/166 §1)。
	if c.SkillUsed {
		r[offSkillUsed-1] = '1'
	} else if r[offSkillUsed-1] == '1' {
		r[offSkillUsed-1] = ' '
	}
	for i := 0; i < 10; i++ {
		if i < len(c.Flags2) {
			r[73+i] = c.Flags2[i]
		}
	}
	r[88] = byte(c.SkillPts)
	// 經驗值(位移 90–93,MBF 單精度)。
	//
	// ⚠ **只在值真的變了才覆寫。** 原版的「零經驗」寫成 `00 00 00 40`
	// (MBF 解出來是 2.7e-20,畫面印 0),而 PutMBF(0) 寫的是 `00 00 00 00` ——
	// 兩者行為相同但位元組不同。無條件覆寫會讓「載入後原樣存回」的
	// 逐位元組往返測試失敗,而那個失敗與真正的欄位錯誤長得一樣。
	// docs/re/150 §1.2。
	if MBF(r[offExp-1:offExp+3]) != c.Exp {
		copy(r[offExp-1:offExp+3], PutMBF(c.Exp))
	}
	return r
}
