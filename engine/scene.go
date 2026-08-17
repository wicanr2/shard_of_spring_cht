package main

// 場景介面與派工表。docs/spec/14-remake-worklist.md §4(C1)。
//
// 這一輪換掉的是 `Update()` 裡那條 270 行的 early-return 鏈:誰先吃到按鍵
// 原本藏在控制流裡,現在是 `inputChain()` 這張**看得見的表**。
//
// ⚠ **這是重構,不是新功能** —— 每個場景的 Update 內容逐段搬過來,
// 順序一模一樣。判準是 T1/T2(端到端)與六張截圖逐位元組相同。
//
// ⚠ **兩張表,不是一張**:
//
//   - `inputChain()` 是**優先序** —— 這一格的按鍵歸誰,第一個 Handles 為真的吃掉。
//   - `drawOrder()` 是**圖層** —— 誰蓋在誰上面,每一格全部都畫。
//
// 兩者本來就不同(戰鬥吃鍵最優先,但畫在迷宮之上、覆蓋層之下),
// 先前混在一起看不出來。
//
// ⚠ **狀態還在 `*Game` 上。** 這一輪只抽輸入與繪圖的邊界,
// 場景結構體都只握一個 `*Game`;把狀態切進各自的套件是 C2
// (docs/spec/14 §4)。⛔ 不要因為「介面已經在了」就當成已經分層。

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/music"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/world"
)

// Input 是這一格收到的輸入。**整個 Update() 只收一次**,再往下傳 ——
// 先前每個分支各自呼叫 inpututil,其中一半繞過了 g.testKeys 接縫,
// 於是那些場景在測試裡按不到任何鍵(docs/spec/14 §8 的涵蓋範圍邊界)。
type Input struct {
	Keys  []ebiten.Key
	Runes []rune
}

// Pressed 回傳這一格有沒有剛按下某個鍵。
func (in Input) Pressed(k ebiten.Key) bool {
	for _, x := range in.Keys {
		if x == k {
			return true
		}
	}
	return false
}

// Transition 是場景處理完這一格之後要做的事。
//
// ⚠ 目前只有兩種。⛔ **不要先加「未來可能要有」的轉場**
// (docs/spec/14 §4:只抽現在真的有實作的那一層)。
type Transition int

const (
	TransitionStay Transition = iota // 什麼都不做,繼續跑
	TransitionQuit                   // 結束程式(主選單的 Q)
)

// Scene 是畫面上的一個單位。
//
//	Handles → 這一格的輸入歸不歸我管(優先序由 inputChain 的順序決定)
//	Update  → 吃掉這一格的輸入
//	Draw    → 畫自己(每一格都會被呼叫,自己判斷該不該畫)
//	Prompt  → 底部指令提示列要顯示什麼(C3)
//
// ⚠ **Prompt 與 Handles 必須是同一個場景說了算。** 提示列先前自己寫了一份
// switch 去猜「現在是哪個畫面」,兩份判斷各自演化 —— 結果戰鬥那一行停在
// 「空白鍵:推進一回合」,而那個鍵在 M10 之後已經不存在。
// 現在提示列直接問**實際會接手這一格的場景**(message.go 的 activeScene)。
type Scene interface {
	Name() string
	Handles(in Input) bool
	Update(in Input) Transition
	Draw(dst *ebiten.Image)
	Prompt() string
}

// dirs 是方向鍵 → 朝向編號(1 北 2 東 3 南 4 西)。世界地圖與迷宮共用。
// 小鍵盤的 1–4 是原版的操作(docs/re/71 的轉譯層)。
var dirs = map[ebiten.Key]int{
	ebiten.KeyUp: 1, ebiten.KeyRight: 2, ebiten.KeyDown: 3, ebiten.KeyLeft: 4,
	ebiten.KeyDigit1: 1, ebiten.KeyDigit2: 2, ebiten.KeyDigit3: 3, ebiten.KeyDigit4: 4,
}

// inputChain 是按鍵的優先序,**順序就是規則**。
//
// ⚠ 順序照搬重構前 `Update()` 的 early-return 鏈,一項都沒動:
// 覆蓋層 → 戰鬥(四個子畫面在前)→ 另存新檔 → 技能點 → N)ames →
// 創角 → 名冊 → 外殼 → 城鎮 → 迷宮機關 → 迷宮 → 世界地圖。
//
// 每一格重建一次切片。⚠ **不要快取成 g 的欄位** —— 場景結構體只握 `*Game`,
// 沒有自己的狀態,快取省下的是幾個指標的配置,換來的是「這張表什麼時候建的」
// 這個問題。
func (g *Game) inputChain() []Scene {
	return []Scene{
		overlayScene{g},
		inspectScene{g},
		castCursorScene{g},
		combatPotionScene{g},
		castSPScene{g},
		castMenuScene{g},
		useMenuScene{g},
		combatScene{g},
		saveAsScene{g},
		skillAllocScene{g},
		rosterHotkey{g},
		createScene{g},
		rosterScene{g},
		shellScene{g},
		townScene{g},
		mazePromptScene{g},
		mazeScene{g},
		worldScene{g},
	}
}

