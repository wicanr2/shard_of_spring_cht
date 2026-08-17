package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/music"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/ui"
)

// 戰鬥場景。docs/spec/07-combat-scene.md。
//
// ⚠ M4 的範圍**不含**戰場移動與 15×15 捲動視野(§7)——
// 那要先解「攻擊的射程條件」,而 docs/spec/01 沒有那一條。
// 這裡是「按一次鍵推一回合」的最小可玩形式,規則本身是完整的。

// 開戰橫幅的兩句,字面照 WRLDMOVE.tsv 第 44/45 列(MAZEMOVE 有同樣兩句)。
// 原版用全形空白把字拉開,這裡照做 —— 那是它在畫面上的樣子。
const (
	CombatBanner = "　　戰　鬥　!"
	BattleBanner = "　　交　戰　!"
)

// startCombat 依遭遇建一場戰鬥。回 false 表示這次沒有觸發。
func (g *Game) startCombat() bool {
	if len(g.members) == 0 || len(g.monsters) == 0 {
		return false
	}
	// 遭遇的怪物由**區域編號**挑(docs/re/169):隨機挑一隻,
	// `|難度階級 − 區域| > 1` 就重挑。區域由「你人在哪裡」決定。
	zone := rules.WorldZone(g.party.X, g.party.Y)
	if g.level != nil {
		zone = rules.MazeZone(g.level.entry.MazeFile,
			g.mazeState.Major, g.mazeState.Minor)
	}
	pick, ok := g.pickMonster(zone)
	if !ok {
		// ⚠ 原版沒有次數上限,湊不到就當掉。引擎**明講**而不是
		// 悄悄退回「最接近的一隻」—— 那會讓原版不存在的怪物出現。
		g.warnings = append(g.warnings,
			fmt.Sprintf("區域 %d 找不到難度階級落在 ±%d 內的怪物(docs/re/169 §5)",
				zone, rules.ZoneTolerance))
		return false
	}
	g.field = combat.Build(g.members, []original.Monster{g.monsters[pick]},
		g.items, g.rand)
	g.field.PartySlots = g.group.MemberSlotNumbers() // 站位看槽號(docs/re/210)
	g.field.Place()
	g.field.ResetPoints(&g.points)
	g.settled = false // 新戰場 → 還沒結算過(settle 的冪等旗標)
	g.dispelled = nil // D)ispell 是「這一場一次」(docs/re/188 §2.2)
	g.actor = g.firstActor()
	// WRLDMOVE:44 / MAZEMOVE:90「    C O M B A T !」—— 原版遭遇時的橫幅。
	g.field.Log = append(g.field.Log, CombatBanner,
		fmt.Sprintf("遭遇 %s!", g.monsters[pick].Name))
	return true
}

