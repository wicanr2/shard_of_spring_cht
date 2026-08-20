package combat

import (
	"testing"

	"shardofspring/internal/music"
)

// 驗收 3(docs/spec/13 §8.3):一般命中出 HT、狂暴出 HK,兩者不會同時出現。
//
// ⚠ 兩個岔路要**各測一次** —— 只測其中一邊,把音效寫死成常數也會通過。
func TestAttackQueuesTheHitSound(t *testing.T) {
	for _, c := range []struct {
		name    string
		berserk int
		second  float64
		want    string
	}{
		{"一般命中", 0, 0.999999, music.FxHit},
		{"狂暴劈砍", 1, 0.999999, music.FxHack},
		{"有狂暴但沒擲中", 1, 0.0, music.FxHit},
	} {
		f := &Field{Items: map[int]Item{}}
		f.Units[PartyBase] = Unit{Name: "人", HP: 20, Facing: North, ToHit: 100,
			Str: 20, Weapon: BareHandMin, Berserk: c.berserk}
		f.Units[MonsterBase] = Unit{Name: "怪", HP: 200, Facing: South}
		f.Rand = &ScriptRand{Floats: []float64{0.0, c.second}, Values: []int{5}}
		f.Attack(PartyBase, MonsterBase)
		if len(f.Sounds) != 1 || f.Sounds[0] != c.want {
			t.Errorf("%s:音效 %v,應為 [%s]", c.name, f.Sounds, c.want)
		}
	}
}

// 驗收 4:被打死的單位額外出一個 DD,而且**一次死一個響一次**
// (docs/re/228 §4:那支 SUB 清的是單一個單位)。
func TestDeathQueuesOneSoundPerUnit(t *testing.T) {
	f := &Field{Items: map[int]Item{}}
	f.Units[PartyBase] = Unit{Name: "人", HP: 20, Facing: North, ToHit: 100,
		Str: 20, Weapon: BareHandMin}
	f.Units[MonsterBase] = Unit{Name: "怪甲", HP: 1, Facing: South}
	f.Units[MonsterBase+1] = Unit{Name: "怪乙", HP: 1, Facing: South}
	f.Rand = &ScriptRand{Floats: []float64{0.0, 0.0, 0.0, 0.0}, Values: []int{5, 5}}
	f.Attack(PartyBase, MonsterBase)
	f.Attack(PartyBase, MonsterBase+1)
	want := []string{music.FxHit, music.FxDie, music.FxHit, music.FxDie}
	if len(f.Sounds) != len(want) {
		t.Fatalf("音效 %v,應為 %v", f.Sounds, want)
	}
	for i := range want {
		if f.Sounds[i] != want[i] {
			t.Fatalf("音效 %v,應為 %v", f.Sounds, want)
		}
	}
}

// 沒打中就只有一句 `and missed!`,**沒有音效** ——
// 原版的 HT/HK 兩個呼叫端都在命中那一側(docs/re/228 §4)。
func TestMissMakesNoSound(t *testing.T) {
	f := &Field{Items: map[int]Item{}}
	f.Units[PartyBase] = Unit{Name: "人", HP: 20, Facing: North, ToHit: -1000,
		Str: 20, Weapon: BareHandMin}
	f.Units[MonsterBase] = Unit{Name: "怪", HP: 20, Facing: South}
	f.Rand = &ScriptRand{Floats: []float64{0.999999}, Values: []int{5}}
	if _, hit, _ := f.Attack(PartyBase, MonsterBase); hit {
		t.Fatal("命中門檻拉到 −1000 應該必失手")
	}
	if len(f.Sounds) != 0 {
		t.Errorf("失手不該有音效,得 %v", f.Sounds)
	}
}

// 規則層排的每一個代碼,播放層都要認得 —— 兩個套件各有一份常數,
// 這條把它們綁在一起。
func TestEveryQueuedCodeHasAWaveform(t *testing.T) {
	for _, c := range []string{music.FxHit, music.FxHack, music.FxDie, music.FxSpell} {
		if music.Effect(c) == nil {
			t.Errorf("規則層會排 %s,但 music.Effect 認不得它", c)
		}
	}
}
