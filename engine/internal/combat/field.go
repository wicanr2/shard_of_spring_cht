package combat

import (
	"sort"
	"strconv"
)

// Field 是一場戰鬥。docs/spec/07-combat-scene.md。
type Field struct {
	// Tactics = 隊上有人會「策略」技能 → 畫面可以顯示怪物的鎖定對象
	// (手冊 p.35、docs/re/186 §2)。**只影響顯示。**
	Tactics bool
	Units   [Slots]Unit
	Order   []int // 先攻表:單位編號,速度高的在前
	Round   int
	// ⚠ 型別是 FloatRand 不是 Rand —— 命中與狂暴擲骰要用 Float01()
	// 算 round(RND×N+1)(docs/re/185),不能只靠 Roll()。
	Rand FloatRand
	// Items 是查表用的:編號 → ITEMS.DAT 的兩個欄位。
	Items map[int]Item
	// Log 是這一場的訊息,給訊息列顯示、也給測試檢查。
	Log []string
	// PartySlots 是每位隊員佔 `GROUPS.DAT` 的第幾個成員槽(1–9),
	// 與 `Units[PartyBase…]` 同序。**站位看的是它**(docs/re/210)。
	// 空的時候退回「隊伍裡的第幾個人」—— 只有搬到有間隔的槽時兩者才不同。
	PartySlots []int
}

// 戰鬥訊息的字面。**逐字照 `translations/module-text/CMBT.tsv` 第 69–82 列**
// (F3,docs/spec/19 §1)——畫面要說原版說的話,不是實作時自己寫的中文。
//
// 原版把一次攻擊拼成一句:
//
//	<攻擊者> attacks <目標> with <武器> and hits for <N> damage. It dies!
//
// 兩個岔路都是讀出來的:`and hacks for`／`and hits for` 由狂暴決定
// (docs/re/153 §7),`for no damage.` 由 `傷害 < 1` 決定(docs/re/153 §8)。
// 死亡那一句分「他」與「牠」兩種,原版用的就是兩段不同的字串。
const (
	msgAttacks  = " 攻擊 "         // 69 `attacks `
	msgWith     = " 使用 "         // 70 `with`
	msgMissed   = "但沒打中!"      // 71 `and missed!`
	msgNoDamage = "沒有造成傷害。" // 72 `for no damage.`
	msgHacksFor = "劈砍造成 "      // 74 `and hacks for `
	msgHitsFor  = "命中造成 "      // 76 `and hits for `
	msgDamage   = " 點傷害。"      // 77 ` damage.`
	msgHeDies   = " 他死了!"       // 81 ` He Dies!`(隊員)
	msgItDies   = " 牠死了!"       // 82 ` It dies!`(怪物)
	msgPoisoned = "並中毒了!"      // 79 `and is poisoned!`(docs/re/191)
	// 5/7 `Hands` —— 武器編號 60 的名字,兩列分別是「已鑑定」與「未鑑定」
	// 兩張表的第 60 格,而**兩格的原文都是 `Hands`**(docs/re/192 §4),
	// 所以引擎只用一個「拳頭」不會漏掉任何一種說法。
	msgBareHands = "拳頭"
)

// naturalWeapons 是武器編號 59–62 的名字。
//
// 那四格**不在 `ITEMS.DAT` 裡**(檔案只有 57 筆)—— 是模組開場在
// `CMBT 0x10463` 手工補進名稱陣列尾巴的(docs/re/192 §2)。
//
// ⚠ **這張表是武器與防具共用的** —— 屬性 4 與屬性 5 查同一個陣列
// (docs/re/75 §1:未裝備時屬性 4 填 60、屬性 5 填 59)。
// ⚠ **只有 61 找不到來源**:怪物欄5 是 0–13 與 62、角色走 0–56 與兩個哨兵。
// ⛔ 不要為了把它用上去編一條「爬蟲用咬擊」的規則。
var naturalWeapons = map[int]string{
	59: "無",         // 9  `None` —— 沒穿防具
	60: msgBareHands, // 5  `Hands` —— 沒拿武器
	61: "咬擊",       // 10 `Bite` —— 沒有任何資料產得出這個編號
	62: "獠牙",       // 6  `Fangs` —— 蛇與食屍鬼,就是中毒那五隻(docs/re/191)
}

