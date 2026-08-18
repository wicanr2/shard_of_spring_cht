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

// 傷害公式的三條分支(docs/re/153 §5/§6)。
//
// ⚠ 三條分支的差別在**擲骰的面數是哪一項**,不在回傳值 ——
// 所以要驗 ScriptRand 記下來的面數,只比傷害數字分不開。
func TestDamageBranchesPickDifferentFaces(t *testing.T) {
	cases := []struct {
		name  string
		atk   Unit
		w     Item
		faces int
		want  int // 擲骰固定 5 時的傷害
	}{
		{"持武器", Unit{Weapon: 3, Str: 11}, Item{Main: 7}, 7, 5 + 2},
		{"赤手", Unit{Weapon: BareHandMin, Str: 9}, Item{Main: 7}, 9, 5},
		{"空手道", Unit{Weapon: BareHandMin, Str: 9, ToHit: 12, Karate: 1,
			Action: ActionFighter}, Item{Main: 7}, 12 - 5, 5},
		// ⚠ 外層閘門是**屬性 14 == 1**(docs/re/163 §4)。行動類型不是戰士時,
		// 就算技能旗標是 1 也不走空手道那一支 —— 少了這一層,巫師會用到戰士的式子。
		{"巫師空手(技能旗標無效)", Unit{Weapon: BareHandMin, Str: 9, ToHit: 12,
			Karate: 1, Action: ActionWizard}, Item{Main: 7}, 9, 5},
	}
	for _, c := range cases {
		r := &ScriptRand{Values: []int{5}}
		got := Damage(c.atk, Unit{}, c.w, Item{}, r)
		if len(r.Faces) != 1 || r.Faces[0] != c.faces {
			t.Errorf("%s:擲骰面數 %v,應為 %d", c.name, r.Faces, c.faces)
		}
		if got != c.want {
			t.Errorf("%s:傷害 %d,應為 %d", c.name, got, c.want)
		}
	}
}

// 力量加值:`floor((力量 − 7) × 0.5)`,而且**只在持武器時**加。
//
// ⚠ 力量 < 7 是負的,而且要 floor 不是截斷:floor(-0.5) = -1、int(-0.5) = 0。
func TestStrengthBonusFloors(t *testing.T) {
	for _, c := range []struct{ str, want int }{
		{7, 0}, {8, 0}, {9, 1}, {11, 2}, {6, -1}, {5, -1}, {4, -2},
	} {
		r := &ScriptRand{Values: []int{20}}
		got := Damage(Unit{Weapon: 3, Str: c.str}, Unit{}, Item{Main: 30}, Item{}, r)
		if got != 20+c.want {
			t.Errorf("力量 %d 的加值是 %d,應為 %d", c.str, got-20, c.want)
		}
	}
	r := &ScriptRand{Values: []int{5}}
	if got := Damage(Unit{Weapon: BareHandMin, Str: 20}, Unit{}, Item{}, Item{}, r); got != 5 {
		t.Errorf("赤手傷害 %d,應為 5(面數已經是力量,不再加加值)", got)
	}
}

// 傷害的減項:屬性 17(Armored skin)與防具值;小於 1 就是 0。
func TestDamageSubtractsArmor(t *testing.T) {
	r := &ScriptRand{Values: []int{10}}
	got := Damage(Unit{Weapon: 3, Str: 7}, Unit{ArmSkin: 4}, Item{Main: 20}, Item{Main: 6}, r)
	if want := 10 - 4 - 6; got != want {
		t.Errorf("傷害 %d,應為 %d(10 − 4 − 6)", got, want)
	}
	r = &ScriptRand{Values: []int{2}}
	if got := Damage(Unit{Weapon: 3, Str: 7}, Unit{ArmSkin: 9}, Item{Main: 20}, Item{}, r); got != 0 {
		t.Errorf("傷害算出來是負的時應回 0,得 %d(docs/re/153 §8)", got)
	}
}

