// Package rules 是手冊附錄六張統計表化成的規則。
//
// 來源:docs/re/140-manual-stat-tables.md(手冊 p.47–49)。
// 證據等級是第 3 級(手冊),其中三項另有反組譯佐證 —— 逐條在下面註明。
//
// ⚠ **有封閉式的才寫封閉式。** 巫師的生命成長、法力成長與降魔矩陣
// 在表上沒有規律,硬湊一條「差不多對」的公式會在少數幾列上錯,
// 而那幾列在遊戲裡看起來完全正常。照表就不會有這個問題。
package rules

// ---------------------------------------------------------------------------
// 速度:行動點數與它的兩個後果。docs/re/140 §1
// ---------------------------------------------------------------------------

// 行動的點數成本。手冊 p.34/p.50 兩處一致。
const (
	CostMove   = 2 // 前進一格
	CostTurn   = 1 // 轉向(左/右/後轉都是 1)
	CostAttack = 3 // 攻擊
	CostCast   = 3 // 施法 —— 用完**結束該角色的回合**,見 EndsTurn
	CostUse    = 3 // 使用物品,同上
	CostDispel = 3 // 降魔,同上
)

// MovePoints 回傳一個角色一回合的行動點數。
//
// **行動點數 = 速度**(手冊 p.34)。p.48 的「最多走幾格 / 最多攻擊幾次」
// 兩欄是這條規則的後果,18 列逐列吻合 —— 那是一次可否證的檢定,
// 不是同一個來源說了兩次。
func MovePoints(speed int) int { return speed }

// MaxMoves / MaxAttacks 是手冊 p.48 那張表,寫成它的成因。
func MaxMoves(speed int) int   { return MovePoints(speed) / CostMove }
func MaxAttacks(speed int) int { return MovePoints(speed) / CostAttack }

// Action 是戰鬥中的一個動作。
//
// ⚠ **不能用成本來分辨動作** —— 攻擊、施法、用物品、降魔的成本都是 3,
// 而其中只有攻擊**不會**結束回合。用 `cost == 3` 判斷會讓攻擊完直接跳過該角色,
// 而畫面上那看起來只像「這個人動作比較少」。
type Action int

const (
	ActMove Action = iota
	ActTurn
	ActAttack
	ActCast
	ActUse
	ActDispel
)

// Cost 回傳一個動作的行動點數成本。
func (a Action) Cost() int {
	switch a {
	case ActMove:
		return CostMove
	case ActTurn:
		return CostTurn
	default:
		return CostAttack // 攻擊/施法/用物品/降魔都是 3
	}
}

// EndsTurn 回傳這個動作做完是否直接結束該角色的回合。
// 手冊 p.35:施法、用物品、降魔三條都寫「會取消行動能力」。
func (a Action) EndsTurn() bool {
	return a == ActCast || a == ActUse || a == ActDispel
}

// ---------------------------------------------------------------------------
// 力量 → 傷害加值。docs/re/140 §2
// ---------------------------------------------------------------------------

// StrengthBonus 是手冊 p.48 `BONUS DAMAGE BY STRENGTH`,STR 3–20 全部 18 列吻合。
//
//	bonus = INT((STR − 7) / 2)      BASIC 的 INT 向下取整
//
// ⛔ **這個值目前沒有落點。** docs/re/140 §2:`k₁`(ds:9460h)走的是乘法,
// 而這是加項,形狀不合 —— 硬塞進傷害公式會做出一個「看起來合理但沒有根據」的版本。
// 提供這個函式是為了讓落點解出來的那一天只要改一處。
func StrengthBonus(str int) int { return floorDiv(str-7, 2) }

// floorDiv 是向下取整的整數除法(Go 的 / 對負數是往零取整,與 BASIC 的 INT 不同)。
//
// ⚠ 這個差別只在**負數**上顯現:STR 6 → Go 給 0、BASIC 給 −1。
// 而屬性很少低到那裡,所以用錯了平常看不出來。
func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// ---------------------------------------------------------------------------
// 技巧 → 命中率。docs/re/140 §3(手冊與反組譯同值,已確認)
// ---------------------------------------------------------------------------

const BehindBonus = 12 // 從背後攻擊的加成

// HitChance 回傳命中百分比。手冊 p.49 表與 CMBT.EXE 的反組譯同值。
//
// ⚠ 手冊 p.50 的**說明文字**寫 `+3`,與它自己的表(+12)打架 ——
// 以表為準,因為表與反組譯對得上(docs/re/140 §3)。
func HitChance(skill int, fromBehind bool) int {
	p := 4 * skill
	if fromBehind {
		p += BehindBonus
	}
	return p
}

// ---------------------------------------------------------------------------
// 升級。docs/re/140 §9
// ---------------------------------------------------------------------------

