package town

import (
	"testing"

	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

func skillHero(pts int, skills string) original.Character {
	if skills == "" {
		skills = "0000000000"
	}
	return original.Character{Name: "測試戰士", Class: byte(rules.ClassHero), SkillPts: pts, Skills: skills}
}

func skillWizard(pts int, skills string) original.Character {
	if skills == "" {
		skills = "0000000000"
	}
	return original.Character{Name: "測試巫師", Class: byte(rules.ClassWizard), SkillPts: pts, Skills: skills}
}

// 驗收 #3:學一項技能後 Skills 對應那一格變 '1'、SkillPts 扣掉正確成本。
func TestLearnSkill_OK(t *testing.T) {
	c := skillHero(2, "")
	if got := LearnSkill(&c, 3); got != LearnOK { // 釘頭鎚,成本 1
		t.Fatalf("LearnSkill = %v,想要 LearnOK", got)
	}
	if c.Skills[2] != '1' {
		t.Errorf("Skills[2] = %q,想要 '1'", c.Skills[2])
	}
	if c.SkillPts != 1 {
		t.Errorf("SkillPts = %d,想要 2-1=1", c.SkillPts)
	}
	// 其餘格子不動
	for i, b := range c.Skills {
		if i != 2 && b != '0' {
			t.Errorf("Skills[%d] = %q,不該被動到", i, b)
		}
	}
}

// 編號超出 1–10。
func TestLearnSkill_BadNumber(t *testing.T) {
	c := skillHero(99, "")
	for _, n := range []int{0, -1, 11, 99} {
		got := LearnSkill(&c, n)
		if got != LearnBadNumber {
			t.Errorf("n=%d: LearnSkill = %v,想要 LearnBadNumber", n, got)
		}
	}
	if c.SkillPts != 99 {
		t.Errorf("SkillPts 被動到了:%d", c.SkillPts)
	}
}

// 驗收 #4:戰士與巫師用各自的表——同一格是不同技能,成本也不同。
// 編號 1:戰士是「劍」(成本 2),巫師是「火誌」(成本 5)。
func TestLearnSkill_HeroAndWizardUseDifferentTables(t *testing.T) {
	hero := skillHero(2, "")
	if got := LearnSkill(&hero, 1); got != LearnOK {
		t.Fatalf("戰士學編號 1:LearnSkill = %v,想要 LearnOK(成本 2,剛好夠)", got)
	}
	if hero.SkillPts != 0 {
		t.Errorf("戰士成本應為 2,剩 %d 點", hero.SkillPts)
	}
	heroName, _ := SkillName(rules.ClassHero, 1)
	if heroName != "劍" {
		t.Fatalf("戰士編號 1 應為「劍」,得 %q", heroName)
	}

	wiz := skillWizard(2, "")
	if got := LearnSkill(&wiz, 1); got != LearnNotEnough {
		t.Fatalf("巫師學編號 1(成本 5,只有 2 點):LearnSkill = %v,想要 LearnNotEnough", got)
	}
	if wiz.SkillPts != 2 {
		t.Errorf("擋下時不該扣點,剩 %d 點", wiz.SkillPts)
	}
	wizName, _ := SkillName(rules.ClassWizard, 1)
	if wizName != "火誌" {
		t.Fatalf("巫師編號 1 應為「火誌」,得 %q", wizName)
	}
	if heroName == wizName {
		t.Fatal("兩張表同一格應該是不同技能")
	}
}

// 驗收 #5:已學過,擋下且不扣點。
func TestLearnSkill_Already(t *testing.T) {
	c := skillHero(5, "0010000000") // 編號 3(釘頭鎚)已經學過
	before := c.SkillPts
	got := LearnSkill(&c, 3)
	if got != LearnAlready {
		t.Fatalf("LearnSkill = %v,想要 LearnAlready", got)
	}
	if c.SkillPts != before {
		t.Errorf("擋下卻扣了點:%d → %d", before, c.SkillPts)
	}
	if c.Skills != "0010000000" {
		t.Errorf("Skills 被動到了:%q", c.Skills)
	}
}

// 驗收 #5:點數不足,擋下且不扣點。
func TestLearnSkill_NotEnough(t *testing.T) {
	c := skillHero(0, "") // 護甲成本 4,一點都沒有
	got := LearnSkill(&c, 7)
	if got != LearnNotEnough {
		t.Fatalf("LearnSkill = %v,想要 LearnNotEnough", got)
	}
	if c.SkillPts != 0 {
		t.Errorf("擋下卻扣了點:%d", c.SkillPts)
	}
	if c.Skills[6] != '0' {
		t.Errorf("Skills[6] 被動到了:%q", c.Skills[6])
	}
}

// 驗收 #6(端到端):學會打獵(戰士第 9 項)之後 CanHunt 真的回 SkillOK。
func TestLearnSkill_HuntEndToEnd(t *testing.T) {
	c := skillHero(2, "")
	if got := CanHunt(c, true); got != SkillNoSkill {
		t.Fatalf("學會之前 CanHunt = %v,想要 SkillNoSkill", got)
	}
	if got := LearnSkill(&c, SkillHunting); got != LearnOK {
		t.Fatalf("學打獵:LearnSkill = %v,想要 LearnOK", got)
	}
	if got := CanHunt(c, true); got != SkillOK {
		t.Fatalf("學會之後 CanHunt = %v,想要 SkillOK", got)
	}
}
