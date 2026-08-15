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
		t.Fatalf("巨魔選巫師應被擋,得 %v", r)
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
		{0, 0, AttrOffsetB}, // 兩次都擲到 0 → 下界就是 B
		{0.999, 0.999, int(2*AttrRangeA) + AttrOffsetB - 1}, // 逼近上界
		{0.5, 0.5, int(AttrRangeA) + AttrOffsetB},           // 兩次半 → A + B
		{0.3, 0.3, 5}, // 1.8 + 1.8 + 2 = 5.6 → 取整一次 → 5
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
	// 0.25×6 = 1.5、0.75×6 = 4.5 → 和 6.0 + 2 = 8
	// 各自取整:1 + 4 + 2 = 7            ← 分岔
	r := &fixedFloat{seq: []float64{0.25, 0.75}}
	if got := RollAttribute(r); got != 8 {
		t.Errorf("INT(1.5 + 4.5 + 2) 應為 8,得 %d "+
			"—— 得 7 表示各自取整了(docs/re/156 §1)", got)
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
	// 地精附贈 Spirit runes = 巫師技能表第 5 格
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

// 重擲是**四捨五入**,首擲是**截尾**(docs/re/185 §2.1)。
//
// 0.3×6 = 1.8,兩次是 1.8+1.8+2 = 5.6:
//
//	首擲(floor)= 5、重擲(round)= 6
//
// 這組值兩者不同,分得出 RerollAttribute 是不是真的換了取整方式。
func TestRerollAttributeRoundsNotFloors(t *testing.T) {
	seq := []float64{0.3, 0.3}
	if got := RollAttribute(&fixedFloat{seq: seq}); got != 5 {
		t.Errorf("首擲 INT(1.8+1.8+2)應為 5(截尾),得 %d", got)
	}
	if got := RerollAttribute(&fixedFloat{seq: seq}); got != 6 {
		t.Errorf("重擲 round(1.8+1.8+2)應為 6(四捨五入),得 %d", got)
	}
}

// ⭐ **只有重擲擲得出上界 14**;首擲的上界是 13(docs/re/185 §2 表列 #1)。
//
// 逼近上限:兩次都取 0.999999 → x = 5.999994×2 + 2 = 13.999988。
// 截尾停在 13,四捨五入進到 14 —— 這條測試釘住「重擲多一格」這件事本身,
// 不是釘住某個中間值。
func TestRerollAttributeReachesFourteen(t *testing.T) {
	seq := []float64{0.999999, 0.999999}
	if got := RollAttribute(&fixedFloat{seq: seq}); got != 13 {
		t.Errorf("首擲逼近上界應為 13,得 %d —— 首擲不該擲得出 14", got)
	}
	if got := RerollAttribute(&fixedFloat{seq: seq}); got != 14 {
		t.Errorf("重擲逼近上界應為 14,得 %d", got)
	}
}

// Reroll() 呼叫的必須是 RerollAttribute(四捨五入),不是 RollAttribute。
// 用同一組 0.3/0.3 值:選 RollAttribute 會得到 5,選對了是 6。
func TestRerollUsesRoundingNotFloor(t *testing.T) {
	r := &fixedFloat{seq: []float64{0.3, 0.3}}
	v := Rolled{Speed: 1, Str: 2, Int: 3, End: 4, Skill: 5}
	got := Reroll(v, 3, r)
	if got.Int != 6 {
		t.Errorf("重擲第 3 項應為 6(四捨五入),得 %d —— 若得 5 表示"+
			"Reroll 還在呼叫 RollAttribute(截尾)", got.Int)
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

// 技能成本:20 項全部有值,而且**戰士表的前五項**與實跑觀察到的一致
// (docs/re/178 §1 —— 那五項是這張表的正對照)。
func TestSkillCostsMatchTitlesDat(t *testing.T) {
	hero := [10]int{2, 2, 1, 2, 2, 2, 4, 3, 2, 2}
	wiz := [10]int{5, 5, 5, 5, 5, 2, 2, 2, 2, 3}
	for i := 1; i <= 10; i++ {
		if c, ok := SkillCost(rules.ClassHero, i); !ok || c != hero[i-1] {
			t.Errorf("戰士技能 %d:得 (%d,%v),應為 %d", i, c, ok, hero[i-1])
		}
		if c, ok := SkillCost(rules.ClassWizard, i); !ok || c != wiz[i-1] {
			t.Errorf("巫師技能 %d:得 (%d,%v),應為 %d", i, c, ok, wiz[i-1])
		}
	}
	// 界外要說「不知道」,不要回一個看起來合理的預設值
	if _, ok := SkillCost(rules.ClassHero, 0); ok {
		t.Error("技能 0 不存在,應回 ok=false")
	}
	if _, ok := SkillCost(rules.ClassHero, 11); ok {
		t.Error("技能 11 不存在,應回 ok=false")
	}
}

// 巫師的初始法力 = 智能,戰士 0(手冊 p.12,docs/re/178 §3)。
func TestInitialSPIsIntellect(t *testing.T) {
	cs := blankRoster()
	// 地精是巫師;智能會被種族修正 +5,所以拿建好的角色自己的 Int 比
	id, r := Create(cs, rules.Gnome, rules.ClassWizard, Rolled{Int: 6, End: 8}, "阿地")
	if r != CreateOK {
		t.Fatalf("創造失敗:%v", r)
	}
	w := cs[id-1]
	if w.SP != w.Int || w.MaxSP != w.Int {
		t.Errorf("巫師初始法力應 = 智能 %d,得 SP=%d MaxSP=%d", w.Int, w.SP, w.MaxSP)
	}
	id2, _ := Create(cs, rules.Human, rules.ClassHero, Rolled{Int: 6, End: 8}, "阿人")
	if h := cs[id2-1]; h.SP != 0 || h.MaxSP != 0 {
		t.Errorf("戰士不該有法力,得 SP=%d MaxSP=%d", h.SP, h.MaxSP)
	}
}
