package combat

import "shardofspring/internal/rules"

// 戰場。docs/spec/12-combat-board.md。
//
// 15×15,四周一圈是離場用的「圓點」(手冊 p.33)。
// 座標 (0,0) 在左上,x 往東、y 往南 —— 戰場不索引任何原版檔案,
// 所以取最不容易寫反的一種。

const BoardSize = 15

// OnEdge 回傳這一格是不是最外圈(踩上去可以離場)。
func OnEdge(x, y int) bool {
	return x == 0 || y == 0 || x == BoardSize-1 || y == BoardSize-1
}

// InBoard 回傳座標是否在盤內。
func InBoard(x, y int) bool {
	return x >= 0 && x < BoardSize && y >= 0 && y < BoardSize
}

// delta 回傳朝向的位移。Absent(0)沒有方向。
func (f Facing) delta() (int, int) {
	switch f {
	case North:
		return 0, -1
	case East:
		return 1, 0
	case South:
		return 0, 1
	case West:
		return -1, 0
	}
	return 0, 0
}

// Points 是這一回合每個單位剩下的行動點數,索引與 Units 相同。
type Points [Slots]int

// ResetPoints 依速度重新給點。docs/spec/12 §2:**點數 = 速度**。
func (f *Field) ResetPoints(p *Points) {
	for i, u := range f.Units {
		if u.Alive() && u.OnField() {
			p[i] = rules.MovePoints(u.Speed)
		} else {
			p[i] = 0
		}
	}
}

// Occupant 回傳站在 (x,y) 的單位編號;沒有人回 -1。
//
// ⚠ 只看**在場且活著**的單位 —— 死者與離場者不佔位置,
// 否則屍體會把格子擋住,而畫面上看不出來為什麼走不過去。
func (f *Field) Occupant(x, y int) int {
	for i, u := range f.Units {
		if u.Alive() && u.OnField() && u.X == x && u.Y == y {
			return i
		}
	}
	return -1
}

// ActResult 說明一個動作為什麼沒做成。
type ActResult int

const (
	ActOK ActResult = iota
	ActNoPoints
	ActBlocked  // 目標格有人或出界
	ActNoTarget // 面前沒有敵人
)

func (r ActResult) String() string {
	switch r {
	case ActNoPoints:
		return "行動點數不足"
	case ActBlocked:
		return "那一格過不去"
	case ActNoTarget:
		return "面前沒有敵人"
	}
	return ""
}

// spend 扣點數。點數不足時**不扣也不做事**(docs/spec/12 §7 驗收 2)。
func (p *Points) spend(i int, a rules.Action) bool {
	c := a.Cost()
	if p[i] < c {
		return false
	}
	p[i] -= c
	return true
}

// Turn 讓第 i 個單位轉向。成本 1。
func (f *Field) Turn(p *Points, i int, dir Facing) ActResult {
	if !f.Units[i].OnField() {
		return ActBlocked
	}
	if f.Units[i].Facing == dir {
		return ActOK // 已經朝那邊,不收費也不算錯
	}
	if !p.spend(i, rules.ActTurn) {
		return ActNoPoints
	}
	f.Units[i].Facing = dir
	return ActOK
}

// Step 讓第 i 個單位往目前朝向前進一格。成本 2。
//
// 走上最外圈 → **離場**(朝向設 0,docs/spec/12 §4)。
// ⚠ 離場不是死亡 —— 生命值不動。
func (f *Field) Step(p *Points, i int) ActResult {
	u := &f.Units[i]
	if !u.OnField() {
		return ActBlocked
	}
	dx, dy := u.Facing.delta()
	nx, ny := u.X+dx, u.Y+dy
	if !InBoard(nx, ny) {
		return ActBlocked
	}
	if f.Occupant(nx, ny) >= 0 {
		return ActBlocked
	}
	if !p.spend(i, rules.ActMove) {
		return ActNoPoints
	}
	u.X, u.Y = nx, ny
	if OnEdge(nx, ny) {
		u.Facing = Absent // 離場
		p[i] = 0
		f.Log = append(f.Log, u.Name+" 離開了戰場")
	}
	return ActOK
}

// StrikeFront 攻擊朝向的那一格。成本 3。
//
// docs/spec/12 §3:**沒有目標時不扣點數** ——
// 扣了會讓玩家對著空氣揮拳把回合耗光,而畫面上只看到「什麼都沒發生」。
func (f *Field) StrikeFront(p *Points, i int) ActResult {
	u := f.Units[i]
	if !u.OnField() {
		return ActBlocked
	}
	dx, dy := u.Facing.delta()
	j := f.Occupant(u.X+dx, u.Y+dy)
	if j < 0 || f.Units[j].IsMonster == u.IsMonster {
		return ActNoTarget
	}
	if !p.spend(i, rules.ActAttack) {
		return ActNoPoints
	}
	f.Attack(i, j)
	return ActOK
}

