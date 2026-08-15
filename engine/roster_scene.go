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
	// joinPick 非 nil 時,下一個按鍵是「編進第幾隊」的答案(1–5)。
	//
	// 從主選單的 C)har Utilities 進名冊時沒有「目前隊伍」(g.slot == 0,
	// docs/spec/15 §4),J)oin 沒辦法像遊戲中那樣直接假設「就是現在這隊」——
	// 要現場問玩家選第幾隊(docs/spec/15 §5.1 的延伸)。
	joinPick bool
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
	if r.joinPick {
		g.rosterJoinPick(k)
		return
	}
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
		r.msg = name + " 已刪除!" // CHARUTIL:53
	}
}

// rosterJoin 開始「加入隊伍」。
//
// 有目前隊伍時(g.slot >= 1,遊戲中按 N 進來的名冊)直接編進那一隊 ——
// 這是原本就有的行為,沒有目前隊伍時它不成立。
// 沒有目前隊伍時(g.slot == 0,從主選單的 C)har Utilities 進來)沒有
// 「現在這隊」可以假設 —— 改成問玩家要編進第幾隊(docs/spec/15 §5.1)。
func (g *Game) rosterJoin() {
	r := g.roster
	c := &g.chars[r.cursor]
	if !c.Occupied() {
		r.msg = "這個槽位沒有角色"
		return
	}
	if g.slot >= 1 {
		g.joinCharacterToSlot(c, g.slot)
		return
	}
	r.joinPick = true
	r.msg = "編進第幾隊?按 1–5(ESC 取消)"
}

// rosterJoinPick 處理「編進第幾隊」的回答。
func (g *Game) rosterJoinPick(k ebiten.Key) {
	r := g.roster
	if k == ebiten.KeyEscape {
		r.joinPick = false
		r.msg = ""
		return
	}
	n, ok := digitKey(k)
	if !ok {
		return // 不是 1–5,不是 ESC —— 什麼都不做,繼續等
	}
	r.joinPick = false
	c := &g.chars[r.cursor]
	g.joinCharacterToSlot(c, n)
}

// joinCharacterToSlot 把角色 c 編進第 slot 隊(1–5)。
//
// 這支函式是本輪修的核心:分兩條路徑,依 slot 是不是「目前正在玩的那隊」
// 決定要動記憶體還是動整份 GROUPS.DAT。**兩條路徑都要處理「這隊原本是
// 空白」的情況**(docs/spec/15 §5.1)—— 判準是 wasBlank 在呼叫
// town.JoinParty **之前**存好,JoinParty 一寫入成員,Blank() 就再也
// 分辨不出「原本空白」還是「已經在玩」了。
func (g *Game) joinCharacterToSlot(c *original.Character, slot int) {
	r := g.roster
	if slot < 1 || slot > original.GroupSlots {
		r.msg = fmt.Sprintf("隊伍編號 %d 超出 1–%d", slot, original.GroupSlots)
		return
	}

	if slot == g.slot {
		// 正在玩的這一隊。docs/spec/15 §5:隊伍選擇畫面擋掉了空白的槽,
		// 選得進來就已經不是空白 —— wasBlank 恆為 false,這裡不必套用
		// 開局初值,只是為了與下面那條路徑對稱才留著判斷。
		wasBlank := g.group.Blank()
		if !town.JoinParty(c, &g.group, slot) {
			r.msg = fmt.Sprintf("第 %d 隊已滿(上限 %d 人)", slot, original.PartySlots)
			return
		}
		if wasBlank {
			applyNewPartyDefaults(&g.group)
		}
		r.msg = fmt.Sprintf("%s 加入第 %d 隊", c.Name, slot)
		g.refreshMembers()
		return
	}

	// 不是目前這隊(含 g.slot == 0,從主選單編隊):GROUPS.DAT 裡還有其他
	// 四槽的資料,要整份讀出來、改這一槽、整份寫回,不能只寫一筆
	// (docs/spec/15 §5.1 —— 這裡額外多做的是「整份讀寫」,原理不變)。
	groups, err := g.readGroupSlots()
	if err != nil {
		r.msg = "讀取隊伍失敗:" + err.Error()
		return
	}
	wasBlank := groups[slot-1].Blank()
	if !town.JoinParty(c, &groups[slot-1], slot) {
		r.msg = fmt.Sprintf("第 %d 隊已滿(上限 %d 人)", slot, original.PartySlots)
		return
	}
	if wasBlank {
		// ⚠ **判準是「這筆記錄原本是不是空白」**——已經在玩的隊伍加人
		// 不能被這一步重設,那會把玩家的座標、金幣、走過的時鐘全部洗掉
		// (docs/spec/06「未初始化的判定」)。這裡能這樣判,是因為
		// wasBlank 在 JoinParty 寫入成員**之前**就存好了。
		applyNewPartyDefaults(&groups[slot-1])
	}
	if err := writeGroups(g.savePath, groups); err != nil {
		r.msg = "隊伍存檔失敗:" + err.Error()
		return
	}
	r.msg = fmt.Sprintf("%s 加入第 %d 隊", c.Name, slot)
}

