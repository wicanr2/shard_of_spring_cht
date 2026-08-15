package combat

// 戰利品:經驗值。docs/re/140 §9、docs/formats/03 欄 8。

// TotalExp 回傳被打倒的怪物身上的經驗值總和。
//
// 經驗值是 `MONSTERS.DAT` 位移 31(欄 8),docs/formats/03 註明它
// **在戰鬥中只寫不讀** —— 也就是說它被搬進戰鬥陣列屬性 19 之後就沒人動它,
// 用途只能是結算。
//
// ⚠ 只算**倒下的怪物**(`IsMonster && !Alive()`)。
// 「活著」與「在場」是兩個欄位(docs/spec/07 §6)—— 逃走的怪物 `Alive()` 仍為真,
// 用 `OnField()` 判斷會把逃走的也算成打倒。
func TotalExp(units []Unit) int {
	total := 0
	for _, u := range units {
		if !u.IsMonster || u.Alive() {
			continue
		}
		total += u.Exp
	}
	return total
}

// ExpShare 是分給每個生還隊員的經驗值。
//
// ⚠ **怎麼分未解。** 原版沒有讀到分配那一段;可能是每人全額,也可能均分。
// 這裡選**均分給生還者**,是一個**具名的假設**,不是讀出來的規則 ——
// 換成全額只要改這一個函式。
//
// 生還者為 0 時回 0(全滅就沒有人拿得到)。
func ExpShare(total, survivors int) int {
	if survivors <= 0 {
		return 0
	}
	return total / survivors
}

// ExpSplitAssumption 是給畫面顯示用的說明,免得玩家把假設當成原版行為。
const ExpSplitAssumption = "⚠ 經驗值的分配方式未解,本引擎均分給生還者"

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

