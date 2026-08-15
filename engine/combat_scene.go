package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/music"
	"shardofspring/internal/original"
	"shardofspring/internal/ui"
)

// 戰鬥場景。docs/spec/07-combat-scene.md。
//
// ⚠ M4 的範圍**不含**戰場移動與 15×15 捲動視野(§7)——
// 那要先解「攻擊的射程條件」,而 docs/spec/01 沒有那一條。
// 這裡是「按一次鍵推一回合」的最小可玩形式,規則本身是完整的。

// startCombat 依遭遇建一場戰鬥。回 false 表示這次沒有觸發。
func (g *Game) startCombat() bool {
	if len(g.members) == 0 || len(g.monsters) == 0 {
		return false
	}
	// 依隊伍所在格的地形挑一隻怪。⚠ **遭遇表未解**(哪種地形出哪些怪)——
	// 這裡用種子決定,是**佔位**,不是規則。
	pick := g.rand.Roll(len(g.monsters)) - 1
	g.field = combat.Build(g.members, []original.Monster{g.monsters[pick]},
		g.items, g.rand)
	g.field.Place() // docs/spec/12 §5:初始佈陣(原版陣型未解,這裡是佔位)
	g.field.ResetPoints(&g.points)
	g.actor = g.firstActor()
	g.field.Log = append(g.field.Log,
		fmt.Sprintf("遭遇 %s！", g.monsters[pick].Name))
	return true
}

// stepCombat 是 M4 的「一次推完一整回合」。**畫面上已經不用它**
// (M10 改成戰場操作,docs/spec/12)—— 留著是因為它是自動對打的最短路徑,
// 測試與除錯用得到。⚠ 它**不算行動點數**,所以不要拿它當規則參考。
func (g *Game) stepCombat() {
	f := g.field
	if f == nil || f.Outcome() != combat.Ongoing {
		return
	}
	if combat.ReorderEachRound {
		f.Sort()
	}
	f.Round++
	f.Log = append(f.Log, fmt.Sprintf("── 第 %d 回合 ──", f.Round))

	for _, i := range f.Order {
		if !f.Units[i].Alive() || !f.Units[i].OnField() {
			continue
		}
		target, ok := g.pickTarget(i)
		if !ok {
			continue
		}
		f.Attack(i, target)
		if f.Outcome() != combat.Ongoing {
			break
		}
	}
	if o := f.Outcome(); o != combat.Ongoing {
		f.Log = append(f.Log, "戰鬥結束："+o.String())
		if _, msg := g.awardExp(f.Units[:]); msg != "" {
			f.Log = append(f.Log, msg)
		}
	}
}

// pickTarget 挑一個還活著的敵方單位。
//
// ⚠ **這不是原版的目標選擇策略**(docs/spec/07 §7 明列為不做)。
// 取第一個活著的,是一個可預測的佔位 —— 可預測比「看起來聰明」重要,
// 因為 M4 的驗收是可重現性。
func (g *Game) pickTarget(attacker int) (int, bool) {
	lo, hi := combat.PartyBase, combat.PartyBase+combat.PartyMax
	if attacker >= combat.PartyBase {
		lo, hi = combat.MonsterBase, combat.MonsterBase+combat.MonsterMax
	}
	for i := lo; i < hi; i++ {
		if g.field.Units[i].Alive() && g.field.Units[i].OnField() {
			return i, true
		}
	}
	return 0, false
}

// --- 戰場操作。docs/spec/12-combat-board.md -------------------------------

// firstActor 回傳先攻表裡第一個還能動的**隊員**。
//
// ⚠ 只有隊員由玩家操作;怪物在 endTurn 時自動走完(docs/spec/12 §5)。
func (g *Game) firstActor() int {
	for i := combat.PartyBase; i < combat.PartyBase+combat.PartyMax; i++ {
		u := g.field.Units[i]
		if u.Alive() && u.OnField() && g.points[i] > 0 {
			return i
		}
	}
	return -1
}

// boardKey 處理戰場上的一次按鍵。回 true 表示這個鍵被吃掉了。
func (g *Game) boardKey(k ebiten.Key) bool {
	f := g.field
	if f.Outcome() != combat.Ongoing || g.actor < 0 {
		return false
	}
	var r combat.ActResult
	switch k {
	case ebiten.KeyUp:
		r = g.moveOrTurn(combat.North)
	case ebiten.KeyRight:
		r = g.moveOrTurn(combat.East)
	case ebiten.KeyDown:
		r = g.moveOrTurn(combat.South)
	case ebiten.KeyLeft:
		r = g.moveOrTurn(combat.West)
	case ebiten.KeyA:
		r = f.StrikeFront(&g.points, g.actor)
	case ebiten.KeyEnter, ebiten.KeyKPEnter:
		g.endTurn()
		return true
	default:
		return false
	}
	if r != combat.ActOK {
		f.Log = append(f.Log, f.Units[g.actor].Name+"："+r.String())
	}
	if g.points[g.actor] <= 0 || !f.Units[g.actor].OnField() {
		g.nextActor()
	}
	return true
}

// moveOrTurn 沿用世界地圖的「先轉再走」手感:朝向不同時只轉身。
//
// ⚠ 這是**本引擎的操作選擇**,不是原版規則 —— 原版的戰場是
// `←`/`→` 轉向、`<RETURN>` 前進(手冊 p.34)。兩者花的點數一樣,
// 差別只在按鍵數。畫面上的指令提示照本引擎的寫。
func (g *Game) moveOrTurn(dir combat.Facing) combat.ActResult {
	if g.field.Units[g.actor].Facing != dir {
		return g.field.Turn(&g.points, g.actor, dir)
	}
	return g.field.Step(&g.points, g.actor)
}

