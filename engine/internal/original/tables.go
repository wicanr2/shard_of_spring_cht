package original

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// MONSTERS.DAT — docs/formats/03
// 74 筆 × 36 bytes:位移 1–16 名稱,位移 17–36 十個 2-byte 整數。
// ---------------------------------------------------------------------------

const monsterRecLen = 36

// Monster 的欄位名稱取自 docs/formats/03 的語意欄,不是原文欄號。
type Monster struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	Speed  int    `json:"speed"`   // 欄1 位移17 → 戰鬥屬性 2(乘亂數後存入)
	Str    int    `json:"str"`     // 欄2 位移19 → 屬性 6
	ToHit  int    `json:"to_hit"`  // 欄3 位移21 → 屬性 9
	HPDie  int    `json:"hp_die"`  // 欄4 位移23 → 屬性 3(生命值的骰基,乘亂數)
	Weapon int    `json:"weapon"`  // 欄5 位移25 → 屬性 4(0 → 60 = 赤手)
	Class  int    `json:"class"`   // 欄6 位移27 → 屬性 11(類別 / 圖組)
	Armor  int    `json:"armor"`   // 欄7 位移29 → 屬性 5
	Exp    int    `json:"exp"`     // 欄8 位移31 → 屬性 19
	Tier   int    `json:"tier"`    // 欄9 位移33 → 屬性 13
	SP     int    `json:"sp"`      // 欄10 位移35 → 屬性 7
}

func ParseMonsters(d []byte) ([]Monster, error) {
	if len(d)%monsterRecLen != 0 {
		return nil, fmt.Errorf("MONSTERS.DAT 長度 %d 不是 %d 的倍數", len(d), monsterRecLen)
	}
	out := make([]Monster, 0, len(d)/monsterRecLen)
	for i := 0; i+monsterRecLen <= len(d); i += monsterRecLen {
		r := d[i : i+monsterRecLen]
		out = append(out, Monster{
			Index:  len(out),
			Name:   str(r, 1, 16),
			Speed:  u16(r, 17),
			Str:    u16(r, 19),
			ToHit:  u16(r, 21),
			HPDie:  u16(r, 23),
			Weapon: u16(r, 25),
			Class:  u16(r, 27),
			Armor:  u16(r, 29),
			Exp:    u16(r, 31),
			Tier:   u16(r, 33),
			SP:     u16(r, 35),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// SPELLS.DAT / ITEMS.DAT — docs/formats/04,逗號分隔文字,6 欄
// ---------------------------------------------------------------------------

// Spell 依 docs/formats/04 的欄位語意。
type Spell struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	School   int    `json:"school"`     // 欄2:1–5,= 命中後的狀態編號
	Effect   int    `json:"effect"`     // 欄3:效果類別 1–13
	Power    int    `json:"power"`      // 欄4:每點威力(類別 3–6 時正負決定增減)
	UnitCost int    `json:"unit_cost"`  // 欄5:每一級的法力單價;等級 = INT(投入 / 這個值)
	HitMsg   string `json:"hit_msg"`    // 欄6:命中訊息
}

// Item 的欄 4/5/6 是雙重身分,分類不在資料裡在呼叫端(docs/formats/04)。
// 這裡兩組名字都留著,由使用端依情境選,**不要在轉換階段就決定**。
type Item struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Alias string `json:"alias"` // 欄2:顯示用別名(小寫)
	// 欄3:⚠ 是**基準價**不是售價。售價 = INT(基準價 × 該店倍率),
	// 倍率在 TOWNDATA.DAT(docs/re/126)。
	BasePrice int `json:"base_price"`
	Col4      int `json:"col4"` // 裝備:傷害/護甲值  魔法道具:法術編號
	Col5      int `json:"col5"` // 裝備:命中加值      魔法道具:投入的法術點數
	Col6      int `json:"col6"` // 裝備:種類代碼      魔法道具:發動成功率(分母 26)
}

// splitCSV 切原版的逗號分隔列。原版用 CRLF 分隔、0x1A 結尾。
func splitCSV(d []byte) [][]string {
	s := strings.ReplaceAll(string(d), "\x1a", "")
	var out [][]string
	for _, line := range strings.Split(s, "\r\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, strings.Split(line, ","))
	}
	return out
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// spellSentinel 是檔案最後一列 `ET CETERA,0,0,0,0,0`,**不讀**
// (docs/formats/04:33 列 + 1 列哨兵)。
const spellSentinel = "ET CETERA"

func ParseSpells(d []byte) ([]Spell, error) {
	var out []Spell
	for _, f := range splitCSV(d) {
		if len(f) < 6 {
			return nil, fmt.Errorf("SPELLS.DAT 第 %d 列只有 %d 欄", len(out), len(f))
		}
		if strings.TrimSpace(f[0]) == spellSentinel {
			break // 哨兵,以及它之後的任何東西
		}
		out = append(out, Spell{
			Index: len(out), Name: strings.TrimSpace(f[0]),
			School: atoi(f[1]), Effect: atoi(f[2]),
			Power: atoi(f[3]), UnitCost: atoi(f[4]),
			HitMsg: f[5],
		})
	}
	return out, nil
}

func ParseItems(d []byte) ([]Item, error) {
	var out []Item
	for _, f := range splitCSV(d) {
		if len(f) < 6 {
			return nil, fmt.Errorf("ITEMS.DAT 第 %d 列只有 %d 欄", len(out), len(f))
		}
		out = append(out, Item{
			Index: len(out), Name: strings.TrimSpace(f[0]), Alias: strings.TrimSpace(f[1]),
			BasePrice: atoi(f[2]), Col4: atoi(f[3]), Col5: atoi(f[4]), Col6: atoi(f[5]),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// TOWNDATA.DAT — docs/re/126
// 61 筆 × 45 bytes:城鎮名 16 + 商店名 16 + 13 bytes 二進位。
// 記錄位移 38(0-based)是 MBF 單精度的**價格倍率**。
// ---------------------------------------------------------------------------

const shopRecLen = 45

type Shop struct {
	Index int     `json:"index"`
	Town  string  `json:"town"`
	Name  string  `json:"name"`
	// PriceMult:售價 = INT(ItemDef.BasePrice × PriceMult)。
	// ⚠ 三筆記錄(訓練所 / 學院 / 競技場)這裡不是合法 MBF,值為 0,
	// 語意未解(docs/re/126 §4)—— **不要當成「免費」**。
	PriceMult float64 `json:"price_mult"`
}

func ParseShops(d []byte) ([]Shop, error) {
	if len(d)%shopRecLen != 0 {
		return nil, fmt.Errorf("TOWNDATA.DAT 長度 %d 不是 %d 的倍數", len(d), shopRecLen)
	}
	out := make([]Shop, 0, len(d)/shopRecLen)
	for i := 0; i+shopRecLen <= len(d); i += shopRecLen {
		r := d[i : i+shopRecLen]
		out = append(out, Shop{
			Index:     len(out),
			Town:      strings.TrimSpace(string(r[0:16])),
			Name:      strings.TrimSpace(string(r[16:32])),
			PriceMult: MBF(r[38:42]),
		})
	}
	return out, nil
}

// Towns 回傳不重複的城鎮名,依首次出現的順序(13 個)。
func Towns(shops []Shop) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range shops {
		if !seen[s.Town] {
			seen[s.Town] = true
			out = append(out, s.Town)
		}
	}
	return out
}
