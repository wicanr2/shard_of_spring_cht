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
	// stepKeep 是**最後一步**:CHARUTIL:33「Do you wish to keep this
	// character (Y/N) ?」。原版問在命名之後,印在整個畫面最下方 ——
	// 看得到成品才決定留不留(2026-08-18 實跑 `r3e4-name.png`)。
	stepKeep
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
		for n, on := range c.picked {
			if on {
				c.rolled = town.Reroll(c.rolled, n, g.rand)
			}
		}
		c.picked = map[int]bool{}
		c.round++
		// ⚠ **ESC 一律消耗一輪,選沒選項目都一樣**,而總共只有三輪
		// (原版實跑:`Adjustment 1/2/3`,第三次 ESC 直接進 `Choose class`)。
		// 先前寫成「沒選項目才算做完」= 無限重擲,而重擲**嚴格有利**
		// (屬性是 2…13 的三角分佈,不滿意就再擲)—— 玩家可以把五項刷到 13。
		if c.round > town.CreateAdjustRounds {
			c.nextAfterAdjust()
		}

	case stepClass:
		// 原版是 `A) Warrior / B) Wizard`(實跑 `r3d4.png`)——
		// ⚠ 助憶鍵照原版,不照中文詞的第一個字母。
		switch k {
		case ebiten.KeyA:
			c.class = rules.ClassHero
			c.step = stepName
		case ebiten.KeyB:
			c.class = rules.ClassWizard
			c.step = stepName
		}

	case stepKeep:
		// ⚠ 原版把這一句問在**最後**(命名之後),印在整個畫面最下方:
		// 看完成品再決定留不留。`N` 是**放棄**,回名冊 —— 不是「重擲一組」。
		switch k {
		case ebiten.KeyY:
			g.finishCreate()
		case ebiten.KeyN:
			g.create = nil
			if g.roster != nil {
				g.roster.msg = "沒有建立角色。"
			}
		}

	case stepName:
		if k == ebiten.KeyBackspace && c.name != "" {
			r := []rune(c.name)
			c.name = string(r[:len(r)-1])
			return
		}
		if k == ebiten.KeyEnter || k == ebiten.KeyKPEnter {
			if strings.TrimSpace(c.name) == "" {
				c.msg = "還沒輸入名稱。"
				return
			}
			c.step = stepKeep
		}
	}
}

// createRunes 收玩家打的字。名稱上限是 `town.NameMaxRunes` = **9**
// (roster.go:欄位 10、輸入 9,兩個數字講的不是同一件事;出貨資料裡最長的
// 兩個名字正好 9 個字元,第 10 個 byte 一律空白)。
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
		line(fmt.Sprintf("第 %d／%d 輪調整:按 1–5 選要重擲的項目,ESC 執行。",
			c.round, town.CreateAdjustRounds))
		line("ESC 一律用掉一輪,三輪用完就進到下一步。")
	case stepClass:
		line("選職業:A) 戰士　B) 巫師") // CHARUTIL:`A) Warrior / B) Wizard`
	case stepKeep:
		line("要保留這位角色嗎?(Y/N)") // CHARUTIL:33
		line("(N 會放棄這位角色,回到名冊)")
	case stepName:
		line(fmt.Sprintf("名稱(最多 %d 字):%s_", town.NameMaxRunes, c.name))
		line("Enter 確定,ESC 取消")
	}
	y += lh * 0.5
	if c.msg != "" {
		line("⚠ " + c.msg)
	}
}

// nextAfterAdjust 是三輪調整用完之後往哪走:能選職業就選,不能就直接命名。
//
// ⚠ **先選種族再選職業** —— 五個種族只有人類能選(檔頭)。
func (c *createState) nextAfterAdjust() {
	if allowed := rules.Races[c.race].Classes; len(allowed) == 1 {
		c.class = allowed[0]
		c.step = stepName
		return
	}
	c.step = stepClass
}