// Outcome 是戰鬥的結束狀態。
type Outcome int

const (
	Ongoing      Outcome = iota
	MonstersDead         // 'MONSTERS ALL DEAD'
	PartyDead            // 'PARTY DIES!'
	PartyRan             // 'PARTY RAN!'
)

// docs/spec/19-module-text.md(F1):字面照 translations/module-text/CMBT.tsv
// 第 23/24/29 列(「MONSTERS ALL DEAD」/「PARTY DIES!」/「PARTY RAN!」)。
func (o Outcome) String() string {
	switch o {
	case MonstersDead:
		return "怪物全滅"
	case PartyDead:
		return "隊伍全滅!"
	case PartyRan:
		return "隊伍逃跑了!"
	}
	return "進行中"
}

// item 查一件裝備;查不到回零值(= 沒有加值、沒有傷害)。
func (f *Field) item(id int) Item { return f.Items[id] }

// Sort 依速度排先攻表。docs/spec/01 §2:速度高的先動。
//
// ⚠ **排序必須穩定**(docs/spec/07 §8 驗收 6)。速度相同時若順序不定,
// 「同一顆種子跑兩次結果相同」就不成立 —— 而那正是 M4 的驗收條件。
// Go 的 map 迭代是隨機的,所以先攻表**只能從索引順序建**,不能從 map。
func (f *Field) Sort() {
	// ⚠ **每次都從索引順序重建**,不是拿上一次的 Order 再排一次
	// (docs/re/159 §3:原版排序前先把順序表填成 0…13)。
	// 穩定排序下兩者不同 —— 沿用舊順序會讓上一回合的次序變成同速時的排法。
	f.Order = f.Order[:0]
	for i := range f.Units {
		u := f.Units[i]
		if u.Alive() && u.OnField() {
			f.Order = append(f.Order, i)
		}
	}
	sort.SliceStable(f.Order, func(a, b int) bool {
		return f.Units[f.Order[a]].Speed > f.Units[f.Order[b]].Speed
	})
}

// Attack 讓 atk 攻擊 def,回傳(擲骰、是否命中、傷害)。
func (f *Field) Attack(atk, def int) (roll int, hit bool, dmg int) {
	a, d := f.Units[atk], f.Units[def]
	aw, da := f.item(a.Weapon), f.item(d.Armor)
	roll, hit = Hits(a, d, aw, da, f.Rand, ToHitFaces)
	head := a.Name + msgAttacks + d.Name + f.weaponPhrase(a)
	if !hit {
		f.Log = append(f.Log, head+msgMissed)
		return roll, false, 0
	}
	dmg = Damage(a, d, aw, da, f.Rand)
	// 狂暴:同一次攻擊的**第二次擲骰** > 75 且攻擊者有屬性 16(docs/re/153 §7)。
	// 原版把兩次擲骰都算在同一次攻擊裡,所以這裡一定要再擲一次,
	// 不能沿用命中那一次的值 —— 沿用會讓「命中很險」與「打得很重」綁在一起。
	//
	// ⚠ 擲骰**無條件進行**,即使傷害是 0。挪到 `dmg > 0` 底下會改變亂數的
	// 消耗順序,同一顆種子就跑出不同的戰鬥(docs/spec/07 §8 驗收 6)。
	verb := msgHitsFor
	// ⚠ round(RND×100+1),不是 Roll(100)——同一個成語,同一個修正
	// (docs/re/185 §2 表列 #4)。
	if second := rollRound(f.Rand, ToHitFaces); Berserk(a, second) {
		dmg *= 2
		verb = msgHacksFor
	}
	Apply(&f.Units[def], dmg)
	msg := head
	if dmg == 0 {
		msg += msgNoDamage
	} else {
		msg += verb + strconv.Itoa(dmg) + msgDamage
	}
	if f.Units[def].HP == 0 {
		if d.IsMonster {
			msg += msgItDies
		} else {
			msg += msgHeDies
		}
	}
	// 中毒判定排在死亡判定**之後** —— 原版打死了就直接走死亡分支,
	// 不擲那顆骰(docs/re/191 §1)。poison() 自己擋活著這一項。
	msg += f.poison(atk, def)
	f.Log = append(f.Log, msg)
	return roll, true, dmg
}

