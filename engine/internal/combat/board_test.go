package combat

import (
	"testing"

	"shardofspring/internal/rules"
)

// docs/spec/12 §7 的驗收,一條一個測試。

func boardField() (*Field, *Points) {
	f := &Field{Items: map[int]Item{}, Rand: &ScriptRand{}}
	f.Units[PartyBase] = Unit{
		Name: "我", HP: 10, Speed: 9, Facing: North, X: 7, Y: 10,
	}
	f.Units[MonsterBase] = Unit{
		Name: "怪", HP: 10, Speed: 6, Facing: South, X: 7, Y: 9, IsMonster: true,
	}
	var p Points
	f.ResetPoints(&p)
	return f, &p
}

func TestPointsEqualSpeed(t *testing.T) {
	f, p := boardField()
	if got := p[PartyBase]; got != f.Units[PartyBase].Speed {
		t.Errorf("行動點數應等於速度 %d,得 %d", f.Units[PartyBase].Speed, got)
	}
}

func TestCostsAreTwoOneThree(t *testing.T) {
	f, p := boardField()
	f.Units[MonsterBase].X = 0 // 把怪物移開,免得擋路
	start := p[PartyBase]

	f.Turn(p, PartyBase, East)
	if start-p[PartyBase] != 1 {
		t.Errorf("轉向應扣 1,扣了 %d", start-p[PartyBase])
	}
	before := p[PartyBase]
	f.Step(p, PartyBase)
	if before-p[PartyBase] != 2 {
		t.Errorf("前進應扣 2,扣了 %d", before-p[PartyBase])
	}
}

// 點數不足時動作**做不出來**,而不是變成負數。
func TestNoPointsMeansNoAction(t *testing.T) {
	f, p := boardField()
	f.Units[MonsterBase].X = 0 // 先把路清開,這條測的是點數不是阻擋
	p[PartyBase] = 1
	if r := f.Step(p, PartyBase); r != ActNoPoints {
		t.Fatalf("點數 1 不該走得動,得 %v", r)
	}
	if p[PartyBase] != 1 {
		t.Errorf("失敗不該扣點數,剩 %d", p[PartyBase])
	}
	if f.Units[PartyBase].Y != 10 {
		t.Error("失敗不該移動")
	}
}

// 攻擊只打朝向的那一格;沒有目標時**不扣點數**。
func TestStrikeOnlyHitsTheFacedSquare(t *testing.T) {
	f, p := boardField()
	before := p[PartyBase]
	if r := f.StrikeFront(p, PartyBase); r != ActOK {
		t.Fatalf("正北方有怪,應打得到,得 %v", r)
	}
	if before-p[PartyBase] != 3 {
		t.Errorf("攻擊應扣 3,扣了 %d", before-p[PartyBase])
	}

	f.Units[PartyBase].Facing = South // 轉開,面前沒有東西
	before = p[PartyBase]
	if r := f.StrikeFront(p, PartyBase); r != ActNoTarget {
		t.Fatalf("面前沒人應回 ActNoTarget,得 %v", r)
	}
	if p[PartyBase] != before {
		t.Errorf("沒有目標不該扣點數,扣了 %d", before-p[PartyBase])
	}
}

func TestStrikeIgnoresOwnSide(t *testing.T) {
	f, p := boardField()
	f.Units[MonsterBase].X = 0
	f.Units[PartyBase+1] = Unit{Name: "同伴", HP: 5, Facing: South, X: 7, Y: 9}
	if r := f.StrikeFront(p, PartyBase); r != ActNoTarget {
		t.Errorf("面前是同伴不該打,得 %v", r)
	}
}

