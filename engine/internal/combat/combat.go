// Package combat 是戰鬥規則。
//
// 規則出自 docs/spec/01-combat.md,執行期形狀出自 docs/spec/07-combat-scene.md。
// **規格改了才改這裡**;每一條規則在下面註明章節。
package combat

import "strconv"

// 單位陣列的槽位配置。docs/spec/07 §1。
//
// ⚠ **順序不可以改。** 怪物在前、隊伍在後這件事被逃跑判定直接依賴
// (掃索引 9…人數+8,docs/spec/01 §7)。
const (
	Slots       = 15
	MonsterBase = 0
	MonsterMax  = 9 // 索引 0–8
	PartyBase   = 9
	PartyMax    = 5 // 索引 9–13
	// 索引 14 未使用 —— 原版陣列開 15 個,實際用 14 個。
)

// Facing 是朝向。**0 代表不在場**,不是朝北(docs/spec/07 §1)。
type Facing int

const (
	Absent Facing = 0
	North  Facing = 1
	East   Facing = 2
	South  Facing = 3
	West   Facing = 4
)

// Unit 是單位陣列的一列。欄位名對應 docs/spec/01 §1 的屬性編號。
type Unit struct {
	X, Y     int    // 屬性 0/1:戰場座標。⚠ M4 只存不用(docs/spec/07 §7)
	Speed    int    // 屬性 2
	HP       int    // 屬性 3
	Weapon   int    // 屬性 4:武器編號
	Armor    int    // 屬性 5:防具編號
	Str      int    // 屬性 6
	SP       int    // 屬性 7
	Status   int    // 屬性 8:= 法術系別編號
	ToHit    int    // 屬性 9
	Facing   Facing // 屬性 10
	Kind     int    // 屬性 11:怪物類別 / 圖組
	StatMag  int    // 屬性 12:狀態效果強度
	Tier     int    // 屬性 13:難度階級(角色固定 99)
	Target   int    // 屬性 15
	Berserk  int    // 屬性 16(僅 Hero)
	ArmSkin  int    // 屬性 17(僅 Hero)—— ⚠ 傷害公式減的就是這一項
	Exp      int    // 屬性 19
	Name     string // 顯示用,不是原版屬性
	IsMonster bool
}

// Alive 回傳這個單位還在不在場上。
//
// ⚠ **「活著」與「在場」是兩個欄位**(docs/spec/07 §6):
// 生命值 ≤ 0 是死亡,朝向 ≤ 0 是離場。合併成一個會讓逃跑判定失效。
func (u Unit) Alive() bool   { return u.HP > 0 }
func (u Unit) OnField() bool { return u.Facing > Absent }

// Item 是 ITEMS.DAT 的兩個進戰鬥公式的欄位(docs/formats/04)。
//
// ⚠ 欄4/欄5 是**型別相依**的:武器上是傷害/命中加值,防具上是防護/加值
// (docs/re/74:同一欄被兩段程式碼用兩種方式解讀)。
type Item struct {
	Main  int // 欄4:武器 = 傷害、防具 = 防護
	Bonus int // 欄5:加值
}

// BareHandMin 是「赤手空拳」的判斷門檻:武器編號 ≥ 60 視為沒有武器
// (docs/spec/01 §5)。
const BareHandMin = 60

// DamageK1 是傷害公式的乘數(原版 ds:9460h)。**未解**(docs/re/136)。
//
// ⛔ 1 是**單位元**,不是原版的值。不要因為「打起來手感不錯」就換一組數字 ——
// 那會讓未解項變成看起來已解的錯誤結論。解出來只要改這裡。
const DamageK1 = 1.0

// DamageK2 是傷害公式的加數(原版 ds:9464h)。
//
// ⚠ **0 不是佔位,是正確的。** 原版的 ds:9464h 是擲骰的偏移常數 ——
// `INT(RND × N) + 1` 的那個 `+1`,17 個使用點形狀一致(docs/re/136 §3)。
// 而本引擎的 Roll(n) 直接回 1…n,**偏移已經折進去了**,再加一次會多算。
const DamageK2 = 0.0