// ArmorRating 是這個單位的防護總量:防具的欄4 + 護甲技能。
//
// ⚠ 這兩項就是傷害公式裡的減項(docs/spec/01 §5 的 `− 護甲技能 − 防具值`)——
// 面板顯示的數字與公式用的是**同一份**,不是另外算一套。
func (f *Field) ArmorRating(i int) int {
	if i < 0 || i >= len(f.Units) {
		return 0
	}
	u := f.Units[i]
	return f.item(u.Armor).Main + u.ArmSkin
}

// WeaponName 是這個單位的攻擊方式(面板的 `Attacks with:`)。
func (f *Field) WeaponName(i int) string {
	if i < 0 || i >= len(f.Units) {
		return msgBareHands
	}
	return f.weaponName(f.Units[i].Weapon, f.Units[i].WeaponKnown)
}

// weaponName 把武器編號翻成名字:先查 `ITEMS.DAT`,查不到再查自然武器表。
//
// 原版兩張表是**同一個陣列**(編號 0–56 從檔案讀、59–62 手工補),
// 所以這裡的兩段查表其實是一段(docs/re/192 §2)。
func (f *Field) weaponName(w int, known bool) string {
	it := f.item(w)
	// 未鑑定 → 小寫名(docs/re/192 §4:原版兩條分支只差查表的基底)。
	// ⚠ 小寫名空白時退回正式名 —— 缺欄位不該讓武器變成「赤手空拳」。
	if !known && it.Alias != "" {
		return it.Alias
	}
	if n := it.Name; n != "" {
		return n
	}
	if n, ok := naturalWeapons[w]; ok {
		return n
	}
	return msgBareHands
}

// weaponPhrase 回傳「 使用 <武器>」。原版那句 `with` 是無條件印的。
//
// 武器格 59–62 是自然武器(docs/re/192),99 是「沒裝備」的哨兵 ——
// 兩張表都查不到才退回 msgBareHands。
func (f *Field) weaponPhrase(u Unit) string {
	return msgWith + f.weaponName(u.Weapon, u.WeaponKnown)
}

// Outcome 判定戰鬥是否結束。docs/spec/07 §6。
//
// ⚠ 三個條件用**兩個不同的欄位**:全滅看生命值、逃離看朝向。
// 合併成「還能行動的成員數」會讓逃跑判定失效。
func (f *Field) Outcome() Outcome {
	monstersAlive, partyAlive, partyOnField := false, false, false
	for i := MonsterBase; i < MonsterBase+MonsterMax; i++ {
		if f.Units[i].Alive() {
			monstersAlive = true
		}
	}
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		if f.Units[i].Alive() {
			partyAlive = true
		}
		if f.Units[i].OnField() {
			partyOnField = true
		}
	}
	switch {
	case !partyAlive:
		return PartyDead
	case !partyOnField:
		return PartyRan // 還活著,但沒有人在場上
	case !monstersAlive:
		return MonstersDead
	}
	return Ongoing
}

// ActionCost 是一個行動要花的點數。docs/spec/01 §3。
const (
	TurnCost   = 1 // 轉身
	ActionCost = 3 // 其他行動
)
