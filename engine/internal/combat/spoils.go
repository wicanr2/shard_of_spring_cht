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

