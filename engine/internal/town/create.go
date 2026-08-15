package town

import (
	"math"

	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// 角色創造。流程出自 `CHARUTIL.EXE` 的實跑(docs/re/143):
//
//	C)reate → Choose Race: H)uman D)warf T)roll E)lf G)nome
//	        → 五項屬性一次擲出,種族修正**另外一欄**顯示
//	        → `Enter #'s to Be rerolled Or ESC when Done`(輸入 1–5 重擲該項)
//	        → 職業 → 名稱
//
// ⚠ 畫面把**擲出的值**與**種族修正**分開顯示,所以修正不是骰進去的,
// 是事後加的 —— 這也是為什麼手冊 p.49 那張表叫「修正」而不是「加值範圍」。

// Roller 是擲一項屬性用的亂數來源。傳進來而不是內建,
// 是為了讓測試能用固定序列(docs/spec/07 的可重現性要求)。
type Roller interface {
	Roll(faces int) int // 回 1…faces
}

// 擲屬性的算式。**形狀是讀出來的**(docs/re/156):
//
//	屬性 = INT(RND × A + RND × A + B)
//
// 兩次亂數用**同一個範圍**(原版第二次乘法沒有重載 di),
// 而 `INT()` **只在總和之後做一次** —— 那讓它是個對稱三角分佈。
//
// ⚠ `INT(x + y + B)` 與 `INT(x) + INT(y) + B` 的支撐集幾乎一樣、分佈不同。
// 先前用 `4d3`(四顆骰的和)支撐集也吻合,但峰度明顯更高 ——
// **支撐集相同而分佈不同,玩起來只覺得「極端值比較少」**,沒有人會發現。
//
// A 與 B 的**值**仍未解(執行期變數)。15 個實測樣本(4…12、平均 8.4)
// 把它們約束到 B = 4、A ≈ 4.9;取整數 A = 5 → 支撐集 4…13、平均 8.5。
// 這兩個數字是**具名的假設**,形狀不是。
const (
	AttrRangeA = 5.0 // ds:6C2A
	AttrOffsetB = 4  // ds:6C2E
)

// AttrRollAssumption 是給畫面顯示用的說明。
const AttrRollAssumption = "⚠ 屬性算式的形狀已解(兩個亂數相加再取整一次)," +
	"但兩個常數未解 —— 本引擎用 A=5、B=4(docs/re/156)"

// SkillPoints 回傳創造時可用的技能點數。
//
// **= 智能**(手冊 p.12 + 實跑:智能 3 的巨魔點掉成本 2 的技能後剩 1,
// docs/re/143 §3)。⚠ 只有一個樣本,分不開「= 智能」與「= 智能 + 0」。
func SkillPoints(intellect int) int { return intellect }

// SkillCost 回傳一項技能的點數成本。
//
// ⚠ **只觀察到戰士表的前五項**(劍 2 / 斧 2 / 釘錘 1 / 空手 2 / 夜視 2)。
// 其餘五項與法師的十項**未解** —— 回 DefaultSkillCost 並在畫面上標出來,
// ⛔ 不要把「沒觀察到」填成 1 或 2 裝作已知。
func SkillCost(class rules.Class, n int) (int, bool) {
	if class == rules.ClassHero {
		if c, ok := heroSkillCost[n]; ok {
			return c, true
		}
	}
	return DefaultSkillCost, false
}

// DefaultSkillCost 是未觀察到成本時的佔位值。
const DefaultSkillCost = 2

var heroSkillCost = map[int]int{1: 2, 2: 2, 3: 1, 4: 2, 5: 2}

// FloatRoller 是能給 [0,1) 浮點的亂數來源。原版的屬性算式要的是浮點,
// 不是骰子(docs/re/156)。
type FloatRoller interface{ Float01() float64 }

// RollAttribute 擲一項屬性(**不含**種族修正)。
//
//	INT(RND × A + RND × A + B)
func RollAttribute(r FloatRoller) int {
	return int(math.Floor(r.Float01()*AttrRangeA + r.Float01()*AttrRangeA + AttrOffsetB))
}

// Rolled 是一次擲出的五項屬性,**種族修正尚未加上去**。
// 欄位順序與創造畫面相同(Speed / Strength / Intellect / Endurance / Skill)。
type Rolled struct {
	Speed, Str, Int, End, Skill int
}

// RollAll 擲出五項。
func RollAll(r FloatRoller) Rolled {
	return Rolled{
		Speed: RollAttribute(r), Str: RollAttribute(r), Int: RollAttribute(r),
		End: RollAttribute(r), Skill: RollAttribute(r),
	}
}

// Reroll 重擲第 n 項(1–5,對應畫面上的編號)。超出範圍時原樣回傳。
func Reroll(v Rolled, n int, r FloatRoller) Rolled {
	switch n {
	case 1:
		v.Speed = RollAttribute(r)
	case 2:
		v.Str = RollAttribute(r)
	case 3:
		v.Int = RollAttribute(r)
	case 4:
		v.End = RollAttribute(r)
	case 5:
		v.Skill = RollAttribute(r)
	}
	return v
}

// CreateResult 說明一次創造為什麼失敗。
type CreateResult int

const (
	CreateOK CreateResult = iota
	CreateBadClass
	CreateNoSlot
	CreateBadName
)

func (r CreateResult) String() string {
	switch r {
	case CreateBadClass:
		return "這個種族不能選這個職業"
	case CreateNoSlot:
		return "名冊已滿"
	case CreateBadName:
		return "名稱不能空白"
	}
	return ""
}

// Create 依種族、職業、擲出的屬性與名稱造一個角色,寫進名冊第一個空槽。
//
// ⚠ **初始生命值 = 體能**(手冊 p.13「它也是你一開始的生命點數」;
// 創造畫面上 `Endurance 5` 對應 `H.P.: 5`,兩個來源一致)。
// ⚠ 初始法力值**未解** —— 法師的 `S.P.` 在創造畫面上是空的,
// 手冊只說「智能決定巫師一開始的法力點數」而沒給算式。這裡填 0 並標出來。
func Create(chars []original.Character, race rules.Race, class rules.Class,
	v Rolled, name string) (int, CreateResult) {

	if !rules.AllowsClass(race, class) {
		return 0, CreateBadClass
	}
	if len([]rune(name)) == 0 || len([]rune(name)) > NameMaxRunes {
		return 0, CreateBadName
	}
	slot := -1
	for i := range chars {
		if !chars[i].Occupied() {
			slot = i
			break
		}
	}
	if slot < 0 {
		return 0, CreateNoSlot
	}
	spd, str, intel, end, skill := rules.ApplyRaceModifiers(
		race, v.Speed, v.Str, v.Int, v.End, v.Skill)

	c := original.Character{
		Party: original.NoParty, // '0' = 有角色但無隊伍(docs/re/144 §5)
		Name:  name,
		ID:    slot + 1,
		Race:  byte(race), Class: byte(class),
		Speed: spd, Str: str, Int: intel, End: end, ToHit: skill,
		MaxHP: end, HP: end, // 初始生命 = 體能
		Weapon: original.NotEquipped, Armor: original.NotEquipped,
		Level:    1,
		Skills:   raceSkills(race, class),
		Flags2:   "0000000000",
		SkillPts: intel, // 技能點數 = 智能,一點都還沒花(docs/re/144 §4)
	}
	// 背包十格全空。⚠ 哨兵是 99 不是 0 —— 填 0 會讓每一格都看起來裝著第 0 號道具。
	for i := range c.Pack {
		c.Pack[i] = original.NotEquipped
	}
	chars[slot] = c
	return slot + 1, CreateOK
}

// InitialSPUnresolved 給畫面用。
const InitialSPUnresolved = "⚠ 法師的初始法力值未解 —— 這裡填 0"

// raceSkills 回傳十個技能旗標,把種族附贈的技能打開。
//
// ⚠ 同一格在戰士表與法師表是**不同的技能**(docs/formats/01),
// 所以旗標要配上職業才有意義 —— 手冊 p.49 給的附贈技能名已經在
// rules.Races 裡換算成該職業表的編號。
func raceSkills(race rules.Race, class rules.Class) string {
	flags := []byte("0000000000")
	for _, n := range rules.Races[race].Skills {
		if n >= 1 && n <= len(flags) {
			flags[n-1] = '1'
		}
	}
	return string(flags)
}