// drawOrder 是圖層順序:排在後面的蓋在前面之上。
//
// ⚠ 與 inputChain **不同**,而且不該一致 —— 見檔頭。
// 外殼接管畫面時走的是另一條路(Game.Draw 直接畫外殼 + 覆蓋層)。
func (g *Game) drawOrder() []Scene {
	return []Scene{
		worldScene{g}, // 地圖底層 + 四個外框 + 隊伍側欄
		mazeScene{g},
		townScene{g},
		rosterScene{g},
		createScene{g},
		skillAllocScene{g},
		combatScene{g},
		castMenuScene{g},
		castSPScene{g},
		useMenuScene{g},
		inspectScene{g},
		overlayScene{g},
		mazePromptScene{g},
		saveAsScene{g},
	}
}

// ── 覆蓋層 ────────────────────────────────────────────────────────────────

// overlayScene 是敘述覆蓋層(docs/spec/04 §3:出現時遊戲暫停)。
// A6 按鍵表(docs/spec/15 §8)借用同一個機制,不專屬迷宮敘述。
type overlayScene struct{ g *Game }

func (s overlayScene) Prompt() string       { return "任意鍵：繼續" }
func (s overlayScene) Name() string         { return "overlay" }
func (s overlayScene) Handles(Input) bool   { return s.g.overlay != "" }
func (s overlayScene) Draw(d *ebiten.Image) { s.g.drawOverlay(d) }
func (s overlayScene) Update(in Input) Transition {
	if len(in.Keys) > 0 {
		s.g.overlay = ""
	}
	return TransitionStay
}

// ── 戰鬥中的四個子畫面 ────────────────────────────────────────────────────

// castCursorScene 是施法的選格階段(手冊 p.34 的 I/J/K/M + 空白鍵)。
type castCursorScene struct{ g *Game }

func (s castCursorScene) Prompt() string {
	// CMBT:118+119「Where do you want to cast it?」+ 114「(ESC to Exit)」
	return castWhere + castEscExit + "　方向鍵／I・J・K・M：移動游標　　空白鍵：施放"
}
func (s castCursorScene) Name() string       { return "cast-cursor" }
func (s castCursorScene) Handles(Input) bool { return s.g.field != nil && s.g.cursor != nil }
func (s castCursorScene) Draw(*ebiten.Image) {} // 游標畫在戰場上(drawCombat)
func (s castCursorScene) Update(in Input) Transition {
	for _, k := range in.Keys {
		if s.g.cursorKey(k) {
			break
		}
	}
	return TransitionStay
}

// combatPotionScene 是「自己用 / 丟給隊友」的子流程(docs/spec/19 §2-1)。
type combatPotionScene struct{ g *Game }

func (s combatPotionScene) Prompt() string     { return s.potionPrompt() }
func (s combatPotionScene) Name() string       { return "combat-potion" }
func (s combatPotionScene) Handles(Input) bool { return s.g.field != nil && s.g.combatPotion != nil }
func (s combatPotionScene) Draw(*ebiten.Image) {} // 提示畫在道具選單裡
func (s combatPotionScene) Update(in Input) Transition {
	for _, k := range in.Keys {
		if s.g.combatPotionKey(k) {
			break
		}
	}
	return TransitionStay
}

// inspectScene 是戰場的單位檢視面板(CMBT:179–192)。**唯讀**,開著時吃掉全部輸入。
type inspectScene struct{ g *Game }

func (s inspectScene) Prompt() string {
	return "↑／↓：換單位　　ESC：關閉　" + inspectScroll
}
func (s inspectScene) Name() string         { return "inspect" }
func (s inspectScene) Handles(Input) bool   { return s.g.field != nil && s.g.inspect != nil }
func (s inspectScene) Draw(d *ebiten.Image) { s.g.drawInspect(d) }
func (s inspectScene) Update(in Input) Transition {
	for _, k := range in.Keys {
		if s.g.inspectKey(k) {
			break
		}
	}
	return TransitionStay
}

// castSPScene 是施法的「投入幾點法力」那一步(CMBT:101)。
//
// ⚠ 排在 castMenuScene **前面**:兩者不會同時開著(進這一步時 castList 已清空),
// 但派工表的順序是規格的一部分,把子步驟放在它的母步驟前面比較不會出錯。
type castSPScene struct{ g *Game }

