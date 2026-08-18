package combat

import "shardofspring/internal/original"

// 戰鬥的 `D)ispell`(降魔／驅散不死生物)。規則全部出自 docs/re/188。
//
// 三道閘門在呼叫端(engine/dispell_scene.go)——它們讀的是**角色記錄**
// (`Priesthood` 技能旗標、職業、這場用過沒有),不是戰鬥屬性陣列。
// 這一支只管「場上有沒有不死生物」與「這一隻驅不驅得掉」。

// UndeadClasses 是不死生物的 `MONSTERS.DAT` 欄6(類別／圖組)。
//
// 原版比的是**戰鬥屬性 11** 的 65／73／81(`CMBT 0x16CBA`–`0x16CDD`,已確認),
// 而屬性 11 = `(欄6 − 1) × 8 + 41` —— ⚠ **那個換算是推的,不是讀到的**
// **{5, 6, 7}**,由 docs/re/220 訂正 —— 先前是 `{4, 5, 6}`,差一位。
//
// 原版讀到的不死**圖組**是 `{65, 73, 81}`(docs/re/188 §3)。
// 圖組換算成檔號是 `檔號 = (圖組 − 25) / 8` → **5 / 6 / 7**。
//
// ⚠ 內容印證:`MONST5` 是骷髏、`MONST6` 是骷髏巫師、`MONST7` 是幽靈 ——
// 三個都是不死。舊的 `{4,5,6}` 把**幽靈排除在外**,還為此寫了一句
// 「本作把它們歸成靈體」—— 那是**替一個錯誤的數字編出來的解釋**,
// 而它讀起來完全合理。⚠ 欄6 也根本沒有 4 這個值(那是隊員巫師的檔)。
var UndeadClasses = [3]int{5, 6, 7}

// DispellFactor 是成功門檻的乘數。`ds:9D98` 的 DGROUP 初值 = MBF **3.6**
// (docs/re/188 §4)。
const DispellFactor = 3.6

// IsUndead 回傳這個單位算不算不死生物。**怪物才算** ——
// 隊員的屬性 11 走另一組值(41／57),不可能落在不死那一組。
func IsUndead(u Unit) bool {
	if !u.IsMonster {
		return false
	}
	for _, c := range UndeadClasses {
		if u.Kind == c {
			return true
		}
	}
	return false
}

// DispellThreshold 是驅散一隻怪物的成功門檻(d100 ≤ 門檻 → 成功)。
//
//	門檻 = (施法者智能 − 怪物難度階級 + 1) × 3.6
//
// 智能高、階級低就容易成功。⚠ 門檻可以是負的(智能低、階級高),
// 那時 d100 永遠打不過 —— 照原版,**不夾在 0**。
func DispellThreshold(intellect, tier int) float64 {
	return float64(intellect-tier+1) * DispellFactor
}

// HasUndead 回傳場上有沒有活著的不死生物。
//
// ⚠ 原版在**進迴圈之前先掃一次**(`0x16C92`),沒有才印
// 「None of these monsters are undead!」—— 少了這一步,一隻都沒有的時候
// 會安靜地什麼都不做,玩家不知道是自己按錯還是驅散失敗。
func (f *Field) HasUndead() bool {
	for i := MonsterBase; i < MonsterBase+MonsterMax; i++ {
		if f.Units[i].Alive() && IsUndead(f.Units[i]) {
			return true
		}
	}
	return false
}

// Dispell 對場上每一隻活著的不死生物各擲一次。回傳訊息;被驅散掉的怪物
// 生命值歸零。
//
// ⚠ **逐隻各擲一次**,不是「一次判定全場」(docs/re/188 §4 的迴圈)。
// ⚠ 擲骰是 `INT(RND × 100) + 1`(值域 1…100,配了 `INT 3D:03` 截尾)——
// 與命中擲骰的 `round(RND×100+1)` **不是同一個成語**(docs/re/185),
// 所以這裡用 Roll(100) 而不是 rollRound。
func (f *Field) Dispell(intellect int) []string {
	var out []string
	for i := MonsterBase; i < MonsterBase+MonsterMax; i++ {
		u := f.Units[i]
		if !u.Alive() || !IsUndead(u) {
			continue
		}
		out = append(out, MsgAttemptingDispell+u.Name)
		if float64(f.Rand.Roll(ToHitFaces)) <= DispellThreshold(intellect, u.Tier) {
			f.Units[i].HP = 0
			out = append(out, u.Name+MsgDispellDies)
		} else {
			out = append(out, u.Name+MsgIgnoresPriest)
		}
	}
	return out
}

// 驅散的訊息字面,照 `translations/module-text/CMBT.tsv` 第 142–159 列(F3)。
const (
	MsgNoPriesthood   = "沒有降魔術,無法驅散。"       // 142+143+144+145
	MsgAlreadyDispell = "這回合已經驅散過了。"         // 146+147+148
	MsgDispellNotWiz  = "不是巫師,無法驅散。"         // 149+150+151+152
	MsgNoUndead       = "這些怪物都不是不死生物!"      // 153+154+155
	MsgAttemptingDispell = "正試圖驅散 "               // 156+157
	MsgIgnoresPriest  = "　無視祭司。"                 // 158
	MsgDispellDies    = "　死去。"                     // 159
)

// PriesthoodSkill 是「降魔術」在**巫師表**的技能編號(`CHARS.DAT` 位移 51)。
//
// ⚠ 同一段位移在**戰士表**是 `Persuasion`(docs/re/94 §1)——
// 查表之前要先確定職業,查錯表的結論會自洽而錯(docs/re/188 §1)。
const PriesthoodSkill = 10

// WizardClassChar 是巫師的職業字元(`CHARS.DAT` 位移 15)。
const WizardClassChar = '2'

// HasPriesthood 回傳這位角色有沒有降魔術。⚠ **只有巫師算數** ——
// 戰士的同一格是說服,不是降魔術。
func HasPriesthood(c original.Character) bool {
	if c.Class != WizardClassChar {
		return false
	}
	return skill(c, PriesthoodSkill) > 0
}