// startScriptedCombat 依 docs/spec/17-scripted-fights.md 用腳本清單建一場戰鬥,
// **不經過** re/169 的區域隨機挑怪。target 是迷宮事件的目標編號
// (maze.TargetPriest / maze.TargetFinalBoss)。
//
// 回 false 表示這個目標沒有腳本清單(docs/re/180 §6:其餘 13 處寫入點還沒
// 盤到)—— 呼叫端(maze_scene.go 的 fireTrigger)要自己決定怎麼處理,
// 這裡不代勞、也不退回隨機遭遇(隨機遭遇的觸發點是「你在哪裡」,
// 不是「這個劇情事件查不到腳本清單」,兩者不能互相頂替)。
//
// ⚠ **每次都是全新查表 + 全新配置 *Field**,不維護任何跨戰鬥的清單狀態
// (docs/spec/17 §4 最後一列)。原版的 `ds:372C` 陣列沒有找到重置點,
// 理論上殘留舊值會讓下一場多出怪物,但那個 bug 在原版裡有沒有真的發生過
// 沒有被驗證過。這裡的做法是:rules.ScriptedFight 是唯讀表,每次呼叫都
// 重新查、重新用 combat.Build 配置一個全新的 *Field —— 上一場的怪物活在
// 上一個 *Field 裡,這一場用的是全新的一個,兩者不共用任何陣列,天生就不會
// 殘留。**這是刻意與原版不同的決定**,理由就是上面那句「沒有被驗證過」。
func (g *Game) startScriptedCombat(target int) bool {
	idxs, ok := rules.ScriptedFight[target]
	if !ok {
		return false
	}
	monsters := make([]original.Monster, 0, len(idxs))
	names := make([]string, 0, len(idxs))
	for _, i := range idxs {
		if i < 0 || i >= len(g.monsters) {
			// 表本身照 docs/re/180 §3 抄,理論上不會發生;
			// 真的發生時明講,不要悄悄跳過湊出一場怪物數不對的戰鬥。
			g.warnings = append(g.warnings, fmt.Sprintf(
				"腳本戰鬥 %d 的怪物索引 %d 超出 MONSTERS.DAT 範圍(docs/re/180 §3)", target, i))
			continue
		}
		monsters = append(monsters, g.monsters[i])
		names = append(names, g.monsters[i].Name)
	}
	if len(monsters) == 0 {
		return false
	}
	g.field = combat.Build(g.members, monsters, g.items, g.rand)
	g.field.PartySlots = g.group.MemberSlotNumbers() // 站位看槽號(docs/re/210)
	// ⚠ 怪物那一半仍是近似:原版擲座標找空格的兩個範圍未解(docs/re/164 §3)
	g.field.Place()
	g.field.ResetPoints(&g.points)
	g.settled = false // 新戰場 → 還沒結算過(settle 的冪等旗標)
	g.dispelled = nil // D)ispell 是「這一場一次」(docs/re/188 §2.2)
	g.actor = g.firstActor()
	if target == maze.TargetPriest {
		// 固定字串,兼做「這場是不是祭司事件」的識別 —— 見
		// rules.PriestEncounterMark 的說明(endTurn() 靠它決定要不要
		// 顯示 rules.PriestBlessing)。
		g.field.Log = append(g.field.Log, rules.PriestEncounterMark)
	} else {
		// 腳本戰鬥走另一句(WRLDMOVE:45 / MAZEMOVE:91「    B A T T L E !」)——
		// ⚠ 原版兩句的分工**沒有讀到**,這裡的分派是引擎自己定的。
		g.field.Log = append(g.field.Log, BattleBanner,
			fmt.Sprintf("遭遇 %s!", strings.Join(names, "、")))
	}
	if target == maze.TargetFinalBoss {
		// docs/spec/15 §7:外殼已經接好 g.bossFight → 結局畫面的轉場,
		// 這裡只設旗標,不碰 main.go。
		g.bossFight = true
	}
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
	g.settle() // 結束了才會做事(settle 自己判斷),與戰場操作走同一條路
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
	// ⚠ **最後一擊也要結算。** 勝負在怪物倒下的那一刻就定了,而結算原本
	// 只寫在 endTurn 裡 —— 隊員砍死最後一隻怪之後如果還有行動點數,
	// 或還有別人沒動,endTurn 就不會被呼叫,於是**打贏了卻沒拿到經驗與金幣**。
	// 沒有任何症狀:畫面照樣顯示「戰鬥結束」之外的一切,ESC 也照樣回世界地圖。
	if f.Outcome() != combat.Ongoing {
		g.settle()
		return true
	}
	if g.points[g.actor] <= 0 || !f.Units[g.actor].OnField() {
		g.nextActor()
	}
	return true
}