func (s castSPScene) Prompt() string {
	return "數字：投入點數　　Enter：確認　　Backspace：修改　　ESC：取消"
}
func (s castSPScene) Name() string         { return "cast-sp" }
func (s castSPScene) Handles(Input) bool   { return s.g.field != nil && s.g.castSP != nil }
func (s castSPScene) Draw(d *ebiten.Image) { s.g.drawCastSP(d) }
func (s castSPScene) Update(in Input) Transition {
	for _, k := range in.Keys {
		if s.g.castSPKey(k) {
			break
		}
	}
	return TransitionStay
}

// castMenuScene:施法選單開著時只吃字母與翻頁鍵。
type castMenuScene struct{ g *Game }

func (s castMenuScene) Prompt() string {
	return "字母：選法術　　PgDn／PgUp：翻頁　　ENTER／ESC：離開"
}
func (s castMenuScene) Name() string         { return "cast-menu" }
func (s castMenuScene) Handles(Input) bool   { return s.g.field != nil && len(s.g.castList) > 0 }
func (s castMenuScene) Draw(d *ebiten.Image) { s.g.drawCastMenu(d) }
func (s castMenuScene) Update(in Input) Transition {
	g := s.g
	// CMBT:113「Hit PgDn key」—— 原版就是用 PgDn 翻法術清單。
	if in.Pressed(ebiten.KeyPageDown) {
		if g.castPage+1 < g.castPages() {
			g.castPage++
		}
		return TransitionStay
	}
	if in.Pressed(ebiten.KeyPageUp) {
		if g.castPage > 0 {
			g.castPage--
		}
		return TransitionStay
	}
	// ⚠ 掃**全部 26 個字母**而不是只掃這一頁有幾個 —— 按到清單以外的字母
	// 要走 pickSpell 的「沒有這個法術!」那一支(CMBT:91/92),
	// 只掃有效範圍的話那句話永遠不會出現。
	for i := 0; i < 26; i++ {
		if in.Pressed(ebiten.KeyA + ebiten.Key(i)) {
			g.pickSpell(i)
			break
		}
	}
	// CMBT:90「 (ENTER exits)」—— 原版用 ENTER 離開,ESC 一併收下。
	if in.Pressed(ebiten.KeyEscape) || in.Pressed(ebiten.KeyEnter) ||
		in.Pressed(ebiten.KeyKPEnter) {
		g.castList = nil
	}
	return TransitionStay
}

// useMenuScene:道具選單開著時只吃字母(docs/spec/12 §5.3 的 U)。
type useMenuScene struct{ g *Game }

func (s useMenuScene) Prompt() string       { return "字母：選道具　　ESC：取消" }
func (s useMenuScene) Name() string         { return "use-menu" }
func (s useMenuScene) Handles(Input) bool   { return s.g.field != nil && len(s.g.useList) > 0 }
func (s useMenuScene) Draw(d *ebiten.Image) { s.g.drawUseMenu(d) }
func (s useMenuScene) Update(in Input) Transition {
	for i := 0; i < len(s.g.useList) && i < 26; i++ {
		if in.Pressed(ebiten.KeyA + ebiten.Key(i)) {
			s.g.pickUseItem(i)
			break
		}
	}
	if in.Pressed(ebiten.KeyEscape) {
		s.g.useList = nil
	}
	return TransitionStay
}

// ── 戰場本體 ──────────────────────────────────────────────────────────────

// combatScene 是戰場操作(docs/spec/12)。
//
// ⚠ 先前的「空白鍵推一整回合」已經移除 —— 兩套並存會讓同一場戰鬥有兩種規則
// (一套算行動點數、一套不算),而畫面上分不出剛才用了哪一套。
type combatScene struct{ g *Game }

