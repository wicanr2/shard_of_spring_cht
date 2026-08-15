package town

import (
	"testing"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// maxRoll 永遠骰到**該次的最大面**,浮點也永遠回接近 1 的值。
//
// ⚠ 不能寫成「永遠回 99」—— `Train` 除了成長還會跑屬性成長
// (`GrowAttributes`,面數 5),而 99 超出 `Roller` 的契約(回 1…faces),
// 會拿去當索引。回 `faces` 同時滿足兩邊。
//
// Float01 回 `1 − 2⁻⁵³` 而不是 1.0 —— 原版的 `RND` 值域是 **[0,1)**,
// 回 1.0 會讓成長量比真正的上界多 1,而那個 +1 看起來完全合理。
type maxRoll struct{}

func (maxRoll) Roll(faces int) int { return faces }
func (maxRoll) Float01() float64   { return 1 - 1.0/(1<<53) }

// capped 是「一定骰到上界」的來源 —— 讓那些在驗**係數**的測試
// 不受骰子影響。⚠ 分開兩件事:係數對不對、亂數怎麼取,各自有自己的測試。
func capped() GrowthRand { return maxRoll{} }

func hero(level, end int) original.Character {
	return original.Character{
		Name: "測試", Class: byte(rules.ClassHero), Level: level, End: end,
		MaxHP: 10, HP: 4,
	}
}

func TestTrainNeedsTheRightGuild(t *testing.T) {
	c := hero(1, 10)
	if got := Train(&c, 999_999, 1, capped()); got != TrainWrongGuild {
		t.Fatalf("戰士進魔法訓練所應被擋下,得 %v", got)
	}
	if c.Level != 1 {
		t.Error("被擋下卻升級了")
	}
}

func TestTrainNeedsExperience(t *testing.T) {
	c := hero(1, 10)
	if got := Train(&c, 299, 0, capped()); got != TrainNotEnoughExp {
		t.Fatalf("299 經驗應不足,得 %v", got)
	}
	if got := Train(&c, 300, 0, capped()); got != TrainOK {
		t.Fatalf("300 經驗應可升級,得 %v", got)
	}
	if c.Level != 2 {
		t.Fatalf("等級應為 2,得 %d", c.Level)
	}
}

// 升級只加上限,**當前值一點都不動**(docs/re/184)。
func TestTrainAddsGrowthToTheCapOnly(t *testing.T) {
	c := hero(1, 10) // 體能 10 戰士:INT(10 × ~1 × 2/3) + 1 = 7
	Train(&c, 300, 0, capped())
	if c.MaxHP != 17 {
		t.Errorf("最大生命應為 10+7=17,得 %d", c.MaxHP)
	}
	if c.HP != 4 {
		t.Errorf("當前生命不該被動到(應留在 4),得 %d", c.HP)
	}
}

// 巫師才有法力成長。這條擋的是「兩張表用錯欄」——
// 用戰士欄給巫師加血,數字仍然合理,畫面上看不出來。
func TestWizardGrowsSPAndUsesWizardColumn(t *testing.T) {
	c := original.Character{
		Name: "法", Class: byte(rules.ClassWizard), Level: 1,
		End: 10, Int: 12, MaxHP: 8, HP: 8, MaxSP: 5, SP: 5,
	}
	Train(&c, 300, 1, capped())
	if c.MaxHP != 8+5 { // 體質 10 的**巫師**欄是 5,不是戰士的 7
		t.Errorf("巫師最大生命應為 8+5=13,得 %d", c.MaxHP)
	}
	// 智能 12:INT(12 × r × 0.5) + 2,r < 1 → 5 + 2 = 7
	// ⚠ 手冊 p.48 寫 8 —— 見 TestManualSPTableIsOffByOne
	if c.MaxSP != 5+7 {
		t.Errorf("最大法力應為 5+7=12,得 %d", c.MaxSP)
	}
}

func TestTrainIsFreeOfCharge(t *testing.T) {
	// 手冊 p.37:訓練完全免費。介面上沒有金幣參數,這裡確認升級不需要它。
	c := hero(1, 10)
	if Train(&c, 300, 0, capped()) != TrainOK {
		t.Error("升級不該需要金幣")
	}
}

// ── 屬性成長(docs/re/183)────────────────────────────────────────

// 三個常數各自能獨立弄錯,所以分開驗。
func TestGrowAttributesRollsThreeTimesOnFive(t *testing.T) {
	r := &combat.ScriptRand{Values: []int{1, 3, 5}}
	c := original.Character{Speed: 8, Str: 8, Int: 8, End: 8, ToHit: 8}
	GrowAttributes(&c, r)
	if len(r.Faces) != AttrGrowthRolls {
		t.Fatalf("應擲 %d 次,得 %d 次", AttrGrowthRolls, len(r.Faces))
	}
	for i, f := range r.Faces {
		if f != AttrGrowthPick {
			t.Errorf("第 %d 次的面數應為 %d,得 %d", i+1, AttrGrowthPick, f)
		}
	}
	// roll 1/3/5 → 位移 16/20/24 = 速度/智能/命中能力
	if c.Speed != 9 || c.Int != 9 || c.ToHit != 9 {
		t.Errorf("速度/智能/命中能力應各 +1,得 %d/%d/%d", c.Speed, c.Int, c.ToHit)
	}
	if c.Str != 8 || c.End != 8 {
		t.Errorf("沒被選中的不該動,得 力量 %d、體能 %d", c.Str, c.End)
	}
}

// 順序是位移順序,換一格不會壞掉但會加錯屬性 —— 所以逐格釘住。
func TestGrowAttributesSlotOrderMatchesCharsDat(t *testing.T) {
	for _, tc := range []struct {
		roll int
		get  func(original.Character) int
		name string
	}{
		{1, func(c original.Character) int { return c.Speed }, "速度(位移16)"},
		{2, func(c original.Character) int { return c.Str }, "力量(位移18)"},
		{3, func(c original.Character) int { return c.Int }, "智能(位移20)"},
		{4, func(c original.Character) int { return c.End }, "體能(位移22)"},
		{5, func(c original.Character) int { return c.ToHit }, "命中能力(位移24)"},
	} {
		c := original.Character{Speed: 1, Str: 1, Int: 1, End: 1, ToHit: 1}
		GrowAttributes(&c, &combat.ScriptRand{Values: []int{tc.roll}})
		if tc.get(c) != 1+AttrGrowthRolls {
			t.Errorf("roll=%d 三次都該加在%s,得 %d", tc.roll, tc.name, tc.get(c))
		}
	}
}

// **有放回**:同一項可以被選中多次。⛔ 不要「修掉」這個重複。
func TestGrowAttributesHasReplacement(t *testing.T) {
	c := original.Character{Str: 5}
	GrowAttributes(&c, &combat.ScriptRand{Values: []int{2}})
	if c.Str != 8 {
		t.Errorf("同一項連中三次應為 5+3=8,得 %d", c.Str)
	}
}

// 已經滿 20 的照樣會被選中,那一次就白費 —— 不是重骰。
func TestGrowAttributesWastesRollsOnMaxedStats(t *testing.T) {
	c := original.Character{Str: AttrGrowthCap, End: 5}
	// 兩次選力量(已滿)、一次選體能
	GrowAttributes(&c, &combat.ScriptRand{Values: []int{2, 2, 4}})
	if c.Str != AttrGrowthCap {
		t.Errorf("滿的屬性不該超過 %d,得 %d", AttrGrowthCap, c.Str)
	}
	if c.End != 6 {
		t.Errorf("白費的兩次不該補到別項:體能應為 6,得 %d", c.End)
	}
}

// 升級本身要帶動屬性成長 —— 少接這一條的話,上面四個測試全過但遊戲裡不會長。
func TestTrainGrowsAttributes(t *testing.T) {
	c := hero(1, 10)
	c.Speed, c.Str, c.Int, c.End, c.ToHit = 8, 8, 8, 8, 8
	before := c.Speed + c.Str + c.Int + c.End + c.ToHit
	Train(&c, 300, 0, &combat.ScriptRand{Values: []int{1}})
	after := c.Speed + c.Str + c.Int + c.End + c.ToHit
	if after-before != AttrGrowthRolls {
		t.Errorf("升一級應總共加 %d 點屬性,得 %d", AttrGrowthRolls, after-before)
	}
}

// 技能點:每級 +1,智能被選中就再多拿,而且會累積(docs/re/183 §6)。
func TestTrainAwardsSkillPoints(t *testing.T) {
	// roll 1 = 速度,智能沒長 → 只拿保底的 1 點
	c := hero(1, 10)
	c.Int, c.SkillPts = 12, 4
	Train(&c, 300, 0, &combat.ScriptRand{Values: []int{1}})
	if c.SkillPts != 4+1 {
		t.Errorf("智能沒長時應為 4+1=5,得 %d", c.SkillPts)
	}

	// roll 3 = 智能,三次都中 → 智能 +3,技能點 +3+1
	d := hero(1, 10)
	d.Int, d.SkillPts = 12, 0
	Train(&d, 300, 0, &combat.ScriptRand{Values: []int{3}})
	if d.Int != 15 {
		t.Fatalf("測試前提壞了:智能應為 15,得 %d", d.Int)
	}
	if d.SkillPts != 3+1 {
		t.Errorf("智能長 3 時應為 3+1=4,得 %d", d.SkillPts)
	}
}

// 智能已滿 20 的話成長是 0,技能點只拿保底 1 點 ——
// 「白費的擲骰」在這裡也要真的白費,不能因為選中就給點。
func TestTrainSkillPointsFollowActualGrowthNotTheRoll(t *testing.T) {
	c := hero(1, 10)
	c.Int, c.SkillPts = AttrGrowthCap, 0
	Train(&c, 300, 0, &combat.ScriptRand{Values: []int{3}}) // 三次都選智能
	if c.Int != AttrGrowthCap {
		t.Fatalf("智能不該超過 %d,得 %d", AttrGrowthCap, c.Int)
	}
	if c.SkillPts != SkillPtsPerLevel {
		t.Errorf("智能沒真的長,應只有保底 %d 點,得 %d", SkillPtsPerLevel, c.SkillPts)
	}
}

func TestGuildTeaches(t *testing.T) {
	if GuildTeaches(0) != byte(rules.ClassHero) {
		t.Error("位移 36 = 0 應是武術(戰士)")
	}
	if GuildTeaches(1) != byte(rules.ClassWizard) {
		t.Error("位移 36 = 1 應是魔法(巫師)")
	}
}

// 成長算式:INT(屬性 × 亂數 × 係數) + 加項(docs/re/184)。
// 用固定的浮點序列驗係數 —— 兩次 Float01 取大,所以給兩個值。
func TestLevelGainHPFormula(t *testing.T) {
	for _, c := range []struct {
		floats []float64
		end    int
		wizard bool
		want   int
	}{
		// 因子取大 = 1.0 的極限 → 上界 INT(體能 × K) + 1
		{[]float64{0.9, 0.999999}, 20, false, 14}, // INT(20 × 2/3) + 1 = 14
		{[]float64{0.999999, 0.1}, 20, true, 9},   // INT(20 × 0.434783) + 1 = 9
		{[]float64{0.5, 0.5}, 20, false, 7},       // INT(20 × 0.5 × 2/3) + 1 = 7
		{[]float64{0.0, 0.0}, 20, false, 1},       // 因子 0 → 只剩加項
	} {
		got := LevelGainHP(&combat.ScriptRand{Floats: c.floats}, c.end, c.wizard)
		if got != c.want {
			t.Errorf("體能 %d、巫師 %v、亂數 %v → %d,應為 %d",
				c.end, c.wizard, c.floats, got, c.want)
		}
	}
}

// **取大不取小** —— 取小的話下面第一列會得到 INT(20 × 0.1 × 2/3) + 1 = 2。
func TestGrowthRollTakesTheLargerOfTwo(t *testing.T) {
	for _, order := range [][]float64{{0.1, 0.9}, {0.9, 0.1}} {
		got := LevelGainHP(&combat.ScriptRand{Floats: order}, 20, false)
		if got != 13 { // INT(20 × 0.9 × 2/3) + 1 = 13
			t.Errorf("亂數 %v 應取大的 0.9 → 13,得 %d", order, got)
		}
	}
}

// 法力係數與加項各自能弄錯,分開驗。
func TestLevelGainSPFormula(t *testing.T) {
	for _, c := range []struct {
		floats []float64
		intel  int
		want   int
	}{
		{[]float64{0.999999, 0.1}, 20, 11}, // INT(20 × 0.999999 × 0.5) + 2 = 11
		{[]float64{0.999999, 0.1}, 11, 7},  // INT(11 × 0.5) + 2 = 7 ← 手冊寫 8
		{[]float64{0.0, 0.0}, 20, 2},       // 因子 0 → 只剩加項
	} {
		got := LevelGainSP(&combat.ScriptRand{Floats: c.floats}, c.intel)
		if got != c.want {
			t.Errorf("智能 %d、亂數 %v → %d,應為 %d", c.intel, c.floats, got, c.want)
		}
	}
}

// ⚠ **手冊 p.48 的 SP 表與本實作全表差 1,而差多少取決於一個未決點。**
//
// 引擎目前用**截尾**(`int()`)。若原版的浮點轉整數其實是**四捨五入**,
// 偶數智能會與手冊吻合、只有奇數 ≥ 11 差 1。兩種模式在 HP 那條**分不出來**
// (戰士係數 0.666667 比 2/3 大,餘裕蓋過 MBF 的 1−r),
// **只有 SP 分得出來**,因為它的係數 0.5 是精確的。
//
// ⛔ 這條測試釘的是**目前的選擇**,不是已確認的原版行為。
// 裁決方法(第 1 級證據,一次實測就夠):在原版把巫師的智能練到 20,
// 反覆升級,量最大法力成長是 **11**(截尾)還是 **12**(四捨五入)。
func TestManualSPTableIsOffByOne(t *testing.T) {
	manual := map[int]int{10: 7, 11: 8, 12: 8, 13: 9, 19: 12}
	for intel, m := range manual {
		got := LevelGainSP(&combat.ScriptRand{Floats: []float64{0.999999, 0.1}}, intel)
		if got != m-1 {
			t.Errorf("智能 %d:本實作應為 %d(手冊 %d),得 %d", intel, m-1, m, got)
		}
	}
}

// 升級**不回血**、也不回法力 —— 只加上限(docs/re/184)。
func TestTrainDoesNotHealOnLevelUp(t *testing.T) {
	c := original.Character{
		Name: "巫", Class: byte(rules.ClassWizard), Level: 1,
		End: 10, Int: 12, MaxHP: 8, HP: 3, MaxSP: 5, SP: 1,
	}
	Train(&c, 300, 1, capped())
	if c.HP != 3 || c.SP != 1 {
		t.Errorf("當前生命/法力不該被動到,得 %d/%d", c.HP, c.SP)
	}
	if c.MaxHP <= 8 || c.MaxSP <= 5 {
		t.Errorf("上限應該有長,得 %d/%d", c.MaxHP, c.MaxSP)
	}
}

// 生命與法力**共用**同一個總量上限 255(ds:6D8C)。
func TestTotalsClampAt255(t *testing.T) {
	c := original.Character{
		Name: "巫", Class: byte(rules.ClassWizard), Level: 1,
		End: 20, Int: 20, MaxHP: MaxTotalPoints, MaxSP: MaxTotalPoints,
	}
	Train(&c, 300, 1, capped())
	if c.MaxHP != MaxTotalPoints || c.MaxSP != MaxTotalPoints {
		t.Errorf("總量應夾在 %d,得 %d/%d", MaxTotalPoints, c.MaxHP, c.MaxSP)
	}
}