// nextActor 換下一個隊員;都走完了就讓怪物動,然後開新回合。
func (g *Game) nextActor() {
	for i := g.actor + 1; i < combat.PartyBase+combat.PartyMax; i++ {
		u := g.field.Units[i]
		if u.Alive() && u.OnField() && g.points[i] > 0 {
			g.actor = i
			return
		}
	}
	g.endTurn()
}

// endTurn 結束玩家這一輪:怪物依序行動,然後重發點數。
func (g *Game) endTurn() {
	f := g.field
	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		if f.Outcome() != combat.Ongoing {
			break
		}
		u := f.Units[i]
		if u.Alive() && u.OnField() {
			f.MonsterTurn(&g.points, i)
		}
	}
	if o := f.Outcome(); o != combat.Ongoing {
		f.Log = append(f.Log, "戰鬥結束："+o.String())
		if _, msg := g.awardExp(f.Units[:]); msg != "" {
			f.Log = append(f.Log, msg)
		}
		if o == combat.PartyDead {
			// ⚠ `USERLIB` 那五段的**用途是推測**(docs/re/148 §2):
			// 它慢一半、而 USERLIB 有死亡與結局兩個匯出槽。
			// 全滅時放它是本引擎的選擇,不是讀到呼叫端。
			g.play(music.Userlib)
		}
		g.actor = -1
		return
	}
	if combat.ReorderEachRound {
		f.Sort()
	}
	f.Round++
	f.ResetPoints(&g.points)
	g.actor = g.firstActor()
	if g.actor < 0 {
		// 全隊都動不了(離場或倒下)—— 直接讓下一輪跑,免得卡住
		g.actor = -1
	}
}

// drawBoard 畫 15×15 的戰場。
func (g *Game) drawBoard(dst *ebiten.Image, x0, y0 float64) float64 {
	f, p := g.field, g.panel
	lh := p.LineHeight()
	cell := lh * 0.9
	for y := 0; y < combat.BoardSize; y++ {
		for x := 0; x < combat.BoardSize; x++ {
			ch := "·" // 空格
			if combat.OnEdge(x, y) {
				ch = "○" // 圓點 = 出口(手冊 p.33)
			}
			if g.cursor != nil && g.cursor.x == x && g.cursor.y == y {
				ch = "✚" // 施法游標
			} else if i := f.Occupant(x, y); i >= 0 {
				if f.Units[i].IsMonster {
					ch = "怪"
				} else if i == g.actor {
					ch = "◆" // 輪到誰
				} else {
					ch = "人"
				}
			}
			p.Draw(dst, ch, x0+float64(x)*cell, y0+float64(y)*cell)
		}
	}
	return y0 + float64(combat.BoardSize)*cell
}

// drawCombat 在主視野畫戰鬥的狀態與訊息。
func (g *Game) drawCombat(dst *ebiten.Image) {
	f, p := g.field, g.panel
	if f == nil || p == nil {
		return
	}
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)
	lh := p.LineHeight()

	p.Draw(dst, fmt.Sprintf("戰鬥 — 第 %d 回合", f.Round), x, y)
	y += lh * 1.3
	y = g.drawBoard(dst, x, y) + lh*0.5
	if g.cursor != nil {
		p.Draw(dst, fmt.Sprintf("選施法目標:%s　I/J/K/M 移動游標、空白鍵施放、ESC 取消",
			g.cursor.spell.Name), x, y)
		y += lh * 1.2
	} else if g.actor >= 0 {
		p.Draw(dst, fmt.Sprintf("輪到 %s　行動點數 %d／%d　方向鍵移動(先轉再走)　A 攻擊　C 施法　Enter 結束",
			f.Units[g.actor].Name, g.points[g.actor], f.Units[g.actor].Speed), x, y)
		y += lh * 1.2
	}

	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		u := f.Units[i]
		if u.Name == "" {
			continue
		}
		p.Draw(dst, unitLine(u), x, y)
		y += lh
	}
	y += lh * 0.5
	for i := combat.PartyBase; i < combat.PartyBase+combat.PartyMax; i++ {
		u := f.Units[i]
		if u.Name == "" {
			continue
		}
		p.Draw(dst, unitLine(u), x, y)
		y += lh
	}

	// 訊息列:只顯示最後幾條,新的在下面。
	my := float64(layout.Message.Y + ui.PanelPad)
	max := int((float64(layout.Message.H) - 2*ui.PanelPad) / lh)
	// 折行:訊息面板 340 px 內距 16 → 內寬 308 px = 30 欄
	// (docs/spec/04 §5)。**不截斷** —— 截掉的字看不出來,折行看得出來。
	const msgCols = 30
	var lines []string
	for _, s := range f.Log {
		lines = append(lines, ui.Wrap(s, msgCols)...)
	}
	if len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	for _, s := range lines {
		p.Draw(dst, s, float64(layout.Message.X+ui.PanelPad), my)
		my += lh
	}
}

func unitLine(u combat.Unit) string {
	if !u.Alive() {
		return ui.PadTo(u.Name, ui.CombatNameCols) + "  倒下"
	}
	return fmt.Sprintf("%s  HP %3d  速 %2d", ui.PadTo(u.Name, ui.CombatNameCols), u.HP, u.Speed)
}