func (s combatScene) Prompt() string       { return s.combatPrompt() }
func (s combatScene) Name() string         { return "combat" }
func (s combatScene) Handles(Input) bool   { return s.g.field != nil }
func (s combatScene) Draw(d *ebiten.Image) { s.g.drawCombat(d) }
func (s combatScene) Update(in Input) Transition {
	g := s.g
	if in.Pressed(ebiten.KeyC) {
		g.openCast()
		return TransitionStay
	}
	if in.Pressed(ebiten.KeyU) {
		g.openUseItem()
		return TransitionStay
	}
	// ⚠ **開檢視面板的鍵是引擎挑的**(inspect_scene.go 有說明):
	// CMBT 的指令鏈有 `?` 與 `/`,但哪一個對到這個面板沒有讀到。
	if in.Pressed(ebiten.KeySlash) {
		g.openInspect()
		return TransitionStay
	}
	// CMBT:52/53「Sound is now on./off.」——`S` 在 CMBT 的指令鏈裡
	// (docs/re/70),而戰場沒有別的 `S` 指令。
	// ⚠ 世界地圖與迷宮的 `S` 是**本引擎的存檔鍵**,不動它。
	if in.Pressed(ebiten.KeyS) {
		g.field.Log = append(g.field.Log, g.toggleSound())
		return TransitionStay
	}
	// D)ispell —— `D` 在 CMBT 的指令鏈裡,語意由 docs/re/188 解出來。
	if in.Pressed(ebiten.KeyD) {
		g.dispell()
		return TransitionStay
	}
	for _, k := range in.Keys {
		if g.boardKey(k) {
			return TransitionStay
		}
	}
	// 戰後的兩個「要撿嗎?(Y/N)」(CMBT:62/63)——金幣先問、道具後問。
	//
	// ⚠ **先前這兩個問句印得出來卻答不了**:`takeGold` 只有測試在呼叫,
	// 畫面上按 Y 沒有任何反應。那個缺口沒有症狀 ——
	// 問句照樣出現,玩家只會以為自己按錯鍵(docs/spec/19-coverage §2.1 的同一類)。
	if g.field.Outcome() != combat.Ongoing {
		if yes, no := in.Pressed(ebiten.KeyY), in.Pressed(ebiten.KeyN); yes || no {
			switch {
			case g.pendingGold > 0:
				g.takeGold(yes)
			case g.pendingLoot != nil:
				g.takeLoot(yes)
			}
			return TransitionStay
		}
	}
	if in.Pressed(ebiten.KeyEscape) && g.field.Outcome() != combat.Ongoing {
		g.leaveCombat()
	}
	return TransitionStay
}

// leaveCombat 收掉一場已經分出勝負的戰鬥,決定接下來去哪個畫面。
//
// docs/re/181 §4 + docs/spec/18 §1 第 3 項:打完一場由迷宮事件引發的戰鬥,
// 要把那個目標的事件作廢,否則走出迷宮再走回來會再打一次。
// 要在 g.field 被清成 nil 之前先讀 f.Log[0] —— combat_scene.go 的
// startScriptedCombat 已經把它當「這場是不是祭司事件」的識別
// (rules.PriestEncounterMark 的說明),settle() 顯示祝福文字用的是同一個判準。
func (g *Game) leaveCombat() {
	outcome := g.field.Outcome()
	f := g.field
	g.field = nil
	// 沒回答的「要撿嗎?」一律視為不撿(docs/re/200 §1.2)——
	// ⚠ 靜靜地幫玩家撿起來會讓那句問話變成假的。
	g.pendingGold, g.pendingLoot = 0, nil
	switch {
	case outcome == combat.PartyDead:
		// A4 全滅(docs/spec/15 §6)。**不能帶著死掉的隊伍回世界地圖
		// 繼續走路** —— 直接進全滅畫面,按鍵後回主選單。
		// 死亡曲(music.Userlib)已經在結算的那一刻放過(combat_scene.go
		// 的 settle),這裡不重放。
		if g.shell != nil {
			g.shell.mode = shellWipe
		}
	case outcome == combat.MonstersDead && g.bossFight:
		// A5 結局(docs/spec/15 §7):這場是迷宮事件目標 533
		// (maze.TargetFinalBoss)引發的劇情戰鬥,打贏了。
		//
		// 533 打完遊戲基本上結束了,但仍然照規則作廢這個目標
		// (不因為「反正要回主選單了」就跳過)。
		g.disableMazeEvent(maze.TargetFinalBoss)
		g.bossFight = false
		if g.shell != nil {
			g.shell.mode = shellEnding
		}
		g.play(music.FileEnding, music.Ending)
	case outcome == combat.MonstersDead && len(f.Log) > 0 && f.Log[0] == rules.PriestEncounterMark:
		// 山丘巨人挾持祭司(maze.TargetPriest = 204)打贏了 ——
		// 同一條規則:作廢目標 204,否則可以無限次「救祭司」。
		g.disableMazeEvent(maze.TargetPriest)
		g.party.Encounter = world.RollEncounter(g.rand)
	default:
		// 每打完一場重填:INT(RND × 10) + 25(docs/re/214)。
		g.party.Encounter = world.RollEncounter(g.rand)
	}
}

// ── 覆蓋在遊戲畫面之上的輸入類畫面 ────────────────────────────────────────

// saveAsScene 是另存新檔的文字輸入(docs/spec/18 §2,save_ui.go)。
//
// ⚠ 要優先於 N)ames 等其他按鍵 —— 否則打名稱打到 `n` 會被別的畫面搶走。
type saveAsScene struct{ g *Game }

