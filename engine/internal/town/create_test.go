package town

import (
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

// 骰出來的值一定落在觀察到的範圍內(docs/re/143 §5:4–12)。
// 這條把「骰法的假設」與「觀察到的支撐集」綁在一起 ——
// 換一組骰子時如果跑出範圍外的值,這裡會擋下來。
func TestRollStaysInObservedRange(t *testing.T) {
	for _, face := range []int{1, AttrFaces} {
		r := &fixedRoller{seq: []int{face}}
		v := RollAttribute(r)
		if v < 4 || v > 12 {
			t.Errorf("擲出 %d,超出實跑觀察到的 4–12", v)
		}
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
	// ⚠ 期望值由 AttrDice/AttrFaces 算出,**不寫死** ——
	// 骰法是未解的假設(docs/re/143 §5),寫死會讓改假設時測試假性失敗。
	r := &fixedRoller{seq: []int{AttrFaces}}
	want := AttrDice * AttrFaces
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
	r := &fixedRoller{seq: []int{AttrFaces}}
	v := Rolled{Speed: 1, Str: 2, Int: 3, End: 4, Skill: 5}
	if got := Reroll(v, 9, r); got != v {
		t.Errorf("編號超出 1–5 應原樣回傳,得 %+v", got)
	}
}
