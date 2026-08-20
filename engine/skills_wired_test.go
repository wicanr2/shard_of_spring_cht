package main

import (
	"strings"
	"testing"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/town"
	"shardofspring/internal/world"
)

// docs/re/229 §1:沒有光的能見度預設 2,隊上有**戰士**帶夜視 → 3。
//
// ⚠ 職業要一起判。位移 46 在巫師表是「風誌」—— 只看旗標的話,
// 一個帶風系的巫師會讓全隊在黑暗裡多看一圈,而畫面上看起來完全正常。
func TestNightVisionNeedsHeroAndSkill(t *testing.T) {
	mk := func(class byte, nightVision bool) original.Character {
		c := original.Character{Class: class}
		flags := []byte("0000000000")
		if nightVision {
			flags[town.SkillNightVision-1] = '1'
		}
		c.Skills = string(flags)
		return c
	}
	for _, c := range []struct {
		name    string
		members []original.Character
		want    int
	}{
		{"沒有人有", []original.Character{mk('1', false), mk('2', false)}, world.VisDarkBase},
		{"戰士有夜視", []original.Character{mk('1', true)}, world.VisDarkNightVision},
		{"巫師同一格(風誌)不算", []original.Character{mk('2', true)}, world.VisDarkBase},
		{"隊上任何一個戰士有就算", []original.Character{mk('2', false), mk('1', true)}, world.VisDarkNightVision},
	} {
		g := &Game{members: c.members}
		g.refreshNightVision()
		if g.party.VisDark != c.want {
			t.Errorf("%s:VisDark = %d,應為 %d", c.name, g.party.VisDark, c.want)
		}
		// 存檔那一份要跟著走,否則讀回來又變回舊值。
		if g.group.VisDark != c.want {
			t.Errorf("%s:group.VisDark = %d,應為 %d", c.name, g.group.VisDark, c.want)
		}
	}
}

// 出貨的 PARTY #5 那一格是 2 —— 與程式碼寫的預設值相同(docs/re/229 §1.2)。
// 這條把那個獨立印證釘在測試裡:常數改掉就會紅。
func TestVisDarkBaseMatchesTheShippedSave(t *testing.T) {
	if world.VisDarkBase != 2 {
		t.Errorf("預設能見度 %d —— 出貨存檔量到的是 2(docs/re/229 §1.2)",
			world.VisDarkBase)
	}
	if world.VisDarkNightVision <= world.VisDarkBase {
		t.Errorf("夜視 %d 沒有比預設 %d 大 —— 那就不是「看得更遠」",
			world.VisDarkNightVision, world.VisDarkBase)
	}
}

// docs/re/229 §2.2:三個詳細度,而且兩條分支**各自把職業釘死**。
//
// ⚠ 只測「有技能就升級」會漏掉職業那一半,而位移 47/50 在另一張表
// 是打獵與武器知識 —— 混用會讓戰士帶打獵就看得到怪物數值。
func TestInspectTierNeedsClassAndSkill(t *testing.T) {
	for _, c := range []struct {
		name string
		u    combat.Unit
		want int
	}{
		{"素戰士", combat.Unit{Action: combat.ActionFighter}, combat.InspectBase},
		{"戰士+策略", combat.Unit{Action: combat.ActionFighter, Tactics: 1}, combat.InspectTactics},
		{"戰士+同一格的怪物知識旗標(其實是打獵)",
			combat.Unit{Action: combat.ActionFighter, MonsterLore: 1}, combat.InspectBase},
		{"巫師+怪物知識", combat.Unit{Action: combat.ActionWizard, MonsterLore: 1}, combat.InspectLore},
		{"巫師+同一格的策略旗標(其實是武器知識)",
			combat.Unit{Action: combat.ActionWizard, Tactics: 1}, combat.InspectBase},
		{"怪物", combat.Unit{Action: 0, Tactics: 1, MonsterLore: 1}, combat.InspectBase},
	} {
		if got := combat.InspectTier(c.u); got != c.want {
			t.Errorf("%s:詳細度 %d,應為 %d", c.name, got, c.want)
		}
	}
}

// 面板的三個樣子:低詳細度看不到數值,而**看不到不是錯誤**。
func TestInspectPanelRespectsTheTier(t *testing.T) {
	mkGame := func(actor combat.Unit) *Game {
		f := &combat.Field{}
		f.Units[combat.PartyBase] = actor
		f.Units[combat.MonsterBase] = combat.Unit{
			Name: "怪", HP: 10, IsMonster: true, Speed: 7, ToHit: 30, Str: 12,
			Target: combat.PartyBase,
		}
		f.Units[combat.PartyBase].Name = "人"
		g := &Game{field: f, actor: combat.PartyBase}
		g.inspect = &inspectState{idx: combat.MonsterBase}
		return g
	}
	// ⚠ 用 Contains 不是切前綴 —— 中文是多位元組,`l[:3]` 只切得到
	// 一個全形空白,而那個比較**永遠不會相等**(第一版就踩了)。
	has := func(lines []string, sub string) bool {
		for _, l := range lines {
			if strings.Contains(l, sub) {
				return true
			}
		}
		return false
	}
	hasNumbers := func(lines []string) bool { return has(lines, inspectSpeed) }
	hasSeeks := func(lines []string) bool { return has(lines, "目標>") }

	base := mkGame(combat.Unit{Action: combat.ActionFighter})
	if l := base.inspectLines(); hasNumbers(l) || hasSeeks(l) {
		t.Errorf("詳細度 1 不該有數值也不該有目標:%v", l)
	}
	tac := mkGame(combat.Unit{Action: combat.ActionFighter, Tactics: 1})
	tac.field.Tactics = true
	if l := tac.inspectLines(); hasNumbers(l) || !hasSeeks(l) {
		t.Errorf("詳細度 2 應該只多目標那一行:%v", l)
	}
	lore := mkGame(combat.Unit{Action: combat.ActionWizard, MonsterLore: 1})
	lore.field.Tactics = true // 就算隊上有人會策略,巫師也拿不到「目標>」
	if l := lore.inspectLines(); !hasNumbers(l) || hasSeeks(l) {
		t.Errorf("詳細度 3 應該只多數值,不該有目標那一行:%v", l)
	}
}

// 兩項技能接上了 → 就要從「學了沒有效果」的清單上撤掉。
//
// ⚠ 這條擋的是**忘了撤**:清單留著一個已經有效果的技能,
// 玩家會避開一個其實有用的技能,而那比漏標更糟。
func TestWiredSkillsAreOffTheNoEffectList(t *testing.T) {
	if town.SkillNoEffect(rules.ClassHero, town.SkillNightVision) {
		t.Error("夜視已經接上了(docs/re/229 §1),不該再標成沒有效果")
	}
	if town.SkillNoEffect(rules.ClassWizard, town.SkillMonsterLore) {
		t.Error("怪物知識已經接上了(docs/re/229 §3.2),不該再標成沒有效果")
	}
}
