package combat

import "math"

// 戰利品:經驗值。整段算式讀出來了 —— docs/re/152。
//
//	每人所得 = INT( 九個怪物槽的屬性 19 總和 ÷ 有資格人數 )
//
// 經驗值來自 `MONSTERS.DAT` 位移 31(欄 8),開場搬進戰鬥陣列屬性 19。

// TotalExp 回傳這一場戰鬥的經驗總額。
//
// ⚠ **不篩死活。** 原版的累加迴圈跑列 0–8 九個怪物槽,
// 裡面**沒有任何「這隻死了沒」的判斷**(docs/re/152 §1.1);
// 屬性 19 全模組只有一個寫入點(開場)與一個讀取點(這裡),
// 沒有人在怪物死掉時清掉它。
//
// 也就是說**逃走的怪物照樣給經驗**。多數戰鬥打到怪物全滅才結束,
// 所以這個差別在畫面上看不出來 —— 只有怪物逃走的那一場會分岔。
func TotalExp(units []Unit) int {
	total := 0
	for i, u := range units {
		if i >= PartyBase || !u.IsMonster {
			continue
		}
		total += u.Exp
	}
	return total
}

// ExpShare 是分給每個有資格的隊員的經驗值。
//
// 整數除法 —— 原版是浮點除完再 `INT()`(docs/re/152 §1.3),
// 對非負數兩者相同。
//
// 有資格的人數為 0 時回 0(全隊陣亡就沒有人拿得到)。
func ExpShare(total, eligible int) int {
	if eligible <= 0 {
		return 0
	}
	return total / eligible
}

// EarnsExp 回傳這個單位戰後分不分得到經驗。
//
// 原版的結算迴圈把兩個條件 `and` 起來(docs/re/150 §2.1):
//
//	屬性 10(朝向)> 0     還在戰場上
//	屬性  8(狀態)< 5     未陣亡
//
// ⚠ **第一個條件不是生命值。** 屬性 3 才是生命值,而原版讀的是屬性 10 ——
// 兩者多數情況同時成立(死了通常也不在場),
// **只有「逃走但活著」那一種狀態分得開**,而那正是原版特別處理的情況。
func (u Unit) EarnsExp() bool { return !u.IsMonster && u.OnField() && u.Status < StatusDead }

// StatusDead 是狀態 5(`D E A D`)。與 original.StatusDead 同值 ——
// 這裡不 import original,戰鬥層只認屬性編號。
const StatusDead = 5

// StatusIncapacitated 是「不能收東西」的門檻:狀態 **≥ 2** 的人
// 在戰後掉落那一支會被跳過(`CMBT 0x130FA` 的 `cmp …, 2` / `jl`)。
// 中毒(1)還算數,被縛(2)以上不算。
const StatusIncapacitated = 2

// 戰後金幣。規則出自 docs/re/207(在那之前是具名佔位)。
const (
	// GoldBaseFloor 是保底項的底數:`ds:96B8` 的 DGROUP 初值 = MBF **1.7**。
	GoldBaseFloor = 1.7
	// GoldBaseRoll 是浮動項的底數:`ds:93C0` = MBF **2.1**。
	GoldBaseRoll = 2.1
	// GoldPlusOne 是最後加的常數:`ds:9464` = MBF 1。
	GoldPlusOne = 1
)

// MonsterGold 是一隻怪物的金幣(docs/re/207 §3):
//
//	INT( 1.7^階級 + RND × 2.1^階級 + 1 )
//
// ⚠ **1.7 與 2.1 是底數,階級是指數** —— 反過來寫(`階級^1.7`)在低階
// 幾乎看不出差別,到高階差好幾個量級。原版把階級存進 `ds:96B4` 再交給
// `INT 3F:23`,而那支常式的 `di` 邊(= 階級那一邊)才是指數:
// `[di] 為 0 → 回 1.0`(`x^0 = 1`),整數快速路徑取的也是它。
//
// ⚠ **兩個底數不同是有意義的**:1.7 那一項是保底、2.1 那一項乘上亂數 ——
// 所以階級越高,「這一場能拿多少」的變異越大。⛔ 不要為了簡化寫成同一個底數。
//
// ⚠ `INT()` 是**截尾**(`INT 3D:03`),不是四捨五入 ——
// 這裡沒有配 `3F:77`(docs/re/185)。
func MonsterGold(tier int, r FloatRand) int {
	if tier < 1 {
		tier = 1
	}
	t := float64(tier)
	return int(math.Pow(GoldBaseFloor, t) +
		r.Float01()*math.Pow(GoldBaseRoll, t) + GoldPlusOne)
}

// TotalGold 回傳這一場戰鬥的金幣:每一隻在場的怪物各擲一次,加起來。
//
// ⚠ **只算怪物。** 隊員的屬性 13 固定 99(docs/spec/01 §1),
// 代進 `2.1^99` 會直接溢位成 +Inf —— 而那不會 panic,只會讓金幣變成一個
// 天文數字然後在轉 int 時變成未定義。
func TotalGold(units []Unit, r FloatRand) int {
	total := 0
	for i, u := range units {
		if i >= PartyBase || !u.IsMonster {
			continue
		}
		total += MonsterGold(u.Tier, r)
	}
	return total
}

// 戰後隨機掉落一件道具。規則出自 docs/re/200 §3。
const (
	// LootGoldFactor:`ds:96E8` 的 DGROUP 初值 = MBF 0.575。
	LootGoldFactor = 0.575
	// LootHalf:`ds:9460` = MBF 0.5,兩個 RND 項各乘一次。
	LootHalf = 0.5
	// LootBase:`ds:96EE` = MBF 2 —— 與值域下界是同一個 2。
	LootBase = 2
	// 值域:`0x130D6`/`0x130DF` 的兩個比較,超出就整段重擲。
	// 上界 46 把火把/油燈/鑰匙/傳送器擋在外面,下界 2 擋掉匕首與小斧。
	LootMin = 2
	LootMax = 46
	// lootTries 是重擲上限。⚠ **原版沒有上限**;金幣很高時 46 以下的機率
	// 會變小,而引擎不能掛在這裡。用完就回 LootMin —— 那是**引擎的決定**。
	lootTries = 500
)

// LootIndex 擲出戰後掉落的道具編號(docs/re/200 §3.2):
//
//	G   = round(總金幣 × 0.575)
//	編號 = INT( G×RND₁×0.5 + G×RND₂×0.5 + 2 )，不在 2–46 就重擲
//
// ⚠ **兩個 RND 是獨立的兩次**,不能併成一次:值域一樣但分佈完全不同
// (兩個均勻分佈相加是三角分佈,docs/re/156 記過同一個坑)。
func LootIndex(gold int, r FloatRand) int {
	g := float64(roundHalfUp(float64(gold) * LootGoldFactor))
	for i := 0; i < lootTries; i++ {
		v := int(math.Floor(g*r.Float01()*LootHalf + g*r.Float01()*LootHalf + LootBase))
		if v >= LootMin && v <= LootMax {
			return v
		}
	}
	return LootMin
}

