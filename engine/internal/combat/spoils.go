package combat

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

// TotalGold 回傳這一場戰鬥的金幣。
//
// ⚠ **這是具名的佔位,不是原版的公式。**
//
// 原版的管線讀出來了(docs/re/152 §2.3),形狀是:
//
//	每隻怪物  FAC ← 難度階級 ^ ds:96B8 … 再一次次方 … × RND … + 1
//	全部加完  總額 × ds:96E8,再走一次 × RND × 0.5
//
// 其中**兩個常數已經讀出來了**(docs/re/177 §6):兩次次方的指數分別是
// `ds:96B8 = 1.7` 與 `ds:93C0 = 2.1`。**還缺的是組裝順序** ——
// `INT 3F:71`/`3F:95`/`3F:85` 的運算元約定沒拆完,所以不知道
// 哪一個乘上 RND、哪一個是底數。手冊沒有任何一頁講戰後金幣。
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

// GoldAssumption 是給畫面顯示用的說明。
//
// ⚠ **長度要放得下提示列**(976 px ≈ 48 個全形字,docs/spec/04 §2)——
// 太長會被右邊界切掉,而那在測試裡沒有症狀,只有截圖看得出來。
// ⚠ 這句話會畫在**提示列**上,所以長度有上限(layout.PromptCols:
// 開源字型 96 欄、倚天點陣 80 欄)。寫太長會折成第三行,而提示列只放得下兩行 ——
// 掉出去的那一行照樣畫得出來,只是壓在畫布邊緣。
const GoldAssumption = "⚠ 戰後金幣的係數未解(re/177 §7):每隻怪物擲 1…難度階級,形狀對"