// settle 結算一場已經分出勝負的戰鬥:發經驗與金幣、寫結束訊息、收掉行動權。
//
// ⚠ **冪等**(靠 g.settled)—— 玩家的最後一擊、怪物回合結束、
// stepCombat 三條路都會走到這裡,不擋的話同一場會發兩次經驗。
// 旗標在 startCombat / startScriptedCombat 建新戰場時歸零。
func (g *Game) settle() {
	f := g.field
	if f == nil || g.settled {
		return
	}
	o := f.Outcome()
	if o == combat.Ongoing {
		return
	}
	g.settled = true
	f.Log = append(f.Log, "戰鬥結束："+o.String())
	if msg := g.awardSpoils(f.Units[:]); msg != "" {
		f.Log = append(f.Log, msg)
	}
	if o == combat.PartyDead {
		// ⚠ `USERLIB` 那五段的**用途是推測**(docs/re/148 §2):
		// 它慢一半、而 USERLIB 有死亡與結局兩個匯出槽。
		// 全滅時放它是本引擎的選擇,不是讀到呼叫端。
		g.play(music.FileDeath, music.Userlib)
	}
	// docs/spec/17 §3:打贏 204(祭司事件)之後顯示祭司的後續文字。
	// ⚠ 祝福的效果未解(docs/re/161 §4 只讀到這句字串)——
	// 只顯示文字,⛔ 不附加任何屬性加成。
	// 用 f.Log[0] 判斷「這場是不是祭司事件」—— 理由見
	// rules.PriestEncounterMark 的說明。
	if o == combat.MonstersDead && len(f.Log) > 0 && f.Log[0] == rules.PriestEncounterMark {
		f.Log = append(f.Log, rules.PriestBlessing)
	}
	g.actor = -1
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
	if f.Outcome() != combat.Ongoing {
		g.settle()
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

// drawBoard 畫戰場的視窗:**9×9 圖塊**(docs/re/175 實跑量到的),陣列本身 31 寬。
func (g *Game) drawBoard(dst *ebiten.Image, x0, y0 float64) float64 {
	f, p := g.field, g.panel
	lh := p.LineHeight()
	cell := lh * 0.9
	// 視窗跟著目前行動的單位;沒有人能動時跟著隊伍的基準格。
	cx, cy := combat.PartyBaseX, combat.PartyBaseY
	if g.actor >= 0 && g.actor < len(f.Units) {
		cx, cy = f.Units[g.actor].X, f.Units[g.actor].Y
	}
	ox, oy := combat.ViewOrigin(cx, cy)
	for vy := 0; vy < combat.ViewH; vy++ {
		for vx := 0; vx < combat.ViewW; vx++ {
			x, y := ox+vx, oy+vy
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
			p.Draw(dst, ch, x0+float64(vx)*cell, y0+float64(vy)*cell)
		}
	}
	return y0 + float64(combat.ViewH)*cell
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
		// CMBT:110/111「Use arrow keys to position cursor.」
		p.Draw(dst, fmt.Sprintf("選施法目標:%s　用方向鍵移動游標。(I/J/K/M 同效)空白鍵施放、ESC 取消",
			g.cursor.spell.Name), x, y)
		y += lh * 1.2
	} else if g.actor >= 0 {
		// ⚠ **按鍵表不寫在這裡** —— 提示列已經有一份(scene.go 的
		// combatPrompt),兩份會各自演化;而且這一行加上按鍵表之後
		// 長度超過主視野的框,字直接壓到右邊的隊伍面板上。
		// CMBT:41「FACING 」+ 42「MP=」—— 原版戰場的狀態列就這兩個標籤。
		p.Draw(dst, fmt.Sprintf("輪到 %s　朝向 %s　行動點=%d／%d",
			f.Units[g.actor].Name, facingName(f.Units[g.actor].Facing),
			g.points[g.actor], f.Units[g.actor].Speed), x, y)
		y += lh * 1.2
	}

	// ⚠ 畫到主視野框外的字**照樣印得出來**,只是壓在別的面板上
	// (docs/spec/04 §5 的同一條)。字級換大(倚天 24 px)之後單位表就會滿出來,
	// 所以這裡跟訊息面板一樣自己收邊。
	bottom := float64(layout.View.Bottom()) - ui.PanelPad
	dropped := 0
	line := func(s string) {
		// 留一行給「還有幾個沒列出來」——⛔ **不可以默默截斷**:
		// 少列一個單位在畫面上完全沒有症狀,玩家會以為場上就這些人。
		if y+lh*2 > bottom {
			dropped++
			return
		}
		p.Draw(dst, s, x, y)
		y += lh
	}
	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		if u := f.Units[i]; u.Name != "" {
			line(unitLine(u) + g.tacticsLine(u))
		}
	}
	y += lh * 0.5
	for i := combat.PartyBase; i < combat.PartyBase+combat.PartyMax; i++ {
		if u := f.Units[i]; u.Name != "" {
			line(unitLine(u))
		}
	}
	if dropped > 0 {
		p.Draw(dst, fmt.Sprintf("…還有 %d 個單位(視窗放不下)", dropped), x, y)
	}

	// 訊息列:只顯示最後幾條,新的在下面。折行與取尾要一起算 ——
	// 先折再取尾,取尾之後才知道放得下哪幾行(message.go 的 drawMessageLines
	// 只負責畫)。
	var lines []string
	for _, s := range f.Log {
		lines = append(lines, ui.Wrap(s, layout.MsgCols)...)
	}
	if max := g.messageLines(); len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	g.drawMessageLines(dst, lines)
}

