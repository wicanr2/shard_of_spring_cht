package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
	"shardofspring/internal/rules"
	"shardofspring/internal/town"
	"shardofspring/internal/ui"
)

// 角色創造畫面。流程照 `CHARUTIL.EXE` 的實跑(docs/re/143 §2):
//
//	選種族 → 擲五項(種族修正另一欄)→ 重擲回合 → 選職業 → 輸入名稱
//
// ⚠ **先選種族再選職業**,因為職業由種族決定 —— 五個種族只有人類能選。
// 反過來寫會讓玩家選完職業才被告知種族不合。

type createStep int

const (
	stepRace createStep = iota
	stepAdjust
	stepClass
	stepName
)

type createState struct {
	step   createStep
	race   rules.Race
	class  rules.Class
	rolled town.Rolled
	// picked 是這一輪選中要重擲的項目(1–5)。原版是**切換**,不是累加。
	picked map[int]bool
	round  int // 對應原版的 `Adjustment N`
	name   string
	msg    string
}

func (g *Game) openCreate() {
	g.create = &createState{step: stepRace, picked: map[int]bool{}, round: 1}
}

func (g *Game) createKey(k ebiten.Key) {
	c := g.create
	if k == ebiten.KeyEscape && c.step != stepAdjust {
		g.create = nil
		return
	}
	switch c.step {
	case stepRace:
		races := map[ebiten.Key]rules.Race{
			ebiten.KeyH: rules.Human, ebiten.KeyD: rules.Dwarf,
			ebiten.KeyT: rules.Troll, ebiten.KeyE: rules.Elf,
			ebiten.KeyG: rules.Gnome,
		}
		r, ok := races[k]
		if !ok {
			return
		}
		c.race = r
		c.rolled = town.RollAll(g.rand)
		c.step = stepAdjust

	case stepAdjust:
		if n := int(k-ebiten.Key1) + 1; n >= 1 && n <= 5 {
			c.picked[n] = !c.picked[n] // 切換,與原版一致
			return
		}
		if k != ebiten.KeyEscape {
			return
		}
		if len(c.picked) > 0 {
			for n, on := range c.picked {
				if on {
					c.rolled = town.Reroll(c.rolled, n, g.rand)
				}
			}
			c.picked = map[int]bool{}
			c.round++
			return
		}
		// 沒選任何一項的 ESC = 這一步做完了
		if allowed := rules.Races[c.race].Classes; len(allowed) == 1 {
			c.class = allowed[0]
			c.step = stepName
		} else {
			c.step = stepClass
		}

	case stepClass:
		switch k {
		case ebiten.KeyF:
			c.class = rules.ClassHero
			c.step = stepName
		case ebiten.KeyW:
			c.class = rules.ClassWizard
			c.step = stepName
		}

	case stepName:
		if k == ebiten.KeyBackspace && c.name != "" {
			r := []rune(c.name)
			c.name = string(r[:len(r)-1])
			return
		}
		if k == ebiten.KeyEnter || k == ebiten.KeyKPEnter {
			g.finishCreate()
		}
	}
}

// createRunes 收玩家打的字。名稱上限 10(docs/re/143 §4:創造的提示是 10,
// 改名的提示才是 9 —— 記錄的欄位是 10 bytes,以 10 為準)。
func (g *Game) createRunes(rs []rune) {
	c := g.create
	if c == nil || c.step != stepName {
		return
	}
	for _, r := range rs {
		if r < ' ' {
			continue
		}
		if len([]rune(c.name)) >= town.NameMaxRunes {
			return
		}
		c.name += string(r)
	}
}

func (g *Game) finishCreate() {
	c := g.create
	name := strings.TrimSpace(c.name)
	id, r := town.Create(g.chars, c.race, c.class, c.rolled, name)
	if r != town.CreateOK {
		c.msg = r.String()
		return
	}
	if err := g.writeChars(); err != nil {
		g.warnings = append(g.warnings, "名冊存檔失敗:"+err.Error())
	}
	g.create = nil
	// docs/spec/20-skill-allocation.md:創角發的技能點要有地方花——接技能點
	// 分配畫面(蓋掉名冊,不另開視窗)。「建立了 XXX」的名冊訊息延到玩家按 0
	// 離開分配畫面之後才設,兩段合起來才是完整的創角流程。
	g.openSkillAlloc(id, -1, func(g *Game) {
		if g.roster != nil {
			g.roster.msg = fmt.Sprintf("建立了 %s(第 %d 槽)", name, id)
			g.roster.cursor = id - 1
		}
	})
}

// drawCreate 畫創造畫面。版面照原版:左邊角色表、右邊提示。
func (g *Game) drawCreate(dst *ebiten.Image) {
	c, p := g.create, g.panel
	if c == nil || p == nil {
		return
	}
	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)
	line := func(s string) {
		p.Draw(dst, s, x, y)
		y += lh
	}

	line("建立角色")
	y += lh * 0.5
	if c.step == stepRace {
		line("選種族:")
		for _, r := range []rules.Race{rules.Human, rules.Dwarf, rules.Troll, rules.Elf, rules.Gnome} {
			info := rules.Races[r]
			cls := "戰士／巫師"
			if len(info.Classes) == 1 && info.Classes[0] == rules.ClassHero {
				cls = "只能戰士"
			} else if len(info.Classes) == 1 {
				cls = "只能巫師"
			}
			line(fmt.Sprintf("%c) %s　%s", r, info.Name, cls))
		}
		return
	}

	info := rules.Races[c.race]
	line("種族:" + info.Name)
	y += lh * 0.3
	rows := []struct {
		n    int
		name string
		roll int
		mod  int
	}{
		{1, "速度", c.rolled.Speed, info.Speed},
		{2, "力量", c.rolled.Str, info.Str},
		{3, "智能", c.rolled.Int, info.Int},
		{4, "體能", c.rolled.End, info.End},
		{5, "技巧", c.rolled.Skill, info.Skill},
	}
	for _, r := range rows {
		mark := "  "
		if c.picked[r.n] {
			mark = "▶ " // 選中要重擲
		}
		// ⚠ 擲出值與種族修正**分兩欄**顯示,與原版一致 —— 修正是事後加的
		line(fmt.Sprintf("%s%d) %s　%3d　%+d　→ %d",
			mark, r.n, r.name, r.roll, r.mod, r.roll+r.mod))
	}
	y += lh * 0.3
	line(fmt.Sprintf("生命值 %d（= 加完修正的體能）", c.rolled.End+info.End))
	y += lh * 0.3

	switch c.step {
	case stepAdjust:
		line(fmt.Sprintf("第 %d 輪調整:按 1–5 選要重擲的項目,ESC 執行;", c.round))
		line("沒選任何一項時按 ESC 進到下一步。")
	case stepClass:
		line("選職業:F) 戰士　W) 巫師")
	case stepName:
		line(fmt.Sprintf("名稱(最多 %d 字):%s_", town.NameMaxRunes, c.name))
		line("Enter 確定,ESC 取消")
	}
	y += lh * 0.5
	if c.msg != "" {
		line("⚠ " + c.msg)
	}
}
