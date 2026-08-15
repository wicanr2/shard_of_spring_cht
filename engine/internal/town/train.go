package town

import (
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// 訓練所。手冊 p.37(docs/re/140 §6):
//
//   - 升級**完全免費**,只看經驗夠不夠
//   - 戰士的訓練所只收戰士,巫師的只收巫師(TOWNDATA.DAT 位移 36:0 = 武術、1 = 魔法)

// TrainResult 說明一次訓練的結果。
type TrainResult int

const (
	TrainOK TrainResult = iota
	TrainWrongGuild
	TrainNotEnoughExp
	TrainMaxLevel
)

// docs/spec/19-module-text.md(F1):TrainWrongGuild 照
// translations/module-text/TOWN.tsv 第 38 列(「You are the wrong class!」)。
// TrainNotEnoughExp 的完整原文(TOWN:40/41)含經驗差額,由呼叫端
// (town_scene.go 的 trainMember)另外組字串,不在這裡處理。
func (r TrainResult) String() string {
	switch r {
	case TrainWrongGuild:
		return "你的職業不對!"
	case TrainNotEnoughExp:
		return "經驗還不夠"
	case TrainMaxLevel:
		return "已經是最高等級"
	}
	return ""
}

// GuildTeaches 回傳這間訓練所收哪個職業。
// extra 是 TOWNDATA.DAT 位移 36:0 = 武術(戰士)、1 = 魔法(巫師)。
func GuildTeaches(extra int) byte {
	if extra == 1 {
		return byte(rules.ClassWizard)
	}
	return byte(rules.ClassHero)
}

// Train 讓一位角色在訓練所升一級。
//
// 成長量是**現骰後夾上限**(見 LevelGain)。
//
// 經驗值**不扣**(手冊沒說要扣,而累計欄的語意是「累計到多少」)。
func Train(c *original.Character, exp, guildExtra int, r Roller) TrainResult {
	if c.Class != GuildTeaches(guildExtra) {
		return TrainWrongGuild
	}
	if c.Level >= rules.MaxLevel {
		return TrainMaxLevel
	}
	if !rules.CanLevelUp(c.Level, exp) {
		return TrainNotEnoughExp
	}
	wizard := c.Class == byte(rules.ClassWizard)

	c.Level++
	gainHP := LevelGain(r, c.End, rules.MaxHPGain(c.End, wizard))
	c.MaxHP += gainHP
	c.HP += gainHP // 升級時把新增的部分也補滿,舊傷不補

	if wizard {
		gainSP := LevelGain(r, c.Int, rules.MaxSPGain(c.Int))
		c.MaxSP += gainSP
		c.SP += gainSP
	}
	intBefore := c.Int
	GrowAttributes(c, r)
	// 升級發的技能點 = 舊的剩餘 + 智能成長 + 1(docs/re/183 §6)。
	// ⚠ 順序有意義:要在屬性成長**之後**才知道智能長了多少。
	c.SkillPts += (c.Int - intBefore) + SkillPtsPerLevel
	return TrainOK
}

// SkillPtsPerLevel 是每級無條件發的技能點。
//
// `TOWN.EXE 0x1140C` 的 `inc ax` —— **無條件**,不看職業也不看等級
// (docs/re/183 §6)。智能在這一級被屬性成長選中的話還會再多拿。
const SkillPtsPerLevel = 1

// ── 升級的屬性成長(docs/re/183)──────────────────────────────────

// AttrGrowthRolls / AttrGrowthPick / AttrGrowthCap 是屬性成長的三個常數,
// **全部是讀出來的**(docs/re/183 §2/§3):
//
//	3   `TOWN.EXE 0x113D6 cmp ax, 3` —— 迴圈上界的**立即數**,
//	    不看等級、種族、職業(兩個職業在 0x11301 匯流後才進迴圈)
//	5   `ds:724C` 的 DGROUP 初值(MBF 5),`INT(RND(1) × 5)` 的乘數
//	20  `0x11366 cmp ax, 14h` —— 超過就寫回 20
const (
	AttrGrowthRolls = 3
	AttrGrowthPick  = 5
	AttrGrowthCap   = 20
)

// attrSlots 是 `roll×2 + 16` 掃過的五格,順序就是 CHARS.DAT 的排列順序
// (位移 16/18/20/22/24)。⚠ **順序不能改** —— roll 是位移的來源,
// 換順序等於換成長機率的對象。
func attrSlots(c *original.Character) [AttrGrowthPick]*int {
	return [AttrGrowthPick]*int{&c.Speed, &c.Str, &c.Int, &c.End, &c.ToHit}
}

// GrowAttributes 執行升級的屬性成長:**擲 3 次,每次五選一 +1,夾 20**。
//
// ⚠ **有放回、無排重** —— 同一次升級可以讓同一項加到 +2 或 +3
// (機率見 docs/re/183 §5:三項各 +1 佔 48%,某項 +2 也佔 48%)。
// 這是原版的行為,不是可以「順手修掉」的重複。
//
// ⚠ **已經滿 20 的屬性照樣會被選中**,那一次擲骰就白費 ——
// 所以高等級角色的成長會自然變慢。⛔ 不要改成「重骰直到選中沒滿的」,
// 那會發明一條原版沒有的規則。
func GrowAttributes(c *original.Character, r Roller) {
	slots := attrSlots(c)
	for i := 0; i < AttrGrowthRolls; i++ {
		p := slots[r.Roll(AttrGrowthPick)-1] // Roll 回 1…5,原版是 0…4
		if *p < AttrGrowthCap {
			*p++
		}
	}
}

// LevelGain 回傳升一級實際加多少點:**擲骰,超過上限就用上限**。
//
//	成長 = min(擲骰(屬性), 上限)
//
// 上限是手冊 p.48/p.49 兩張表(`MAX … GAIN PER LEVEL`)—— 欄名寫著「最多」,
// 所以那兩張表給的不是成長量本身,是成長量的**天花板**。
//
// **「現骰」是確定的** —— 專案負責人實測原版,同一個角色反覆升級成長量**會變**
// (2026-08-15,第 1 級證據)。
//
// ⚠ **但骰面是具名假設**:裁定沒有指定骰幾面,原版那段程式碼也沒有讀到。
// 這裡取**屬性本身**(生命骰體能、法力骰智能),理由有二:
//
//   - 只有骰面**可能超過上限**,「超過就用上限」才有作用 ——
//     若骰 1…上限,那句規則永遠不會生效;
//   - 全遊戲的擲骰成語都是 `INT(RND × N) + 1`(docs/re/152 §3),
//     而這裡手邊唯一的 N 就是那個屬性。
//
// ⛔ **這不是讀出來的。** 要裁決得去讀升級那段程式碼,或在原版裡
// 同一個角色反覆升級看成長量會不會變。
//
// 效果:低屬性的人幾乎每次都吃滿(上限本來就低),高屬性的人**平均拿不到上限** ——
// 體能 20 的戰士上限 14,骰 1…20 夾完平均只有 9.45。
func LevelGain(r Roller, attr, cap int) int {
	if cap <= 0 {
		return 0
	}
	if attr < 1 {
		attr = 1
	}
	if v := r.Roll(attr); v < cap {
		return v
	}
	return cap
}

// AttrNames 是五項屬性的顯示名,順序同 attrSlots(CHARS.DAT 位移 16–24)。
// 譯名照 translations/glossary.md。
//
// ⚠ 原版印的名字來自執行期填的查表 `ds:6D28 + (roll+20)×4`,靜態讀不到內容
// (docs/re/183 §6),所以這裡用的是角色表已經在用的那組名字。
var AttrNames = [AttrGrowthPick]string{"速度", "力量", "智能", "體能", "命中"}

// AttrSnapshot 取五項屬性的當下值,配 AttrGrowth 用來算「長了哪幾項」。
func AttrSnapshot(c original.Character) [AttrGrowthPick]int {
	s := attrSlots(&c)
	var out [AttrGrowthPick]int
	for i, p := range s {
		out[i] = *p
	}
	return out
}

// AttrGrowth 回傳升級後五項各長了多少(對應 'Stats are up by:')。
func AttrGrowth(before [AttrGrowthPick]int, c original.Character) [AttrGrowthPick]int {
	after := AttrSnapshot(c)
	var d [AttrGrowthPick]int
	for i := range d {
		d[i] = after[i] - before[i]
	}
	return d
}

// GrowthNote 是訓練所畫面的說明。
const GrowthNote = "升級成長:擲骰(屬性)夾在手冊 p.48/49 的上限。⚠ 骰面是假設,原版那段沒讀到"

// NeedExp 回傳這個等級升下一級所需的累計經驗;已達頂級回 0。
func NeedExp(level int) int {
	if level < 1 || level >= rules.MaxLevel {
		return 0
	}
	return rules.ExpForLevel[level]
}