func (s saveAsScene) Prompt() string {
	return "打字：輸入檔名　　Enter：存檔　　ESC：取消"
}
func (s saveAsScene) Name() string         { return "save-as" }
func (s saveAsScene) Handles(Input) bool   { return s.g.saveAs != nil }
func (s saveAsScene) Draw(d *ebiten.Image) { s.g.drawSaveAs(d) }
func (s saveAsScene) Update(in Input) Transition {
	s.g.saveAsRunes(in.Runes)
	for _, k := range in.Keys {
		s.g.saveAsKey(k)
		if s.g.saveAs == nil {
			break
		}
	}
	return TransitionStay
}

// skillAllocScene 是技能點分配(docs/spec/20)。創角完成 / 升級成功之後
// 蓋在原畫面上,兩個入口共用。
//
// ⚠ 排在創角 / 名冊 / 城鎮 / N 鍵**之前** —— 不管背後是名冊還是城鎮,
// 這個畫面開著時要吃光所有按鍵,不能讓底下的畫面漏接。
type skillAllocScene struct{ g *Game }

func (s skillAllocScene) Prompt() string {
	return "數字：技能編號　　Enter：確定　　0＋Enter：結束　　ESC：離開"
}
func (s skillAllocScene) Name() string         { return "skill-alloc" }
func (s skillAllocScene) Handles(Input) bool   { return s.g.skillAlloc != nil }
func (s skillAllocScene) Draw(d *ebiten.Image) { s.g.drawSkillAlloc(d) }
func (s skillAllocScene) Update(in Input) Transition {
	for _, k := range in.Keys {
		s.g.skillAllocKey(k)
		if s.g.skillAlloc == nil {
			break
		}
	}
	return TransitionStay
}

// rosterHotkey 是遊戲中的 N)ames 快捷鍵(docs/spec/11 §5)。
//
// ⚠ **這不是一個畫面,是一個全域熱鍵** —— 它只在按下 N 的那一格接手,
// 其餘的鍵照樣往下傳給城鎮 / 迷宮 / 世界地圖。
// 它在鏈上的位置就是原本 `Update()` 裡那一行的位置,不要移動:
// 移到前面會蓋掉技能點分配,移到後面則城鎮與迷宮裡按 N 會失效。
type rosterHotkey struct{ g *Game }

func (s rosterHotkey) Prompt() string     { return "" }
func (s rosterHotkey) Name() string       { return "roster-hotkey" }
func (s rosterHotkey) Draw(*ebiten.Image) {}
func (s rosterHotkey) Handles(in Input) bool {
	// ⚠ 名冊**已經開著**時不接手 —— 名冊裡的 `N` 是 N)ew Name(改名),
	// 被這個熱鍵吃掉的話改名鍵在畫面上完全沒有反應。
	if s.g.roster != nil && s.g.roster.open {
		return false
	}
	playing := s.g.shell == nil || s.g.shell.mode == shellPlaying
	return playing && in.Pressed(ebiten.KeyN)
}
func (s rosterHotkey) Update(Input) Transition {
	s.g.openRoster()
	return TransitionStay
}

// createScene 是建立角色(蓋在名冊之上)。
type createScene struct{ g *Game }

func (s createScene) Prompt() string       { return s.g.createPrompt() }
func (s createScene) Name() string         { return "create" }
func (s createScene) Handles(Input) bool   { return s.g.create != nil }
func (s createScene) Draw(d *ebiten.Image) { s.g.drawCreate(d) }
func (s createScene) Update(in Input) Transition {
	s.g.createRunes(in.Runes)
	for _, k := range in.Keys {
		s.g.createKey(k)
		if s.g.create == nil {
			break
		}
	}
	return TransitionStay
}

// rosterScene 是名冊(docs/spec/11 §5)。
//
// ⚠ ESC 回哪裡由 g.shell.mode 決定,rosterKey 本身不記「是從哪裡打開的」
// (docs/spec/15 §4,見 openMainMenu 的說明)。
type rosterScene struct{ g *Game }

func (s rosterScene) Prompt() string {
	// 助憶鍵照原版(docs/re/143):C)reate R)emove N)ew Name / D)isband J)oin
	// I)nformation E)xit。
	return "↑↓選　C造角色　R刪除　N改名　J入隊　D解散　I資訊　E離開"
}
func (s rosterScene) Name() string         { return "roster" }
func (s rosterScene) Handles(Input) bool   { return s.g.roster != nil && s.g.roster.open }
func (s rosterScene) Draw(d *ebiten.Image) { s.g.drawRoster(d) }
func (s rosterScene) Update(in Input) Transition {
	// 文字輸入要在按鍵之前收 —— Enter 會把 rename 收掉,
	// 收完之後那一格的 Runes 就沒有地方去了。
	s.g.rosterRenameRunes(in.Runes)
	for _, k := range in.Keys {
		s.g.rosterKey(k)
	}
	return TransitionStay
}

