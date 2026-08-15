package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/magic"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/ui"
)

// 施法介面。docs/spec/09-magic-items.md。
//
// 流程:戰鬥中按 C → 列出當前施法者可施的法術 → 按字母選 → 投入該法術的
// 一級單價 → 對敵方套用。
//
// ⚠ **投入點數固定投一級**是簡化。原版讓玩家自己輸入(`'Spell Pts ?'`,
// docs/spec/02 §1),而投入量會改變威力與狀態強度 —— 所以這裡不是「等價」,
// 是「M6 只做到一級」。提示列會標出來。

// castable 回傳這位角色目前施得出來的法術,以及各自要投入多少(一級)。
func (g *Game) castable(c original.Character) []original.Spell {
	var out []original.Spell
	for _, s := range g.spells {
		invest := s.UnitCost
		if invest < 1 {
			invest = 1
		}
		if magic.CanCast(c, s, invest) == magic.OK {
			out = append(out, s)
		}
	}
	return out
}

// openCast 打開施法選單。回 false 表示沒有人施得出來。
// openCast 開施法選單。**施法的是目前輪到的那個人**(docs/spec/12 §2)——
// 先前是「掃過全隊,找第一個會法術的」,那在戰場上是錯的:
// 施法要扣**那個人**的行動點數,而且做完結束**他的**回合。
func (g *Game) openCast() bool {
	if g.field == nil {
		return false
	}
	i := g.actor
	if i < combat.PartyBase || i >= combat.PartyBase+combat.PartyMax {
		g.field.Log = append(g.field.Log, "現在沒有人可以行動")
		return false
	}
	if g.points[i] < rules.ActCast.Cost() {
		g.field.Log = append(g.field.Log,
			g.field.Units[i].Name+"：行動點數不足,施不了法")
		return false
	}
	if i-combat.PartyBase >= len(g.members) {
		return false
	}
	if list := g.castable(g.members[i-combat.PartyBase]); len(list) > 0 {
		g.castUnit, g.castList = i, list
		return true
	}
	g.field.Log = append(g.field.Log, g.field.Units[i].Name+" 施不出法術")
	return false
}

// pickSpell 用字母選一個法術並施放。
func (g *Game) pickSpell(n int) {
	if n < 0 || n >= len(g.castList) {
		return
	}
	s := g.castList[n]
	invest := s.UnitCost
	if invest < 1 {
		invest = 1
	}
	caster := &g.field.Units[g.castUnit]

	// 目標:敵方所有還活著的單位。單體法術由 magic.Apply 自己取第一個。
	var targets []*combat.Unit
	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		if g.field.Units[i].Alive() {
			targets = append(targets, &g.field.Units[i])
		}
	}
	// 增益類對自己人。⚠ **原版怎麼選目標未解** —— 這裡是實作決定。
	//
	// ⚠ 類別 3–6 的**欄4 帶正負**(docs/formats/04):正 = 增益、負 = 傷害。
	// `COLUMN OF FIRE` 是類別 5 且威力 −3,那是**傷害法術**,不是治療 ——
	// 只看類別會把它送去打自己人,而畫面上只顯示「XXX:-3」,看不出打錯了。
	friendly := false
	switch s.Effect {
	case magic.EffRaise, magic.EffCure, magic.EffUnbind, magic.EffProtect:
		friendly = true
	case magic.EffAttr3, magic.EffStrength, magic.EffHitPoints, magic.EffSpeed:
		friendly = s.Power > 0
	}
	if friendly {
		targets = nil
		for i := combat.PartyBase; i < combat.PartyBase+combat.PartyMax; i++ {
			if g.field.Units[i].Name != "" {
				targets = append(targets, &g.field.Units[i])
			}
		}
	}

	caster.SP -= invest
	if caster.SP < 0 {
		caster.SP = 0
	}
	r := magic.Apply(s, invest, caster, targets)
	// 施法扣 3 點,而且**做完直接結束這個人的回合**(手冊 p.35,docs/spec/12 §2)。
	// ⚠ 攻擊的成本也是 3 但**不會**結束回合 —— 不能用成本判斷。
	if g.castUnit == g.actor {
		g.points[g.actor] = 0
		g.nextActor()
	}
	g.field.Log = append(g.field.Log,
		fmt.Sprintf("%s 施放 %s(投入 %d)", caster.Name, s.Name, invest))
	g.field.Log = append(g.field.Log, r.Message)
	g.castList = nil
}

// drawCastMenu 在訊息列上方畫可施法術的清單。
func (g *Game) drawCastMenu(dst *ebiten.Image) {
	if len(g.castList) == 0 || g.panel == nil {
		return
	}
	// 蓋掉底下的訊息 —— 兩層字疊在一起是「看得到但讀不出來」,
	// 而那比缺字更難察覺。
	rc := layout.Message
	vector.DrawFilledRect(dst, float32(rc.X+2), float32(rc.Y+2),
		float32(rc.W-4), float32(rc.H-4), cgaBlack, false)
	p := g.panel
	lh := p.LineHeight()
	x := float64(layout.Message.X + ui.PanelPad)
	y := float64(layout.Message.Y + ui.PanelPad)
	p.Draw(dst, g.field.Units[g.castUnit].Name+" 的法術：", x, y)
	y += lh
	for i, s := range g.castList {
		if y > float64(layout.Message.Y+layout.Message.H)-lh*2 {
			p.Draw(dst, fmt.Sprintf("…還有 %d 個(未分頁)", len(g.castList)-i), x, y)
			break
		}
		invest := s.UnitCost
		if invest < 1 {
			invest = 1
		}
		p.Draw(dst, fmt.Sprintf("%c) %s  %d 點", 'A'+i, s.Name, invest), x, y)
		y += lh
	}
}
