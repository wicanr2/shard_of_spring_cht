package combat

// 怪物 AI 的規則測試(docs/re/186)。三條規則都是**讀出來的**,
// 所以這裡釘的是常數本身,不是「看起來合理的行為」。

import "testing"

// TestMonsterAnchorsAreTheReadTable 釘住錨點表。
//
// ⚠ 這張表是 `CMBT 0x115B1`–`0x11611` 的字面常數,不是設計選擇 ——
// 改它等於改遊戲規則,要先有新的反組譯證據。
func TestMonsterAnchorsAreTheReadTable(t *testing.T) {
	want := [8][2]int{
		{6, 6}, {11, 6}, {16, 6},
		{6, 11}, {16, 11},
		{6, 16}, {11, 16}, {16, 16},
	}
	if MonsterAnchors != want {
		t.Errorf("錨點表被改了:\n得到 %v\n期望 %v", MonsterAnchors, want)
	}
	// 中間那格(11,11)不在表上 —— 那是隊伍的區塊。
	for _, a := range MonsterAnchors {
		if a[0] == 11 && a[1] == 11 {
			t.Error("(11,11) 是隊伍的區塊,不該是怪物錨點")
		}
	}
	if MonsterAnchorFaces != 7 {
		t.Errorf("擲骰面數 %d,原版是 7(ds:94B4)", MonsterAnchorFaces)
	}
	if MonsterJitterFaces != 5 {
		t.Errorf("抖動面數 %d,原版是 5(ds:94B8)", MonsterJitterFaces)
	}
}

// TestMonsterSpawnStaysInFifteenSquare 確認出場範圍就是手冊的 15×15(6…20)。
func TestMonsterSpawnStaysInFifteenSquare(t *testing.T) {
	f := &Field{Rand: NewRand(99)}
	for i := 0; i < 500; i++ {
		x, y := f.rollMonsterSpot()
		if x < 6 || x > 20 || y < 6 || y > 20 {
			t.Fatalf("出場位置 (%d,%d) 跑出 6…20 —— 錨點 + 抖動接不起來", x, y)
		}
	}
}

// TestMonsterSpawnNeverOnAnOccupiedCell:重擲的條件就是「那一格是空的」。
func TestMonsterSpawnNeverOnAnOccupiedCell(t *testing.T) {
	f := &Field{Rand: NewRand(7)}
	// 把六個錨點區塞滿,只留 (16,16) 那一塊 —— 它是**永遠挑不到**的第 8 個,
	// 所以擲骰一定湊不到,會落到線性掃描的退路。
	for i := 0; i < MonsterMax; i++ {
		f.Units[i] = Unit{Name: "佔位", HP: 1, Facing: South, IsMonster: true,
			X: 6 + i, Y: 6}
	}
	x, y := f.rollMonsterSpot()
	if f.Occupant(x, y) >= 0 {
		t.Errorf("擲到了有人的格子 (%d,%d)", x, y)
	}
}

// TestRetargetKeepsTargetWhileAlive 是 D1 的核心:**活著就不換**。
func TestRetargetKeepsTargetWhileAlive(t *testing.T) {
	f := newAITestField()
	first := f.Retarget(MonsterBase)
	if first < PartyBase {
		t.Fatalf("應該鎖定一名隊員,得到 %d", first)
	}
	for i := 0; i < 20; i++ {
		if got := f.Retarget(MonsterBase); got != first {
			t.Fatalf("目標還活著卻換人了:%d → %d", first, got)
		}
	}
}

// TestRetargetSwitchesWhenTargetDies / …Flees:兩個換人的條件各一條。
func TestRetargetSwitchesWhenTargetDies(t *testing.T) {
	f := newAITestField()
	first := f.Retarget(MonsterBase)
	f.Units[first].HP = 0 // 倒下
	got := f.Retarget(MonsterBase)
	if got == first {
		t.Error("目標倒下之後應該重挑")
	}
	if got < PartyBase || !f.Units[got].Alive() {
		t.Errorf("重挑的目標 %d 不是活著的隊員", got)
	}
}

func TestRetargetSwitchesWhenTargetLeavesField(t *testing.T) {
	f := newAITestField()
	first := f.Retarget(MonsterBase)
	f.Units[first].Facing = Absent // 逃跑 = 朝向清 0(docs/re/103)
	if got := f.Retarget(MonsterBase); got == first {
		t.Error("目標離場之後應該重挑")
	}
}

func TestRetargetWithNobodyLeft(t *testing.T) {
	f := newAITestField()
	for i := PartyBase; i < PartyBase+PartyMax; i++ {
		f.Units[i].HP = 0
	}
	if got := f.Retarget(MonsterBase); got != -1 {
		t.Errorf("全隊倒下時應該回 −1,得到 %d", got)
	}
}

func newAITestField() *Field {
	f := &Field{Rand: NewRand(2026)}
	f.Units[MonsterBase] = Unit{Name: "地精", HP: 10, Facing: South,
		IsMonster: true, X: 11, Y: 10}
	for i := 0; i < PartyMax; i++ {
		f.Units[PartyBase+i] = Unit{Name: "隊員", HP: 10, Facing: North,
			X: 12 + i%3, Y: 13 + i/3}
	}
	return f
}