// shellScene 是外殼:標題 / 主選單 / 存檔選擇 / 匯入 / 隊伍選擇 / 全滅 / 結局
// (docs/spec/15)。
//
// ⚠ 排在 create / roster **之後** —— C)har Utilities 蓋在主選單上時,
// 上面兩個先接手;名冊關閉(ESC)後 g.shell.mode 沒被動過,
// 下一格自然落回這裡,也就「回到主選單」。
type shellScene struct{ g *Game }

func (s shellScene) Prompt() string     { return "" }
func (s shellScene) Name() string       { return "shell" }
func (s shellScene) Draw(*ebiten.Image) {} // 外殼接管整個畫布,見 Game.Draw
func (s shellScene) Handles(Input) bool {
	return s.g.shell != nil && s.g.shell.mode != shellPlaying
}
func (s shellScene) Update(in Input) Transition { return s.g.shellUpdate(in) }

// ── 城鎮 ──────────────────────────────────────────────────────────────────

type townScene struct{ g *Game }

// townScene.Prompt 依所在畫面換句子。措辭照 TOWN.tsv(F3):
// 第 5 列是城鎮地圖、第 10 列是商店(有分頁)、第 12 列是沒有分頁的那一版。
//
// ⚠ 原版的商店提示有兩個版本,差別只在**要不要翻頁** —— 這裡照做,
// 一頁放得下就不提翻頁,免得畫面叫玩家按一個沒有作用的鍵。
func (s townScene) Prompt() string {
	g := s.g
	if g.town == nil {
		// 還沒進城鎮(或測試直接建構場景)—— 提示列**不能是空的**
		// (scene_test.go 的 TestEveryScenePrompt),退回通用那一句。
		return townGenericPrompt
	}
	switch g.town.mode {
	case townBuildings:
		return "選擇要進入的建築物,# 查看角色,(ESC離開)"
	case townShop:
		if g.town.pages(g.itemList) > 1 {
			return "選購道具,ESC離開,任意鍵翻頁。"
		}
		return "選購道具,ESC離開。"
	}
	return townGenericPrompt
}

// townGenericPrompt 給沒有原版對應句子的子畫面(旅店/治療所/酒館/訓練所/營地)。
const townGenericPrompt = "字母：選項　　+／-：翻頁　　ESC：返回／離開城鎮"
func (s townScene) Name() string         { return "town" }
func (s townScene) Handles(Input) bool   { return s.g.town != nil && s.g.town.mode != townClosed }
func (s townScene) Draw(d *ebiten.Image) { s.g.drawTown(d) }
func (s townScene) Update(in Input) Transition {
	// 咒語輸入要先收字元 —— 按鍵那一圈會把 Enter 當成送出。
	s.g.utterRunes(in.Runes)
	for _, k := range in.Keys {
		if k == ebiten.KeyEscape && s.g.town.mode == townBuildings {
			s.g.town = nil
			s.g.saveMsg = "隊伍離開了……" // TOWN:7「The party leaves..」
			break
		}
		s.g.townKey(k)
	}
	return TransitionStay
}

// ── 迷宮 ──────────────────────────────────────────────────────────────────

// mazePromptScene 是迷宮機關的問答(寶石謎題、治療池、Eldron 氏族)。
type mazePromptScene struct{ g *Game }

func (s mazePromptScene) Prompt() string       { return s.g.mazePromptHint() }
func (s mazePromptScene) Name() string         { return "maze-prompt" }
func (s mazePromptScene) Handles(Input) bool   { return s.g.prompt != nil }
func (s mazePromptScene) Draw(d *ebiten.Image) { s.g.drawPrompt(d) }
func (s mazePromptScene) Update(in Input) Transition {
	s.g.promptRunes(in.Runes)
	for _, k := range in.Keys {
		s.g.promptKey(k)
		if s.g.prompt == nil {
			break
		}
	}
	return TransitionStay
}

type mazeScene struct{ g *Game }