// 走上最外圈 = 離場。⚠ 離場不是死亡,生命值不能動。
func TestSteppingOnEdgeLeavesTheField(t *testing.T) {
	f, p := boardField()
	f.Units[MonsterBase].X = 0
	u := &f.Units[PartyBase]
	u.X, u.Y, u.Facing = 7, 1, North
	hp := u.HP
	if r := f.Step(p, PartyBase); r != ActOK {
		t.Fatalf("往北走到第 0 列應該成功,得 %v", r)
	}
	if f.Units[PartyBase].OnField() {
		t.Error("走上最外圈應離場")
	}
	if f.Units[PartyBase].HP != hp {
		t.Error("離場不該扣生命值 —— 離場與死亡是兩個欄位")
	}
	if !f.Units[PartyBase].Alive() {
		t.Error("離場的人仍然活著")
	}
}

func TestAllPartyOffFieldMeansRan(t *testing.T) {
	f, p := boardField()
	_ = p
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		f.Units[i].Facing = Absent
	}
	if got := f.Outcome(); got != PartyRan {
		t.Errorf("全隊離場應為 PartyRan,得 %v", got)
	}
}

// 兩個單位不能站同一格。
func TestUnitsCannotShareASquare(t *testing.T) {
	f, p := boardField()
	// 怪物就在正北方 (7,9)
	if r := f.Step(p, PartyBase); r != ActBlocked {
		t.Errorf("目標格有人應被擋,得 %v", r)
	}
}

// 死者不佔位置 —— 否則屍體會擋路,而畫面上看不出為什麼走不過去。
func TestDeadUnitsDoNotBlock(t *testing.T) {
	f, p := boardField()
	f.Units[MonsterBase].HP = 0
	if r := f.Step(p, PartyBase); r != ActOK {
		t.Errorf("死者不該擋路,得 %v", r)
	}
}

// 同一顆種子跑兩次,位置序列完全相同。
func TestBoardIsReproducible(t *testing.T) {
	run := func() [2]int {
		f := &Field{Items: map[int]Item{}, Rand: NewRand(1234)}
		f.Units[MonsterBase] = Unit{Name: "怪", HP: 50, Speed: 9, X: 7, Y: 2,
			Facing: South, IsMonster: true}
		f.Units[PartyBase] = Unit{Name: "我", HP: 50, Speed: 9, X: 7, Y: 10,
			Facing: North}
		var p Points
		for round := 0; round < 3; round++ {
			f.ResetPoints(&p)
			f.MonsterTurn(&p, MonsterBase)
		}
		return [2]int{f.Units[MonsterBase].X, f.Units[MonsterBase].Y}
	}
	if a, b := run(), run(); a != b {
		t.Errorf("同種子兩次的位置不同:%v vs %v", a, b)
	}
}

// 佈陣:每列三個,欄偏移 −1/0/1(docs/re/160)。
//
// ⚠ 對照的是原版截圖的形狀:上排三個、下排**靠左**兩個 ——
// 不是置中。置中在畫面上也很合理,而那正是分不開的地方,所以要逐格對。
func TestPartyOffsetThreePerRow(t *testing.T) {
	want := []struct{ dx, dy int }{
		{-1, 0}, {0, 0}, {1, 0}, // 上排三個
		{-1, 1}, {0, 1}, // 下排靠左兩個
		{1, 1}, {-1, 2},
	}
	for i, w := range want {
		dx, dy := PartyOffset(i + 1)
		if dx != w.dx || dy != w.dy {
			t.Errorf("第 %d 位偏移 (%d,%d),應為 (%d,%d)", i+1, dx, dy, w.dx, w.dy)
		}
	}
}

// 佈陣不可以把人放到最外圈 —— 站在圓點上等於還沒開打就能離場。
func TestPlaceKeepsPartyOffTheRim(t *testing.T) {
	f := &Field{}
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		f.Units[i] = Unit{HP: 5, Facing: South}
	}
	f.Place()
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		u := f.Units[i]
		if u.X <= 0 || u.X >= BoardW-1 || u.Y <= 0 || u.Y >= BoardH-1 {
			t.Errorf("第 %d 位站在 (%d,%d) —— 那是最外圈", i-PartyBase+1, u.X, u.Y)
		}
	}
	// 兩個人不可以站同一格
	seen := map[[2]int]bool{}
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		k := [2]int{f.Units[i].X, f.Units[i].Y}
		if seen[k] {
			t.Errorf("兩個單位站在同一格 %v", k)
		}
		seen[k] = true
	}
}

