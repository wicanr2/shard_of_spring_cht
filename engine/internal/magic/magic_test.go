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
		{"巫師 + 有技能 + 夠力", wizard("1000000000"), 4, OK},
		{"不是巫師", original.Character{Class: '1', SP: 100, Skills: "1111111111"}, 4, FailNotWizard},
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

// 驗收 6:仍未解的類別不套用效果,訊息標「未解」。
//
// ⚠ 類別 3 **已經不在這個清單裡**(docs/re/171 §3:它是命中能力)——
// 這條測試從三個縮到兩個,是解出來的結果,不是放寬。
func TestUnresolvedEffectsChangeNothing(t *testing.T) {
	for _, eff := range []int{EffTransference, 99} {
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
	// 只剩類別 13(TRANSFERENCE)一個未解。
	// ⚠ 類別 3 的三個已經解出來了(docs/re/171 §3),所以這裡從 4 變成 1 ——
	// **數字變了要回去看規格**,這一次規格也跟著改了。
	if unresolved != 1 {
		t.Errorf("未解的法術 %d 個,docs/spec/09 §3 預期 1"+
			"(只剩類別 13)—— 數字變了要回去看規格", unresolved)
	}
}

// 類別 3 改的是**命中能力**(docs/re/171 §3),而且與 4/5/6 同一條路徑。
func TestEffectClassThreeChangesToHit(t *testing.T) {
	for _, c := range []struct {
		eff   int
		field func(combat.Unit) int
		name  string
	}{
		{EffToHit, func(u combat.Unit) int { return u.ToHit }, "命中能力"},
		{EffStrength, func(u combat.Unit) int { return u.Str }, "力量"},
		{EffHitPoints, func(u combat.Unit) int { return u.HP }, "生命值"},
		{EffSpeed, func(u combat.Unit) int { return u.Speed }, "速度"},
	} {
		tgt := combat.Unit{HP: 10, Str: 5, Speed: 7, ToHit: 11}
		before := c.field(tgt)
		r := Apply(original.Spell{Name: "X", Effect: c.eff, Power: 3},
			1, &combat.Unit{}, []*combat.Unit{&tgt})
		if r.Unresolved {
			t.Errorf("類別 %d 不該再標未解", c.eff)
		}
		if got := c.field(tgt); got == before {
			t.Errorf("類別 %d 應該改動%s,但沒變(%d)", c.eff, c.name, got)
		}
	}
	// ⛔ 這條的依據是**讀到的類別 → 屬性欄對應**,不是從
	// `Becomes clumsy` 這個名字推的 —— docs/spec/09 §3 當初明文禁止那樣做。
}

// ── F3:法術結果的措辭,以及它翻出來的一條規則 ─────────────────────────

// TestUnbindCoversAllThreeBindingStatuses 釘住「解除束縛吃三種狀態」。
//
// ⚠ 這**不是**措辭調整,是規則:引擎原本只解 `Status == 2`。
// 依據是 CMBT 自己的字串 `is not bound in chains and still air and ice`
// (CMBT:134–137)—— 原版把三種並列,所以三種都在這個法術的範圍內。
// 少解兩種在畫面上沒有症狀:凝滯/冰封的角色照樣顯示狀態,
// 玩家只會以為「這個法術對他沒用」。
func TestUnbindCoversAllThreeBindingStatuses(t *testing.T) {
	s := original.Spell{Name: "UNBIND", Effect: EffUnbind}
	for _, st := range []int{StatusBound, StatusStill, StatusFrozen} {
		tgt := combat.Unit{Name: "隊員", HP: 10, Status: st, StatMag: 3}
		r := Apply(s, 1, &combat.Unit{}, []*combat.Unit{&tgt})
		if tgt.Status != 0 {
			t.Errorf("狀態 %d 應該被解掉,得到 %d", st, tgt.Status)
		}
		if !strings.Contains(r.Message, "掙脫了!") {
			t.Errorf("狀態 %d 解掉之後要說「掙脫了!」,得到 %q", st, r.Message)
		}
	}
	// 中毒**不算**束縛 —— 那是解毒(類別 9)的事。
	poisoned := combat.Unit{Name: "隊員", HP: 10, Status: 1}
	r := Apply(s, 1, &combat.Unit{}, []*combat.Unit{&poisoned})
	if poisoned.Status != 1 {
		t.Error("中毒不該被解除束縛清掉")
	}
	if !strings.Contains(r.Message, MsgNotBound) {
		t.Errorf("沒有被束縛時要說原版那一長串,得到 %q", r.Message)
	}
}

// TestZeroPowerSaysNoDifference:威力算出來是 0 時不要印「力量 +0」。
func TestZeroPowerSaysNoDifference(t *testing.T) {
	s := original.Spell{Name: "STRENGTH", Effect: EffStrength, Power: 0}
	tgt := combat.Unit{Name: "隊員", HP: 10, Str: 12}
	r := Apply(s, 1, &combat.Unit{}, []*combat.Unit{&tgt})
	if tgt.Str != 12 {
		t.Errorf("威力 0 不該動屬性,力量變成 %d", tgt.Str)
	}
	if !strings.Contains(r.Message, MsgNoDifference) {
		t.Errorf("威力 0 要說「%s」,得到 %q", MsgNoDifference, r.Message)
	}
}

// TestCureWithNothingToCureSaysNoDifference:身上沒狀態時解毒等於沒解。
func TestCureWithNothingToCureSaysNoDifference(t *testing.T) {
	s := original.Spell{Name: "CURE", Effect: EffCure}
	ok := combat.Unit{Name: "隊員", HP: 10, Status: 0}
	if r := Apply(s, 1, &combat.Unit{}, []*combat.Unit{&ok}); !strings.Contains(r.Message, MsgNoDifference) {
		t.Errorf("沒有狀態可解時要說「%s」,得到 %q", MsgNoDifference, r.Message)
	}
	sick := combat.Unit{Name: "隊員", HP: 10, Status: 1}
	if r := Apply(s, 1, &combat.Unit{}, []*combat.Unit{&sick}); !strings.Contains(r.Message, "被治癒了。") {
		t.Errorf("真的解掉時要說「被治癒了。」,得到 %q", r.Message)
	}
}

// TestDamageSpellReportsDeath:法術打死人要說「並死亡!」(CMBT:122)。
func TestDamageSpellReportsDeath(t *testing.T) {
	s := original.Spell{Name: "FIREBALL", Effect: EffSingleDamage, Power: 50}
	tgt := combat.Unit{Name: "地精", HP: 3, Facing: combat.South, IsMonster: true}
	r := Apply(s, 1, &combat.Unit{}, []*combat.Unit{&tgt})
	if tgt.Alive() {
		t.Fatalf("fixture 失效:50 點傷害應該打死 3 點生命的怪物,剩 %d", tgt.HP)
	}
	if !strings.Contains(r.Message, MsgDies) {
		t.Errorf("打死了要說「%s」,得到 %q", MsgDies, r.Message)
	}
	// 沒打死就不要說
	alive := combat.Unit{Name: "巨龍", HP: 200, Facing: combat.South, IsMonster: true}
	if r := Apply(s, 1, &combat.Unit{}, []*combat.Unit{&alive}); strings.Contains(r.Message, MsgDies) {
		t.Errorf("沒打死不該說死了:%q", r.Message)
	}
}
