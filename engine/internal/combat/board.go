package combat

import "shardofspring/internal/rules"

// 戰場。docs/spec/12-combat-board.md。
//
// 座標 (0,0) 在左上,x 往東、y 往南 —— 戰場不索引任何原版檔案,
// 所以取最不容易寫反的一種。

// BoardW 是格陣列的寬度。**讀出來的**:原版索引是 `列 × 31 + 欄`
// (docs/re/164 §1,三處 `mov dx, 31` + `imul`)。
//
// ⚠ 手冊 p.33 的「15×15」講的是**畫面**,不是這個陣列 ——
// 隊伍站在 x ∈ {12,13,14},只有 15 寬的話會貼在右下角,
// 而原版截圖是下方置中(docs/re/164 §4)。
const BoardW = 31

// BoardH 是高度。⚠ **未解**(docs/re/164 §5):原版的通行判定讀格值(< 33),
// 沒有任何座標界限的字面值可以量。這裡取正方形,是**具名假設**。
const BoardH = BoardW

// ViewW / ViewH 是畫面上看得到的視窗:**9 × 9 圖塊**,與迷宮／世界地圖相同。
//
// **這是量到的**(docs/re/175):實跑一場戰鬥,遊戲視窗內寬 153 px = 9 × 17,
// 畫面上的地形色塊 119 px = 7 × 17 —— 兩個獨立量測都是圖塊尺寸的整數倍。
//
// ⚠ 手冊 p.33 的「15×15」對不上視野(9)也對不上列距(31),仍未解。
// ⛔ 先前有一版把這裡寫成 15 —— 那是**用新推論換掉舊推論**,而舊的才是對的。
// 要改就要先量。
//
// ⚠ 視窗**跟著誰捲動**沒有讀到,引擎跟著目前行動的單位。
const (
	ViewW = 9
	ViewH = 9
)

// ViewOrigin 回傳以 (cx,cy) 為中心的視窗左上角,夾在盤內。
func ViewOrigin(cx, cy int) (x0, y0 int) {
	clamp := func(v, view, size int) int {
		v -= view / 2
		if v < 0 {
			return 0
		}
		if v > size-view {
			return size - view
		}
		return v
	}
	return clamp(cx, ViewW, BoardW), clamp(cy, ViewH, BoardH)
}

// OnEdge 回傳這一格是不是最外圈(踩上去可以離場)。
//
// ⚠ 原版**不是這樣判的** —— 它讀格陣列 `ds:6AD4` 的值(docs/re/164 §5),
// 而那份內容是執行期建的,靜態讀不到。用座標近似是**實作決定**。
func OnEdge(x, y int) bool {
	return x == 0 || y == 0 || x == BoardW-1 || y == BoardH-1
}

