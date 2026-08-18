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
	// disbandPick:D)isband 正在問「解散第幾隊」(1–5,0 離開)。
	disbandPick bool
	// removePick:R)emove 正在問「真的要移除嗎」(0 取消)。
	// ⚠ 刪角色是不可逆的,原版在這裡有一句問話(CHARUTIL:52)。
	removePick bool
	// rename 非 nil = N)ew Name 的文字輸入開著。
	rename *renameState
	// info 非空 = I)nformation 的隊伍資訊頁開著,內容是要顯示的幾行。
	info []string
}

// renameState 是 N)ew Name 的輸入狀態(CHARUTIL:7/8)。
type renameState struct {
	slot int    // 名冊索引
	text string // 目前打到哪
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
//
// 助憶鍵照原版(DOSBox 實跑,docs/re/143):
//
//	* Characters *  C)reate  R)emove  N)ew Name
//	* Parties *     D)isband  J)oin  I)nformation  E)xit
func (g *Game) rosterKey(k ebiten.Key) {
	r := g.roster
	switch {
	case r.info != nil:
		// CHARUTIL:50「Any key to continue, (Q quits)...」—— 任意鍵關掉。
		r.info = nil
		return
	case r.rename != nil:
		g.rosterRenameKey(k)
		return
	case r.joinPick:
		g.rosterJoinPick(k)
		return
	case r.removePick:
		g.rosterRemovePick(k)
		return
	case r.disbandPick:
		g.rosterDisbandPick(k)
		return
	}
	switch k {
	case ebiten.KeyEscape, ebiten.KeyE: // E)xit
		r.open = false
		g.saveRoster()
	// ⚠ 游標**只停在有角色的槽** —— 原版的名冊只列有人的那幾列
	// (2026-08-18 實跑 `r3a1-charutil.png`),空槽根本不在畫面上,
	// 停在一個看不見的列上會讓按鍵看起來沒有反應。
	case ebiten.KeyUp:
		g.moveRosterCursor(-1)
	case ebiten.KeyDown:
		g.moveRosterCursor(1)
	case ebiten.KeyC: // C)reate
		g.openCreate()
	case ebiten.KeyJ: // J)oin
		g.rosterJoin()
	case ebiten.KeyN: // N)ew Name
		g.rosterRename()
	case ebiten.KeyI: // I)nformation
		g.rosterInfo()
	case ebiten.KeyD: // D)isband —— 解散**整支隊伍**,不是把一個人移出去
		r.disbandPick = true
		r.msg = "要解散第幾隊　" + rosterDisbandAsk // CHARUTIL:54
	case ebiten.KeyR: // R)emove
		c := &g.chars[r.cursor]
		if p, in := c.InParty(); in {
			r.msg = fmt.Sprintf("%s 還在第 %d 隊,要先解散那一隊", c.Name, p)
			return
		}
		if !c.Occupied() {
			r.msg = "那一槽是空的"
			return
		}
		// 原版在刪除之前問一句(CHARUTIL:52,docs/re/203 §2)。
		// ⚠ 先前引擎按一下 R 就直接刪掉,**沒有任何確認** ——
		// 那是不可逆的動作,而畫面上看不出剛才刪了誰。
		r.removePick = true
		r.msg = "要移除 " + c.Name + " 嗎?　" + rosterRemoveAsk
	}
}

// rosterRename 開始改名(CHARUTIL:7)。
func (g *Game) rosterRename() {
	r := g.roster
	c := g.chars[r.cursor]
	if !c.Occupied() {
		r.msg = "這個槽位沒有角色"
		return
	}
	r.rename = &renameState{slot: r.cursor}
	r.msg = "要改名哪位角色?　" + c.Name
}

// renamePrompt 是 CHARUTIL:8。**數字寫死**,因為它要與原版逐字相同
// (`tools/check_module_text.py` 比對的是字面);
// `TestRenamePromptMatchesLimit` 釘住它與 `town.NameMaxRunes` 不會走散。
const renamePrompt = "請輸入新名字(最多9個字元)："

// rosterRenameRunes 收字元。名字沿用建立角色那一套上限。
//
// 上限是 **9**(`town.NameMaxRunes`,docs/re/199):原版創造與改名兩處都把
// 輸入常式的上限設成 9,讀完才補到 10 個字元存進記錄。
// **欄位 10、輸入 9,兩個數字講的不是同一件事。**
func (g *Game) rosterRenameRunes(rs []rune) {
	st := g.roster.rename
	if st == nil {
		return
	}
	for _, ch := range rs {
		if ch < ' ' || ch == 127 {
			continue
		}
		if len([]rune(st.text)) >= town.NameMaxRunes {
			return
		}
		st.text += string(ch)
	}
}

// rosterRenameKey 處理改名的控制鍵。
func (g *Game) rosterRenameKey(k ebiten.Key) {
	r := g.roster
	st := r.rename
	switch k {
	case ebiten.KeyEscape:
		r.rename, r.msg = nil, ""
	case ebiten.KeyBackspace:
		if rs := []rune(st.text); len(rs) > 0 {
			st.text = string(rs[:len(rs)-1])
		}
	case ebiten.KeyEnter, ebiten.KeyKPEnter:
		if st.text == "" {
			return
		}
		c := &g.chars[st.slot]
		old := c.Name
		town.Rename(c, st.text)
		g.syncMember(*c)
		r.rename = nil
		r.msg = old + " → " + c.Name
	}
}

// rosterDisbandPick 收「解散第幾隊」的答案。
func (g *Game) rosterDisbandPick(k ebiten.Key) {
	r := g.roster
	if k == ebiten.KeyEscape {
		r.disbandPick, r.msg = false, ""
		return
	}
	n := countKey(k) // town_scene.go
	if n < 0 {
		return // 不是數字就繼續等
	}
	r.disbandPick = false
	if n == 0 {
		r.msg = ""
		return
	}
	g.disbandParty(n)
}

// disbandParty 解散第 n 隊:把成員全部移出。
//
// ⚠ **不解散正在玩的那一隊。** 那會讓遊戲當場沒有隊員可以行動,
// 而原版的 `CHARUTIL` 是**獨立的程式**,根本不會在遊戲進行中被叫起來 ——
// 引擎把名冊併進遊戲裡,這道保護是併進來之後才需要的。
func (g *Game) disbandParty(n int) {
	r := g.roster
	if n == g.slot {
		r.msg = fmt.Sprintf("第 %d 隊正在遊戲中,不能解散", n)
		return
	}
	var freed int
	for i := range g.chars {
		if p, in := g.chars[i].InParty(); in && p == n {
			g.chars[i].Party = original.NoParty
			freed++
		}
	}
	if freed == 0 {
		r.msg = fmt.Sprintf("沒有隊伍 #%d 可解散!", n) // CHARUTIL:58+59
		return
	}
	groups, err := g.readGroupSlots()
	if err != nil {
		r.msg = "讀取隊伍失敗:" + err.Error()
		return
	}
	if n >= 1 && n <= len(groups) {
		for i := range groups[n-1].Members {
			groups[n-1].Members[i] = 0
		}
	}
	if err := writeGroups(g.savePath, groups); err != nil {
		r.msg = "隊伍存檔失敗:" + err.Error()
		return
	}
	r.msg = fmt.Sprintf("第 %d 隊已解散(%d 人)", n, freed)
}

// rosterInfo 組出 I)nformation 的隊伍資訊頁(CHARUTIL:39/42/44/45/50)。
func (g *Game) rosterInfo() {
	r := g.roster
	groups, err := g.readGroupSlots()
	if err != nil {
		r.msg = "讀取隊伍失敗:" + err.Error()
		return
	}
	out := []string{}
	for i, grp := range groups {
		n := i + 1
		if grp.Blank() {
			out = append(out, fmt.Sprintf("隊伍 #%d　（空）", n))
			continue
		}
		// ⚠ 原版印的是**月份名**與**地點名**,不是編號與座標
		// (`Saved in the month of the Spirit` / `Currently in the Wilderness`)。
		// 座標是內部狀態,原版從不對玩家顯示;「日」原版也沒有這一欄。
		month := monthName(grp.Month)
		if month == "" {
			month = fmt.Sprintf("第 %d 月", grp.Month) // 13 月以上沒有名字
		}
		out = append(out, fmt.Sprintf("隊伍 #%d　金幣：%.0f　食糧：%d", n, grp.Gold, grp.Provisions))
		// CHARUTIL:42「Saved in the month of the 」+ 45「Currently in the 」。
		out = append(out, fmt.Sprintf("　　存檔於%s　目前位於%s", month, g.groupPlace(grp)))
	}
	// 角色表(CHARUTIL:46)——**原版把它放在這一頁**,不是主畫面。
	out = append(out, "", rosterHeader)
	for i, ch := range g.chars {
		if !ch.Occupied() {
			continue
		}
		out = append(out, fmt.Sprintf("%2d) %s　%s %s　%d 級　%s",
			i+1, ui.PadTo(ch.Name, 10), ch.RaceName(), ch.ClassName(),
			ch.Level, ch.StatusName()))
	}
	// 原版的資訊頁還有一段「誰在哪一隊」的兩欄表(CHARUTIL:69)。
	out = append(out, "", rosterPartyHeader)
	out = append(out, g.partyMembershipRows()...)
	r.info = append(out, "", "按任意鍵繼續,(Q離開)……") // CHARUTIL:50
}

// partyMembershipRows 把「誰在哪一隊」排成兩欄,欄位對齊 rosterPartyHeader。
//
// ⚠ 欄位起點**從表頭量出來**(rosterColumns),不是另外寫一組數字 ——
// 改表頭就一定連帶改資料列。
func (g *Game) partyMembershipRows() []string {
	c := rosterColumns(rosterPartyHeader)
	if len(c) < 6 {
		return nil
	}
	cell := func(i int, ch original.Character) string {
		party := "—"
		if n, in := ch.InParty(); in {
			party = fmt.Sprint(n)
		}
		s := ui.PadTo(fmt.Sprintf("%d)", i+1), int(c[1]-c[0]))
		s += ui.PadTo(ch.Name, int(c[2]-c[1]))
		return s + party
	}
	var rows []string
	var left string
	for i, ch := range g.chars {
		if !ch.Occupied() {
			continue
		}
		if left == "" {
			left = cell(i, ch)
			continue
		}
		rows = append(rows, ui.PadTo(left, int(c[3]-c[0]))+cell(i, ch))
		left = ""
	}
	if left != "" {
		rows = append(rows, left)
	}
	return rows
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
	r.msg = "編進第幾隊(1–5)(ESC離開)?" // CHARUTIL:23
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
	// 位移 83:新隊伍不在任何迷宮(docs/re/204 §1)。
	// ⚠ 哨兵值與「未裝備」相同都是 99,但那是巧合不是同一件事。
	grp.MazeNum = original.NotInMaze
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
	// 表頭照原版一整行畫(CHARUTIL:46),資料列的欄位起點**從同一行量出來** ——
	// 這樣改表頭就一定連帶改欄位,不會兩邊各自漂走。
	// ⚠ 原版最後一欄是「狀態」;引擎多一欄「隊伍」,因為它把原版的兩個畫面
	// (角色清單與隊伍歸屬)併成一張表(docs/re/203 §3)。
	// ⚠ 主畫面是**兩欄** `#) 名字   隊伍`,而且**只列有角色的槽**
	// (2026-08-18 實跑 `r3a1-charutil.png`:五列,不是 25 列)。
	// 原版那張 `種族/職業/等級/狀態` 的表是 `I)nformation` 頁在用的 ——
	// 兩張表的畫面歸屬先前剛好對調了。
	head := rosterMainHeader
	c := rosterColumns(head)
	cNum, cName, cParty := c[0], c[1], c[2]
	p.Draw(dst, head, x, y)
	y += lh * 1.3
	shown := 0
	for i, ch := range g.chars {
		if !ch.Occupied() {
			continue
		}
		shown++
		if i == r.cursor {
			p.Draw(dst, "▶", x, y)
		}
		p.DrawRight(dst, fmt.Sprint(i+1), col(cNum)+ui.ColUnit*2, y)
		p.Draw(dst, ch.Name, col(cName), y)
		party := "—"
		if n, in := ch.InParty(); in {
			party = fmt.Sprint(n)
		}
		p.Draw(dst, party, col(cParty), y)
		y += lh
	}
	if shown == 0 {
		p.Draw(dst, "（名冊是空的 —— 按 C 建立角色）", x, y)
		y += lh
	}
	// I)nformation 的資訊頁蓋在名單上 —— 兩層字疊在一起讀不出來。
	if r.info != nil {
		y = float64(layout.View.Y+ui.PanelPad) + lh*1.3
		for _, ln := range r.info {
			p.Draw(dst, ln, x, y)
			y += lh
		}
		return
	}
	msg := r.msg
	if r.rename != nil {
		// CHARUTIL:8「Please enter the new name (9 char max): 」——
		// 數字與原版一致(docs/re/199)。
		msg = renamePrompt + r.rename.text + "_"
	}
	g.drawMessage(dst, msg)
}

// 名冊的三句提問(CHARUTIL:46/52/54/69)。
//
// ⚠ 原版的 `(0 exits)` 有**兩種空格寫法**,分屬兩個不同的提問:
// 移除角色是 `(0 exits) ? `(52)、解散隊伍是 `(0 exits) ?`(54)。
// 兩句都照抄,不要統一成一句。
const (
	rosterRemoveAsk  = "(0離開) ?" // 52
	rosterDisbandAsk = "(0離開)?"  // 54
	// rosterHeader 是 **I)nformation** 那一頁的角色表表頭(CHARUTIL:46)。
	// ⚠ 欄位間距照原版的空白數,資料列用 ui.Cols 量同一組欄位起點,兩邊才對得齊。
	// ⚠ **它不是主畫面的表頭** —— 主畫面是兩欄的 rosterMainHeader
	// (2026-08-18 實跑:兩張表的畫面歸屬先前對調了)。
	rosterHeader = "#) 名字         種族   職業    等級    狀態"
	// rosterMainHeader 是名冊**主畫面**的表頭:兩欄,只列有角色的槽。
	rosterMainHeader = "#) 名字            隊伍"
	// rosterPartyHeader 是 I)nformation 裡「誰在哪一隊」那一段的表頭(69),
	// 原版是兩欄並排。
	rosterPartyHeader = " #) 名字        隊伍          #) 名字        隊伍"
)

// rosterRemovePick 處理「真的要移除嗎」的回答。0 取消,R 確認。
func (g *Game) rosterRemovePick(k ebiten.Key) {
	r := g.roster
	r.removePick = false
	if k != ebiten.KeyR {
		r.msg = "" // 0 或任何別的鍵 = 取消
		return
	}
	c := &g.chars[r.cursor]
	name := c.Name
	town.Delete(c)
	r.msg = name + " 已刪除!" // CHARUTIL:53
}

// rosterColumns 量出表頭每一欄的起始欄號(以空白分隔的每一段算一欄)。
//
// ⚠ 用 ui.Cols 算,全形字算兩欄。資料列照這組欄號排,
// **改表頭就一定連帶改欄位** —— 兩邊各寫一組數字才會漂走。
func rosterColumns(head string) []float64 {
	var out []float64
	rs := []rune(head)
	prevSpace := true
	for i, r := range rs {
		if r == ' ' {
			prevSpace = true
			continue
		}
		if prevSpace {
			out = append(out, float64(ui.Cols(string(rs[:i]))))
		}
		prevSpace = false
	}
	return out
}

// groupPlace 回傳一支隊伍存檔時人在哪裡,照原版的說法:
// 在地城裡印**地城名**,否則印 `Wilderness`(野外)。
//
// ⚠ 存檔只記迷宮**檔號**(GROUPS.DAT 位移 83),而入口 0/1/4/6 共用 DG1、
// 7–11 共用 DG6 —— 檔號還原不出入口編號,所以共用檔號的地城只能取第一個
// 相符的名字(同 maze_scene.go 的 resumeMaze,docs/re/222 §4)。
func (g *Game) groupPlace(grp original.Group) string {
	const notInMaze = 99 // GROUPS.DAT 位移 83:99 = 不在迷宮(docs/re/169)
	if grp.MazeNum != notInMaze && grp.MazeNum != 0 {
		for i, e := range g.mazeData {
			if e.MazeFile == grp.MazeNum && i < len(g.dungeonNames) {
				return "地城： " + g.dungeonNames[i] // CHARUTIL:44「maze: 」
			}
		}
		return fmt.Sprintf("地城： DG%d", grp.MazeNum)
	}
	return "野外" // CHARUTIL:`Wilderness`
}

// moveRosterCursor 把游標移到**下一個有角色的槽**。
//
// ⚠ 名冊只列有人的那幾列,游標停在看不見的空槽上會讓按鍵看起來沒反應。
// 沒有任何角色時游標留在原地(不會無限迴圈)。
func (g *Game) moveRosterCursor(d int) {
	r := g.roster
	for i := r.cursor + d; i >= 0 && i < len(g.chars); i += d {
		if g.chars[i].Occupied() {
			r.cursor = i
			return
		}
	}
}