// 陣列寬 31、基準 13 是讀出來的(docs/re/164 §1/§2)——
// 這個測試會在有人把它改回 15 或改動基準時失敗。
func TestBoardStrideAndBase(t *testing.T) {
	if BoardW != 31 {
		t.Errorf("格陣列寬度 %d,原版索引是 列×31+欄(docs/re/164 §1)", BoardW)
	}
	if PartyBaseX != 13 || PartyBaseY != 13 {
		t.Errorf("隊伍基準 (%d,%d),原版兩軸都是 +13(docs/re/164 §2)",
			PartyBaseX, PartyBaseY)
	}
	f := &Field{}
	for i := PartyBase; i < PartyBase+5; i++ {
		f.Units[i] = Unit{HP: 5, Facing: South}
	}
	f.Place()
	// 五個人落在 x ∈ {12,13,14}、y ∈ {13,14}
	for i := PartyBase; i < PartyBase+5; i++ {
		u := f.Units[i]
		if u.X < 12 || u.X > 14 || u.Y < 13 || u.Y > 14 {
			t.Errorf("第 %d 位在 (%d,%d),應落在 x 12–14、y 13–14",
				i-PartyBase+1, u.X, u.Y)
		}
	}
}

// 視窗夾在盤內,而且不會露出盤外的格子。
func TestViewOriginClamps(t *testing.T) {
	for _, c := range [][4]int{
		{0, 0, 0, 0},
		{BoardW - 1, BoardH - 1, BoardW - ViewW, BoardH - ViewH},
		{15, 15, 15 - ViewW/2, 15 - ViewH/2},
	} {
		x, y := ViewOrigin(c[0], c[1])
		if x != c[2] || y != c[3] {
			t.Errorf("ViewOrigin(%d,%d) = (%d,%d),應為 (%d,%d)",
				c[0], c[1], x, y, c[2], c[3])
		}
	}
}

// 站位看的是 `GROUPS.DAT` 的**槽號**,不是「隊伍裡的第幾個人」(docs/re/210)。
//
// ⚠ 只有「搬到有間隔的槽」才分得出兩者。這裡刻意用槽 1 與槽 7(A 與 G):
// 照槽號一個在上排、一個在下排;照人數順序兩個並排在上排。
func TestPlaceUsesSlotNumberNotOrder(t *testing.T) {
	f := &Field{PartySlots: []int{1, 7}}
	f.Units[PartyBase] = Unit{Name: "甲", HP: 10}
	f.Units[PartyBase+1] = Unit{Name: "乙", HP: 10}
	f.Place()

	dx1, dy1 := PartyOffset(1)
	dx7, dy7 := PartyOffset(7)
	if got := [2]int{f.Units[PartyBase].X, f.Units[PartyBase].Y}; got !=
		[2]int{PartyBaseX + dx1, PartyBaseY + dy1} {
		t.Errorf("槽 1 站在 %v,想要 %v", got,
			[2]int{PartyBaseX + dx1, PartyBaseY + dy1})
	}
	if got := [2]int{f.Units[PartyBase+1].X, f.Units[PartyBase+1].Y}; got !=
		[2]int{PartyBaseX + dx7, PartyBaseY + dy7} {
		t.Errorf("槽 7 站在 %v,想要 %v", got,
			[2]int{PartyBaseX + dx7, PartyBaseY + dy7})
	}
	// 正對照:兩者真的不同,否則這條測試分不出來
	if dy1 == dy7 {
		t.Fatal("槽 1 與槽 7 落在同一列 —— 這組值分不出「槽號」與「順序」")
	}
}

