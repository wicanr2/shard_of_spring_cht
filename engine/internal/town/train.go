package town

import (
	"math"

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
// 成長量是讀出來的算式(見 LevelGainHP / LevelGainSP)。
//
// 經驗值**不扣** —— 這條先前是從「累計欄的語意」推的,現在是**讀出來的**
// (docs/re/184:升級常式沒有任何一處寫位移 90–93)。
func Train(c *original.Character, exp, guildExtra int, r GrowthRand) TrainResult {
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
	// ⚠ **只加上限,不動當前值** —— 原版的升級常式沒有寫位移 28 / 32
	// (docs/re/184)。先前這裡寫 `c.HP += gainHP`,那是「新增的部分補滿」
	// 的推論,原版沒有這回事:升級不會回血。
	c.MaxHP = clampTotal(c.MaxHP + LevelGainHP(r, c.End, wizard))
	if wizard {
		c.MaxSP = clampTotal(c.MaxSP + LevelGainSP(r, c.Int))
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

// GrowthRand 是升級成長要的亂數來源:屬性成長用 Roll、HP/SP 用 Float01。
type GrowthRand interface {
	Roller
	FloatRoller
}

// 升級成長的係數,**全部是 DGROUP 初值**(docs/re/184):
//
//	ds:71C2  0.666667   戰士的生命係數(= 2/3)
//	ds:71BA  0.434783   巫師的生命係數(= 1/2.3)
//	ds:71BE  1          生命的加項
//
// 法力那條的 0.5 與 2 同樣是 DGROUP 常數(ds:6D98 / ds:722E)。
const (
	HPFactorHero   = 0.666667
	HPFactorWizard = 0.434783
	HPAddend       = 1
	SPFactor       = 0.5
	SPAddend       = 2
)

// MaxTotalPoints 是生命與法力的**總量**上限。
//
// `ds:6D8C`,在 `TOWN.EXE 0x10306` 寫成 `0FFh` —— 全檔唯一的寫入端,
// 而且**生命與法力共用同一個**(docs/re/184)。
// ⚠ 它夾的是**總量**不是成長量。
const MaxTotalPoints = 255

func clampTotal(v int) int {
	if v > MaxTotalPoints {
		return MaxTotalPoints
	}
	return v
}

// toInt 是原版把浮點存回整數變數的行為:**四捨五入**,不是截尾。
//
// `INT 3F:77` 有兩種分支,由 `ds:1Fh` 分流:`== 4` 走純 `shr`(截尾)、
// 其餘走 `shr` + `adc`(四捨五入)。**`BRUN30` 載入映像的 `ds:1Fh` 是 `0x00`**
// (`seg002` 線性 0x1FE00 → 檔位移 0x10000,`ds:00`–`ds:1F` 整塊為零),
// 而全 13 支 EXE 掃不到任何一處寫 `ds:1Fh` → 走四捨五入(docs/re/184 §5)。
//
// ⚠ **獨立佐證**:全 8 支模組共 72 個 `3F:77`,其中 42 個前面配了顯式
// `INT 3D:03`。若 `3F:77` 本來就向下取整,那 42 個 `INT()` 全是白打的。
//
// ⛔ 所以「用 `int()` 截尾」在這個專案裡是**錯的預設值** ——
// 原版要向下取整時會自己寫 `INT()`,沒寫就是四捨五入。
func toInt(x float64) int { return int(math.Floor(x + 0.5)) }

// growthRoll 回傳成長算式用的亂數因子:**兩次 RND 取大**。
//
// 原版擲兩次 `RND` 存進 `ds:71B0` / `ds:71B4`,比較後把大的留在 `71B0`
// (`TOWN 0x11235`–`0x11243`:`3F:9F` 比較、`jb` → `xchg` → `3F:7B` 複製)。
//
// ⚠ **取大不取小**,期望值 2/3 而不是 1/2 —— 這在平均成長量上看得見。
func growthRoll(r FloatRoller) float64 {
	a, b := r.Float01(), r.Float01()
	if a < b {
		return b
	}
	return a
}

// LevelGainHP 回傳升一級的生命成長。
//
//	成長 = 四捨五入(體能 × max(R1,R2) × K) + 1   K:戰士 2/3、巫師 1/2.3
//
// **這是讀出來的算式**(docs/re/184),不是「擲骰後夾上限」的近似。
// 兩者的關係是:亂數因子趨近 1 時,這條式子的上界正好等於手冊 p.49 那張表 ——
// **`INT(體能 × K) + 1` 對手冊 36 格全中**,所以那張表是這條公式的推論,
// 不是程式裡的查表(程式裡根本沒有那張表)。
//
// ⚠ 專案負責人裁定的「現骰」(2026-08-15,第 1 級證據)**成立**;
// 「夾上限」那一半被算式取代 —— 算式本身不可能超過上限,不需要再夾。
func LevelGainHP(r FloatRoller, endurance int, wizard bool) int {
	k := HPFactorHero
	if wizard {
		k = HPFactorWizard
	}
	if endurance < 0 {
		endurance = 0
	}
	return toInt(float64(endurance)*growthRoll(r)*k) + HPAddend
}

// LevelGainSP 回傳巫師升一級的法力成長。戰士沒有這一項。
//
//	成長 = 四捨五入(智能 × max(R1,R2) × 0.5) + 2
//
// ⚠ **手冊 p.48 那張表對不上**,而且不是「差幾格」的問題:手冊的數字是
// 設計者紙筆算的 `INT(屬性 × K) + C`,不是量出來的 —— 證據是那張表自己
// **在智能 11 換了公式**(3–10 用 `INT(I/2)+2`、11–19 用 `INT((I+1)/2)+2`),
// 量出來的表不會中途換規則(docs/re/184 §4)。
//
// 依 CLAUDE.md §6 的證據優先序,反組譯(第 2 級)勝過手冊(第 3 級)。
//
// 一次就能定案的實測(第 1 級證據):**智能 10 的巫師**連升 10–15 級 ——
// 看到過一次「+7」就證實四捨五入(截尾為真時 7 的機率是 0),全程 ≤ 6 就是截尾。
func LevelGainSP(r FloatRoller, intellect int) int {
	if intellect < 0 {
		intellect = 0
	}
	return toInt(float64(intellect)*growthRoll(r)*SPFactor) + SPAddend
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
const GrowthNote = "升級成長:INT(屬性 × 兩次亂數取大 × 係數) + 加項;生命與法力總量共用上限 255"

// NeedExp 回傳這個等級升下一級所需的累計經驗;已達頂級回 0。
func NeedExp(level int) int {
	if level < 1 || level >= rules.MaxLevel {
		return 0
	}
	return rules.ExpForLevel(level)
}