// ExpForLevel[n] 是升到第 n+1 級所需的**累計**經驗(手冊 p.47)。
//
// ⚠ 手冊那張表的 `INCREASE FROM LAST LEVEL` 欄第 1 列不自洽(寫 200),
// 第 2 列起 18 列全部自洽 —— **累計欄才是可用的那一欄**。
var ExpForLevel = [...]int{
	1: 300, 2: 700, 3: 1_100, 4: 1_800, 5: 2_800,
	6: 4_600, 7: 7_500, 8: 12_600, 9: 21_600, 10: 37_700,
	11: 66_400, 12: 118_000, 13: 210_800, 14: 377_600, 15: 677_600,
	16: 1_217_500, 17: 2_189_300, 18: 3_938_200, 19: 7_086_100, 20: 12_752_200,
}

// MaxLevel 是經驗表的最後一級。
const MaxLevel = 20

// CanLevelUp 回傳某等級的角色在目前經驗下能不能升級。
//
// 訓練所升級**完全免費**(手冊 p.37),只看經驗夠不夠 ——
// 所以這裡沒有金幣參數,那不是漏掉。
func CanLevelUp(level, exp int) bool {
	if level < 1 || level >= MaxLevel {
		return false
	}
	return exp >= ExpForLevel[level]
}

// ---------------------------------------------------------------------------
// 升級的成長上限。docs/re/140 §結論表:兩張都**照表**,不湊公式。
// ---------------------------------------------------------------------------

// hpGain[END] = {戰士, 巫師}。手冊 p.49 `MAX H.P. GAIN PER LEVEL`。
var hpGain = map[int][2]int{
	3: {3, 2}, 4: {3, 2}, 5: {4, 3}, 6: {5, 3}, 7: {5, 4},
	8: {6, 4}, 9: {7, 4}, 10: {7, 5}, 11: {8, 5}, 12: {9, 6},
	13: {9, 6}, 14: {10, 7}, 15: {11, 7}, 16: {11, 7}, 17: {12, 8},
	18: {13, 8}, 19: {13, 9}, 20: {14, 9},
}

// spGain[INT]。手冊 p.48 `MAX S.P. GAIN PER LEVEL FOR WIZARDS`。
//
// ⚠ 手冊這張表只列到 INT 19 —— **20 那一列不存在**,不是抄漏。
// 20 的值沿用 19,並在這裡註明,免得下一輪把它當成抄錄缺失去補。
var spGain = map[int]int{
	3: 3, 4: 4, 5: 4, 6: 5, 7: 5, 8: 6, 9: 6, 10: 7, 11: 8,
	12: 8, 13: 9, 14: 9, 15: 10, 16: 10, 17: 11, 18: 11, 19: 12, 20: 12,
}

// MaxHPGain 回傳升一級最多增加多少生命值。wizard 為 true 時走巫師欄。
func MaxHPGain(endurance int, wizard bool) int {
	v, ok := hpGain[clamp(endurance, 3, 20)]
	if !ok {
		return 0
	}
	if wizard {
		return v[1]
	}
	return v[0]
}

// MaxSPGain 回傳巫師升一級最多增加多少法力。戰士沒有這一項。
func MaxSPGain(intellect int) int { return spGain[clamp(intellect, 3, 20)] }

// ---------------------------------------------------------------------------
// 降魔成功率。手冊 p.48 的矩陣,照表。
// ---------------------------------------------------------------------------

// dispelByGap[d] 是 `INT − 怪物等級 = d` 時的成功率(%)。
//
// 手冊 p.48 那張 20×10 的矩陣**每一格只取決於這個差** ——
// 200 格全部落在下面這 20 個值上,`—` 的格子剛好是 d < 0
// (rules_test.go 把整張表展開逐格比對過)。
//
// ⚠ 這 20 個值的增量是 4,3,4,4,3,4,3,4,4,3… 沒有整數週期。
// 用 `round(3.63×d)+3` 這種係數能全中,但那個 3.63 是湊出來的,
// 不是從任何地方讀到的 —— **照表比較誠實,而且一樣短**。
var dispelByGap = [...]int{
	3, 7, 10, 14, 18, 21, 25, 28, 32, 36,
	39, 43, 46, 50, 54, 57, 61, 64, 68, 72,
}

// DispelChance 回傳有 `Priesthood` 的角色降伏某等級不死怪物的成功率(%)。
//
// 差距為負回 0 —— 表上那些格印的是 `—`,意思是**辦不到**。
func DispelChance(intellect, monsterLevel int) int {
	d := intellect - monsterLevel
	if d < 0 {
		return 0
	}
	if d >= len(dispelByGap) {
		return dispelByGap[len(dispelByGap)-1] // 表外封頂在 72
	}
	return dispelByGap[d]
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
