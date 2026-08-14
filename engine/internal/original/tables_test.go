package original

import (
	"os"
	"path/filepath"
	"testing"
)

// gameDir 是原版檔的位置。⚠ 原版不進版控(CLAUDE.md §1),
// 所以沒有原版時測試**跳過而不是失敗** —— 但要說清楚是跳過,
// 不能讓「沒跑到」看起來像「通過」。
func gameDir(t *testing.T) string {
	t.Helper()
	for _, d := range []string{"/game/sharspri", "../../../game/sharspri"} {
		if _, err := os.Stat(filepath.Join(d, "SPELLS.DAT")); err == nil {
			return d
		}
	}
	t.Skip("找不到原版 game/sharspri —— 這一項沒有被驗證")
	return ""
}

func read(t *testing.T, name string) []byte {
	t.Helper()
	d, err := os.ReadFile(filepath.Join(gameDir(t), name))
	if err != nil {
		t.Fatalf("讀 %s:%v", name, err)
	}
	return d
}

// ---------------------------------------------------------------------------
// 以下的期望值全部來自**精訊資訊 1987 官方中文說明書**(docs/manual/),
// 不是從資料檔自己算出來的。這是 CLAUDE.md §2.1 條件 3 要的那種印證:
// 一邊是 1987 年印刷的紙本,一邊是資料檔的位元組,沒有共同的錯誤來源。
// ---------------------------------------------------------------------------

// 手冊 p.28 的五系最低法力總表(docs/re/125 §2)。
// ⚠ MAGIC TORCH 手冊 p.28 表格印 3、p.26 正文印 2 —— 資料檔是 2,
// 表格才是印刷錯誤。這裡採正文的 2。
var manualSpellCost = map[string]int{
	"COLUMN OF FIRE": 1, "FLAME STRIKE": 16, "FIRE STORM": 10, "MELT": 11,
	"FLAME SHIELD": 4, "MAGIC TORCH": 2,
	"SWORD": 2, "CHAINS": 10, "DEATH BLADE": 15, "STRENGTH": 1,
	"BREAK BONDS": 11, "ARMOR": 2,
	"TEMPEST": 6, "STILL AIR": 11, "WINGS OF VICTORY": 1, "WINGS": 4,
	"FREEDOM": 13, "WIND WALK": 10, "BREATH OF LIFE": 5,
	"HAIL STORM": 7, "CHILL": 1, "SLOW": 3, "FREEZE": 9, "ICE SHIELD": 3,
	"CRYSTALIGHT": 2,
	"SPIRIT WRACK": 20, "WEAKEN": 1, "CLUMSINESS": 2, "HEAL": 1,
	"RESURRECT": 25, "CURE POISON": 9, "TRANSFERENCE": 3, "SANCTUARY": 3,
}