func (s mazeScene) Prompt() string {
	return "方向鍵／1234：移動　　ESC：離開地城　　S：存檔　　A：另存新檔"
}
func (s mazeScene) Name() string         { return "maze" }
func (s mazeScene) Handles(Input) bool   { return s.g.level != nil }
func (s mazeScene) Draw(d *ebiten.Image) { s.g.drawMaze(d) }
func (s mazeScene) Update(in Input) Transition {
	g := s.g
	for key, d := range dirs {
		if in.Pressed(key) {
			g.stepMaze(maze.Facing(d))
		}
	}
	if in.Pressed(ebiten.KeyEscape) {
		// ⚠ 原版怎麼離開迷宮**未解**(docs/re/146 §2)。實跑已經排除
		// `ESC` 與 `E`(兩者在原版的迷宮裡都沒有作用),`Q` 是離開遊戲。
		// 這裡沿用 ESC 是**本引擎的選擇**,不是原版行為。
		g.level = nil
		g.syncMazeNum()
	}
	// docs/spec/18 §3.2 MazeFile + 驗收 4:原版在迷宮裡也存得了檔
	// (GROUPS.DAT 位移 79/81 就是為此存在的)。
	if in.Pressed(ebiten.KeyS) {
		g.saveHere()
	}
	if in.Pressed(ebiten.KeyA) {
		g.openSaveAs() // 另存新檔(docs/spec/18 §2,save_ui.go)
	}
	return TransitionStay
}

// ── 世界地圖(鏈的最後一項:沒有別人接手就是它)────────────────────────────

type worldScene struct{ g *Game }

func (s worldScene) Prompt() string {
	return "方向鍵／1234：移動　　N：名冊　　S：存檔　　A：另存新檔"
}
func (s worldScene) Name() string       { return "world" }
func (s worldScene) Handles(Input) bool { return true }

// Draw 畫地圖底層 + 四個外框 + 隊伍側欄。
//
// ⚠ 地圖只有在**沒有別的畫面接管主視野**時才畫 —— 這四個旗標照搬重構前
// Game.Draw 裡的判斷,一項都沒動。外框與側欄則是無條件畫(同重構前)。
func (s worldScene) Draw(dst *ebiten.Image) {
	g := s.g
	inCombat := g.field != nil
	inMaze := g.level != nil && !inCombat
	inTown := g.town != nil && g.town.mode != townClosed && !inCombat && !inMaze
	inRoster := (g.roster != nil && g.roster.open) || g.create != nil

	// 9×9 視野,隊伍固定在正中央(docs/spec/05 §3、§4)。
	const half = layout.ViewTiles / 2
	for vy := 0; !inCombat && !inMaze && !inTown && !inRoster && vy < layout.ViewTiles; vy++ {
		for vx := 0; vx < layout.ViewTiles; vx++ {
			// 天色會縮視野(docs/re/213):繪製半徑 = 生效能見度,
			// 白天 4 → 9×9 全開,入夜 2 → 5×5,深夜 1 → 3×3。
			// ⚠ 原版把起點往內推 `(4 − 能見度) × 17` 像素,
			// 縮小之後仍然置中 —— 那個算式反過來證明半徑就是能見度。
			if vx < half-g.party.Visibility || vx > half+g.party.Visibility ||
				vy < half-g.party.Visibility || vy > half+g.party.Visibility {
				continue
			}
			mx, my := g.party.X-half+vx, g.party.Y-half+vy
			v := g.world.At(mx, my)
			px := float32(layout.View.X + vx*layout.TileDst)
			py := float32(layout.View.Y + vy*layout.TileDst)

			// 值 11(海洋,全圖 55.63%)原版**一個像素都不畫**
			// (docs/re/132 §1),顯示的就是底色。這裡同樣什麼都不做 ——
			// 畫一張「海的圖」會讓畫面比原版多東西。
			if src, _ := original.WorldTileOrigin(v); src == original.SrcBackdrop {
				continue
			}

			if img, ok := g.tiles[v]; ok {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(layout.ArtScale, layout.ArtScale)
				op.GeoM.Translate(float64(px), float64(py))
				// 最近鄰 —— 整數倍放大不該有插值(docs/spec/04 §1)。
				op.Filter = ebiten.FilterNearest
				dst.DrawImage(img, op)
			} else {
				g.noSrc[v] = true
				vector.DrawFilledRect(dst, px, py,
					layout.TileDst, layout.TileDst, missing, false)
			}
		}
	}

	if !inCombat && !inMaze && !inTown && !inRoster {
		// 隊伍所在格的框
		c := float32(layout.View.X + half*layout.TileDst)
		r := float32(layout.View.Y + half*layout.TileDst)
		vector.StrokeRect(dst, c, r, layout.TileDst, layout.TileDst, 3, cgaWhite, false)
	}

	frame := func(rc layout.Rect) {
		vector.StrokeRect(dst, float32(rc.X), float32(rc.Y),
			float32(rc.W), float32(rc.H), 2, cgaWhite, false)
	}
	frame(layout.View)
	frame(layout.Party)
	frame(layout.Message)
	frame(layout.Prompt)

	g.drawParty(dst)
}
func (s worldScene) Update(in Input) Transition {
	g := s.g
	for key, d := range dirs {
		if !in.Pressed(key) {
			continue
		}
		if g.party.Step(world.Facing(d), g.world) != world.Moved {
			continue
		}
		// 站到拱門那一格且朝南 → 印敘述(docs/re/198)
		g.archwayCheck()
		// 踩到地城入口 → 進迷宮(docs/spec/08 §6)
		if g.enterMaze(g.party.X, g.party.Y) {
			return TransitionStay
		}
		// 踩到城鎮 → 進城(docs/spec/11 §2)
		if v := g.world.At(g.party.X, g.party.Y); v >= 30 && v <= 32 {
			if g.enterTown(g.party.X, g.party.Y) {
				return TransitionStay
			}
		}
		// docs/formats/02 位移 25:歸零時觸發遭遇檢查
		if g.party.Encounter == 0 {
			g.startCombat()
		}
	}
	if in.Pressed(ebiten.KeyS) {
		g.saveHere()
	}
	if in.Pressed(ebiten.KeyA) {
		g.openSaveAs()
	}
	return TransitionStay
}

