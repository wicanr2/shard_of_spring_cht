package combat

import "sort"

// Field 是一場戰鬥。docs/spec/07-combat-scene.md。
type Field struct {
	Units [Slots]Unit
	Order []int // 先攻表:單位編號,速度高的在前
	Round int
	Rand  Rand
	// Items 是查表用的:編號 → ITEMS.DAT 的兩個欄位。
	Items map[int]Item
	// Log 是這一場的訊息,給訊息列顯示、也給測試檢查。
	Log []string
}

// FullComma 是全形逗號。訊息一律用中文標點(docs/spec/06 §7)。
const FullComma = "，"

// Outcome 是戰鬥的結束狀態。
type Outcome int

const (
	Ongoing      Outcome = iota
	MonstersDead         // 'MONSTERS ALL DEAD'
	PartyDead            // 'PARTY DIES!'
	PartyRan             // 'PARTY RAN!'
)

func (o Outcome) String() string {
	switch o {
	case MonstersDead:
		return "怪物全滅"
	case PartyDead:
		return "全隊陣亡"
	case PartyRan:
		return "全隊逃離"
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
	if !hit {
		f.Log = append(f.Log, a.Name+" 攻擊 "+d.Name+"，落空")
		return roll, false, 0
	}
	dmg = Damage(a, d, aw, f.item(d.Armor), f.Rand)
	Apply(&f.Units[def], dmg)
	msg := a.Name + " 擊中 " + d.Name
	if f.Units[def].HP == 0 {
		msg += FullComma + d.Name + " 倒下"
	}
	f.Log = append(f.Log, msg)
	return roll, true, dmg
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