// ToHitFaces 是命中擲骰的面數(原版 ds:977Eh)。**未解**(docs/re/136 §7)。
//
// ⛔ 這個數字決定整個命中率的尺度 —— 填錯會讓 docs/spec/01 §4 的
// `×4` 與 `+30` 全部失去意義,而畫面上只會看到「好像太容易打中」。
// 100 是**佔位**,挑它只因為命中值的量級看起來是百分比。
const ToHitFaces = 100

// Unresolved 是要顯示在訊息列的未解項,讓它們在**執行時**也看得見。
var Unresolved = []string{
	"傷害乘數 k1 未解(暫用 1)",
	"命中擲骰面數未解(暫用 " + strconv.Itoa(ToHitFaces) + ")",
}


// ReorderEachRound:先攻是否每回合重排。
//
// ⚠ **這是實作決定,不是 RE 結論**(docs/spec/01 §2 明寫「未確認」)。
// 選 true 的理由:速度可能被法術改動,不重排會讓那些法術沒有效果。
// 解出來時改這一個地方。
const ReorderEachRound = true

// Rand 是擲骰來源。Roll(n) 回 1..n。
//
// 介面化是為了兩件事:固定種子的可重現性(docs/spec/07 §2),
// 以及測試可以餵腳本化的序列來逐項驗公式。
type Rand interface{ Roll(faces int) int }

// DamageFaces 是傷害公式兩個亂數的面數,原版都是常數 26
// (docs/re/78:`mov bx, 1Ah` 出現兩次)。
const DamageFaces = 26

// ToHit 回傳命中門檻。docs/spec/01 §4。
//
//	命中 = (命中能力[攻] − 防具加值[防] + 武器加值[攻]) × 4
//	       + 30  若 狀態[防] > 1
//	       + 12  若 朝向[攻] == 朝向[防]
//
// ⚠ `+30` 的條件是 `> 1` **不是 ≤ 1** —— docs/re/98 抓到過一次反號,
// 而反號之後的規則仍然「看起來合理」(狀態好的更容易被打中),沒有任何訊號。
func ToHit(atk, def Unit, atkWeapon, defArmor Item) int {
	v := (atk.ToHit - defArmor.Bonus + atkWeapon.Bonus) * 4
	if def.Status > 1 {
		v += 30
	}
	if atk.Facing == def.Facing {
		v += 12 // 背後攻擊:兩者朝同一方向 = 從背後打
	}
	return v
}

// Hits 擲骰判定是否命中。擲骰 ≤ 門檻 → 命中。
func Hits(atk, def Unit, atkWeapon, defArmor Item, r Rand, faces int) (int, bool) {
	roll := r.Roll(faces)
	return roll, roll <= ToHit(atk, def, atkWeapon, defArmor)
}

// Damage 回傳一次命中的傷害。docs/spec/01 §5。
//
//	傷害 = (武器傷害 × R₁ − 屬性17[防] − 防具值) × k₁ + R₂ + k₂
//
// 全程浮點,最後轉整數(原版走 MBF 單精度,docs/re/79)。
// ⚠ 括號位置是從運算順序推的,不是從括號本身讀出(docs/re/79 §3)。
func Damage(atk, def Unit, weapon, armor Item, r Rand) int {
	dmg := float64(weapon.Main)
	if atk.Weapon >= BareHandMin {
		dmg = 1 // 赤手空拳
	}
	v := dmg*float64(r.Roll(DamageFaces)) -
		float64(def.ArmSkin) - float64(armor.Main)
	v = v*DamageK1 + float64(r.Roll(DamageFaces)) + DamageK2
	if v < 0 {
		// ⚠ 原版是否夾在 0 **未讀到**。這裡夾住是為了不讓負傷害變成治療;
		// 若之後讀到原版允許負值,改這一行。
		return 0
	}
	return int(v)
}

// Apply 把傷害套到防禦者身上。docs/spec/01 §6。
//
//	屬性3 −= 傷害;若 < 1 則設為 0
func Apply(def *Unit, dmg int) {
	def.HP -= dmg
	if def.HP < 1 {
		def.HP = 0
	}
}
