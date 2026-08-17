package combat

import (
	"math"
	"testing"
)

// 戰後分經驗的資格(docs/re/150 §2.1):朝向 > 0 **且** 狀態 < 5。
//
// ⚠ 四種組合都要測。只測「陣亡的不分」的話,`HP > 0` 那條錯規則也會通過 ——
// 兩者只在**「逃走但活著」**這一格分岔,而那正是原版特別處理的情況。
func TestEarnsExpUsesFacingNotHP(t *testing.T) {
	cases := []struct {
		name   string
		facing Facing
		hp     int
		status int
		want   bool
	}{
		{"在場、正常", South, 5, 0, true},
		{"在場、中毒", South, 5, 1, true},
		{"在場、束縛", South, 5, 2, true},
		{"逃走了(朝向 0)但還活著", Absent, 5, 0, false},
		{"陣亡(狀態 5)", South, 0, 5, false},
		{"在場但 HP 0、狀態未標死亡", South, 0, 0, true},
	}
	for _, c := range cases {
		u := Unit{Facing: c.facing, HP: c.hp, Status: c.status}
		if got := u.EarnsExp(); got != c.want {
			t.Errorf("%s = %v,應為 %v", c.name, got, c.want)
		}
	}
	// 怪物不分經驗
	if (Unit{Facing: South, HP: 5, IsMonster: true}).EarnsExp() {
		t.Error("怪物不該分到經驗")
	}
}

// 經驗總額**不篩死活**(docs/re/152 §1.1)。
//
// ⚠ 這條容易寫反:直覺是「打倒的怪物才給經驗」,而原版的累加迴圈
// 跑九個怪物槽、裡面沒有任何死活判斷。多數戰鬥打到全滅才結束,
// 所以兩種算法**在畫面上分不開** —— 只有怪物逃走的那一場會分岔。
func TestTotalExpDoesNotFilterByDeath(t *testing.T) {
	units := make([]Unit, PartyBase+PartyMax)
	units[0] = Unit{IsMonster: true, Exp: 100, HP: 0, Facing: Absent} // 被打倒
	units[1] = Unit{IsMonster: true, Exp: 50, HP: 3, Facing: Absent}  // 逃走了,還活著
	units[2] = Unit{IsMonster: true, Exp: 7, HP: 9, Facing: South}    // 還在場上
	units[PartyBase] = Unit{Exp: 999, HP: 5, Facing: South}           // 隊員不算

	if got := TotalExp(units); got != 157 {
		t.Errorf("經驗總額 %d,應為 157(100 + 50 + 7,隊員的 999 不算)", got)
	}
}

// 均分是整數除法(docs/re/152 §1.3:原版浮點除完再 INT())。
func TestExpShareTruncates(t *testing.T) {
	if got := ExpShare(157, 4); got != 39 {
		t.Errorf("157 ÷ 4 = %d,應為 39(取整,不是四捨五入)", got)
	}
	if got := ExpShare(157, 0); got != 0 {
		t.Errorf("沒有人有資格時應回 0,得 %d", got)
	}
}

// 金幣:每隻怪物擲一次 `INT(1.7^階級 + RND × 2.1^階級 + 1)`,隊員不算
// (docs/re/207 §3)。
func TestTotalGoldPerMonster(t *testing.T) {
	units := make([]Unit, PartyBase+PartyMax)
	units[0] = Unit{IsMonster: true, Tier: 5}
	units[1] = Unit{IsMonster: true, Tier: 3}
	units[2] = Unit{IsMonster: true, Tier: 0} // 階級 0 → 夾成 1
	units[PartyBase] = Unit{Tier: 99}         // 隊員的階級固定 99,不該被算進去

	// 亂數固定 0 → 只剩保底項:INT(1.7^T + 1)
	r := &fixedFloat{v: 0}
	want := int(math.Pow(1.7, 5)+1) + int(math.Pow(1.7, 3)+1) + int(math.Pow(1.7, 1)+1)
	if got := TotalGold(units, r); got != want {
		t.Errorf("亂數 0 時應為 %d,得 %d(隊員的 99 不該算)", want, got)
	}
	if got := TotalGold(make([]Unit, PartyBase+PartyMax), &fixedFloat{}); got != 0 {
		t.Errorf("沒有怪物應為 0,得 %d", got)
	}
}

// ⚠ 這一條擋的是「把底數與指數寫反」——`階級^1.7` 在低階幾乎看不出差別,
// 到高階差好幾個量級,而畫面上只會顯示一個看起來很大的數字。
func TestGoldBaseAndExponentAreNotSwapped(t *testing.T) {
	const tier = 13 // 最終首領那一級
	got := MonsterGold(tier, &fixedFloat{v: 0})
	if want := int(math.Pow(1.7, tier) + 1); got != want {
		t.Fatalf("階級 %d 的保底金幣是 %d,應為 %d", tier, got, want)
	}
	if swapped := int(math.Pow(float64(tier), 1.7) + 1); got == swapped {
		t.Errorf("底數與指數寫反了也會得到 %d —— 這條測試分不出來", got)
	}
}

// 浮動項乘的是 2.1^階級(另一個底數),不是同一個。
func TestGoldRollUsesTheOtherBase(t *testing.T) {
	const tier = 5
	lo := MonsterGold(tier, &fixedFloat{v: 0})
	hi := MonsterGold(tier, &fixedFloat{v: 1})
	if want := int(math.Pow(1.7, tier) + math.Pow(2.1, tier) + 1); hi != want {
		t.Errorf("亂數 1 時應為 %d,得 %d", want, hi)
	}
	if hi <= lo {
		t.Error("浮動項沒有作用")
	}
}

// fixedFloat 讓 Float01 永遠回同一個值;Roll 回面數(用不到,但要滿足介面)。
type fixedFloat struct{ v float64 }

func (f *fixedFloat) Roll(faces int) int { return faces }
func (f *fixedFloat) Float01() float64   { return f.v }