// InBoard 回傳座標是否在盤內。
func InBoard(x, y int) bool {
	return x >= 0 && x < BoardW && y >= 0 && y < BoardH
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

// AreaRadius 是群體傷害的作用半徑:以游標為中心**兩軸各 ±2**,合起來 5×5
// (docs/re/195 §2 —— `CMBT 0x15A19` 的 `游標 + 2` 與 `0x15A1C` 的 `游標 − 2`)。
//
// ⚠ 原版畫的範圍框(`0x159C2` 的 1…5 × 1…5 雙重迴圈)與判定用的是**同一塊**。
const AreaRadius = 2

// UnitsInArea 回傳作用範圍內、活著而且在場的單位索引。
//
// ⚠ **「類別 1 不選目標」不等於「打敵方全部」**(docs/re/195 §2):
// 它跳過的是「指定一個單位」,游標仍然決定範圍。
func (f *Field) UnitsInArea(cx, cy int) []int {
	var out []int
	for i, u := range f.Units {
		if !u.Alive() || !u.OnField() {
			continue
		}
		if abs(u.X-cx) <= AreaRadius && abs(u.Y-cy) <= AreaRadius {
			out = append(out, i)
		}
	}
	return out
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

// PartyBaseX / PartyBaseY 是隊伍陣型的基準,**兩軸共用同一個字面值 13**
// (docs/re/164 §2:`add bx, 0Dh` 出現在欄與列兩條路徑上)。
const (
	PartyBaseX = 13
	PartyBaseY = 13
)

// Place 把單位擺到初始位置。
//
// 隊伍是**讀出來的**(docs/re/160 的陣型 + docs/re/164 的基準 13)。
//
// ⚠ **怪物那一半是近似**:原版是「擲一組隨機座標 → 查格值是不是空的 →
// 不合就重擲」(docs/re/164 §3),而兩個擲骰範圍是執行期變數,未解。
// 這裡用可預測的排列填,**不假裝那是原版的分佈**。
func (f *Field) Place() {
	const baseX, baseY = PartyBaseX, PartyBaseY
	n := 0
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		if !f.Units[i].Alive() {
			continue
		}
		// ⚠ **站位看的是 `GROUPS.DAT` 的槽號,不是「隊伍裡的第幾個人」**
		// (docs/re/210):原版的迴圈跑槽 1…9,用槽號去查偏移表。
		// 兩者只有在「搬到有間隔的槽」時才不同 —— 例如只有 A 與 G 有人,
		// 照槽號站是一個在上排、一個在下排,照人數順序站則兩個並排在上排。
		slot := n + 1
		if n < len(f.PartySlots) {
			slot = f.PartySlots[n]
		}
		dx, dy := PartyOffset(slot)
		f.Units[i].X, f.Units[i].Y = baseX+dx, baseY+dy
		f.Units[i].Facing = North
		n++
	}
	for i := MonsterBase; i < MonsterBase+MonsterMax; i++ {
		if !f.Units[i].Alive() {
			continue
		}
		x, y := f.rollMonsterSpot()
		f.Units[i].X, f.Units[i].Y = x, y
		f.Units[i].Facing = South // 常數 3:出場全部面南(docs/re/96)
	}
}

// MonsterAnchors 是怪物出場的 8 個錨點(docs/re/186 §1)——
// `CMBT 0x115B1`–`0x11611` 的字面常數,**讀出來的**。
//
// 排出來是 3×3 的格局而**中間那格空著**:中間 (11,11) 正是隊伍所在的區塊
// (隊伍基準 13,13)。錨點 6/11/16 各加 0…4 的抖動,接起來就是 6…20 的
// 15 格 —— 手冊 p.33 的「15×15」因此不只是畫面,也是怪物實際會出現的範圍。
var MonsterAnchors = [8][2]int{
	{6, 6}, {11, 6}, {16, 6},
	{6, 11} /* 中間留給隊伍 */, {16, 11},
	{6, 16}, {11, 16}, {16, 16},
}

const (
	// MonsterAnchorFaces:`INT(RND × 7) + 1` 的面數。
	//
	// ⚠ **值域是 1…7,而錨點表有 8 列** —— 第 8 列 (16,16) 永遠挑不到。
	// 常數 `ds:94B4 = 7.0` 是讀出來的;「原版是不是本來想寫 8」無從得知,
	// 所以照抄(docs/re/186 §1.1)。
	MonsterAnchorFaces = 7
	// MonsterJitterFaces:每軸再加 `INT(RND × 5)` = 0…4(`ds:94B8 = 5.0`)。
	MonsterJitterFaces = 5
	// monsterPlaceTries 是重擲上限。⚠ **原版沒有上限**,湊不到就當掉;
	// 引擎不能當,所以有限次之後改用線性掃描找空格 —— 那是**本引擎的選擇**,
	// 不是原版行為。
	monsterPlaceTries = 200
)

// rollMonsterSpot 依 docs/re/186 §1 擲一個出場位置。
func (f *Field) rollMonsterSpot() (int, int) {
	for try := 0; try < monsterPlaceTries; try++ {
		a := MonsterAnchors[f.Rand.Roll(MonsterAnchorFaces)-1]
		x := a[0] + f.Rand.Roll(MonsterJitterFaces) - 1
		y := a[1] + f.Rand.Roll(MonsterJitterFaces) - 1
		if InBoard(x, y) && f.Occupant(x, y) < 0 {
			return x, y
		}
	}
	// 退路:從錨點區掃第一個空格。⚠ 原版不會走到這裡(它會一直重擲)。
	for _, a := range MonsterAnchors {
		for dy := 0; dy < MonsterJitterFaces; dy++ {
			for dx := 0; dx < MonsterJitterFaces; dx++ {
				x, y := a[0]+dx, a[1]+dy
				if InBoard(x, y) && f.Occupant(x, y) < 0 {
					return x, y
				}
			}
		}
	}
	return MonsterAnchors[0][0], MonsterAnchors[0][1]
}

// Retarget 依 docs/re/186 §2 更新一隻怪物鎖定的目標(屬性 15)。
//
// 規則:**目標倒下(狀態陣亡)或離場才重挑**,否則保持不變 ——
// 這就是手冊 p.35 的 `TACTICS` 看得到的那個對象。
// 重挑是 `INT(RND × 隊伍人數) + 9`,也就是**在隊員裡均勻隨機**
// (`ds:34F8` = 隊伍人數、`ds:94D2 = 9.0` = 隊員起始索引)。
//
// 回傳鎖定的單位索引;找不到任何可鎖的隊員回 −1。
func (f *Field) Retarget(i int) int {
	if t := f.Units[i].Target; t >= PartyBase && t < PartyBase+PartyMax {
		if u := f.Units[t]; u.Alive() && u.OnField() {
			return t // 還活著也還在場上 → 不換
		}
	}
	// 只在**還能鎖定的**隊員裡挑 —— 原版擲的是隊伍人數,而死掉的人
	// 仍然佔著槽位;擲到死人時原版下一輪會再擲,引擎直接跳過,結果同形。
	var live []int
	for u := PartyBase; u < PartyBase+PartyMax; u++ {
		if f.Units[u].Alive() && f.Units[u].OnField() {
			live = append(live, u)
		}
	}
	if len(live) == 0 {
		f.Units[i].Target = 0
		return -1
	}
	t := live[f.Rand.Roll(len(live))-1]
	f.Units[i].Target = t
	return t
}

// axisBias 回傳這個單位的軸向偏好(屬性 18),第一次用時擲硬幣決定。
//
// `CMBT 0x13E8E`:**只有還是 0 才擲**,所以同一個單位整場的偏好是穩定的
// (docs/re/158 §1)。
func (f *Field) axisBias(i int) int {
	if f.Units[i].Bias == 0 {
		f.Units[i].Bias = -1
		if f.Rand.Float01() < BiasCoin {
			f.Units[i].Bias = 1
		}
	}
	return f.Units[i].Bias
}

// stepToward 回傳「在這一軸上朝目標走」該面哪個方向;同一格回 false。
func stepToward(horizontal bool, u, t Unit) (Facing, bool) {
	if horizontal {
		switch {
		case u.X < t.X:
			return East, true
		case u.X > t.X:
			return West, true
		}
		return North, false
	}
	switch {
	case u.Y < t.Y:
		return South, true
	case u.Y > t.Y:
		return North, true
	}
	return North, false
}

// MonsterTurn 讓一隻怪物用完它的行動點數:朝**鎖定的目標**靠近,相鄰就攻擊。
//
// 逐步的選格照 docs/re/215:
//
//	先軸 = 屬性 18 為 +1 → 東西;−1 → 南北
//	該軸上自己與目標的座標一比,決定往哪個方向;相等就換另一軸
//	要走的那一格站不了 → **屬性 18 取負**(下一步改先試另一軸)
//
// ⚠ **偏好不是隨機抖動,是「這隻怪習慣先橫走還是先直走」** ——
// 整場固定,只有撞牆才翻面。⛔ 不要改成每步重擲:那會讓怪物在牆邊原地抖。
func (f *Field) MonsterTurn(p *Points, i int) {
	for p[i] > 0 {
		u := f.Units[i]
		if !u.Alive() || !u.OnField() {
			return
		}
		j := f.Retarget(i)
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
		// 否則照軸向偏好選一軸靠近一格(docs/re/215)
		horizontal := f.axisBias(i) > 0
		moved := false
		for try := 0; try < 2; try++ {
			dir, ok := stepToward(horizontal, u, t)
			horizontal = !horizontal // 這一軸不成就換另一軸
			if !ok {
				continue
			}
			if f.Turn(p, i, dir) != ActOK {
				return
			}
			if f.Step(p, i) == ActOK {
				moved = true
				break
			}
			// 站不了 → 偏好取負(`CMBT 0x140DE` 的 neg)。
			// ⚠ 點數已經花在轉身上,這一步就結束了。
			f.Units[i].Bias = -f.Units[i].Bias
			return
		}
		if !moved {
			return // 兩軸都沒得走(已經在同一格)——不要空轉
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
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
