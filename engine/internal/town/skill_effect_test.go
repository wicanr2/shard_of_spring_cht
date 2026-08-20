package town

import (
	"testing"

	"shardofspring/internal/rules"
)

// docs/spec/14 §13.1:三項技能目前沒有效果,而且**兩張表同一格是不同技能**,
// 所以要分職業各驗一次 —— 只驗一邊的話,把判斷寫成「n == 5 || n == 9 || n == 10」
// (不分職業)也會通過,而那會多標兩項(戰士的打獵、巫師的精神之誌)。
func TestSkillNoEffectIsPerClass(t *testing.T) {
	for _, c := range []struct {
		class rules.Class
		n     int
		want  bool
		why   string
	}{
		{rules.ClassHero, SkillNightVision, true, "夜視:沒有消費端"},
		{rules.ClassHero, SkillPersuasion, true, "說服:降價未解"},
		{rules.ClassHero, SkillMonsterLore, false, "戰士的第 9 格是打獵,有實作"},
		{rules.ClassHero, 6, false, "策略有實作"},
		{rules.ClassHero, 4, false, "空手道有實作"},
		{rules.ClassWizard, SkillMonsterLore, true, "怪物知識:沒有消費端"},
		{rules.ClassWizard, SkillNightVision, false, "巫師的第 5 格是精神之誌,有實作"},
		{rules.ClassWizard, 8, false, "物品知識在原版就是死技能,不歸引擎標(re/167 §4)"},
		{rules.ClassWizard, 10, false, "降魔術有實作"},
	} {
		if got := SkillNoEffect(c.class, c.n); got != c.want {
			t.Errorf("%s:SkillNoEffect(%c, %d) = %v,應為 %v", c.why, c.class, c.n, got, c.want)
		}
	}
}

// 有效果的技能不能被標到。⚠ 這條是**上界** —— 少標一項是漏,多標一項
// 會讓玩家避開一個其實有用的技能,後者更糟。
func TestOnlyThreeSkillsAreMarked(t *testing.T) {
	n := 0
	for _, class := range []rules.Class{rules.ClassHero, rules.ClassWizard} {
		for i := 1; i <= 10; i++ {
			if SkillNoEffect(class, i) {
				n++
			}
		}
	}
	if n != 3 {
		t.Errorf("標了 %d 項,應為 3(夜視 / 說服 / 怪物知識)—— "+
			"數字變了要回去看 docs/spec/14 §13.1", n)
	}
}

// 記號不能與「學過」那個 `*` 撞號 —— 兩件事要能同時看得到。
func TestNoEffectMarkIsNotTheLearnedMark(t *testing.T) {
	if SkillNoEffectMark == "*" || SkillNoEffectMark == "" {
		t.Errorf("記號 %q 與學過的 * 撞號或是空的", SkillNoEffectMark)
	}
}
