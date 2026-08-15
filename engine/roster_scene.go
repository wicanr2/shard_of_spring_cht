package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
	"shardofspring/internal/original"
	"shardofspring/internal/town"
	"shardofspring/internal/ui"
)

// 角色名冊畫面。docs/spec/11-town-camp-roster.md §5。
//
// 對照 `CHARUTIL.EXE` 的 `' #) Name       Party'` —— 原版是兩欄並排 25 個槽,
// 本專案放主視野(61 欄)一欄列完。
//
// 原版的指令(DOSBox 實跑,docs/re/143):
//
//	* Characters *  C)reate  R)emove  N)ew Name
//	* Parties *     D)isband  J)oin  I)nformation  E)xit
//
// ⚠ 助憶鍵沿用原版 —— 先前刪除用 `X`,那是自己造的。

type rosterState struct {
	open   bool
	cursor int
	msg    string
}

// openRoster 讀名冊。名冊是 CHARS.DAT 全部 25 槽,不只在隊的那五個。
func (g *Game) openRoster() {
	if g.roster == nil {
		g.roster = &rosterState{}
	}
	g.roster.open = true
	g.roster.msg = ""
}

// rosterKey 處理名冊畫面的按鍵。
func (g *Game) rosterKey(k ebiten.Key) {
	r := g.roster
	switch k {
	case ebiten.KeyEscape:
		r.open = false
		g.saveRoster()
	case ebiten.KeyUp:
		if r.cursor > 0 {
			r.cursor--
		}
	case ebiten.KeyDown:
		if r.cursor < original.CharSlots-1 {
			r.cursor++
		}
	case ebiten.KeyC: // C)reate
		g.openCreate()
	case ebiten.KeyJ: // J)OIN —— 沿用原版的助憶鍵
		g.rosterJoin()
	case ebiten.KeyD: // D)ISBAND / 移出隊伍
		c := &g.chars[r.cursor]
		town.LeaveParty(c, &g.group)
		r.msg = c.Name + " 已離隊"
	case ebiten.KeyX: // 刪除
		c := &g.chars[r.cursor]
		if p, in := c.InParty(); in {
			r.msg = fmt.Sprintf("%s 還在第 %d 隊,要先離隊", c.Name, p)
			return
		}
		name := c.Name
		town.Delete(c)
		r.msg = name + " 已刪除"
	}
}

func (g *Game) rosterJoin() {
	r := g.roster
	c := &g.chars[r.cursor]
	if !c.Occupied() {
		r.msg = "這個槽位沒有角色"
		return
	}
	if !town.JoinParty(c, &g.group, g.slot) {
		r.msg = fmt.Sprintf("第 %d 隊已滿(上限 %d 人)", g.slot, original.PartySlots)
		return
	}
	r.msg = fmt.Sprintf("%s 加入第 %d 隊", c.Name, g.slot)
	g.refreshMembers()
}

// refreshMembers 依 GROUPS.DAT 的成員槽重建隊伍(docs/spec/06 §1:以它為準)。
func (g *Game) refreshMembers() {
	g.members = nil
	for _, id := range g.group.MemberIDs() {
		if id >= 1 && id <= len(g.chars) && g.chars[id-1].Occupied() {
			g.members = append(g.members, g.chars[id-1])
		}
	}
}

// saveRoster 把名冊與隊伍寫回 <assets>/save/。
//
// ⚠ 寫的是複本,**不碰 game/sharspri/**(CLAUDE.md §8)。
func (g *Game) saveRoster() {
	if err := g.writeChars(); err != nil {
		g.warnings = append(g.warnings, "名冊存檔失敗:"+err.Error())
	}
	if err := g.save(); err != nil {
		g.warnings = append(g.warnings, "隊伍存檔失敗:"+err.Error())
	}
}

// drawRoster 畫名冊。
func (g *Game) drawRoster(dst *ebiten.Image) {
	r, p := g.roster, g.panel
	if r == nil || !r.open || p == nil {
		return
	}
	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)

	// ⚠ **用像素定位排欄,不用空白補位** —— 字型是比例字
	// (與 docs/spec/06 §5 的隊伍狀態欄同一條)。ui.PadTo 只用來算容量。
	col := func(c float64) float64 { return x + c*ui.ColUnit }
	const (
		cNum, cName, cRace, cClass, cLevel, cParty = 2.0, 6.0, 22.0, 30.0, 40.0, 46.0
	)
	p.Draw(dst, "#", col(cNum), y)
	p.Draw(dst, "名稱", col(cName), y)
	p.Draw(dst, "種族", col(cRace), y)
	p.Draw(dst, "職業", col(cClass), y)
	p.Draw(dst, "等級", col(cLevel), y)
	p.Draw(dst, "隊伍", col(cParty), y)
	y += lh * 1.3
	for i, c := range g.chars {
		if i == r.cursor {
			p.Draw(dst, "▶", x, y)
		}
		p.DrawRight(dst, fmt.Sprint(i+1), col(cNum)+ui.ColUnit*2, y)
		if !c.Occupied() {
			p.Draw(dst, "（空）", col(cName), y)
			y += lh
			continue
		}
		p.Draw(dst, c.Name, col(cName), y)
		p.Draw(dst, c.RaceName(), col(cRace), y)
		p.Draw(dst, c.ClassName(), col(cClass), y)
		p.DrawRight(dst, fmt.Sprint(c.Level), col(cLevel)+ui.ColUnit*2, y)
		party := "—"
		if n, in := c.InParty(); in {
			party = fmt.Sprint(n)
		}
		p.Draw(dst, party, col(cParty), y)
		y += lh
	}
	if r.msg != "" {
		my := float64(layout.Message.Y + ui.PanelPad)
		for _, ln := range ui.Wrap(r.msg, 30) {
			p.Draw(dst, ln, float64(layout.Message.X+ui.PanelPad), my)
			my += lh
		}
	}
}
