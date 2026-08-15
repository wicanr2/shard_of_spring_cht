package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/ui"
	"shardofspring/internal/world"
)

// 外殼狀態機的測試。docs/spec/15-game-shell.md §9 的八條驗收,
// 每一條都直接呼叫 g.Update()(不跑 ebiten.RunGame,不開視窗)。
//
// ⚠ 能做到「不開視窗、直接呼叫 Update()」全靠 main.go 的 g.testKeys ——
// ebitengine 在沒有 X11 display 的環境連套件初始化都會 panic
// (glfw.Init 需要 display),而 inpututil 的「剛按下」狀態本來就是
// 靠 RunGame 的事件迴圈更新,不跑 RunGame 永遠讀不到任何鍵。
// g.testKeys 非 nil 時,Update() 讀它當作「這一格剛按下的鍵」,
// 繞過 inpututil,兩條路徑在正式執行(RunGame,g.testKeys 恆為 nil)裡
// 完全等價。

// newShellTestGame 建一個最小可跑的 Game,只給外殼狀態機測試用。
//
// GROUPS.DAT fixture:第 1 槽有資料(隊員 1、2,座標 8,8,金幣 75),
// 第 2–5 槽整份空白(0x20)——對照 docs/spec/06「未初始化的判定」。
// CHARS.DAT fixture:槽 1、2 是這兩個隊員,其餘 23 槽空。
func newShellTestGame(t *testing.T) *Game {
	t.Helper()
	dir := t.TempDir()
	saveDir := filepath.Join(dir, "save")
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	chars := make([]original.Character, original.CharSlots)
	chars[0] = original.Character{Party: '1', Name: "灰燼", ID: 1, Level: 3}
	chars[1] = original.Character{Party: '1', Name: "風語", ID: 2, Level: 2}
	// 索引 2:還沒編進任何隊伍 —— 給「主選單造角色 → 編隊」那組測試用
	// (Party 留零值,original.Character.InParty() 判斷不到 '1'–'5' 的範圍,
	// 與 NoParty 同一種讀法)。
	chars[2] = original.Character{Name: "新血", ID: 3, Level: 1}
	var charBytes []byte
	for _, c := range chars {
		charBytes = append(charBytes, c.Bytes()...)
	}
	if err := os.WriteFile(filepath.Join(saveDir, "CHARS.DAT"), charBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	parsedChars, err := original.ParseChars(charBytes)
	if err != nil {
		t.Fatal(err)
	}

	occupied := original.Group{
		WorldX: 8, WorldY: 8, Facing: 3, // 南(docs/spec/15 §5.1 的具名假設同一套朝向編碼)
		Gold: 75, Provisions: 20,
	}
	occupied.Members[0], occupied.Members[1] = 1, 2
	for i := 2; i < len(occupied.Members); i++ {
		occupied.Members[i] = 99 // 空槽哨兵(docs/re/135 §2)
	}
	blank := make([]byte, original.GroupRecLen)
	for i := range blank {
		blank[i] = ' '
	}
	var groupBytes []byte
	groupBytes = append(groupBytes, occupied.Bytes()...)
	for i := 0; i < original.GroupSlots-1; i++ {
		groupBytes = append(groupBytes, blank...)
	}
	savePath := filepath.Join(saveDir, "GROUPS.DAT")
	if err := os.WriteFile(savePath, groupBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	g := &Game{
		assets:   dir,
		savePath: savePath,
		chars:    parsedChars,
		noSrc:    map[int]bool{},
	}
	g.openTitle()
	return g
}

func mustUpdate(t *testing.T, g *Game) {
	t.Helper()
	if err := g.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

// ── 驗收 1:啟動後看到標題,任意鍵進主選單 ──────────────────────────

func TestTitleAnyKeyEntersMainMenu(t *testing.T) {
	g := newShellTestGame(t)
	if g.shell.mode != shellTitle {
		t.Fatalf("初始狀態應為標題,得到 %v", g.shell.mode)
	}
	g.testKeys = []ebiten.Key{ebiten.KeyEnter}
	mustUpdate(t, g)
	if g.shell.mode != shellMainMenu {
		t.Errorf("按任意鍵後應進主選單,得到 %v", g.shell.mode)
	}
}

// ── 驗收 2:主選單四個鍵各有反應;Q 真的關掉程式 ─────────────────────

func TestMainMenuLOpensPartySelect(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.testKeys = []ebiten.Key{ebiten.KeyL}
	mustUpdate(t, g)
	if g.shell.mode != shellPartySelect {
		t.Errorf("L 之後應進隊伍選擇,得到 %v", g.shell.mode)
	}
	if len(g.shell.slots) != original.GroupSlots {
		t.Errorf("應該讀到 %d 槽,得到 %d", original.GroupSlots, len(g.shell.slots))
	}
}

func TestMainMenuCOpensRoster(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.testKeys = []ebiten.Key{ebiten.KeyC}
	mustUpdate(t, g)
	if g.roster == nil || !g.roster.open {
		t.Errorf("C 之後名冊應該開著")
	}
	if g.shell.mode != shellMainMenu {
		t.Errorf("shell.mode 不應該被 C 動到,得到 %v", g.shell.mode)
	}
}

func TestMainMenuPShowsKeyHelp(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.testKeys = []ebiten.Key{ebiten.KeyP}
	mustUpdate(t, g)
	if g.overlay == "" {
		t.Errorf("P 之後應該顯示按鍵表覆蓋層")
	}
	// 任意鍵關閉,回到主選單(docs/spec/04 §3 的覆蓋層機制)。
	g.testKeys = []ebiten.Key{ebiten.KeyEnter}
	mustUpdate(t, g)
	if g.overlay != "" {
		t.Errorf("按鍵後覆蓋層應該關閉")
	}
	if g.shell.mode != shellMainMenu {
		t.Errorf("關閉按鍵表後應該還在主選單,得到 %v", g.shell.mode)
	}
}

func TestMainMenuQQuits(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.testKeys = []ebiten.Key{ebiten.KeyQ}
	if err := g.Update(); err != ebiten.Termination {
		t.Errorf("Q 應該讓 Update() 回傳 ebiten.Termination,得到 %v", err)
	}
}

// shellKeyHelp 折起來要放得進敘述覆蓋層(60 欄 × 5 行,docs/spec/04 §3)。
func TestShellKeyHelpFitsOverlay(t *testing.T) {
	lines := ui.Wrap(shellKeyHelp, 60)
	if len(lines) > 5 {
		t.Errorf("按鍵表折成 %d 行,覆蓋層只有 5 行容量:\n%s",
			len(lines), strings.Join(lines, "\n"))
	}
}

// ── 驗收 3:C 進角色管理,ESC 回主選單;遊戲中 N 進去,ESC 回遊戲 ────────

func TestRosterEscFromMainMenuReturnsToMainMenu(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.testKeys = []ebiten.Key{ebiten.KeyC}
	mustUpdate(t, g)
	if g.roster == nil || !g.roster.open {
		t.Fatalf("前提不成立:C 之後名冊應該開著")
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.roster.open {
		t.Errorf("ESC 之後名冊應該關閉")
	}
	if g.shell.mode != shellMainMenu {
		t.Errorf("從主選單進去的名冊,ESC 後應該回主選單,得到 %v", g.shell.mode)
	}
}

func TestRosterEscFromGameplayReturnsToGameplay(t *testing.T) {
	g := newShellTestGame(t)
	if err := g.loadParty(1); err != nil {
		t.Fatal(err)
	}
	g.shell.mode = shellPlaying

	g.testKeys = []ebiten.Key{ebiten.KeyN}
	mustUpdate(t, g)
	if g.roster == nil || !g.roster.open {
		t.Fatalf("前提不成立:N 之後名冊應該開著")
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.roster.open {
		t.Errorf("ESC 之後名冊應該關閉")
	}
	if g.shell.mode != shellPlaying {
		t.Errorf("遊戲中進去的名冊,ESC 後應該回遊戲(shellPlaying),得到 %v", g.shell.mode)
	}
}

// 回歸測試:saveRoster() 曾經在「還沒選隊伍就從主選單開名冊」時
// 用 g.slot-1 = -1 去索引 GROUPS.DAT 的槽陣列,直接 panic
// (roster_scene.go saveRoster 的說明)。
func TestSaveRosterFromMainMenuDoesNotPanicWithoutParty(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.openRoster()
	if g.slot != 0 {
		t.Fatalf("前提不成立:還沒選隊伍,g.slot 應該是 0,得到 %d", g.slot)
	}
	g.saveRoster() // 不應該 panic
}

// ── 驗收 4:隊伍選擇列出五槽,未初始化顯示「（空）」且選不下去 ─────────

func TestPartySelectListsFiveSlotsBlankUnselectable(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.testKeys = []ebiten.Key{ebiten.KeyL}
	mustUpdate(t, g)
	if g.shell.mode != shellPartySelect {
		t.Fatalf("前提不成立:L 之後應進隊伍選擇,得到 %v", g.shell.mode)
	}
	if len(g.shell.slots) != original.GroupSlots {
		t.Fatalf("前提不成立:應該讀到 %d 槽,得到 %d", original.GroupSlots, len(g.shell.slots))
	}
	if g.shell.slots[0].Blank() {
		t.Fatalf("前提不成立:第 1 槽 fixture 應該有資料")
	}
	for i := 1; i < original.GroupSlots; i++ {
		if !g.shell.slots[i].Blank() {
			t.Errorf("第 %d 槽 fixture 應該是空白(未初始化的判定 = 整筆全為 0x20)", i+1)
		}
	}

	// 選一個空槽:選不下去,留在隊伍選擇畫面並給提示。
	g.testKeys = []ebiten.Key{ebiten.KeyDigit2}
	mustUpdate(t, g)
	if g.shell.mode != shellPartySelect {
		t.Errorf("空槽應該選不下去,shell.mode 卻變成 %v", g.shell.mode)
	}
	if g.shell.msg == "" {
		t.Errorf("選空槽應該顯示提示訊息")
	}
}

// ── 驗收 5:選一支有資料的隊伍 → 進世界地圖,座標/時鐘/金幣來自該槽 ─────

func TestSelectOccupiedPartyEntersGameWithSavedState(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()
	g.testKeys = []ebiten.Key{ebiten.KeyL}
	mustUpdate(t, g)

	g.testKeys = []ebiten.Key{ebiten.KeyDigit1}
	mustUpdate(t, g)

	if g.shell.mode != shellPlaying {
		t.Fatalf("選第 1 隊之後應該進遊戲,得到 %v", g.shell.mode)
	}
	if g.party.X != 8 || g.party.Y != 8 {
		t.Errorf("座標應該來自第 1 槽(8,8),得到 (%d,%d)", g.party.X, g.party.Y)
	}
	if g.group.Gold != 75 {
		t.Errorf("金幣應該來自第 1 槽(75),得到 %.0f", g.group.Gold)
	}
	if len(g.members) != 2 {
		t.Errorf("應該有 2 名成員,得到 %d", len(g.members))
	}
}

// ── 驗收 6:全滅 → 全滅畫面 → 回主選單,不能帶著死掉的隊伍繼續走路 ─────

func TestCombatPartyDeadGoesToWipeThenMainMenu(t *testing.T) {
	g := newShellTestGame(t)
	if err := g.loadParty(1); err != nil {
		t.Fatal(err)
	}
	g.shell.mode = shellPlaying

	// 全零值的 Field:沒有任何隊員存活 → Outcome() == PartyDead。
	g.field = &combat.Field{}
	if got := g.field.Outcome(); got != combat.PartyDead {
		t.Fatalf("前提不成立:Outcome()=%v,應為 PartyDead", got)
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.field != nil {
		t.Errorf("戰鬥結束後 g.field 應該清空")
	}
	if g.shell.mode != shellWipe {
		t.Fatalf("PartyDead 之後應該進全滅畫面,得到 %v", g.shell.mode)
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEnter}
	mustUpdate(t, g)
	if g.shell.mode != shellMainMenu {
		t.Errorf("按鍵後應回主選單,得到 %v", g.shell.mode)
	}
	// 不能帶著死掉的隊伍回世界地圖繼續走路:狀態要清空,
	// 而不是回到 shellPlaying 讓移動鍵繼續生效。
	if g.party != (world.State{}) {
		t.Errorf("回主選單後 g.party 應該清空,得到 %+v", g.party)
	}
	if len(g.members) != 0 || g.slot != 0 {
		t.Errorf("回主選單後隊伍狀態應該清空(members=%v slot=%d)", g.members, g.slot)
	}
}

// ── 驗收 7:打倒目標 533 → 結局畫面 → 回主選單 ────────────────────────
//
// 這一條只驗**轉場邏輯本身**,所以直接設 g.bossFight,不打一整場戰鬥。
// 端對端(從迷宮事件觸發一路打到結局畫面)在
// engine/scripted_fight_test.go 的 TestScriptedBossFightReachesEndingScreen。
func TestCombatBossVictoryGoesToEndingThenMainMenu(t *testing.T) {
	g := newShellTestGame(t)
	if err := g.loadParty(1); err != nil {
		t.Fatal(err)
	}
	g.shell.mode = shellPlaying

	g.field = &combat.Field{}
	g.field.Units[combat.PartyBase] = combat.Unit{HP: 10, Facing: combat.South}
	if got := g.field.Outcome(); got != combat.MonstersDead {
		t.Fatalf("前提不成立:Outcome()=%v,應為 MonstersDead", got)
	}
	g.bossFight = true // 由 startScriptedCombat 設定(docs/spec/17)

	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.shell.mode != shellEnding {
		t.Fatalf("MonstersDead + bossFight 之後應該進結局畫面,得到 %v", g.shell.mode)
	}
	if g.bossFight {
		t.Errorf("bossFight 應該在轉場後重置,避免下一場普通戰鬥誤觸結局")
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEnter}
	mustUpdate(t, g)
	if g.shell.mode != shellMainMenu {
		t.Errorf("按鍵後應回主選單,得到 %v", g.shell.mode)
	}
}

// 對照組:沒有 bossFight 旗標的普通戰鬥勝利**不會**觸發結局,
// 只是照舊重置遭遇倒數 —— 確保結局畫面不會被隨便一場戰鬥誤觸發。
func TestCombatOrdinaryVictoryDoesNotEndGame(t *testing.T) {
	g := newShellTestGame(t)
	if err := g.loadParty(1); err != nil {
		t.Fatal(err)
	}
	g.shell.mode = shellPlaying

	g.field = &combat.Field{}
	g.field.Units[combat.PartyBase] = combat.Unit{HP: 10, Facing: combat.South}

	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.shell.mode != shellPlaying {
		t.Errorf("普通戰鬥勝利不該觸發結局,shell.mode=%v", g.shell.mode)
	}
	if g.party.Encounter != 54 {
		t.Errorf("普通戰鬥勝利後應該重置遭遇倒數(佔位值 54),得到 %d", g.party.Encounter)
	}
}

// ── 驗收 8 是後設驗收:上面每一條都直接呼叫 g.Update(),不跑
// ebiten.RunGame、不開視窗 —— 見本檔開頭的說明與 g.testKeys。

// ── 補件(2026-08-15 把關回饋):新玩家開不了局 ─────────────────────
//
// 把關抓到的洞:驗收 4/5 只驗了「列五槽、空的選不下去」,沒驗「主選單
// 造出來的新角色能不能真的組成一支能玩的隊伍」——rosterJoin() 原本無條件
// 用 g.slot 當隊伍編號,從主選單進來時 g.slot == 0,J)oin 連 100% 會失敗;
// 就算改成能選隊伍編號,不補開局初值,選中的隊伍會停在 (8224,8224)
// (docs/spec/06「未初始化的判定」)。下面兩條分別對應把關回饋要的
// (a)(b) 兩種情境。

// (a) 從主選單造角色 → 編隊 → 該隊在隊伍選擇畫面上出現,且座標不是
// (8224,8224),而是 docs/spec/15 §5.1 的具名開局初值。
func TestJoinBlankSlotFromMainMenuAppliesNewPartyDefaults(t *testing.T) {
	g := newShellTestGame(t)
	g.openMainMenu()

	g.testKeys = []ebiten.Key{ebiten.KeyC}
	mustUpdate(t, g)
	if g.roster == nil || !g.roster.open {
		t.Fatalf("前提不成立:C 之後名冊應該開著")
	}
	// 游標移到「新血」(fixture 索引 2,還沒編隊)。
	g.testKeys = []ebiten.Key{ebiten.KeyDown}
	mustUpdate(t, g)
	g.testKeys = []ebiten.Key{ebiten.KeyDown}
	mustUpdate(t, g)
	if g.roster.cursor != 2 {
		t.Fatalf("前提不成立:游標應該在索引 2,得到 %d", g.roster.cursor)
	}

	g.testKeys = []ebiten.Key{ebiten.KeyJ}
	mustUpdate(t, g)
	if !g.roster.joinPick {
		t.Fatalf("前提不成立:g.slot == 0 時按 J 應該進入「編進第幾隊」的詢問")
	}

	g.testKeys = []ebiten.Key{ebiten.KeyDigit2} // 第 2 隊 —— fixture 裡是空白的
	mustUpdate(t, g)
	if g.roster.joinPick {
		t.Errorf("選完隊伍編號後應該離開詢問狀態")
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEscape} // 關名冊,回主選單
	mustUpdate(t, g)
	if g.shell.mode != shellMainMenu {
		t.Fatalf("前提不成立:關名冊後應該回主選單,得到 %v", g.shell.mode)
	}

	g.testKeys = []ebiten.Key{ebiten.KeyL}
	mustUpdate(t, g)
	if g.shell.mode != shellPartySelect {
		t.Fatalf("前提不成立:L 之後應進隊伍選擇,得到 %v", g.shell.mode)
	}

	slot2 := g.shell.slots[1] // 第 2 隊,0-based 索引 1
	if slot2.Blank() {
		t.Fatalf("第 2 隊編完角色後不應該還是空白")
	}
	if slot2.WorldX == 8224 || slot2.WorldY == 8224 {
		t.Errorf("座標不該是 (8224,8224)(CVI(\" \")= 未填初值的症狀,"+
			"docs/spec/06「未初始化的判定」),得到 (%d,%d)", slot2.WorldX, slot2.WorldY)
	}
	if slot2.WorldX != NewPartyX || slot2.WorldY != NewPartyY {
		t.Errorf("座標應該套用 docs/spec/15 §5.1 的開局初值 (%d,%d),得到 (%d,%d)",
			NewPartyX, NewPartyY, slot2.WorldX, slot2.WorldY)
	}
	if slot2.Facing != NewPartyFacing {
		t.Errorf("朝向應該是開局初值 %d,得到 %d", NewPartyFacing, slot2.Facing)
	}
	if slot2.Gold != NewPartyGold {
		t.Errorf("金幣應該是開局初值 %d,得到 %.0f", NewPartyGold, slot2.Gold)
	}
	if slot2.Provisions != NewPartyProvisions {
		t.Errorf("補給應該是開局初值 %d,得到 %d", NewPartyProvisions, slot2.Provisions)
	}
	ids := slot2.MemberIDs()
	if len(ids) != 1 || ids[0] != 3 {
		t.Errorf("第 2 隊應該只有「新血」(ID 3)一名成員,得到 %v", ids)
	}
}

// (b) 對一支「已經在玩」的隊伍加人,座標(以及金幣、時鐘)不能被重設 ——
// 判準是「這筆記錄原本是不是空白」,已經有進度的隊伍不該被套用開局初值。
// 這裡刻意把 fixture 的座標/金幣/時鐘設成跟 NewParty* 不同的值,
// 這樣「有沒有被誤重設」才驗得出來(如果沿用 newShellTestGame 的
// 第 1 槽,它的座標剛好也是 (8,8),測不出重設與不重設的差別)。
func TestJoinExistingPartyDoesNotResetProgress(t *testing.T) {
	dir := t.TempDir()
	saveDir := filepath.Join(dir, "save")
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	chars := make([]original.Character, original.CharSlots)
	chars[0] = original.Character{Party: '1', Name: "老將", ID: 1, Level: 9}
	chars[1] = original.Character{Name: "新兵", ID: 2, Level: 1} // 還沒編隊
	var charBytes []byte
	for _, c := range chars {
		charBytes = append(charBytes, c.Bytes()...)
	}
	if err := os.WriteFile(filepath.Join(saveDir, "CHARS.DAT"), charBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	parsedChars, err := original.ParseChars(charBytes)
	if err != nil {
		t.Fatal(err)
	}

	// 「已經在玩」的隊伍:座標/金幣/時鐘是玩家累積出來的進度,
	// 刻意跟 NewParty*(8,8 / 南 / 75 / 20 / 1,1,4,2)全部不同。
	existing := original.Group{
		WorldX: 50, WorldY: 61, Facing: 1, // 北,不是 NewPartyFacing(南=3)
		Gold: 9999, Provisions: 3,
		Month: 10, Day: 15, Hour: 12, Sub: 6,
	}
	existing.Members[0] = 1
	for i := 1; i < len(existing.Members); i++ {
		existing.Members[i] = 99
	}
	blank := make([]byte, original.GroupRecLen)
	for i := range blank {
		blank[i] = ' '
	}
	groupBytes := append([]byte{}, existing.Bytes()...)
	for i := 0; i < original.GroupSlots-1; i++ {
		groupBytes = append(groupBytes, blank...)
	}
	savePath := filepath.Join(saveDir, "GROUPS.DAT")
	if err := os.WriteFile(savePath, groupBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	g := &Game{assets: dir, savePath: savePath, chars: parsedChars, noSrc: map[int]bool{}}
	g.openTitle()
	g.openMainMenu()

	// C)har Utilities,游標移到「新兵」(索引 1),J)oin 進第 1 隊
	// (那支隊伍「已經在玩」——這條路徑走的是 joinCharacterToSlot 裡
	// slot != g.slot 的分支,因為 g.slot 這時還是 0)。
	g.testKeys = []ebiten.Key{ebiten.KeyC}
	mustUpdate(t, g)
	g.testKeys = []ebiten.Key{ebiten.KeyDown}
	mustUpdate(t, g)
	g.testKeys = []ebiten.Key{ebiten.KeyJ}
	mustUpdate(t, g)
	if !g.roster.joinPick {
		t.Fatalf("前提不成立:應該進入「編進第幾隊」的詢問")
	}
	g.testKeys = []ebiten.Key{ebiten.KeyDigit1}
	mustUpdate(t, g)

	groups, err := g.readGroupSlots()
	if err != nil {
		t.Fatal(err)
	}
	got := groups[0]
	if got.WorldX != 50 || got.WorldY != 61 {
		t.Errorf("加人後座標不該變,應該還是 (50,61),得到 (%d,%d)", got.WorldX, got.WorldY)
	}
	if got.Facing != 1 {
		t.Errorf("加人後朝向不該變,應該還是 1,得到 %d", got.Facing)
	}
	if got.Gold != 9999 {
		t.Errorf("加人後金幣不該變,應該還是 9999,得到 %.0f", got.Gold)
	}
	if got.Provisions != 3 {
		t.Errorf("加人後補給不該變,應該還是 3,得到 %d", got.Provisions)
	}
	if got.Month != 10 || got.Day != 15 || got.Hour != 12 || got.Sub != 6 {
		t.Errorf("加人後時鐘不該變,應該還是 月10 日15 時12 分6,得到 月%d 日%d 時%d 分%d",
			got.Month, got.Day, got.Hour, got.Sub)
	}
	ids := got.MemberIDs()
	if len(ids) != 2 {
		t.Fatalf("應該有 2 名成員(原本的老將 + 新加入的新兵),得到 %v", ids)
	}
}