// PartyRowWidth 是隊伍佈陣的每列人數。原版 `CMBT 0x11783` 的
// `mov cx, 3` + `idiv` —— **那個 3 是直接讀到的**(docs/re/160)。
const PartyRowWidth = 3

// PartyOffset 回傳第 i 位隊員(i 從 1 起)相對於陣型基準的欄 / 列偏移。
//
//	欄 = (i−1) mod 3 − 1     → −1 / 0 / 1
//	列 = (i−1) ÷ 3
//
// 五個人因此排成上排三個、下排靠左兩個 —— 與原版截圖逐格吻合
// (docs/re/160 §2)。
func PartyOffset(i int) (dx, dy int) {
	n := i - 1
	return n%PartyRowWidth - 1, n / PartyRowWidth
}

// Place 把單位擺到初始位置。
//
// 隊伍的陣型是**讀出來的**(docs/re/160):每列三個。
// ⚠ 陣型的**基準列**(原版 `ds:94C2`)未解 —— 它只把整塊陣型上下平移,
// 這裡取「下半、避開最外圈」:一開始就站在圓點上等於還沒開打就能逃。
//
// ⚠ **怪物那一半仍是佔位** —— 原版怪物的佈陣沒有讀到。
func (f *Field) Place() {
	const baseX, baseY = 7, BoardSize - 4 // 基準:置中偏下
	slot := 1
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		if !f.Units[i].Alive() {
			continue
		}
		dx, dy := PartyOffset(slot)
		f.Units[i].X, f.Units[i].Y = baseX+dx, baseY+dy
		f.Units[i].Facing = North
		slot++
	}
	mx, my := 5, 2
	for i := MonsterBase; i < MonsterBase+MonsterMax; i++ {
		if !f.Units[i].Alive() {
			continue
		}
		f.Units[i].X, f.Units[i].Y = mx, my
		f.Units[i].Facing = South
		mx++
		if mx >= BoardSize-1 {
			mx, my = 5, my+1
		}
	}
}

// MonsterTurn 是怪物的佔位 AI:朝最近的隊員直線靠近,相鄰就攻擊。
//
// ⚠ **這不是原版的策略**(docs/spec/12 §5)。手冊 p.35 提到 `TACTICS`
// 技能可以看出「怪物正追蹤哪一個同伴」—— 所以原版是**每隻怪物鎖定一個人**,
// 形狀相容,但選法不同。可預測比「看起來聰明」重要。
func (f *Field) MonsterTurn(p *Points, i int) {
	for p[i] > 0 {
		u := f.Units[i]
		if !u.Alive() || !u.OnField() {
			return
		}
		j := f.nearestFoe(i)
		if j < 0 {
			return
		}
		t := f.Units[j]
		// 相鄰就轉向並攻擊
		if abs(t.X-u.X)+abs(t.Y-u.Y) == 1 {
			if f.Turn(p, i, dirTo(u.X, u.Y, t.X, t.Y)) != ActOK {
				return
			}
			if f.StrikeFront(p, i) != ActOK {
				return
			}
			continue
		}
		// 否則往差距大的那一軸靠近一格
		dir := approach(u.X, u.Y, t.X, t.Y)
		if f.Turn(p, i, dir) != ActOK {
			return
		}
		if f.Step(p, i) != ActOK {
			return
		}
	}
}

func (f *Field) nearestFoe(i int) int {
	u := f.Units[i]
	best, bestD := -1, 1<<30
	lo, hi := PartyBase, PartyBase+PartyMax
	if !u.IsMonster {
		lo, hi = MonsterBase, MonsterBase+MonsterMax
	}
	for j := lo; j < hi; j++ {
		v := f.Units[j]
		if !v.Alive() || !v.OnField() {
			continue
		}
		if d := abs(v.X-u.X) + abs(v.Y-u.Y); d < bestD {
			best, bestD = j, d
		}
	}
	return best
}

func dirTo(x, y, tx, ty int) Facing {
	switch {
	case ty < y:
		return North
	case ty > y:
		return South
	case tx > x:
		return East
	default:
		return West
	}
}

// approach 挑差距大的那一軸,平手時走橫的 —— 固定規則,不擲骰。
func approach(x, y, tx, ty int) Facing {
	if abs(tx-x) >= abs(ty-y) {
		if tx > x {
			return East
		}
		return West
	}
	if ty > y {
		return South
	}
	return North
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