// 狂暴:屬性 16 非 0 **且** 第二次擲骰 > 75(docs/re/153 §7)。
//
// ⚠ 兩個條件要分開測 —— 只測其中一個,另一個寫成常數 true 也會通過。
func TestBerserkNeedsBothConditions(t *testing.T) {
	for _, c := range []struct {
		berserk, roll int
		want          bool
	}{
		{1, 76, true}, {1, 75, false}, {1, 100, true},
		{0, 100, false}, {0, 76, false},
	} {
		if got := Berserk(Unit{Berserk: c.berserk}, c.roll); got != c.want {
			t.Errorf("狂暴=%d 擲骰=%d → %v,應為 %v", c.berserk, c.roll, got, c.want)
		}
	}
}

// 面數的下界:狂暴的門檻是 75,面數 < 76 會讓那個分支變成死碼。
func TestToHitFacesAllowsBerserk(t *testing.T) {
	if ToHitFaces <= BerserkThreshold {
		t.Errorf("面數 %d ≤ 狂暴門檻 %d —— 狂暴永遠不會觸發,而它是矮人的種族技能",
			ToHitFaces, BerserkThreshold)
	}
}

// 面數 = 100:命中值直接就是百分比(docs/re/154)。
//
// ⚠ 這條不是「鎖住一個佔位」,是**鎖住一條規則**:
// 手冊的表把命中率印成 `技巧 × 4 %`,而程式算的是 `擲骰 ≤ 技巧 × 4`
// —— 兩者相等只在面數 100 時成立。改了面數就等於改了命中率的尺度。
func TestToHitIsAPercentage(t *testing.T) {
	if ToHitFaces != 100 {
		t.Fatalf("面數 %d —— 手冊 p.13 的表是百分比,面數不是 100 的話"+
			"「命中值」與「命中率」就不再是同一個數(docs/re/154)", ToHitFaces)
	}
	// 手冊 p.13 逐列:技巧 3 → 12%、技巧 20 → 80%
	for _, c := range []struct{ skill, pct int }{{3, 12}, {10, 40}, {20, 80}} {
		got := ToHit(Unit{ToHit: c.skill, Facing: North}, Unit{Facing: East},
			Item{}, Item{})
		if got != c.pct {
			t.Errorf("技巧 %d 的命中值 %d,手冊表寫 %d%%", c.skill, got, c.pct)
		}
	}
	// 狂暴門檻在 100 面下是四分之一
	if 100-BerserkThreshold != 25 {
		t.Errorf("狂暴機率 %d%%,應為 25%%", 100-BerserkThreshold)
	}
	// 目前一條:一場遭遇的隻數(docs/re/225)。怪物施法的投入與目標格
	// 已經解掉(docs/re/226),接上去了。
	// 這條的用意是**清單與規格一起改**:新增或解掉一項時它會失敗,
	// 逼人回來更新規格。⛔ 清空了也不要把 Unresolved 拆掉,
	// 下一個未解項要有地方放(combat.go 的說明)。
	if len(Unresolved) != 1 {
		t.Errorf("未解項清單有 %d 條,應為 1:%v", len(Unresolved), Unresolved)
	}
}

// 命中擲骰是 round(RND×faces+1),不是 Roll(faces)(docs/re/185 §2 表列 #2)。
//
// 0.005×100+1 = 1.5:floor 給 1、round 給 2 —— 這組值兩者不同,
// 分得出 Hits() 是不是真的换了取整方式(而不是還在用 Roll())。
func TestHitsRollIsRoundNotFloor(t *testing.T) {
	atk := Unit{ToHit: 100, Facing: North} // 門檻拉滿,讓命中與否只看擲骰本身
	def := Unit{Facing: South}
	r := &ScriptRand{Floats: []float64{0.005}}
	roll, _ := Hits(atk, def, Item{}, Item{}, r, ToHitFaces)
	if roll != 2 {
		t.Errorf("擲骰 %d,應為 2(round(0.005×100+1)=round(1.5)=2;"+
			"得 1 表示還在用 floor 或 Roll())", roll)
	}
}

