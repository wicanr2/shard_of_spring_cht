package combat

import "testing"

// docs/spec/07 §8 驗收 2:命中公式的兩個加項各有一個測試。
//
// ⚠ `+30` 的條件是「狀態 > 1」。docs/re/98 抓到過一次反號,
// 而反號之後的規則仍然像一條合理的規則(狀態好的更容易被打中),
// **沒有任何訊號** —— 所以這裡要正面測邊界值 1 與 2。
func TestToHitStatusBonus(t *testing.T) {
	atk := Unit{ToHit: 10, Facing: North}
	base := ToHit(atk, Unit{Status: 0, Facing: East}, Item{}, Item{})
	for _, c := range []struct {
		status, want int
	}{
		{0, base}, {1, base}, // ≤ 1 沒有加成
		{2, base + 30}, {5, base + 30}, // > 1 才加
	} {
		got := ToHit(atk, Unit{Status: c.status, Facing: East}, Item{}, Item{})
		if got != c.want {
			t.Errorf("狀態 %d:命中 %d,應為 %d(條件是 > 1,不是 ≤ 1)",
				c.status, got, c.want)
		}
	}
}

// 背後攻擊 +12:兩者**朝同一方向**時成立。
func TestToHitBackAttack(t *testing.T) {
	a := Unit{ToHit: 10, Facing: North}
	same := ToHit(a, Unit{Facing: North}, Item{}, Item{})
	diff := ToHit(a, Unit{Facing: South}, Item{}, Item{})
	if same-diff != 12 {
		t.Errorf("同朝向與異朝向差 %d,應為 12", same-diff)
	}
}

// 命中 = (命中能力 − 防具加值 + 武器加值) × 4。**乘 4 在括號外**。
func TestToHitTimesFour(t *testing.T) {
	got := ToHit(Unit{ToHit: 10, Facing: North}, Unit{Facing: East},
		Item{Bonus: 2}, Item{Bonus: 3})
	if want := (10 - 3 + 2) * 4; got != want {
		t.Errorf("命中 %d,應為 %d", got, want)
	}
}

// 驗收 3:赤手空拳(武器編號 ≥ 60)時武器項是常數 1。
func TestDamageBareHanded(t *testing.T) {
	r := &ScriptRand{Values: []int{5, 0}} // R₁ = 5、R₂ = 0
	armed := Damage(Unit{Weapon: 3}, Unit{}, Item{Main: 7}, Item{}, r)
	r.i = 0
	bare := Damage(Unit{Weapon: BareHandMin}, Unit{}, Item{Main: 7}, Item{}, r)
	if armed != 7*5 {
		t.Errorf("持武器傷害 %d,應為 7×5 = 35", armed)
	}
	if bare != 1*5 {
		t.Errorf("赤手空拳傷害 %d,應為 1×5 = 5(武器項換成常數 1)", bare)
	}
}

// 傷害的減項:屬性 17(Armored skin)與防具值。
func TestDamageSubtractsArmor(t *testing.T) {
	r := &ScriptRand{Values: []int{10, 0}}
	got := Damage(Unit{Weapon: 3}, Unit{ArmSkin: 4}, Item{Main: 2}, Item{Main: 6}, r)
	if want := 2*10 - 4 - 6; got != want {
		t.Errorf("傷害 %d,應為 %d(2×10 − 4 − 6)", got, want)
	}
}

// 兩個佔位常數。這條測試的存在就是提醒:解出來時它會失敗,
// 逼人回來更新 docs/re/136 與 docs/spec/07 §3。
//
// ⚠ DamageK2 **不在這條裡** —— 它的 0 是正確的(偏移已折進 Roll),
// 不是佔位。把它一起鎖住會讓下一個人以為它也還沒解。
func TestPlaceholderConstants(t *testing.T) {
	if DamageK1 != 1.0 {
		t.Errorf("k₁=%v —— 若這是解出來的值,請同時更新 docs/re/136 與 "+
			"docs/spec/07 §3,再改這條測試", DamageK1)
	}
	if ToHitFaces != 100 {
		t.Errorf("命中面數=%v —— 同上", ToHitFaces)
	}
	if len(Unresolved) != 2 {
		t.Errorf("未解項清單有 %d 條,應為 2 —— 清單與常數要一起改", len(Unresolved))
	}
}

// 驗收 4:生命值 < 1 設為 0,不會變負。
func TestApplyClampsToZero(t *testing.T) {
	u := Unit{HP: 3}
	Apply(&u, 10)
	if u.HP != 0 {
		t.Errorf("生命值 %d,應夾在 0", u.HP)
	}
	if u.Alive() {
		t.Error("生命值 0 應判成死亡")
	}
}

