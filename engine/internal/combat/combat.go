// Package combat 是戰鬥規則。
//
// 規則出自 docs/spec/01-combat.md,執行期形狀出自 docs/spec/07-combat-scene.md。
// **規格改了才改這裡**;每一條規則在下面註明章節。
package combat

import (
	"math"
	"strconv"
)

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
	X, Y      int    // 屬性 0/1:戰場座標。⚠ M4 只存不用(docs/spec/07 §7)
	Speed     int    // 屬性 2
	HP        int    // 屬性 3
	Weapon    int    // 屬性 4:武器編號
	Armor     int    // 屬性 5:防具編號
	Str       int    // 屬性 6
	SP        int    // 屬性 7
	Status    int    // 屬性 8:= 法術系別編號
	ToHit     int    // 屬性 9
	Facing    Facing // 屬性 10
	Kind      int    // 屬性 11:怪物類別 / 圖組
	StatMag   int    // 屬性 12:狀態效果強度
	Tier      int    // 屬性 13:難度階級(角色固定 99)
	Target    int    // 屬性 15
	Berserk   int    // 屬性 16(僅 Hero)
	ArmSkin   int    // 屬性 17(僅 Hero)—— ⚠ 傷害公式減的就是這一項
	Exp       int    // 屬性 19
	// Karate **不是原版的戰鬥屬性** —— 原版空手時去讀角色記錄的位移 45
	// (技能旗標第 4 格),不是讀陣列(docs/re/153 §5)。
	// 這裡先搬進來,免得傷害公式要回頭去查記錄。
	Karate int
	Name      string // 顯示用,不是原版屬性
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

// DamageK1 是力量加值的乘數(原版 ds:9460h)。
//
// 原版算的是 `INT((力量 − 7) × ds:9460h)`,而手冊 p.11 的力量加值是
// `INT((力量 − 7) / 2)` —— **兩者要相等,它只能是 0.5**(docs/re/153 §6.1)。
//
// ⚠ 它乘的是**力量那一項**,不是整個括號。先前的實作乘在括號上,
// 而那樣算出來的傷害同樣「看起來合理」。
//
// ⚠ 值本身仍然沒有從檔案的位元組讀出來,信心是**證據充分**:
// `−7` 這個偏移是讀到的,而手冊的公式也是 `−7`。
//
// ⚠ **1.0 已被排除。** 同一個變數在 CMBT 0x13EA0 被當成機率門檻用
// (`RND < ds:9460h` 才走某個分支)—— 若它 ≥ 1,那個條件恆真、
// 條件跳躍是廢的(docs/re/153 §6.2)。先前的佔位剛好是 1.0。
const DamageK1 = 0.5

// DamageK2 是傷害公式的加數(原版 ds:9464h)。
//
// ⚠ **0 不是佔位,是正確的。** 原版的 ds:9464h 是擲骰的偏移常數 ——
// `INT(RND × N) + 1` 的那個 `+1`,17 個使用點形狀一致(docs/re/136 §3)。
// 而本引擎的 Roll(n) 直接回 1…n,**偏移已經折進去了**,再加一次會多算。
const DamageK2 = 0.0

// ToHitFaces 是命中擲骰的面數(原版 ds:977Eh)。**仍未解**,但有下界。
//
// 同一個擲骰也用在狂暴的門檻上(`> 75`,docs/re/153 §7)。
// 狂暴是矮人的種族附贈技能,不可能是死碼 —— **所以面數 ≥ 76**。
// 配上「門檻寫 75」這個形狀,100 是最自然的候選。
//
// ⛔ 這個數字決定整個命中率的尺度 —— 填錯會讓 docs/spec/01 §4 的
// `×4` 與 `+30` 全部失去意義,而畫面上只會看到「好像太容易打中」。
const ToHitFaces = 100

// BerserkThreshold 是狂暴加倍的門檻:同一次攻擊的第二次擲骰**大於**它
// (原版 `cmp ds:977Ah, 4Bh` + `jle`,docs/re/153 §7)。
const BerserkThreshold = 75

// KarateSkill 是空手道在戰士技能表裡的編號(docs/formats/01)。
// 空手時若攻擊者有這一項,傷害改走「命中能力 − 5」那條式子。
const KarateSkill = 4

// Unresolved 是要顯示在訊息列的未解項,讓它們在**執行時**也看得見。
var Unresolved = []string{
	"命中擲骰面數未解(已知 ≥ 76,暫用 " + strconv.Itoa(ToHitFaces) + ")",
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

// 這裡曾經有一個 `DamageFaces = 26`。**那個面數不存在**:
// 它來自把 `mov bx, 1Ah` 讀成「26 面骰」,而 0x1A 是浮點累加器的**位址**
// (docs/re/152 §3、153 §9)。傷害擲骰的面數是**武器傷害本身**
// (或力量、或命中能力 − 5),每條分支各不相同。

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

// Damage 回傳一次命中的傷害。docs/re/153。
//
//	持武器  傷害 = 擲骰(武器傷害) − 護甲技能 − 防具值 + floor((力量 − 7) × k₁)
//	赤手    傷害 = 擲骰(力量)     − 護甲技能 − 防具值
//	空手道  傷害 = 擲骰(命中能力 − 5) − 護甲技能 − 防具值
//
// 原版寫的是 `INT(N × RND) + 1`,而 Roll(n) 直接回 1…n —— 同一件事。
// 所以 k₂(ds:9464h)不出現在這裡,它的 +1 已經折進 Roll。
//
// ⚠ **力量加值先前整個漏掉了**,而 k₁ 被乘在整個括號上。
// 兩種寫法算出來的傷害都在合理範圍,分不開 —— 要讀程式才知道。
//
// ⚠ 力量 < 7 時加值是**負的**,而且是 `floor` 不是截斷:
// `floor(-0.5) = -1`,`int(-0.5) = 0`。BASIC 的 INT() 是 floor。
func Damage(atk, def Unit, weapon, armor Item, r Rand) int {
	faces := weapon.Main
	strBonus := int(math.Floor(float64(atk.Str-7) * DamageK1))
	if atk.Weapon >= BareHandMin {
		// 赤手:面數換成力量,而且**沒有**力量加值那一項
		faces, strBonus = atk.Str, 0
		if atk.Karate > 0 {
			faces = atk.ToHit - 5
		}
	}
	if faces < 1 {
		// 擲骰面數 < 1 在原版是 `INT(負數 × RND)`,不會是正的傷害。
		// 夾在 0 讓 Roll 不必處理非正的面數。
		faces = 0
	}
	roll := 0
	if faces > 0 {
		roll = r.Roll(faces)
	}
	dmg := roll - def.ArmSkin - armor.Main + strBonus
	if dmg < 1 {
		// 原版:`cmp ds:9786h, 1` / `jl` → 印 'for no damage.',不扣血
		// (docs/re/153 §8)。**讀到了,不是推的。**
		return 0
	}
	return dmg
}

// Berserk 回傳這一擊要不要加倍,以及該用哪一種訊息。
//
// 原版:攻擊者的屬性 16(狂暴)非 0,**且**同一次攻擊的第二次擲骰 > 75
// (docs/re/153 §7)。加倍那一邊印 `and hacks for`,另一邊印 `and hits for`
// —— 語意與數值互相印證。
func Berserk(atk Unit, secondRoll int) bool {
	return atk.Berserk != 0 && secondRoll > BerserkThreshold
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
