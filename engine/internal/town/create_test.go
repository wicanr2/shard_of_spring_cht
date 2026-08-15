package town

import (
	"shardofspring/internal/combat"
	"testing"

	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// fixedRoller 依序回傳固定值,用完循環 —— 讓創造的測試可重現。
type fixedRoller struct {
	seq []int
	i   int
}

func (r *fixedRoller) Roll(int) int {
	v := r.seq[r.i%len(r.seq)]
	r.i++
	return v
}

// fixedFloat 依序回傳固定的 [0,1) 值 —— 屬性算式吃的是浮點不是骰子
// (docs/re/156)。
type fixedFloat struct {
	seq []float64
	i   int
}

func (r *fixedFloat) Float01() float64 {
	v := r.seq[r.i%len(r.seq)]
	r.i++
	return v
}

func blankRoster() []original.Character {
	return make([]original.Character, original.CharSlots)
}

func TestCreateRefusesForbiddenClass(t *testing.T) {
	cs := blankRoster()
	// 巨魔只能當戰士(手冊 p.49)
	if _, r := Create(cs, rules.Troll, rules.ClassWizard, Rolled{}, "阿魔"); r != CreateBadClass {
		t.Fatalf("巨魔選法師應被擋,得 %v", r)
	}
	if cs[0].Occupied() {
		t.Error("被擋下卻寫進名冊了")
	}
}

// 種族修正要加在**擲出的值**上,而且是手冊 p.49 那一組。
func TestCreateAppliesRaceModifiers(t *testing.T) {
	cs := blankRoster()
	v := Rolled{Speed: 10, Str: 7, Int: 8, End: 10, Skill: 7}
	id, r := Create(cs, rules.Troll, rules.ClassHero, v, "阿魔")
	if r != CreateOK {
		t.Fatalf("應該造得出來,得 %v", r)
	}
	c := cs[id-1]
	// 巨魔:−3 速、+5 力、−5 智、+5 體
	if c.Speed != 7 || c.Str != 12 || c.Int != 3 || c.End != 15 || c.ToHit != 7 {
		t.Errorf("屬性 = 速%d 力%d 智%d 體%d 技%d,應為 7/12/3/15/7",
			c.Speed, c.Str, c.Int, c.End, c.ToHit)
	}
}

// 初始生命 = 體質(手冊 p.13 + 創造畫面 Endurance 5 → H.P. 5)。
func TestCreateInitialHPEqualsEndurance(t *testing.T) {
	cs := blankRoster()
	v := Rolled{Speed: 9, Str: 9, Int: 9, End: 11, Skill: 9}
	id, _ := Create(cs, rules.Human, rules.ClassHero, v, "阿人")
	c := cs[id-1]
	if c.MaxHP != c.End || c.HP != c.End {
		t.Errorf("初始生命應等於體質 %d,得 %d/%d", c.End, c.HP, c.MaxHP)
	}
}

// 新角色沒有隊伍、沒有裝備 —— 裝備欄要寫哨兵 99,不是 0。
// 寫 0 會讓新角色「拿著背包第 0 格」,而那一格是空的,畫面上看不出來。
func TestCreateLeavesEquipmentUnset(t *testing.T) {
	cs := blankRoster()
	id, _ := Create(cs, rules.Human, rules.ClassWizard, Rolled{End: 8}, "阿法")
	c := cs[id-1]
	if c.Weapon != original.NotEquipped || c.Armor != original.NotEquipped {
		t.Errorf("裝備欄應為 %d,得 %d/%d", original.NotEquipped, c.Weapon, c.Armor)
	}
	if _, in := c.InParty(); in {
		t.Error("新角色不該屬於任何隊伍")
	}
}

// 屬性算式:INT(RND × A + RND × A + B),兩次亂數同範圍、只取整一次
// (docs/re/156 §1)。
func TestAttributeRollShape(t *testing.T) {
	for _, c := range []struct {
		a, b float64
		want int
	}{
		{0, 0, AttrOffsetB},                          // 兩次都擲到 0 → 下界就是 B
		{0.999, 0.999, int(2*AttrRangeA) + AttrOffsetB - 1}, // 逼近上界
		{0.5, 0.5, int(AttrRangeA) + AttrOffsetB},    // 兩次半 → A + B
		{0.3, 0.3, int(0.6*AttrRangeA) + AttrOffsetB},
	} {
		r := &fixedFloat{seq: []float64{c.a, c.b}}
		if got := RollAttribute(r); got != c.want {
			t.Errorf("RND=%.3f,%.3f → %d,應為 %d", c.a, c.b, got, c.want)
		}
	}
}

// ⚠ 取整只做**一次**,在總和之後。
//
// 這條測的是 `INT(x+y+B)` 與 `INT(x)+INT(y)+B` 的差別 ——
// 兩者的支撐集幾乎一樣,只有小數部分會相加進位的那些點分岔。
func TestAttributeRollFloorsOnce(t *testing.T) {
	// 0.6×5 = 3.0、0.19×5 = 0.95 → 和 3.95 + 4 = 7.95 → 7
	// 若各自取整:3 + 0 + 4 = 7 —— 相同,換一組會分開的
	// 0.5×5 = 2.5、0.7×5 = 3.5 → 和 6.0 + 4 = 10
	// 各自取整:2 + 3 + 4 = 9   ← 分岔
	r := &fixedFloat{seq: []float64{0.5, 0.7}}
	if got := RollAttribute(r); got != 10 {
		t.Errorf("INT(2.5 + 3.5 + 4) 應為 10,得 %d "+
			"—— 得 9 表示各自取整了(docs/re/156 §1)", got)
	}
}

// 骰出來的值落在實測觀察到的範圍內(docs/re/143 §5:4–12,
// 而 A=5 的支撐集是 4–13)。
func TestRollStaysInPlausibleRange(t *testing.T) {
	rnd := combat.NewRand(20260815)
	lo, hi := 99, -1
	for i := 0; i < 2000; i++ {
		v := RollAttribute(rnd)
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	if lo != AttrOffsetB {
		t.Errorf("下界 %d,應為 B = %d", lo, AttrOffsetB)
	}
	if want := int(2*AttrRangeA) + AttrOffsetB - 1; hi != want {
		t.Errorf("上界 %d,應為 ⌈2A+B⌉−1 = %d", hi, want)
	}
}

func TestCreateGivesRacialSkills(t *testing.T) {
	cs := blankRoster()
	// 地精附贈 Spirit runes = 法師技能表第 5 格
	id, _ := Create(cs, rules.Gnome, rules.ClassWizard, Rolled{End: 8}, "阿地")
	if got := cs[id-1].Skills; got[4] != '1' {
		t.Errorf("地精該有第 5 個技能旗標,得 %q", got)
	}
}

func TestCreateFillsFirstEmptySlot(t *testing.T) {
	cs := blankRoster()
	cs[0].Name = "已存在"
	id, r := Create(cs, rules.Human, rules.ClassHero, Rolled{End: 8}, "阿二")
	if r != CreateOK || id != 2 {
		t.Fatalf("應該填第 2 槽,得 id=%d r=%v", id, r)
	}
	if cs[0].Name != "已存在" {
		t.Error("不該覆蓋已有的角色")
	}
}

func TestRerollOnlyTouchesTheChosenAttribute(t *testing.T) {
	// ⚠ 期望值由 A/B 算出,**不寫死** —— 兩個常數仍是假設(docs/re/156),
	// 寫死會讓改假設時測試假性失敗。
	r := &fixedFloat{seq: []float64{0.5}}
	want := int(AttrRangeA) + AttrOffsetB
	v := Rolled{Speed: 1, Str: 2, Int: 3, End: 4, Skill: 5}
	got := Reroll(v, 3, r)
	if got.Int != want {
		t.Errorf("第 3 項應重擲成 %d,得 %d", want, got.Int)
	}
	if got.Speed != 1 || got.Str != 2 || got.End != 4 || got.Skill != 5 {
		t.Errorf("其餘四項不該動,得 %+v", got)
	}
}

func TestRerollIgnoresOutOfRange(t *testing.T) {
	r := &fixedFloat{seq: []float64{0.5}}
	v := Rolled{Speed: 1, Str: 2, Int: 3, End: 4, Skill: 5}
	if got := Reroll(v, 9, r); got != v {
		t.Errorf("編號超出 1–5 應原樣回傳,得 %+v", got)
	}
}

// R)eorder:順序不只是顯示 —— 戰場佈陣直接用槽序算位置(docs/re/160)。
//
// ⚠ 要同時動 GROUPS 的成員槽與 members 切片。只動一邊的話存檔與畫面對不上,
// 而畫面看起來完全正常 —— 所以這條兩邊都驗。
func TestReorderMovesBothSides(t *testing.T) {
	g := &original.Group{}
	ms := []original.Character{{Name: "甲", ID: 1}, {Name: "乙", ID: 2}, {Name: "丙", ID: 3}}
	for i, c := range ms {
		g.Members[i] = c.ID
	}
	if !Reorder(g, ms, 1, 3) {
		t.Fatal("交換 1↔3 應該成功")
	}
	if ms[0].Name != "丙" || ms[2].Name != "甲" {
		t.Errorf("members 沒換:%s / %s", ms[0].Name, ms[2].Name)
	}
	if g.Members[0] != 3 || g.Members[2] != 1 {
		t.Errorf("成員槽沒換:%d / %d", g.Members[0], g.Members[2])
	}
	for _, c := range [][2]int{{1, 1}, {0, 2}, {1, 9}} {
		if Reorder(g, ms, c[0], c[1]) {
			t.Errorf("(%d,%d) 應該被擋下來", c[0], c[1])
		}
	}
}

// T)rade:搬一格背包,傳出去的若正裝備著要先卸下。
func TestTradeUnequipsWhatItMoves(t *testing.T) {
	from := emptyPackChar()
	to := emptyPackChar()
	from.Pack[2] = 7
	from.Weapon = 2 // 正拿著第 2 格

	if r := Trade(&from, &to, 2); r != TradeOK {
		t.Fatalf("應該傳得過去,得 %v", r)
	}
	if from.Pack[2] != original.NotEquipped {
		t.Errorf("來源那一格應該清空,得 %d", from.Pack[2])
	}
	if from.Weapon != original.NotEquipped {
		t.Error("傳出去的那一格正裝備著,應該一起卸下 —— 否則裝備欄指向空格")
	}
	if to.Pack[0] != 7 {
		t.Errorf("對方的第一個空格應該收到 7,得 %d", to.Pack[0])
	}
	// 空格 / 自己 / 背包滿
	if r := Trade(&from, &to, 5); r != TradeEmptySlot {
		t.Errorf("空格應回 TradeEmptySlot,得 %v", r)
	}
	if r := Trade(&from, &from, 0); r != TradeSamePerson {
		t.Errorf("交給自己應被擋,得 %v", r)
	}
	full := emptyPackChar()
	for i := range full.Pack {
		full.Pack[i] = 3
	}
	src := emptyPackChar()
	src.Pack[0] = 9
	if r := Trade(&src, &full, 0); r != TradeNoRoom {
		t.Errorf("對方背包滿應回 TradeNoRoom,得 %v", r)
	}
}

// emptyPackChar 造一個背包全空的角色 —— **空格的哨兵是 99 不是 0**
// (docs/re/144 §3)。用零值 Character 會讓測試保護錯誤的假設。
func emptyPackChar() original.Character {
	var c original.Character
	for i := range c.Pack {
		c.Pack[i] = original.NotEquipped
	}
	c.Weapon, c.Armor = original.NotEquipped, original.NotEquipped
	return c
}