// 驗收 5:三個結束條件,而且**逃離用朝向、全滅用生命值**。
func TestOutcomes(t *testing.T) {
	newField := func() *Field {
		f := &Field{}
		f.Units[MonsterBase] = Unit{HP: 5, Facing: South, IsMonster: true}
		f.Units[PartyBase] = Unit{HP: 5, Facing: North}
		return f
	}

	if got := newField().Outcome(); got != Ongoing {
		t.Errorf("雙方都在:%v,應為進行中", got)
	}

	f := newField()
	f.Units[MonsterBase].HP = 0
	if got := f.Outcome(); got != MonstersDead {
		t.Errorf("怪物生命值 0:%v,應為怪物全滅", got)
	}

	f = newField()
	f.Units[PartyBase].HP = 0
	if got := f.Outcome(); got != PartyDead {
		t.Errorf("隊伍生命值 0:%v,應為全隊陣亡", got)
	}

	// 逃離:**還活著**,但朝向 0
	f = newField()
	f.Units[PartyBase].Facing = Absent
	if got := f.Outcome(); got != PartyRan {
		t.Errorf("隊伍朝向 0 但生命值 5:%v,應為全隊逃離 —— "+
			"這條錯了表示把「在場」和「活著」合併了", got)
	}
	if !f.Units[PartyBase].Alive() {
		t.Error("逃離的成員仍然活著,Alive() 不該看朝向")
	}
}

// 驗收 6:速度高的先動,速度相同時順序穩定。
//
// ⚠ 穩定性不是可有可無:順序若不定,「同種子跑兩次結果相同」就不成立,
// 而那是 M4 的驗收條件。
func TestInitiativeOrder(t *testing.T) {
	f := &Field{}
	f.Units[0] = Unit{HP: 1, Facing: South, Speed: 5}
	f.Units[1] = Unit{HP: 1, Facing: South, Speed: 9}
	f.Units[2] = Unit{HP: 1, Facing: South, Speed: 5} // 與 0 同速
	f.Units[9] = Unit{HP: 1, Facing: North, Speed: 7}
	f.Units[10] = Unit{HP: 0, Facing: North, Speed: 99} // 死了,不進表

	f.Sort()
	want := []int{1, 9, 0, 2} // 9 → 7 → 5(索引小的在前)
	if len(f.Order) != len(want) {
		t.Fatalf("先攻表 %v,應為 %v", f.Order, want)
	}
	for i := range want {
		if f.Order[i] != want[i] {
			t.Fatalf("先攻表 %v,應為 %v", f.Order, want)
		}
	}
	// 重排多次結果必須相同
	for n := 0; n < 20; n++ {
		f.Sort()
		for i := range want {
			if f.Order[i] != want[i] {
				t.Fatalf("第 %d 次重排得到 %v —— 排序不穩定", n, f.Order)
			}
		}
	}
}

// 驗收 1:同一顆種子跑兩次,傷害序列完全相同。
func TestSameSeedSameFight(t *testing.T) {
	run := func(seed uint64) []int {
		f := &Field{Rand: NewRand(seed), Items: map[int]Item{
			1: {Main: 6, Bonus: 2}, 2: {Main: 3, Bonus: 1},
		}}
		f.Units[0] = Unit{Name: "怪", HP: 60, Facing: South, Speed: 5,
			Weapon: 1, Armor: 2, ToHit: 12, IsMonster: true}
		f.Units[9] = Unit{Name: "人", HP: 60, Facing: North, Speed: 7,
			Weapon: 1, Armor: 2, ToHit: 12}
		var out []int
		for round := 0; round < 30 && f.Outcome() == Ongoing; round++ {
			f.Sort()
			for _, i := range f.Order {
				target := 0
				if i < PartyBase {
					target = PartyBase
				}
				if !f.Units[target].Alive() {
					continue
				}
				_, _, d := f.Attack(i, target)
				out = append(out, d)
			}
		}
		return out
	}
	a, b := run(12345), run(12345)
	if len(a) == 0 {
		t.Fatal("沒有打到任何一回合 —— 這條測試沒有測到東西")
	}
	if len(a) != len(b) {
		t.Fatalf("兩次長度 %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("第 %d 次攻擊 %d vs %d —— 同種子結果不同", i, a[i], b[i])
		}
	}
	if c := run(999); len(c) == len(a) && equal(a, c) {
		t.Error("不同種子得到完全相同的序列 —— 亂數沒有真的被用到")
	}
}

func equal(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 種子 0 不可以讓擲骰卡住(xorshift 的全零狀態是不動點)。
func TestSeedZeroStillVaries(t *testing.T) {
	r := NewRand(0)
	seen := map[int]bool{}
	for i := 0; i < 50; i++ {
		seen[r.Roll(26)] = true
	}
	if len(seen) < 5 {
		t.Errorf("種子 0 的 50 次擲骰只有 %d 種結果 —— 狀態卡住了", len(seen))
	}
}