// NewParty* 是「這支隊伍第一次被組成」時要填的開局初值。
//
// ⚠ **具名假設,不是 RE 結論**(docs/spec/15 §5.1):原版一定有一段程式
// 在建隊時寫入這些值,但沒有讀到那段程式。這裡照抄出貨磁片 PARTY #5 的值——
// 那支隊伍是原廠組好的,它的初值**很可能**就是新隊伍的初值,而且它就在
// 資料裡,不必推。時鐘/遭遇倒數/能見度三項與 docs/spec/06「新遊戲的初始值」
// 是同一份範本,不是這裡另外編的。
const (
	NewPartyX, NewPartyY            = 8, 8 // 出貨 PARTY #5 的座標
	NewPartyFacing                  = 3    // 南
	NewPartyGold                    = 75
	NewPartyProvisions              = 20
	NewPartyMonth, NewPartyDay      = 1, 1
	NewPartyHour, NewPartySub       = 4, 2
	NewPartyEncounter               = 54   // 遭遇倒數,docs/spec/06 同一份範本
	NewPartyVisLit, NewPartyVisDark = 3, 2 // 能見度,同上
)

// applyNewPartyDefaults 把上面那組初值套進一支**原本是空白**的隊伍記錄。
//
// ⚠ 呼叫端要在修改前就把「原本是不是空白」存好再呼叫這支函式 ——
// 這支函式本身不判斷,也不應該判斷:它收到的 grp 這時通常已經被
// town.JoinParty 寫過 Members,Blank() 在這裡已經恆為 false,
// 用它來判斷會永遠得到「不是空白」這個錯誤答案。
func applyNewPartyDefaults(grp *original.Group) {
	grp.WorldX, grp.WorldY = NewPartyX, NewPartyY
	grp.Facing = NewPartyFacing
	grp.Gold = NewPartyGold
	grp.Provisions = NewPartyProvisions
	grp.Month, grp.Day = NewPartyMonth, NewPartyDay
	grp.Hour, grp.Sub = NewPartyHour, NewPartySub
	grp.Encounter = NewPartyEncounter
	grp.VisLit, grp.VisDark = NewPartyVisLit, NewPartyVisDark

	// ⚠ **剩下的欄位一定要明寫成 0**,不能靠「沒填就是零值」——
	// 這筆記錄是從一份**整份 0x20 的空白記錄**解出來的,每個沒被覆寫的
	// word 都是 `CVI(" ") = 8224`,不是 0(docs/spec/06「未初始化的判定」)。
	//
	// 漏掉的症狀很晚才出現:新隊伍在世界地圖上完全正常,
	// 直到第一次查光源或進迷宮才會看到 8224。
	//
	// ⚠ 這七項**不在** docs/spec/15 §5.1 的範本裡(那份只列了上面九項)——
	// 取 0 是「還沒發生過任何事」的自然值,不是讀出來的。
	grp.LightTurns = 0 // 還沒點過光源
	// ⚠ **光源的「沒有」是 99 不是 0**(docs/formats/02 位移 83 的哨兵,
	// docs/re/134 §2)。填 0 會變成「背包第 0 格那件道具」——
	// 而那一格通常真的有東西,所以症狀是「新隊伍莫名其妙帶著一盞燈」。
	grp.LightPick = original.NotEquipped
	grp.PoolUses = 0 // 治療池一次都還沒用(docs/re/155 §2.1)
	grp.MazeX, grp.MazeY = 0, 0
	grp.Fled = 0 // 上一場沒有逃跑
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
//
// ⚠ g.slot == 0 表示還沒有載入任何隊伍 —— 從主選單的 C)har Utilities
// 進名冊時就是這樣(docs/spec/15 §4:C 從主選單就能進去,不必先 L)oad)。
// 這時沒有「目前隊伍」可以寫回 GROUPS.DAT:g.save() 會拿 g.slot-1 = -1
// 去索引陣列,直接 panic。名冊本身(CHARS.DAT)與隊伍槽無關,一定要存;
// 隊伍(GROUPS.DAT)只有在真的有一支隊伍在玩的時候才存。
func (g *Game) saveRoster() {
	if err := g.writeChars(); err != nil {
		g.warnings = append(g.warnings, "名冊存檔失敗:"+err.Error())
	}
	if g.slot >= 1 {
		if err := g.save(); err != nil {
			g.warnings = append(g.warnings, "隊伍存檔失敗:"+err.Error())
		}
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