func TestSpellsMatchManual(t *testing.T) {
	spells, err := ParseSpells(read(t, "SPELLS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if len(spells) != 33 {
		t.Fatalf("法術數 %d,規格說 33(docs/formats/04)", len(spells))
	}
	for _, s := range spells {
		want, ok := manualSpellCost[s.Name]
		if !ok {
			t.Errorf("手冊沒有這個法術:%q", s.Name)
			continue
		}
		if s.UnitCost != want {
			t.Errorf("%s 法力單價:檔案 %d,手冊 %d", s.Name, s.UnitCost, want)
		}
	}
}

// 系別 1–5,且 = 命中後的狀態編號(docs/formats/03)。
func TestSpellSchoolRange(t *testing.T) {
	spells, _ := ParseSpells(read(t, "SPELLS.DAT"))
	for _, s := range spells {
		if s.School < 1 || s.School > 5 {
			t.Errorf("%s 系別 %d 超出 1–5", s.Name, s.School)
		}
	}
}

// 手冊 p.30 的武器表:傷害(欄4)與**基準價**(欄3)。
func TestWeaponsMatchManual(t *testing.T) {
	items, err := ParseItems(read(t, "ITEMS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 57 {
		t.Fatalf("物品數 %d,規格說 57(docs/formats/04)", len(items))
	}
	want := map[string][2]int{ // 名稱 → {傷害, 價格}
		"Dagger": {3, 2}, "Small axe": {4, 6}, "Short sword": {6, 15},
		"Mace": {6, 13}, "Morning star": {7, 20}, "Broad sword": {8, 30},
		"Battle axe": {10, 65}, "2-hand sword": {12, 100},
	}
	seen := 0
	for _, it := range items {
		w, ok := want[it.Name]
		if !ok {
			continue
		}
		seen++
		if it.Col4 != w[0] {
			t.Errorf("%s 傷害:檔案 %d,手冊 %d", it.Name, it.Col4, w[0])
		}
		if it.BasePrice != w[1] {
			t.Errorf("%s 基準價:檔案 %d,手冊 %d", it.Name, it.BasePrice, w[1])
		}
	}
	if seen != len(want) {
		t.Errorf("手冊列的 %d 件武器只在檔案裡找到 %d 件", len(want), seen)
	}
}

// 手冊 p.30 的護甲表**只有防護值對得上,價格對不上** ——
// 那不是解析錯誤,是手冊內文譯自 Apple II 版(docs/re/126 §5)。
// 這個測試只驗防護值,並把價格的分歧固定下來,免得後人以為是 bug。
func TestArmorProtectionMatchesManualButPriceDoesNot(t *testing.T) {
	items, _ := ParseItems(read(t, "ITEMS.DAT"))
	prot := map[string]int{"Cloth": 1, "Leather": 2, "Chain": 3, "Scale": 4, "Plate": 5}
	manualPrice := map[string]int{"Cloth": 5, "Leather": 10, "Chain": 20, "Scale": 75, "Plate": 200}
	diverged := 0
	for _, it := range items {
		p, ok := prot[it.Name]
		if !ok {
			continue
		}
		if it.Col4 != p {
			t.Errorf("%s 防護值:檔案 %d,手冊 %d", it.Name, it.Col4, p)
		}
		if it.BasePrice != manualPrice[it.Name] {
			diverged++
		}
	}
	// Cloth 5 兩邊一致,其餘四件分歧 —— 這是已知且已解釋的事實。
	if diverged != 4 {
		t.Errorf("護甲價格與手冊分歧的件數 = %d,預期 4(docs/re/126 §5);"+
			"數字變了表示解析或資料變了,要回去看", diverged)
	}
}

// ---------------------------------------------------------------------------
// TOWNDATA.DAT 的價格倍率 —— 用手冊 p.31 的**遊戲截圖**驗證(docs/re/126)。
// 截圖是遊戲自己畫的畫面,是獨立於資料檔的第三個來源。
// ---------------------------------------------------------------------------

func TestShopPriceMultiplier(t *testing.T) {
	shops, err := ParseShops(read(t, "TOWNDATA.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if len(shops) != 61 {
		t.Fatalf("商店數 %d,規格說 61(docs/re/126)", len(shops))
	}
	if got := len(Towns(shops)); got != 13 {
		t.Errorf("城鎮數 %d,規格說 13", got)
	}

	find := func(town, name string) Shop {
		for _, s := range shops {
			if s.Town == town && s.Name == name {
				return s
			}
		}
		t.Fatalf("找不到商店 %s / %s", town, name)
		return Shop{}
	}

	if m := find("Athe", "The Blacksmith").PriceMult; m != 1.0 {
		t.Errorf("Athe 鐵匠店倍率 %v,應為 1.0(截圖 13 件品項價格與基準價相同)", m)
	}

	// Atlantis 的護甲店是 1.3,但 MBF 表示不出精確的 1.3。
	// ⚠ 那個誤差**就是證據**:截圖四筆價格都剛好比 基準價×1.3 少 1,
	// 因為 INT() 截斷了 909.99999(docs/re/126 §3)。
	ea := find("Atlantis", "Enchanted Armor")
	items, _ := ParseItems(read(t, "ITEMS.DAT"))
	base := map[string]int{}
	for _, it := range items {
		base[it.Name] = it.BasePrice
	}
	for _, c := range []struct {
		item string
		shot int // 手冊 p.31 截圖上的售價
	}{
		{"Chain +1", 909}, {"Scale +1", 1299}, {"Plate +1", 1949}, {"Plate +2", 2599},
	} {
		b, ok := base[c.item]
		if !ok {
			t.Errorf("ITEMS.DAT 沒有 %q", c.item)
			continue
		}
		if got := int(float64(b) * ea.PriceMult); got != c.shot {
			t.Errorf("%s 售價:算出 %d,截圖 %d(基準 %d × %v)",
				c.item, got, c.shot, b, ea.PriceMult)
		}
	}
}

// ---------------------------------------------------------------------------
// MONSTERS.DAT
// ---------------------------------------------------------------------------

func TestMonstersShape(t *testing.T) {
	ms, err := ParseMonsters(read(t, "MONSTERS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 74 {
		t.Fatalf("怪物數 %d,規格說 74(docs/formats/03)", len(ms))
	}
	for _, m := range ms {
		if m.Name == "" {
			t.Errorf("第 %d 隻怪物沒有名稱", m.Index)
		}
		// 欄9 是難度階級 1–10 或 13,**不是等級**(docs/formats/03)。
		if m.Tier != 13 && (m.Tier < 1 || m.Tier > 10) {
			t.Errorf("%s 難度階級 %d 超出 1–10 與 13", m.Name, m.Tier)
		}
	}
}

// MBF 的邊界:0 指數 → 0,而 1.3 解出來必須是那個「不精確的 1.3」。
func TestMBF(t *testing.T) {
	if v := MBF([]byte{0, 0, 0, 0}); v != 0 {
		t.Errorf("指數 0 應解為 0,得 %v", v)
	}
	if v := MBF([]byte{0, 0, 0, 0x81}); v != 1.0 {
		t.Errorf("1.0 解錯:%v", v)
	}
	v := MBF([]byte{0x66, 0x66, 0x26, 0x81})
	if v == 1.3 {
		t.Errorf("MBF 不該解出精確的 1.3 —— 那表示引入了修正,"+
			"而截斷誤差是有觀測後果的(docs/re/126 §3);得 %v", v)
	}
	if v < 1.2999999 || v > 1.3 {
		t.Errorf("1.3 應落在 [1.2999999, 1.3),得 %v", v)
	}
}
