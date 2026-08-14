package town

import (
	"os"
	"path/filepath"
	"testing"

	"shardofspring/internal/original"
)

// docs/spec/11 §8 驗收 1:售價 = INT(基準價 × 倍率),**先乘再取整**。
//
// ⚠ 倍率是 MBF 單精度,1.1 存進去是 1.100000023841858(docs/re/126 §3)。
// 先取整再乘會把那個截斷誤差抹掉,而抹掉之後價格看起來還是合理的 ——
// 所以這條要用真的 MBF 值測,不能用 1.1。
func TestPriceMultipliesBeforeTruncating(t *testing.T) {
	const mbf11 = 1.100000023841858
	for _, c := range []struct {
		base int
		mult float64
		want int
	}{
		{5, 1, 5},
		{10, mbf11, 11},
		{200, mbf11, 220},
		{15, mbf11, 16}, // 16.5 → 16
		{75, mbf11, 82}, // 82.5 → 82
	} {
		if got := Price(c.base, c.mult); got != c.want {
			t.Errorf("基準 %d × %v = %d,應為 %d", c.base, c.mult, got, c.want)
		}
	}
	// 倍率 0 = 資料缺漏,不要讓東西免費
	if Price(100, 0) != 100 {
		t.Error("倍率 0 應退回 1 倍,不是免費")
	}
}

// 驗收 2 + 3:買入扣金幣、進第一個空格;背包滿與金幣不足都要擋。
func TestBuy(t *testing.T) {
	gold := 100.0
	c := original.Character{}
	c.Pack[0] = 7 // 第 0 格已有東西

	if r := Buy(&gold, &c, 42, 30); r != BuyOK {
		t.Fatalf("買得起卻回 %v", r)
	}
	if c.Pack[1] != 42 {
		t.Errorf("東西進了第 %v 格,應為第 1 格(第一個空的)", c.Pack)
	}
	if gold != 70 {
		t.Errorf("金幣 %v,應為 70", gold)
	}

	if r := Buy(&gold, &c, 43, 1000); r != BuyNoGold {
		t.Errorf("買不起卻回 %v", r)
	}
	if gold != 70 {
		t.Error("買不起時不該扣款")
	}

	// 背包塞滿
	for i := range c.Pack {
		c.Pack[i] = 1
	}
	if r := Buy(&gold, &c, 44, 1); r != BuyPackFull {
		t.Errorf("背包滿卻回 %v", r)
	}
}

// 金幣不會變負 —— 即使價格剛好等於餘額。
func TestGoldNeverNegative(t *testing.T) {
	gold := 30.0
	c := original.Character{}
	Buy(&gold, &c, 1, 30)
	if gold != 0 {
		t.Errorf("金幣 %v,應為 0", gold)
	}
	if gold < 0 {
		t.Error("金幣不可以變負")
	}
}

// 驗收 4:裝備寫的是**背包格號**,卸下寫 99。
func TestEquipStoresPackSlot(t *testing.T) {
	c := original.Character{}
	c.Pack[3] = 55 // 第 3 格放著物品編號 55

	if !Equip(&c, 3, false) {
		t.Fatal("裝備第 3 格應成功")
	}
	if c.Weapon != 3 {
		t.Errorf("武器欄是 %d,應為**格號** 3(不是物品編號 55)", c.Weapon)
	}
	if !Equip(&c, original.NotEquipped, true) {
		t.Fatal("卸下應成功")
	}
	if c.Armor != original.NotEquipped {
		t.Errorf("卸下後防具欄是 %d,應為 %d", c.Armor, original.NotEquipped)
	}
	if Equip(&c, PackSlots, false) {
		t.Error("超出背包格數應被擋")
	}
}

// 驗收 5:組隊同時寫兩邊。
func TestJoinPartyWritesBothSides(t *testing.T) {
	g := &original.Group{}
	for i := range g.Members {
		g.Members[i] = 99
	}
	c := original.Character{ID: 7, Name: "測試"}

	if !JoinParty(&c, g, 3) {
		t.Fatal("組隊應成功")
	}
	if c.Party != '3' {
		t.Errorf("角色的隊伍欄是 %q,應為 '3'", string(c.Party))
	}
	if ids := g.MemberIDs(); len(ids) != 1 || ids[0] != 7 {
		t.Errorf("隊伍成員 %v,應為 [7]", ids)
	}

	// 人數上限 5
	for id := 1; id <= 5; id++ {
		JoinParty(&original.Character{ID: id}, g, 3)
	}
	if n := len(g.MemberIDs()); n > original.PartySlots {
		t.Errorf("隊伍有 %d 人,上限是 %d", n, original.PartySlots)
	}

	LeaveParty(&c, g)
	if c.Party != original.NoParty {
		t.Errorf("離隊後角色的隊伍欄是 %q", string(c.Party))
	}
	for _, id := range g.MemberIDs() {
		if id == 7 {
			t.Error("離隊後成員槽裡還有 7")
		}
	}
}

// 改名截到 9 個字(CHARUTIL 的 '9 char max');刪除要清名稱,
// 因為佔用判定看的是名稱(docs/spec/06 §2)。
func TestRenameAndDelete(t *testing.T) {
	c := original.Character{Name: "舊名"}
	Rename(&c, "一二三四五六七八九十")
	if len([]rune(c.Name)) != NameMaxRunes {
		t.Errorf("改名後 %q 有 %d 字,上限 %d", c.Name, len([]rune(c.Name)), NameMaxRunes)
	}
	Delete(&c)
	if c.Occupied() {
		t.Error("刪除後應判成未佔用 —— 名稱要清掉")
	}
	if c.Party != original.NoParty {
		t.Error("刪除後隊伍欄應為 '*'")
	}
}

// 出貨五人的屬性範圍 —— 給創造畫面當暫用依據(規則本身未解)。
func TestShippedAttrRange(t *testing.T) {
	var d []byte
	var err error
	for _, dir := range []string{"/game/sharspri", "../../../game/sharspri"} {
		if d, err = os.ReadFile(filepath.Join(dir, "CHARS.DAT")); err == nil {
			break
		}
	}
	if err != nil {
		t.Skip(err)
	}
	chars, err := original.ParseChars(d)
	if err != nil {
		t.Fatal(err)
	}
	lo, hi := ShippedAttrRange(chars)
	if lo < 1 || hi > 20 || lo >= hi {
		t.Errorf("屬性範圍 %d–%d 看起來不對", lo, hi)
	}
	t.Logf("出貨五人的屬性範圍:%d–%d(3d6 是 3–18 —— 相容但不等價,"+
		"所以不能寫成 3d6)", lo, hi)
}
