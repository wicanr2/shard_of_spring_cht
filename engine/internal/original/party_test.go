package original

import (
	"bytes"
	"testing"
)

// docs/spec/06 §7 驗收 1:五個角色的欄位。
// 數值取自原版 CHARS.DAT,與 docs/spec/06 §5 的表一致。
func TestParseCharsShipped(t *testing.T) {
	chars, err := ParseChars(read(t, "CHARS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name             string
		race, class      byte
		spd, str, in, en int
		hp, sp, lv       int
	}{
		{"Segrono", 'H', '1', 11, 10, 9, 9, 9, 0, 1},
		{"Hard Axe", 'D', '1', 11, 13, 7, 12, 12, 0, 1},
		{"Grod", 'T', '1', 8, 16, 5, 15, 15, 0, 1},
		{"Fire Hawk", 'E', '2', 13, 3, 15, 7, 7, 15, 1},
		{"Richtatha", 'G', '2', 13, 10, 13, 5, 5, 13, 1},
	}
	for i, w := range want {
		c := chars[i]
		if c.Name != w.name || c.Race != w.race || c.Class != w.class {
			t.Errorf("第 %d 個:%q/%c/%c,應為 %q/%c/%c",
				i+1, c.Name, c.Race, c.Class, w.name, w.race, w.class)
		}
		if c.Speed != w.spd || c.Str != w.str || c.Int != w.in || c.End != w.en {
			t.Errorf("%s 屬性 %d/%d/%d/%d,應為 %d/%d/%d/%d",
				c.Name, c.Speed, c.Str, c.Int, c.End, w.spd, w.str, w.in, w.en)
		}
		if c.HP != w.hp || c.MaxHP != w.hp || c.SP != w.sp || c.MaxSP != w.sp {
			t.Errorf("%s HP %d/%d SP %d/%d,應為 %d/%d %d/%d",
				c.Name, c.HP, c.MaxHP, c.SP, c.MaxSP, w.hp, w.hp, w.sp, w.sp)
		}
		if c.Level != w.lv {
			t.Errorf("%s 等級 %d,應為 %d", c.Name, c.Level, w.lv)
		}
		if c.ID != i+1 {
			t.Errorf("%s 編號 %d,應為 %d", c.Name, c.ID, i+1)
		}
		if c.Weapon != NotEquipped || c.Armor != NotEquipped {
			t.Errorf("%s 裝備 %d/%d,出貨資料應為 99/99", c.Name, c.Weapon, c.Armor)
		}
	}
}

// 驗收 2 + 3:20 個空槽不產生假角色;五人全在第 5 隊。
//
// ⚠ 空槽的整數欄位讀出來是 8224(= 0x2020 = 兩個空白),**不會拋錯**。
// 這一條測的就是「靠名稱判定」有沒有真的擋住那些值。
func TestCharSlotsAndParty(t *testing.T) {
	chars, err := ParseChars(read(t, "CHARS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	occupied := 0
	for i, c := range chars {
		if c.Occupied() {
			occupied++
			continue
		}
		if c.Party != NoParty {
			t.Errorf("第 %d 槽未佔用,位移 1 卻是 %q", i+1, c.Party)
		}
		if _, ok := c.InParty(); ok {
			t.Errorf("第 %d 槽未佔用,卻被判成有隊伍", i+1)
		}
	}
	if occupied != 5 {
		t.Errorf("佔用 %d 槽,原版出貨是 5", occupied)
	}
	p5 := Party(chars, 5)
	if len(p5) != 5 {
		t.Fatalf("第 5 隊 %d 人,手冊說出貨的 PARTY #5 有 5 人(docs/re/133)", len(p5))
	}
	for n := 1; n <= 4; n++ {
		if got := len(Party(chars, n)); got != 0 {
			t.Errorf("第 %d 隊有 %d 人,出貨資料應為 0", n, got)
		}
	}
}

// 驗收 4:第 1–4 筆未初始化、第 5 筆是真實存檔(docs/re/135)。
//
// ⚠ 這條原本寫成「五筆都空白」——那是 formats/02 的一句話,**而它是錯的**。
// 寫成測試之後第一次執行就爆,而那筆存檔在專案裡躺了很久沒人看。
func TestGroupsShippedRecords(t *testing.T) {
	gs, err := ParseGroups(read(t, "GROUPS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if !gs[i].Blank() {
			t.Errorf("第 %d 隊應為空白", i+1)
		}
		// 不判 Blank 的話座標就是 8224 —— 把陷阱本身也鎖住
		if gs[i].WorldX != 8224 || gs[i].WorldY != 8224 {
			t.Errorf("第 %d 隊空白記錄解出座標 (%d,%d),預期 8224;"+
				"若這裡變了,Blank() 的必要性要重新檢查",
				i+1, gs[i].WorldX, gs[i].WorldY)
		}
	}

	g := gs[4]
	if g.Blank() {
		t.Fatal("第 5 隊應該是一份真實存檔(手冊的 PARTY #5)")
	}
	// docs/re/135 §2 的逐欄值
	if got := g.MemberIDs(); len(got) != 5 {
		t.Errorf("第 5 隊 %d 人(%v),應為 5", len(got), got)
	} else {
		for i, id := range got {
			if id != i+1 {
				t.Errorf("成員 %d 是角色 %d,應為 %d", i+1, id, i+1)
			}
		}
	}
	for _, c := range []struct {
		name      string
		got, want int
	}{
		{"世界 x", g.WorldX, 8}, {"世界 y", g.WorldY, 8},
		{"朝向", g.Facing, 3}, {"補給", g.Provisions, 20},
		{"遭遇倒數", g.Encounter, 54},
		{"月", g.Month, 1}, {"日", g.Day, 1}, {"時", g.Hour, 4}, {"時以下", g.Sub, 2},
		{"有光能見度", g.VisLit, 3}, {"無光能見度", g.VisDark, 2},
		{"光源選擇", g.LightPick, NoLight},
	} {
		if c.got != c.want {
			t.Errorf("第 5 隊的%s是 %d,docs/re/135 §2 記 %d", c.name, c.got, c.want)
		}
	}
	if g.Gold != 75 {
		t.Errorf("第 5 隊金幣 %v,docs/re/135 §2 記 75", g.Gold)
	}
	// 空的成員槽用 99,不是 0 也不是空白
	for k := 5; k < MemberSlots; k++ {
		if g.Members[k] != 99 {
			t.Errorf("第 %d 個成員槽是 %d,出貨存檔應為 99", k+1, g.Members[k])
		}
	}
}

// 出貨存檔就站在城鎮 Green Hamlet 的**西邊一格**,朝向卻是南(3)——
// 也就是玩家開局要先往東轉一次才走得進去。這是 DOSBox 實跑裁決的
// (docs/re/141:往東走一格進得去、往南走兩格進不去)。
//
// ⚠ 這條測試的第一版寫「城鎮在正南方」,而它**當時是通過的** ——
// 因為地圖的兩軸接反了,而轉置過的地圖照樣自洽。
// **一個從錯誤前提寫出來的測試會保護那個錯誤**,不會揭發它。
func TestShippedSaveTownIsOneStepEast(t *testing.T) {
	gs, err := ParseGroups(read(t, "GROUPS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	cells, err := DecodeWorldMap(read(t, "WRLDMAP.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	g := gs[4]
	// 索引 = x*WorldH + y,往東一格 = +103(docs/re/141)
	at := func(x, y int) int { return int(cells[x*WorldH+y]) }
	if v := at(g.WorldX, g.WorldY); v != 4 {
		t.Errorf("隊伍所在格 (%d,%d) 的地形是 %d,應為 4(可站人)",
			g.WorldX, g.WorldY, v)
	}
	if v := at(g.WorldX+1, g.WorldY); v != 30 {
		t.Errorf("東邊一格 (%d,%d) 的地形是 %d,應為 30(Green Hamlet)",
			g.WorldX+1, g.WorldY, v)
	}
	// 反面:南邊那一格**不是**城鎮 —— 這一格正是先前接反時被誤認的那一格。
	if v := at(g.WorldX, g.WorldY+1); v == 30 {
		t.Errorf("南邊一格不該是城鎮,得 %d", v)
	}
	if g.Facing != 3 {
		t.Errorf("出貨存檔朝向應為 3(南),得 %d", g.Facing)
	}
}

// 驗收 5:金幣走 MBF 單精度。手冊截圖是 63447,超過 int16 上限。
//
// ⚠ 用 int16 讀會得到 −2089,**而那看起來像個合理的錯誤**(負的金幣),
// 不會有任何例外。所以這條要正面測數值,不能只測「沒有 panic」。
func TestGoldIsSinglePrecision(t *testing.T) {
	for _, v := range []float64{0, 1, 5, 63447, 34531, 26922, 999999, 1} {
		b := PutMBF(v)
		if got := MBF(b); got != v {
			t.Errorf("MBF 往返 %v → %v(位元組 %x)", v, got, b)
		}
	}
	// 走完整條記錄路徑,不只是 MBF 函式本身
	raw := bytes.Repeat([]byte{' '}, GroupRecLen*GroupSlots)
	gs, err := ParseGroups(raw)
	if err != nil {
		t.Fatal(err)
	}
	g := gs[0]
	g.Gold = 63447
	gs2, err := ParseGroups(bytes.Repeat(g.Bytes(), GroupSlots))
	if err != nil {
		t.Fatal(err)
	}
	if gs2[0].Gold != 63447 {
		t.Errorf("金幣經記錄往返後是 %v,應為 63447", gs2[0].Gold)
	}
	// 同一段位元組若當成 2-byte 整數讀,會得到別的東西 ——
	// 這正是 docs/spec/06 §3 警告的那個「看起來合理的錯誤」。
	if v := u16(g.Bytes(), offGold); v == 63447 {
		t.Error("整數讀法竟然也得到 63447 —— 本測試的前提要重查")
	}
}

// 驗收 6:存檔往返逐位元組相同,含未解的位移 1–18。
//
// 造一筆**每個位元組都不同**的假記錄 ——
// 拿出貨檔來測會漏掉「未解欄位有沒有被保留」(它們大多是 0x20,寫錯也看不出來)。
func TestGroupRoundTrip(t *testing.T) {
	raw := make([]byte, GroupRecLen)
	for i := range raw {
		raw[i] = byte(i + 1) // 1..90,沒有 0x20,也沒有重複
	}
	gs, err := ParseGroups(bytes.Repeat(raw, GroupSlots))
	if err != nil {
		t.Fatal(err)
	}
	got := gs[0].Bytes()
	if !bytes.Equal(got, raw) {
		for i := range raw {
			if got[i] != raw[i] {
				t.Fatalf("往返在位移 %d 就不同了:%#02x → %#02x(位移 1–18 與未解欄位必須原樣保留)",
					i+1, raw[i], got[i])
			}
		}
	}
}

// 狀態 0 顯示空白,不是「正常」(docs/spec/06 §5)。
func TestStatusNameOKIsBlank(t *testing.T) {
	if s := (Character{Status: 0}).StatusName(); s != "" {
		t.Errorf("狀態 0 顯示 %q,應為空字串 —— 五人都正常時整欄要空著", s)
	}
	if s := (Character{Status: 5}).StatusName(); s != "死亡" {
		t.Errorf("狀態 5 顯示 %q,應為「死亡」", s)
	}
	if s := (Character{Status: 99}).StatusName(); s != "?" {
		t.Errorf("超出範圍的狀態顯示 %q,應為 %q —— 不可以靜默當成正常", s, "?")
	}
}

// 角色記錄的往返也要逐位元組相同(docs/spec/06 §3 的同一條原則)。
//
// ⚠ 名稱是 10 bytes 但原版只讓玩家輸入 9 個 —— 第 10 格的用途未解,
// 所以往返測試用**出貨的真實記錄**,不是造出來的。
func TestCharRoundTrip(t *testing.T) {
	d := read(t, "CHARS.DAT")
	chars, err := ParseChars(d)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range chars {
		got := c.Bytes()
		want := d[i*CharRecLen : (i+1)*CharRecLen]
		if !bytes.Equal(got, want) {
			for k := range want {
				if got[k] != want[k] {
					t.Fatalf("第 %d 筆在位移 %d 不同:%#02x → %#02x",
						i+1, k+1, want[k], got[k])
				}
			}
		}
	}
}

// 驗收 11(docs/spec/11 §9):寫回時**不能動到未解的尾端位移 86–94**。
//
// 經驗值的落點未解(docs/re/140 §9),而 86–94 是僅存的候選區。
// 這個測試擋的是「順手找個空位存經驗值」——
// 那種改動不會報錯,原版照樣開得起來,只是某個欄位悄悄變了值。
func TestBytesLeavesUnresolvedTailUntouched(t *testing.T) {
	raw := make([]byte, CharRecLen)
	for i := range raw {
		raw[i] = 0x20
	}
	copy(raw[1:], "測試      ")
	// 尾端擺一組可辨識的值
	tail := []byte{0x20, 0x20, 0x20, 0x01, 0x00, 0x00, 0x00, 0x40, 0x00}
	copy(raw[85:], tail)

	c := Character{Name: "測試", Raw: raw}
	out := c.Bytes()
	if got := out[85:94]; string(got) != string(tail) {
		t.Errorf("位移 86–94 被改動了:%X → %X", tail, got)
	}
}