// 沒有槽號資訊時退回「隊伍裡的第幾個人」——舊行為,不要變成 (0,0)。
func TestPlaceFallsBackWithoutSlots(t *testing.T) {
	f := &Field{}
	f.Units[PartyBase] = Unit{Name: "甲", HP: 10}
	f.Place()
	dx, dy := PartyOffset(1)
	if f.Units[PartyBase].X != PartyBaseX+dx || f.Units[PartyBase].Y != PartyBaseY+dy {
		t.Errorf("沒有槽號時應該照順序站第一格,得到 (%d,%d)",
			f.Units[PartyBase].X, f.Units[PartyBase].Y)
	}
}

// ── 逐步選格:軸向偏好(docs/re/215)────────────────────────────────────

// biasField 擺一隻怪與一名隊員,兩軸都有差距 —— 只有這樣才分得出先走哪一軸。
func biasField(bias int) *Field {
	f := &Field{Rand: &halfRand{}}
	f.Units[MonsterBase] = Unit{Name: "怪", HP: 10, Facing: South,
		IsMonster: true, Speed: 9, X: 5, Y: 5, Bias: bias}
	f.Units[PartyBase] = Unit{Name: "人", HP: 10, Facing: North, X: 9, Y: 9}
	return f
}

// 偏好 +1 → 先走東西;−1 → 先走南北。**兩軸都有差距**才測得出來。
func TestBiasPicksTheFirstAxis(t *testing.T) {
	for _, tc := range []struct {
		bias         int
		wantX, wantY int
	}{
		{1, 6, 5},  // 先東
		{-1, 5, 6}, // 先南
	} {
		f := biasField(tc.bias)
		var p Points
		f.ResetPoints(&p)
		// 剛好夠轉一次 + 走一格 —— 多給的話牠會一路走到相鄰,就分不出第一步走哪一軸
	p[MonsterBase] = rules.CostTurn + rules.CostMove
		f.MonsterTurn(&p, MonsterBase)
		u := f.Units[MonsterBase]
		if u.X != tc.wantX || u.Y != tc.wantY {
			t.Errorf("偏好 %+d 應該走到 (%d,%d),得到 (%d,%d)",
				tc.bias, tc.wantX, tc.wantY, u.X, u.Y)
		}
	}
}

// 偏好只擲一次 —— 整場固定(`CMBT 0x13E96` 的 `cmp …, 0` 閘門)。
func TestBiasIsRolledOnlyOnce(t *testing.T) {
	f := biasField(0)
	got := f.axisBias(MonsterBase)
	if got != 1 && got != -1 {
		t.Fatalf("擲出來應該是 ±1,得到 %d", got)
	}
	for i := 0; i < 5; i++ {
		if again := f.axisBias(MonsterBase); again != got {
			t.Fatalf("偏好被重擲了:%d → %d", got, again)
		}
	}
}

// 要走的那一格站不了 → 偏好取負(`CMBT 0x140DE` 的 neg)。
func TestBlockedStepFlipsTheBias(t *testing.T) {
	f := biasField(1)
	// 東邊那一格擺一隻擋路的
	f.Units[MonsterBase+1] = Unit{Name: "擋路", HP: 10, Facing: South,
		IsMonster: true, X: 6, Y: 5}
	var p Points
	f.ResetPoints(&p)
	f.MonsterTurn(&p, MonsterBase)
	if f.Units[MonsterBase].Bias != -1 {
		t.Errorf("撞到人之後偏好應該翻成 −1,得到 %+d", f.Units[MonsterBase].Bias)
	}
	if f.Units[MonsterBase].X != 5 || f.Units[MonsterBase].Y != 5 {
		t.Errorf("擋住就不該動,得到 (%d,%d)",
			f.Units[MonsterBase].X, f.Units[MonsterBase].Y)
	}
}

// halfRand:Float01 恆為 0 → 偏好擲成 +1;Roll 回 1。
type halfRand struct{}

func (halfRand) Roll(faces int) int { return 1 }
func (halfRand) Float01() float64   { return 0 }
