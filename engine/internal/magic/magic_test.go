package magic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
)

func wizard(skills string) original.Character {
	return original.Character{Class: WizardClass, SP: 100, Skills: skills}
}

// docs/spec/09 §7 驗收 1:三個閘門各一個測試。
func TestCastGates(t *testing.T) {
	s := original.Spell{Name: "FIREBALL", School: 1, UnitCost: 2, Power: 3}
	for _, c := range []struct {
		name   string
		ch     original.Character
		invest int
		want   Fail
	}{
		{"法師 + 有技能 + 夠力", wizard("1000000000"), 4, OK},
		{"不是法師", original.Character{Class: '1', SP: 100, Skills: "1111111111"}, 4, FailNotWizard},
		{"技能旗標關著", wizard("0111111111"), 4, FailNoSkill},
		{"法力不足", func() original.Character { c := wizard("1000000000"); c.SP = 2; return c }(), 4, FailNoPoints},
		{"投不到一級", wizard("1000000000"), 1, FailBelowOneLevel},
	} {
		if got := CanCast(c.ch, s, c.invest); got != c.want {
			t.Errorf("%s:得 %v,應為 %v", c.name, got, c.want)
		}
	}
}

// 驗收 2:技能索引是**系別**,不是法術編號。
//
// ⚠ 三十三個法術共用五個旗標。寫成「法術編號 + 41」會讓 6 號以上的法術
// 全部查到界外 —— 這條測試就是要在那時候失敗。
func TestSkillIndexIsSchoolNotSpellIndex(t *testing.T) {
	ch := wizard("0000100000") // 只開第 5 系
	for _, c := range []struct {
		school int
		want   Fail
	}{{5, OK}, {1, FailNoSkill}, {4, FailNoSkill}} {
		s := original.Spell{Index: 30, School: c.school, UnitCost: 1, Power: 1}
		if got := CanCast(ch, s, 3); got != c.want {
			t.Errorf("系別 %d(法術編號 30):得 %v,應為 %v", c.school, got, c.want)
		}
	}
}

// 驗收 3 + 4:等級 = INT(投入 ÷ 單價);威力 = 每點威力 × **投入**。
func TestLevelAndPower(t *testing.T) {
	s := original.Spell{UnitCost: 3, Power: 4}
	for _, c := range []struct{ invest, lv, pw int }{
		{2, 0, 8}, {3, 1, 12}, {8, 2, 32}, {9, 3, 36},
	} {
		if got := Level(s, c.invest); got != c.lv {
			t.Errorf("投 %d:等級 %d,應為 %d", c.invest, got, c.lv)
		}
		if got := Power(s, c.invest); got != c.pw {
			t.Errorf("投 %d:威力 %d,應為 %d(乘投入不是乘等級)", c.invest, got, c.pw)
		}
	}
	// 單價 1 時等級與投入相同 —— 這正是「乘等級」的寫法測不出來的地方
	if Power(original.Spell{UnitCost: 1, Power: 4}, 5) != 20 {
		t.Error("單價 1 的情形應為 4×5")
	}
}

// 驗收 5:狀態強度 = 單價 ÷ 投入,**投得多、值小**。
func TestStatusMagnitudeShrinks(t *testing.T) {
	s := original.Spell{UnitCost: 12}
	a, b := StatusMagnitude(s, 2), StatusMagnitude(s, 6)
	if a != 6 || b != 2 {
		t.Errorf("投 2 → %d、投 6 → %d,應為 6 與 2", a, b)
	}
	if b >= a {
		t.Error("投得多應該讓強度變小(docs/spec/09 §4)")
	}
}

// 驗收 6:類別 3 與 13 不套用效果,訊息標「未解」。
func TestUnresolvedEffectsChangeNothing(t *testing.T) {
	for _, eff := range []int{EffAttr3, EffTransference, 99} {
		u := combat.Unit{HP: 10, Str: 5, Speed: 7}
		tgt := u
		r := Apply(original.Spell{Name: "X", Effect: eff, Power: 5},
			3, &combat.Unit{}, []*combat.Unit{&tgt})
		if !r.Unresolved {
			t.Errorf("類別 %d 應標成未解", eff)
		}
		if !strings.Contains(r.Message, "未解") && !strings.Contains(r.Message, "不在規格") {
			t.Errorf("類別 %d 的訊息沒有標出未解:%q", eff, r.Message)
		}
		if tgt != u {
			t.Errorf("類別 %d 改動了目標:%+v → %+v", eff, u, tgt)
		}
	}
}

// 驗收 7:魔法道具的門檻是**大於** 26。
func TestMagicItemThreshold(t *testing.T) {
	always := &combat.ScriptRand{Values: []int{1}} // 一定成功
	if ItemTriggers(26, 26, always) {
		t.Error("編號 26 不是魔法道具(門檻是 > 26)")
	}
	if !ItemTriggers(27, 26, always) {
		t.Error("編號 27 且成功率 26 應必定發動")
	}
	never := &combat.ScriptRand{Values: []int{combat.ToHitFaces}}
	if ItemTriggers(27, 1, never) {
		t.Error("擲出最大值且成功率 1,不該發動")
	}
}

// 驗收 8:原版 33 個法術全部有分派,沒有掉進「不在規格裡」。
func TestAllShippedSpellsDispatch(t *testing.T) {
	var d []byte
	var err error
	for _, dir := range []string{"/game/sharspri", "../../../game/sharspri"} {
		if d, err = os.ReadFile(filepath.Join(dir, "SPELLS.DAT")); err == nil {
			break
		}
	}
	if err != nil {
		t.Skip(err)
	}
	spells, err := original.ParseSpells(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(spells) != 33 {
		t.Fatalf("解出 %d 個法術,應為 33", len(spells))
	}
	unresolved := 0
	for _, s := range spells {
		tgt := combat.Unit{HP: 50}
		r := Apply(s, 4, &combat.Unit{}, []*combat.Unit{&tgt})
		if strings.Contains(r.Message, "不在規格裡") {
			t.Errorf("%s(類別 %d)沒有分派", s.Name, s.Effect)
		}
		if r.Unresolved {
			unresolved++
		}
	}
	// 類別 3 與 13 是已知未解的。數字變了表示資料或規格變了。
	if unresolved != 4 {
		t.Errorf("未解的法術 %d 個,docs/spec/09 §3 預期 4"+
			"(類別 3 有 3 個、類別 13 有 1 個)—— 數字變了要回去看規格", unresolved)
	}
}
