package combat

import (
	"strings"
	"testing"

	"shardofspring/internal/original"
)

// scriptRand 讓每一次 Float01 依序回指定值,Roll 一律回最小值。
type scriptRand struct {
	floats []float64
	i      int
}

func (r *scriptRand) Roll(faces int) int { return 1 }
func (r *scriptRand) Float01() float64 {
	if r.i < len(r.floats) {
		v := r.floats[r.i]
		r.i++
		return v
	}
	return 0.99
}

// poisonField 擺一隻用毒牙(武器 62)的怪咬一位隊員,而且必定命中、必有傷害。
func poisonField(rnd FloatRand) *Field {
	f := &Field{Rand: rnd}
	f.Units[MonsterBase] = Unit{Name: "眼鏡蛇", HP: 10, Facing: South,
		ToHit: 50, Str: 20, Weapon: PoisonWeapon, Armor: original.NotEquipped, IsMonster: true}
	f.Units[PartyBase] = Unit{Name: "凱恩", HP: 30, Facing: North,
		ToHit: 10, Str: 10, Weapon: original.NotEquipped, Armor: original.NotEquipped}
	return f
}

// TestPoisonOnHit:武器 62 + 擲骰 < 0.15 + 狀態正常 → 中毒(docs/re/191)。
func TestPoisonOnHit(t *testing.T) {
	// 第一顆是命中擲骰、第二顆是狂暴、第三顆才是中毒。
	f := poisonField(&scriptRand{floats: []float64{0.0, 0.0, 0.10}})
	f.Attack(MonsterBase, PartyBase)
	if got := f.Units[PartyBase].Status; got != StatusPoisoned {
		t.Fatalf("應該中毒,狀態 = %d", got)
	}
	if last := f.Log[len(f.Log)-1]; !strings.Contains(last, "並中毒了!") {
		t.Errorf("訊息要接上中毒那一句,得到 %q", last)
	}
}

// TestPoisonMissesTheRoll:擲骰 ≥ 0.15 就不中。
func TestPoisonMissesTheRoll(t *testing.T) {
	f := poisonField(&scriptRand{floats: []float64{0.0, 0.0, 0.15}})
	f.Attack(MonsterBase, PartyBase)
	if f.Units[PartyBase].Status != 0 {
		t.Error("0.15 不小於 0.15,不該中毒")
	}
}

// TestPoisonNeedsPoisonWeapon:武器不是 62 就不會中毒,
// **但那一顆骰照樣消耗掉** —— 原版三項各算完才 AND(docs/re/191 §2)。
func TestPoisonNeedsPoisonWeapon(t *testing.T) {
	r := &scriptRand{floats: []float64{0.0, 0.0, 0.10}}
	f := poisonField(r)
	f.Units[MonsterBase].Weapon = 3
	f.Attack(MonsterBase, PartyBase)
	if f.Units[PartyBase].Status != 0 {
		t.Error("普通武器不該讓人中毒")
	}
	if r.i != 3 {
		t.Errorf("中毒那一顆骰要照樣擲(亂數消耗順序),用掉 %d 顆", r.i)
	}
}

// TestPoisonOnlyHitsParty:怪物不會被咬到中毒(索引 ≤ 8 那一段直接返回),
// 而且**連骰都不擲**。
func TestPoisonOnlyHitsParty(t *testing.T) {
	r := &scriptRand{floats: []float64{0.0, 0.0, 0.10}}
	f := poisonField(r)
	f.Units[MonsterBase+1] = Unit{Name: "地精", HP: 30, Facing: North,
		Weapon: original.NotEquipped, Armor: original.NotEquipped, IsMonster: true}
	f.Units[PartyBase].Weapon = PoisonWeapon // 就算隊員拿得到 62
	f.Attack(PartyBase, MonsterBase+1)
	if f.Units[MonsterBase+1].Status != 0 {
		t.Error("怪物不該中毒")
	}
	if r.i != 2 {
		t.Errorf("防禦者是怪物時不擲中毒骰,用掉 %d 顆", r.i)
	}
}

// TestPoisonSkipsAbnormalStatus:已經有狀態的人不會再中毒(狀態 == 0 才算)。
func TestPoisonSkipsAbnormalStatus(t *testing.T) {
	f := poisonField(&scriptRand{floats: []float64{0.0, 0.0, 0.10}})
	f.Units[PartyBase].Status = 4 // 冰封
	f.Attack(MonsterBase, PartyBase)
	if got := f.Units[PartyBase].Status; got != 4 {
		t.Errorf("原有狀態不該被中毒蓋掉,得到 %d", got)
	}
}

// TestPoisonSkipsTheDead:這一擊打死了就不擲中毒骰(原版走死亡分支)。
func TestPoisonSkipsTheDead(t *testing.T) {
	r := &scriptRand{floats: []float64{0.0, 0.0, 0.10}}
	f := poisonField(r)
	f.Units[PartyBase].HP = 1
	f.Attack(MonsterBase, PartyBase)
	if f.Units[PartyBase].Alive() {
		t.Skip("這一擊沒打死,測不到")
	}
	if f.Units[PartyBase].Status == StatusPoisoned {
		t.Error("死了不該再中毒")
	}
	if r.i != 2 {
		t.Errorf("死亡分支不擲中毒骰,用掉 %d 顆", r.i)
	}
}

// ── 自然武器名(docs/re/192)────────────────────────────────────────────

// TestNaturalWeaponNames:59–62 不在 ITEMS.DAT 裡,名字要走手工補的那四格。
func TestNaturalWeaponNames(t *testing.T) {
	f := &Field{Items: map[int]Item{2: {Name: "短劍"}}}
	for _, c := range []struct {
		w    int
		want string
	}{
		{2, "短劍"},   // 檔案裡的
		{59, "無"},   // `None`
		{60, "拳頭"},  // `Hands`
		{61, "咬擊"},  // `Bite`
		{62, "獠牙"},  // `Fangs`
		{99, "拳頭"},  // 「沒裝備」的哨兵 → 退回赤手
	} {
		if got := f.weaponName(c.w); got != c.want {
			t.Errorf("武器 %d 的名字應該是 %q,得到 %q", c.w, c.want, got)
		}
	}
}

// TestPoisonWeaponIsFangs 把 191 與 192 綁在一起:會下毒的那個編號,
// 名字必須是「獠牙」。兩個結論來自不同的碼段,對不上就是有一邊讀錯了。
func TestPoisonWeaponIsFangs(t *testing.T) {
	if got := naturalWeapons[PoisonWeapon]; got != "獠牙" {
		t.Errorf("武器 %d 應該叫獠牙,得到 %q", PoisonWeapon, got)
	}
}

// TestSnakeAttackSaysFangs:訊息裡真的印得出來(接線,不只是查表)。
func TestSnakeAttackSaysFangs(t *testing.T) {
	f := poisonField(&scriptRand{floats: []float64{0.0, 0.0, 0.99}})
	f.Attack(MonsterBase, PartyBase)
	if last := f.Log[len(f.Log)-1]; !strings.Contains(last, "使用 獠牙") {
		t.Errorf("毒蛇攻擊要說「使用 獠牙」,得到 %q", last)
	}
}
