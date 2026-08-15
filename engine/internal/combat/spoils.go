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
