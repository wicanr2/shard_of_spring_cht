package town

import (
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

// 擲屬性的骰法。**未解** —— 沒有讀到 `CHARUTIL.EXE` 那一段程式碼。
//
// docs/re/143 §5:實跑收集到 15 個擲出值,落在 **4–12**、平均 8.4。
// ⛔ **3d6 已被這批樣本排除**(3d6 有 26% 超過 12,15 個全部 ≤12 的機率約 1.2%)。
// 這裡用 **4d3**(範圍 4–12、平均 8),理由只有「支撐集與觀察到的上下界吻合」——
// 是**具名的假設**,不是規則。
const (
	AttrDice  = 4
	AttrFaces = 3
)

// AttrRollAssumption 是給畫面顯示用的說明。
const AttrRollAssumption = "⚠ 屬性的骰法未解(docs/re/143 §5)—— 本引擎用 4d3,是假設不是規則"

// SkillPoints 回傳創造時可用的技能點數。
//
// **= 智力**(手冊 p.12 + 實跑:智力 3 的巨魔點掉成本 2 的技能後剩 1,
// docs/re/143 §3)。⚠ 只有一個樣本,分不開「= 智力」與「= 智力 + 0」。
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

// RollAttribute 擲一項屬性(**不含**種族修正)。
func RollAttribute(r Roller) int {
	sum := 0
	for i := 0; i < AttrDice; i++ {
		sum += r.Roll(AttrFaces)
	}
	return sum
}

// Rolled 是一次擲出的五項屬性,**種族修正尚未加上去**。
// 欄位順序與創造畫面相同(Speed / Strength / Intellect / Endurance / Skill)。
type Rolled struct {
	Speed, Str, Int, End, Skill int
}

// RollAll 擲出五項。
func RollAll(r Roller) Rolled {
	return Rolled{
		Speed: RollAttribute(r), Str: RollAttribute(r), Int: RollAttribute(r),
		End: RollAttribute(r), Skill: RollAttribute(r),
	}
}

// Reroll 重擲第 n 項(1–5,對應畫面上的編號)。超出範圍時原樣回傳。
func Reroll(v Rolled, n int, r Roller) Rolled {
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
// ⚠ **初始生命值 = 體質**(手冊 p.13「它也是你一開始的生命點數」;
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
		Party: original.NoParty,
		Name:  name,
		ID:    slot + 1,
		Race:  byte(race), Class: byte(class),
		Speed: spd, Str: str, Int: intel, End: end, ToHit: skill,
		MaxHP: end, HP: end, // 初始生命 = 體質
		Weapon: original.NotEquipped, Armor: original.NotEquipped,
		Level:  1,
		Skills: raceSkills(race, class),
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
