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
	// 5/7 `Hands` —— 沒有武器時填進 `with` 後面的那個名字。
	//
	// ⚠ **哪一種單位拿到哪一個名字沒有讀到。** CMBT 的字串表開頭並排著
	// `Hands` / `Fangs` / `Bite` / `None`(第 5–10 列),形狀像一張
	// 「沒有武器時該叫什麼」的小表,但選用的判斷式沒讀 —— 所以這裡一律用
	// 「拳頭」,⛔ 不自己編一條「怪物用獠牙、爬蟲用咬擊」的規則。
	msgBareHands = "拳頭"
)

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
	if n := f.item(f.Units[i].Weapon).Name; n != "" {
		return n
	}
	return msgBareHands
}

// weaponPhrase 回傳「 使用 <武器>」。
//
// 武器格用 60／99 當「沒有武器」的哨兵(docs/spec/01 §5、docs/formats/03),
// 而 `ITEMS.DAT` 只有 0–56 有名字 —— 查不到名字就填 msgBareHands,
// 原版那句 `with` 是無條件印的。
func (f *Field) weaponPhrase(u Unit) string {
	name := f.item(u.Weapon).Name
	if name == "" {
		name = msgBareHands
	}
	return msgWith + name
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
