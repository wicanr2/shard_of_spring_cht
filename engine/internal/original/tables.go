package original

import (
	"encoding/binary"
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
	Speed  int    `json:"speed"`  // 欄1 位移17 → 戰鬥屬性 2(乘亂數後存入)
	Str    int    `json:"str"`    // 欄2 位移19 → 屬性 6
	ToHit  int    `json:"to_hit"` // 欄3 位移21 → 屬性 9
	HPDie  int    `json:"hp_die"` // 欄4 位移23 → 屬性 3(生命值的骰基,乘亂數)
	Weapon int    `json:"weapon"` // 欄5 位移25 → 屬性 4(0 → 60 = 赤手)
	Class  int    `json:"class"`  // 欄6 位移27 → 屬性 11(類別 / 圖組)
	Armor  int    `json:"armor"`  // 欄7 位移29 → 屬性 5
	Exp    int    `json:"exp"`    // 欄8 位移31 → 屬性 19
	Tier   int    `json:"tier"`   // 欄9 位移33 → 屬性 13
	SP     int    `json:"sp"`     // 欄10 位移35 → 屬性 7
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
	School   int    `json:"school"`    // 欄2:1–5,= 命中後的狀態編號
	Effect   int    `json:"effect"`    // 欄3:效果類別 1–13
	Power    int    `json:"power"`     // 欄4:每點威力(類別 3–6 時正負決定增減)
	UnitCost int    `json:"unit_cost"` // 欄5:每一級的法力單價;等級 = INT(投入 / 這個值)
	HitMsg   string `json:"hit_msg"`   // 欄6:命中訊息
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

// ShopKind 是建築類別。docs/re/138 §2:位移 34 的負值。
type ShopKind int

const (
	ShopGoods   ShopKind = 0  // 位移 34 ≥ 0:賣道具,範圍 [First, Last]
	ShopHealer  ShopKind = -1 // 治療所
	ShopTavern  ShopKind = -2 // 酒館;位移 36 = 傳聞編號 1–11
	ShopInn     ShopKind = -3 // 旅店
	ShopTrainer ShopKind = -4 // 訓練所;位移 36:0 = 武術、1 = 魔法
)

func (k ShopKind) String() string {
	switch k {
	case ShopHealer:
		return "治療所"
	case ShopTavern:
		return "酒館"
	case ShopInn:
		return "旅店"
	case ShopTrainer:
		return "訓練所"
	}
	return "商店"
}

type Shop struct {
	Index int    `json:"index"`
	Town  string `json:"town"`
	Name  string `json:"name"`
	// Kind:位移 34 為負時是建築類別,≥ 0 時是 ShopGoods(docs/re/138)。
	Kind ShopKind `json:"kind"`
	// First / Last:賣的道具編號範圍(含)。只在 Kind == ShopGoods 時有意義。
	//
	// ⚠ 先前的實作把 57 件道具全列進每一間店 —— 而那看起來完全正常,
	// 玩家不會知道劍舖不該賣板甲。**「多顯示一些」的錯誤沒有症狀。**
	First int `json:"first"`
	Last  int `json:"last"`
	// Extra:位移 36 在特殊建築上的值(酒館 = 傳聞編號、訓練所 = 0 武 / 1 魔)。
	Extra int `json:"extra"`
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
		// 位移 34/36 是兩個 16-bit 整數(docs/re/138)。
		// ⚠ 商店名是 **18 bytes**(16–33),不是 16 —— 位移 34 才是數字欄。
		a := int(int16(binary.LittleEndian.Uint16(r[34:36])))
		b := int(int16(binary.LittleEndian.Uint16(r[36:38])))
		sh := Shop{
			Index:     len(out),
			Town:      strings.TrimSpace(string(r[0:16])),
			Name:      strings.TrimSpace(string(r[16:34])),
			PriceMult: MBF(r[38:42]),
		}
		if a < 0 {
			sh.Kind, sh.Extra = ShopKind(a), b
		} else {
			sh.Kind, sh.First, sh.Last = ShopGoods, a, b
		}
		out = append(out, sh)
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

// ---------------------------------------------------------------------------
// TOWNDATA.BIN — 城鎮座標表。docs/re/53 §2(座標)、docs/re/141(兩軸訂正)。
// ---------------------------------------------------------------------------

// TownSite 是一個城鎮在世界地圖上的位置與建築數。
//
// ⚠ 這張表**早就解出來了**(docs/re/53 §2 的三重驗證),
// 而 `enterTown` 一直用「按座標排序取第 n 個」的佔位 ——
// 那個佔位在實跑裡被打臉:(24,12) 是 **Arcania**(表的第 4 列),
// 不是排序給出的 Gleon。**答案早就在自己的 docs 裡。**
type TownSite struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Shops int `json:"shops"`
}

// TownSitesLen 是城鎮數。
const TownSitesLen = 13

// ParseTownSites 解 TOWNDATA.BIN:56 個 MBF 單精度,排成 4 欄 × 14 列,第 0 列全零。
//
//	欄 1 = 東西座標、欄 2 = 南北座標、欄 3 = 建築數
//
// ⚠ re/53 §2 把欄 1/2 標成 `Y`/`X`,那是**兩軸接反時的命名**(docs/re/141)——
// 數值沒變,名字反了。回傳的第 0 筆對應原版的第 1 列。
func ParseTownSites(d []byte) ([]TownSite, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	if len(b.Body) < 4*4*14 {
		return nil, fmt.Errorf("TOWNDATA.BIN 只有 %d bytes,需要 224", len(b.Body))
	}
	// ⚠ 排法是**逐欄**存(BASIC 的二維陣列):索引 = 欄 × 14 + 列,
	// 不是 列 × 4 + 欄。寫反了會取到別欄的值,而那些值同樣落在合理範圍內。
	at := func(row, col int) int { return int(MBF(b.Body[(col*14+row)*4:])) }
	out := make([]TownSite, 0, TownSitesLen)
	for row := 1; row <= TownSitesLen; row++ {
		out = append(out, TownSite{X: at(row, 1), Y: at(row, 2), Shops: at(row, 3)})
	}
	return out, nil
}
