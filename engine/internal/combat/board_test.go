package combat

import "testing"

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
		if u.X <= 0 || u.X >= BoardSize-1 || u.Y <= 0 || u.Y >= BoardSize-1 {
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