// ⭐ 值域上界是 **101**,不是 100(docs/re/185 §2 表列 #2)。
//
// Float01() 逼近 1 時,round(RND×100+1) 逼近 101。舊的 `Roll(100)`
// 值域頂多到 100 —— 這條測試專門釘住「上界多了 1」這件事本身。
func TestHitsRollUpperBoundIs101(t *testing.T) {
	atk := Unit{ToHit: 1000, Facing: North}
	def := Unit{Facing: South}
	r := &ScriptRand{Floats: []float64{0.999999}}
	roll, _ := Hits(atk, def, Item{}, Item{}, r, ToHitFaces)
	if roll != 101 {
		t.Errorf("逼近上界的擲骰 %d,應為 101 —— 值域是 1…faces+1,"+
			"不是 1…faces", roll)
	}
}

// 狂暴的第二次擲骰同樣是 round(RND×100+1),要用 Float01()
// 不是 Roll()(docs/re/185 §2 表列 #4)。用 Attack() 整條路徑驗,
// 免得只測 rollRound 這個內部函式漏掉呼叫端沒接上。
func TestBerserkSecondRollIsRoundNotFloor(t *testing.T) {
	f := &Field{Items: map[int]Item{}}
	// 赤手(Weapon: BareHandMin)→ 擲骰面數直接是力量,沒有加值
	// (docs/re/153),傷害基準就是 Roll(Str) 的回傳值。
	// ⚠ 攻擊者要有 Berserk,Berserk() 看的是**攻擊者**那一項。
	f.Units[PartyBase] = Unit{Name: "人", HP: 20, Facing: North, ToHit: 100,
		Str: 20, Weapon: BareHandMin, Berserk: 1}
	f.Units[MonsterBase] = Unit{Name: "怪", HP: 20, Facing: South}
	// Roll() 走 Values、Float01() 走 Floats,兩條計數各自獨立(ScriptRand)。
	// 命中門檻拉滿保證命中;傷害擲骰固定用 Values=[5]。
	// 若狂暴那一擲還在用 f.Rand.Roll(ToHitFaces) 而不是 rollRound(Float01),
	// 它會讀到 Values 的 5(5 ≤ 75,不觸發狂暴)—— 兩條路徑在這組腳本值下
	// **結果不同**,分得出呼叫端有沒有真的接上 Float01。
	f.Rand = &ScriptRand{Floats: []float64{0.0, 0.999999}, Values: []int{5}}
	_, hit, dmg := f.Attack(PartyBase, MonsterBase)
	if !hit {
		t.Fatal("命中門檻拉滿應該必中")
	}
	if dmg != 10 { // 傷害基準 5,狂暴加倍 → 10;沒加倍會停在 5
		t.Errorf("狂暴應加倍傷害為 10,得 %d —— "+
			"檢查第二次擲骰有沒有走 rollRound(Float01)而不是 Roll()", dmg)
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
		// ⚠ 兩個單位都要有力量 —— 傷害公式帶力量加值(docs/re/153 §6),
		// 力量 0 的假單位加值是 floor(−3.5) = −4,傷害全部被夾成 0,
		// 於是「不同種子結果不同」這條驗收**測不到東西**。
		// **假資料要長得像真資料**,否則它保護的是假設不是行為。
		f.Units[0] = Unit{Name: "怪", HP: 60, Facing: South, Speed: 5,
			Weapon: 1, Armor: 2, ToHit: 12, Str: 13, IsMonster: true}
		f.Units[9] = Unit{Name: "人", HP: 60, Facing: North, Speed: 7,
			Weapon: 1, Armor: 2, ToHit: 12, Str: 13}
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

// ── D)ispell(docs/re/188)────────────────────────────────────────────────

// TestUndeadClassesAreTheReadSet 釘住不死生物的判準。
//
// 原版比的是**戰鬥屬性 11** 的 65/73/81(已確認)。換算回 `MONSTERS.DAT`
// 欄6 要靠「圖組 → 檔號」,而那個換算**現在讀出來了**(docs/re/220):
//
//	圖組 = 檔號 × 8 + 25   →   {65,73,81} = 檔 {5,6,7}
//
// ⚠ 這張表先前是 `{4,5,6}`,差一位。舊版還為此寫了一句「幽靈不算不死,
// 本作把它們歸成靈體」——**那是替一個錯誤的數字編出來的解釋**。
// 內容印證:MONST5 骷髏、MONST6 骷髏巫師、MONST7 幽靈,三個都是不死。
//
// 改這張表等於改規則,要先有新的反組譯證據。
func TestUndeadClassesAreTheReadSet(t *testing.T) {
	if UndeadClasses != [3]int{5, 6, 7} {
		t.Errorf("不死類別表被改了:%v", UndeadClasses)
	}
	// 骷髏(5)、骷髏巫師(6)、幽靈(7)—— 都算。
	for _, k := range []int{5, 6, 7} {
		if !IsUndead(Unit{IsMonster: true, Kind: k}) {
			t.Errorf("類別 %d 應該算不死", k)
		}
	}
	// 欄6 = 4 **不存在**(那是隊員巫師的圖檔),不該算。
	if IsUndead(Unit{IsMonster: true, Kind: 4}) {
		t.Error("類別 4 是隊員巫師的圖檔,不該算不死")
	}
	// 隊員永遠不算 —— 他們的屬性 11 走 41/57 那一組。
	if IsUndead(Unit{IsMonster: false, Kind: 5}) {
		t.Error("隊員不該被判成不死生物")
	}
}

// TestSpriteFile 釘住「單位 → MONST 檔號」(docs/re/220)。
//
// ⚠ 怪物的 Kind 存的是**欄6**,隊員的 Kind 存的是**圖組**(41/57)——
// 兩者不同尺度。SpriteFile 就是為了把這件事收在一個地方。
func TestSpriteFile(t *testing.T) {
	// 隊員:CMBT 的字串表裡寫死 monst2.bin / monst4.bin。
	if got := SpriteFile(Unit{Kind: KindFighter}); got != SpriteFileFighter {
		t.Errorf("戰士拿到檔 %d,要 %d", got, SpriteFileFighter)
	}
	if got := SpriteFile(Unit{Kind: KindWizard}); got != SpriteFileWizard {
		t.Errorf("巫師拿到檔 %d,要 %d", got, SpriteFileWizard)
	}
	// 怪物:檔號就是欄6。內容對照見 docs/re/220。
	for _, k := range []int{1, 5, 15, 16, 22} {
		if got := SpriteFile(Unit{IsMonster: true, Kind: k}); got != k {
			t.Errorf("怪物欄6=%d 拿到檔 %d", k, got)
		}
	}
	// 超出範圍回 0 —— ⚠ 不要夾到最後一個檔:那會讓每隻沒有圖的怪
	// 都長成龍,而畫面上完全看不出哪裡不對。
	for _, k := range []int{0, -1, MonstFiles + 1, 99} {
		if got := SpriteFile(Unit{IsMonster: true, Kind: k}); got != 0 {
			t.Errorf("欄6=%d 超出範圍,該回 0,拿到 %d", k, got)
		}
	}
}

// TestDispellThreshold 釘住成功門檻的算式與那個 3.6。
func TestDispellThreshold(t *testing.T) {
	if DispellFactor != 3.6 {
		t.Errorf("乘數 %v,原版 ds:9D98 是 3.6", DispellFactor)
	}
	// 智能 20 打階級 1 → (20−1+1)×3.6 = 72
	if got := DispellThreshold(20, 1); got != 72 {
		t.Errorf("智能 20／階級 1 的門檻應該是 72,得到 %v", got)
	}
	// 智能 10 打階級 5 → (10−5+1)×3.6 = 21.6
	if got := DispellThreshold(10, 5); got < 21.5 || got > 21.7 {
		t.Errorf("智能 10／階級 5 的門檻應該約 21.6,得到 %v", got)
	}
	// ⚠ 門檻可以是負的 —— 照原版**不夾在 0**,那時擲多少都失敗。
	if got := DispellThreshold(3, 10); got >= 0 {
		t.Errorf("智能 3／階級 10 的門檻應該是負的,得到 %v", got)
	}
}

// TestHasUndeadAndDispell:場上沒有不死就回 false;有就逐隻各擲一次。
func TestHasUndeadAndDispell(t *testing.T) {
	f := &Field{Rand: NewRand(5)}
	f.Units[MonsterBase] = Unit{Name: "哥布林", HP: 10, Facing: South,
		IsMonster: true, Kind: 12, Tier: 3}
	if f.HasUndead() {
		t.Error("只有哥布林時不該說有不死生物")
	}
	f.Units[MonsterBase+1] = Unit{Name: "骷髏", HP: 10, Facing: South,
		IsMonster: true, Kind: 5, Tier: 1}
	if !f.HasUndead() {
		t.Error("有骷髏就該說有不死生物")
	}

	// 智能 20 打階級 1 → 門檻 72,種子固定 → 只驗「非不死的沒被碰到」。
	before := f.Units[MonsterBase].HP
	f.Dispell(20)
	if f.Units[MonsterBase].HP != before {
		t.Errorf("哥布林不是不死生物,不該被驅散:%d → %d",
			before, f.Units[MonsterBase].HP)
	}
}

// TestDispellAlwaysKillsAtImpossibleThreshold / …NeverKills:兩端各釘一條,
// 免得把比較方向寫反 —— 反了之後「智能越高越難成功」,而畫面上看不出來。
func TestDispellDirectionOfThreshold(t *testing.T) {
	newField := func() *Field {
		f := &Field{Rand: NewRand(11)}
		f.Units[MonsterBase] = Unit{Name: "骷髏", HP: 10, Facing: South,
			IsMonster: true, Kind: 5, Tier: 1}
		return f
	}
	// 智能 100 → 門檻 360,d100 打不破 → 必定成功
	f := newField()
	f.Dispell(100)
	if f.Units[MonsterBase].Alive() {
		t.Error("門檻遠高於 d100 時應該必定驅散成功")
	}
	// 智能 0、階級 10 → 門檻負的 → 必定失敗
	f = newField()
	f.Dispell(0)
	if !f.Units[MonsterBase].Alive() {
		t.Error("門檻是負的時候不該驅散成功")
	}
}

// 命中的 +30 從**被縛**起算 —— 中毒(狀態 1)吃不到(docs/re/206 §2.1)。
//
// ⚠ 這一條擋的是把條件寫成 `>= 1`:兩種寫法只有「中毒」這一格分得開,
// 而中毒是戰鬥裡最常見的異常狀態。
func TestStatusToHitBonusStartsAtBound(t *testing.T) {
	base := ToHit(Unit{ToHit: 10, Facing: North}, Unit{Status: 0, Facing: South},
		Item{}, Item{})
	poisoned := ToHit(Unit{ToHit: 10, Facing: North}, Unit{Status: 1, Facing: South},
		Item{}, Item{})
	bound := ToHit(Unit{ToHit: 10, Facing: North}, Unit{Status: 2, Facing: South},
		Item{}, Item{})
	if poisoned != base {
		t.Errorf("中毒不該有 +30:正常 %d、中毒 %d", base, poisoned)
	}
	if bound != base+30 {
		t.Errorf("被縛應該 +30:正常 %d、被縛 %d", base, bound)
	}
}

// TestFieldStartsOnRoundOne —— 新建的戰場是**第 1 回合**,不是第 0。
//
// 原版的 `ds:9314` 初始化成 0,但它在每回合開頭 `inc`(docs/re/195 §1),
// 所以打第一回合時值是 1。引擎的第一回合不遞增 —— 從 0 起算的話,
// 群體傷害的閘門要寫成「≤ 1」才擋得住第一回合,而那會**連第二回合一起擋掉**。
// 症狀是玩家連兩回合放不出風暴,而畫面上寫著「第 0 回合」。
func TestFieldStartsOnRoundOne(t *testing.T) {
	f := Build(nil, nil, nil, NewRand(1))
	if f.Round != 1 {
		t.Errorf("新戰場的回合是 %d,應為 1(docs/re/195 §1)", f.Round)
	}
}