// saveHere 是世界地圖與迷宮共用的 S)ave。
//
// ⚠ **存檔也推進時鐘一格**(docs/re/149:只按 S 不移動,位移 33 仍 +1)。
// 這不是直覺的行為 —— 但三次量測都吻合「每個動作一格」。
//
// 重構前這段在世界地圖與迷宮各寫一份,連失敗訊息的冒號都一個全形一個半形。
func (g *Game) saveHere() {
	g.party.Tick()
	if err := g.save(); err != nil {
		g.saveMsg = "存檔失敗:" + err.Error()
		return
	}
	g.saveMsg = fmt.Sprintf("已存到第 %d 隊(%s)", g.slot, g.savePath)
}

// ── 動態的提示列(內容隨子狀態變)────────────────────────────────────────

// combatPrompt 是戰場的指令提示。
//
// ⚠ 這一行先前寫在 party_panel.go 的一個 switch 裡,而且已經**過期**:
// 它寫「空白鍵:推進一回合」,但那個操作在 M10 改成戰場操作時就移除了
// (docs/spec/12);「C:施法(固定投一級)」也在 E5 之後不成立。
// 提示列現在由場景自己說,不會再各自演化。
func (s combatScene) combatPrompt() string {
	if s.g.field != nil && s.g.field.Outcome() != combat.Ongoing {
		return "ESC：離開戰鬥"
	}
	// ⚠ 一行要放得進提示列的欄寬預算(倚天版 80 欄,docs/spec/04 §5)——
	// 超出去不會報錯,字會壓在右邊的面板上(scene_test.go 的 TestEveryScenePrompt)。
	return "方向鍵：移動　A：攻擊　C：施法　D：驅散　U：道具　/：檢視　S：音效　Enter：結束"
}

// potionPrompt 是「自己用 / 丟給隊友」兩個階段的提示(docs/spec/19 §2-1)。
func (s combatPotionScene) potionPrompt() string {
	if s.g.combatPotion != nil && s.g.combatPotion.stage == 2 {
		return "1–5：選要丟給誰"
	}
	return "Y：自己用　　T：丟給隊友"
}

// createPrompt 依創角走到哪一步給提示(create_scene.go 的四步)。
func (g *Game) createPrompt() string {
	if g.create == nil {
		return ""
	}
	switch g.create.step {
	case stepRace:
		return "H／D／T／E／G：選種族　　ESC：取消"
	case stepAdjust:
		// ⚠ ESC 在這一步**不是取消** —— 有勾選就重擲勾選的那幾項,
		// 沒勾選才是「這組我接受了」,往下一步走(create_scene.go)。
		return "1–5：勾選要重擲的屬性　　ESC：重擲勾選的（沒勾選就是接受這組）"
	case stepKeep:
		// CHARUTIL:33「Do you wish to keep this character (Y/N) ?」
		return "要保留這位角色嗎?(Y/N)"
	case stepClass:
		return "F：戰士　　W：巫師　　ESC：取消"
	default:
		return "打字：輸入名字　　Enter：完成　　ESC：取消"
	}
}

// mazePromptHint 依機關種類給提示(docs/re/155、162)。
func (g *Game) mazePromptHint() string {
	if g.prompt == nil {
		return ""
	}
	switch g.prompt.kind {
	case promptGem:
		return "B／G／V／R：依序輸入四個寶石顏色　　ESC：離開"
	case promptPool:
		return "1–5：選要治療的隊員　　ESC：離開"
	default:
		return "打字：輸入名字　　Enter：送出　　ESC：離開"
	}
}
