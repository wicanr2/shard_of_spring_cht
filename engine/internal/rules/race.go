package rules

// 種族與職業。手冊 p.49 `RACE ADVANTAGES/DISADVANTAGES`,
// 對照 docs/re/140 §5 與 docs/formats/01 的位移 14/15。

// Class 是職業碼,值與 CHARS.DAT 位移 15 相同。
type Class byte

const (
	ClassHero   Class = '1'
	ClassWizard Class = '2'
)

// Race 是種族碼,值與 CHARS.DAT 位移 14 相同。
type Race byte

const (
	Human Race = 'H'
	Troll Race = 'T'
	Dwarf Race = 'D'
	Elf   Race = 'E'
	Gnome Race = 'G'
)

// RaceInfo 是一個種族的職業限制、附贈技能與屬性修正。
type RaceInfo struct {
	Name string
	// Classes 是可選的職業。長度 2 表示兩種都可以。
	Classes []Class
	// Skills 是附贈的技能編號(1–10,對照 docs/formats/01 的技能表)。
	// ⚠ 同一個編號在戰士表與法師表是**不同的技能**,所以這裡的編號
	// 只有配上該種族的職業才有意義。
	Skills []int
	// 屬性修正。手冊寫的是 Spd/Str/Int/End/Sk 五項。
	Speed, Str, Int, End, Skill int
}

// Races 是五個種族。順序照手冊表的排列。
var Races = map[Race]RaceInfo{
	Human: {
		Name: "人類", Classes: []Class{ClassHero, ClassWizard},
		Speed: 2, Int: 2, Skill: 2,
	},
	Dwarf: {
		Name: "矮人", Classes: []Class{ClassHero},
		Skills: []int{8, 2}, // Berserking、Axe(戰士表)
		Str:    1, Int: -3, End: 3,
	},
	Troll: {
		Name: "巨魔", Classes: []Class{ClassHero},
		Skills: []int{7}, // Armored skin(戰士表)
		Speed:  -3, Str: 5, Int: -5, End: 5,
	},
	Elf: {
		Name: "精靈", Classes: []Class{ClassWizard},
		Skills: []int{8, 6}, // Item lore、Weapon lore(法師表)
		Speed:  5, Str: -3, Int: 3, End: -3, Skill: -2,
	},
	Gnome: {
		Name: "地精", Classes: []Class{ClassWizard},
		Skills: []int{5}, // Spirit runes(法師表)
		Speed:  2, Str: -2, Int: 5, Skill: -2,
	},
}

// AllowsClass 回傳某種族能不能選某職業。
func AllowsClass(r Race, c Class) bool {
	info, ok := Races[r]
	if !ok {
		return false
	}
	for _, allowed := range info.Classes {
		if allowed == c {
			return true
		}
	}
	return false
}

// ⛔ **基礎屬性的骰法未解**(docs/re/140 §5)。
//
// 上面那張表給的是**修正量**,不是產生基礎值的規則。
// 看起來像 3d6 不等於是 3d6 —— 在讀到那段程式碼之前,
// remake 的角色創造只套修正、基礎值由玩家或既有角色提供。
const CreationUnresolved = "⚠ 角色創造的基礎屬性骰法未解(docs/re/140 §5),這裡只套種族修正"

// ApplyRaceModifiers 把種族修正加到一組基礎屬性上。
func ApplyRaceModifiers(r Race, speed, str, intel, end, skill int) (int, int, int, int, int) {
	info, ok := Races[r]
	if !ok {
		return speed, str, intel, end, skill
	}
	return speed + info.Speed, str + info.Str, intel + info.Int,
		end + info.End, skill + info.Skill
}