// dispell 是戰場的 `D)ispell`。三道閘門照 docs/re/188 §2 的順序:
// 有沒有 Priesthood → 這場用過沒有 → 是不是巫師。
//
// ⚠ **「這場用過沒有」放在記憶體裡,不是角色記錄。** 原版存在 `CHARS.DAT`
// 位移 87,而**它在哪裡清回去沒有讀到**(docs/re/188 §5)——
// 照抄一個不知道怎麼清的持久旗標,會讓玩家永久失去這個指令。
func (g *Game) dispell() {
	f := g.field
	i := g.actor
	if f == nil || i < combat.PartyBase || i >= combat.PartyBase+combat.PartyMax {
		return
	}
	idx := i - combat.PartyBase
	if idx >= len(g.members) {
		return
	}
	c := g.members[idx]
	name := f.Units[i].Name
	switch {
	case !combat.HasPriesthood(c):
		f.Log = append(f.Log, name+combat.MsgNoPriesthood)
		return
	case g.dispelled[c.ID]:
		f.Log = append(f.Log, name+combat.MsgAlreadyDispell)
		return
	case c.Class != combat.WizardClassChar:
		f.Log = append(f.Log, name+combat.MsgDispellNotWiz)
		return
	case !f.HasUndead():
		f.Log = append(f.Log, combat.MsgNoUndead)
		return
	}
	// ⚠ 旗標在**擲骰之前**就設掉 —— 原版跑完整輪也是無條件設(re/188 §2.2),
	// 一隻都沒驅掉照樣算用過。
	if g.dispelled == nil {
		g.dispelled = map[int]bool{}
	}
	g.dispelled[c.ID] = true
	f.Log = append(f.Log, f.Dispell(c.Int)...)
	// 驅散花一次行動,而且結束這個人的回合 —— 與施法同一種成本
	// (docs/spec/12 §2)。⚠ 原版怎麼算**沒有讀到**,這是引擎的選擇。
	g.points[i] = 0
	g.nextActor()
	g.settle()
}

// facingName 是四個朝向的中文。手冊 p.33 的戰場是俯視圖,北在上。
func facingName(f combat.Facing) string {
	switch f {
	case combat.North:
		return "北"
	case combat.East:
		return "東"
	case combat.South:
		return "南"
	case combat.West:
		return "西"
	}
	return "—" // 朝向 0 = 已離場(docs/re/103)
}

func unitLine(u combat.Unit) string {
	if !u.Alive() {
		return ui.PadTo(u.Name, ui.CombatNameCols) + "  倒下"
	}
	return fmt.Sprintf("%s  HP %3d  速 %2d", ui.PadTo(u.Name, ui.CombatNameCols), u.HP, u.Speed)
}

// tacticsLine 在怪物那一列後面補上「→ 它在追誰」。
//
// 手冊 p.35 的 `TACTICS`(精訊譯「策略」)技能:隊上有人會,就看得出
// 每隻怪物正在追蹤哪一位同伴。鎖定對象是戰鬥屬性 15(docs/re/186 §2)。
// ⚠ 沒有那個技能就**什麼都不顯示** —— 不是顯示「未知」,那會洩漏「有目標」這件事。
func (g *Game) tacticsLine(u combat.Unit) string {
	f := g.field
	if f == nil || !f.Tactics || !u.IsMonster || !u.Alive() {
		return ""
	}
	t := u.Target
	if t < combat.PartyBase || t >= combat.PartyBase+combat.PartyMax {
		return ""
	}
	if f.Units[t].Name == "" {
		return ""
	}
	// F3:原版的字面是 `Seeks> `(CMBT:180)。
	return "　目標> " + f.Units[t].Name
}

// pickMonster 依區域挑一隻怪物,照原版的「不合就重擲」。
//
// ⚠ 原版的重擲**沒有上限**(無條件 `jg` 回頭),湊不到就當掉。
// 這裡加上限是**實作決定**;耗盡時回 false,由呼叫端明講,
// ⛔ 不用「取最接近的一隻」代替。
func (g *Game) pickMonster(zone int) (int, bool) {
	// 先看有沒有合格的 —— 沒有的話重擲多少次都沒用。
	any := false
	for _, m := range g.monsters {
		if rules.ZoneAccepts(zone, m.Tier) {
			any = true
			break
		}
	}
	if !any {
		return 0, false
	}
	for i := 0; i < monsterPickTries; i++ {
		p := g.rand.Roll(len(g.monsters)) - 1
		if rules.ZoneAccepts(zone, g.monsters[p].Tier) {
			return p, true
		}
	}
	return 0, false
}

// monsterPickTries 是重擲的上限。原版沒有這個數字(見 pickMonster)。
const monsterPickTries = 200
