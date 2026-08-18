package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/town"
)

// 測試全部**不開視窗**:直接呼叫方法,不跑 ebiten.RunGame,也不呼叫
// drawSkillAlloc(那支需要 g.panel,是唯一要 GPU/字型資源的部分)。
// docs/spec/20 §6 的七條驗收,對應關係寫在每個測試前面。

// 驗收 #1:創角走完後可以分配技能,0 離開;離開後名冊裡的那一份也更新了。
func TestFinishCreate_OpensSkillAllocThenSpendAndClose(t *testing.T) {
	g := &Game{
		chars:  make([]original.Character, 25),
		roster: &rosterState{open: true},
		create: &createState{
			race:   rules.Human,
			class:  rules.ClassHero,
			rolled: town.Rolled{Speed: 5, Str: 5, Int: 2, End: 5, Skill: 5},
			name:   "阿爾",
		},
	}

	g.finishCreate()
	if g.create != nil {
		t.Fatal("finishCreate 之後 g.create 應該是 nil(交給技能點分配畫面接手)")
	}
	if g.skillAlloc == nil {
		t.Fatal("創角完成後應該打開技能點分配畫面")
	}
	if g.roster.msg != "" {
		t.Fatalf("關閉分配畫面之前不該先設名冊訊息,得 %q", g.roster.msg)
	}
	// 人類 Int 修正 +2(rules.Races[Human].Int),SkillPoints = 智能。
	if g.chars[0].SkillPts != 4 {
		t.Fatalf("前置條件跑掉了:SkillPts = %d,想要 4", g.chars[0].SkillPts)
	}

	// 學編號 1(劍,成本 2)。
	g.skillAllocKey(ebiten.KeyDigit1)
	g.skillAllocKey(ebiten.KeyEnter)
	if g.chars[0].Skills[0] != '1' {
		t.Fatalf("學完之後 Skills[0] 應該是 '1',得 %q", g.chars[0].Skills[0])
	}
	if g.chars[0].SkillPts != 2 {
		t.Fatalf("扣完成本應剩 2 點,得 %d", g.chars[0].SkillPts)
	}

	// 0 離開。
	g.skillAllocKey(ebiten.KeyDigit0)
	g.skillAllocKey(ebiten.KeyEnter)
	if g.skillAlloc != nil {
		t.Fatal("按 0 應該關閉技能點分配畫面")
	}
	if g.roster.msg == "" {
		t.Fatal("關閉之後應該接上原本「建立了 XXX」的名冊訊息")
	}
	if g.chars[0].SkillPts != 2 || g.chars[0].Skills[0] != '1' {
		t.Fatalf("離開之後名冊裡的那一份要保留學到的技能與剩餘點數,得 SkillPts=%d Skills=%q",
			g.chars[0].SkillPts, g.chars[0].Skills)
	}
}

// 驗收 #2:升級後可以分配,而且沒花完的點數會留到下次升級。
func TestTrainMember_OpensSkillAllocLeftoverCarriesOver(t *testing.T) {
	c := original.Character{
		ID: 1, Name: "測試", Class: byte(rules.ClassHero), Level: 1,
		End: 10, Exp: 999_999, MaxHP: 10, HP: 10, Skills: "0000000000",
	}
	g := &Game{
		members: []original.Character{c},
		chars:   make([]original.Character, 25),
		rand:    combat.NewRand(1),
		town:    &townState{},
	}
	g.chars[0] = c

	g.trainMember(0, 0) // guildExtra 0 = 武術(戰士)
	if g.skillAlloc == nil {
		t.Fatal("升級成功後應該打開技能點分配畫面")
	}
	pts1 := g.members[0].SkillPts
	if pts1 < 1 {
		t.Fatalf("升級至少發 1 點(SkillPtsPerLevel),得 %d", pts1)
	}

	// 不學任何技能,直接按 0 離開——點數不該憑空消失。
	g.skillAllocKey(ebiten.KeyDigit0)
	g.skillAllocKey(ebiten.KeyEnter)
	if g.skillAlloc != nil {
		t.Fatal("按 0 應該關閉技能點分配畫面")
	}
	if g.members[0].SkillPts != pts1 {
		t.Fatalf("沒花完的點數不該憑空消失:%d → %d", pts1, g.members[0].SkillPts)
	}

	// 再升一級,驗證上一輪沒花完的點數留到這一次(累加,不是被歸零重算)。
	g.trainMember(0, 0)
	if g.skillAlloc == nil {
		t.Fatal("第二次升級也應該打開技能點分配畫面")
	}
	pts2 := g.members[0].SkillPts
	if pts2 <= pts1 {
		t.Fatalf("第二次升級的點數應該是「上一輪剩的 + 這次新發的」,不能小於等於上一輪:%d → %d",
			pts1, pts2)
	}
}

// 驗收 #5(UI 層):已經學過的技能透過畫面重複學一次,不會再扣一次點。
func TestSkillAllocKey_SecondPressRemovesAndRefunds(t *testing.T) {
	c := original.Character{ID: 1, Class: byte(rules.ClassHero), SkillPts: 5, Skills: "0000000000"}
	g := &Game{chars: []original.Character{c}}
	g.openSkillAlloc(1, -1, nil)

	g.skillAllocKey(ebiten.KeyDigit3) // 釘頭鎚,成本 1
	g.skillAllocKey(ebiten.KeyEnter)
	if g.chars[0].SkillPts != 4 {
		t.Fatalf("第一次學完應剩 4 點,得 %d", g.chars[0].SkillPts)
	}

	// ⚠ 同一個編號按第二次 = **取消並退點**(原版右欄的 `(or remove)`)。
	// 而且**要寫回名冊** —— 只改暫存的話玩家按了看起來沒事,離開就白按。
	g.skillAllocKey(ebiten.KeyDigit3)
	g.skillAllocKey(ebiten.KeyEnter)
	if g.chars[0].SkillPts != 5 {
		t.Fatalf("取消應該退點:得 %d,想要 5", g.chars[0].SkillPts)
	}
	if g.chars[0].Skills[2] != '0' {
		t.Errorf("取消之後旗標應該清掉:%q", g.chars[0].Skills)
	}
	if g.skillAlloc.msg == "" {
		t.Error("取消應該有提示訊息")
	}
}

// 驗收 #5(UI 層):點數不足時透過畫面學技能,不會扣點。
func TestSkillAllocKey_NotEnoughDoesNotDeduct(t *testing.T) {
	c := original.Character{ID: 1, Class: byte(rules.ClassHero), SkillPts: 0, Skills: "0000000000"}
	g := &Game{chars: []original.Character{c}}
	g.openSkillAlloc(1, -1, nil)

	g.skillAllocKey(ebiten.KeyDigit7) // 護甲,成本 4,一點都沒有
	g.skillAllocKey(ebiten.KeyEnter)
	if g.chars[0].SkillPts != 0 {
		t.Fatalf("點數不足不該扣成負的:得 %d", g.chars[0].SkillPts)
	}
	if g.chars[0].Skills[6] != '0' {
		t.Errorf("點數不足不該學成:Skills[6] = %q", g.chars[0].Skills[6])
	}
}
