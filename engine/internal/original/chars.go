package original

import (
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
	// NoParty 是位移 1 在「不屬於任何隊伍」時的值。
	// ⚠ 它同時出現在空槽上,兩種語意在出貨資料上分不開(docs/re/133 §3)。
	NoParty = '*'
	// NotEquipped 是裝備格號的哨兵值(位移 34/36)。
	NotEquipped = 99
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
	Pack    [15]int
	StatMag int    // 位移 84:狀態效果強度
	Raw     []byte // 原始 94 bytes —— 存檔往返要逐位元組保留未解欄位
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
			Skills: string(r[41:51]), StatMag: u16(r, 84),
			Raw: append([]byte(nil), r...),
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
