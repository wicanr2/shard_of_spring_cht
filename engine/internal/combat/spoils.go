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

// TotalGold 回傳這一場戰鬥的金幣。
//
// ⚠ **這是具名的佔位,不是原版的公式。**
//
// 原版每隻怪物的形狀是(docs/re/152 §2.3、200 §3.2):
//
//	FAC ← 難度階級 → ^ ds:96B8(1.7) → 存 → ^ ds:93C0(2.1) → 存
//	→ RND × 後者 → + 前者 → + 1 → INT()
//
// **運算子的約定已經定案**(`3F:71` 存、`3F:95` 乘、`3F:85` 加,
// docs/re/77 §1 + 184 §7),擋住的只剩一件事:`3F:23`(次方)遇到
// **非整數指數**(1.7 / 2.1)走的是哪一條路徑 —— 那一支有整數指數的
// 特例處理,而 1.7 不是整數。⛔ 在讀完 `BRUN30 0x1B8A9` 的一般路徑之前
// 不要把 `^1.7` 寫進來。
//
// ⚠ **「總額 × 0.575,再走一次 × RND × 0.5」那一段不是金幣的一部分** ——
// 它是**戰後掉落道具**的擲骰(docs/re/200 §3.2),吃的是金幣總額當上限。
// 先前這裡把它算進金幣的管線裡。
//
// ⚠ 先前這裡寫「四個常數是執行期變數,不在檔案映像裡」——**那是錯的**。
// BASIC 把運算式裡的數字字面值放成 DGROUP 常數,而那些常數有初值、
// 就寫在檔案裡(docs/re/177 §1)。
//
// 這裡用「每隻怪物擲 1…難度階級」:量綱對(來源是難度階級)、
// 形狀對(`INT(RND × N) + 1`),但**係數與兩次次方都不在裡面**。
//
// ⛔ 不要因為「玩起來手感不錯」就調這個數。要改就是去實測。
//
// ⚠ 為什麼不乾脆給 0:給 0 也是一個選擇,而且**確定是錯的** ——
// 玩家永遠買不起東西,整個經濟停擺。有標示的近似值比確定錯誤的零好。
func TotalGold(units []Unit, r Rand) int {
	total := 0
	for i, u := range units {
		if i >= PartyBase || !u.IsMonster {
			continue
		}
		tier := u.Tier
		if tier < 1 {
			tier = 1
		}
		total += r.Roll(tier)
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

// GoldAssumption 是給畫面顯示用的說明。
//
// ⚠ **長度要放得下提示列**(976 px ≈ 48 個全形字,docs/spec/04 §2)——
// 太長會被右邊界切掉,而那在測試裡沒有症狀,只有截圖看得出來。
// ⚠ 這句話會畫在**提示列**上,所以長度有上限(layout.PromptCols:
// 開源字型 96 欄、倚天點陣 80 欄)。寫太長會折成第三行,而提示列只放得下兩行 ——
// 掉出去的那一行照樣畫得出來,只是壓在畫布邊緣。
const GoldAssumption = "⚠ 戰後金幣的係數未解(re/177 §7):每隻怪物擲 1…難度階級,形狀對"
