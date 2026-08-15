package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/render"
	"shardofspring/internal/ui"
	"shardofspring/internal/world"
)

// 遊戲外殼:標題、主選單、隊伍選擇、全滅、結局。docs/spec/15-game-shell.md。
//
// 原版 MENU.EXE 的主選單**沒有「開新遊戲」**:開局是兩步 ——
// 先 C)har Utilities 造角色、組隊,回主選單再 L)oad a Party 選那一隊進遊戲
// (docs/spec/15 §1)。這裡照抄這個流程,不要自己發明「新遊戲／讀取進度」
// 的二分。

// shellMode 是外殼狀態機的畫面。docs/spec/15 §2 的場景圖。
type shellMode int

const (
	shellTitle       shellMode = iota // A1 標題畫面(docs/spec/15 §3)
	shellMainMenu                     // A2 主選單(§4)
	shellPartySelect                  // A3 隊伍選擇 = 開局(§5)
	shellWipe                         // A4 全滅(§6)
	shellEnding                       // A5 結局(§7)
	shellPlaying                      // 遊戲中,外殼不接管 Update/Draw
)

// shellState 是外殼的狀態機。
type shellState struct {
	mode shellMode

	// slots 是隊伍選擇畫面(A3)的 5 個槽。每次 openPartySelect 都重新讀
	// GROUPS.DAT,不快取 —— 從主選單的 C)har Utilities 回來後,GROUPS.DAT
	// 可能已經被 J)oin/D)isband 改過(docs/re/135)。
	slots []original.Group
	// msg 是主選單/隊伍選擇畫面的錯誤或提示訊息。
	msg string
}

// openTitle 是程式進入點的預設狀態(docs/spec/15 §1:預設走標題)。
func (g *Game) openTitle() {
	g.shell = &shellState{mode: shellTitle}
}

// openMainMenu 回到 A2。C)har Utilities 蓋在它上面時不會改動這裡的 mode ——
// 名冊關閉後(rosterKey 的 ESC)控制權自然落回這個狀態,不必額外記錄
// 「呼叫端是誰」(docs/spec/15 §4:呼叫端決定回哪裡,不要讓名冊自己記)。
func (g *Game) openMainMenu() {
	if g.shell == nil {
		g.shell = &shellState{}
	}
	g.shell.mode = shellMainMenu
	g.shell.msg = ""
}

// openPartySelect 讀 GROUPS.DAT 的五個槽,進 A3。
func (g *Game) openPartySelect() error {
	slots, err := g.readGroupSlots()
	if err != nil {
		return err
	}
	g.shell.slots = slots
	g.shell.mode = shellPartySelect
	g.shell.msg = ""
	return nil
}

func (g *Game) readGroupSlots() ([]original.Group, error) {
	b, err := os.ReadFile(g.savePath)
	if err != nil {
		return nil, err
	}
	return original.ParseGroups(b)
}

// writeGroups 把全部 original.GroupSlots 筆記錄寫回 GROUPS.DAT。
//
// 用在「改動的槽不是目前正在玩的那槽」的情況(roster_scene.go 的
// joinCharacterToSlot)—— GROUPS.DAT 是一個檔案裝五筆記錄,
// 只改其中一筆也要整份重寫,不能單獨動一段。
func writeGroups(path string, groups []original.Group) error {
	out := make([]byte, 0, original.GroupRecLen*original.GroupSlots)
	for _, x := range groups {
		out = append(out, x.Bytes()...)
	}
	return os.WriteFile(path, out, 0o644)
}

// selectParty 選第 slot 隊(1–5)進遊戲。
//
// 未初始化的判定是「整筆記錄全為 0x20」(docs/spec/06「未初始化的判定」/
// docs/spec/15 §5)——**不判座標是不是 (8224,8224)**,那是症狀不是原因。
// original.Group.Blank() 就是這個判定。
func (g *Game) selectParty(slot int) {
	if slot < 1 || slot > len(g.shell.slots) || g.shell.slots[slot-1].Blank() {
		g.shell.msg = fmt.Sprintf("第 %d 隊還沒有資料 —— 先去 C)har Utilities 組隊", slot)
		return
	}
	if err := g.loadParty(slot); err != nil {
		g.shell.msg = "載入失敗:" + err.Error()
		return
	}
	g.shell.mode = shellPlaying
}

// returnToMainMenu 清掉這一局的暫存狀態,回到主選單。docs/spec/15 §6/§7。
//
// ⚠ **不自動讀檔**(docs/spec/15 §6):下次要玩哪一隊由玩家在主選單重新
// L)oad 選,這裡只清記憶體裡的狀態,不碰 <assets>/save/ 底下的檔案。
// 全滅之所以不能「帶著死掉的隊伍回世界地圖繼續走路」,靠的就是這裡
// 把 g.party/g.members/g.level/g.town 全部清空 —— 下一輪 Update()
// 落在 shell.mode != shellPlaying,不會再進任何移動邏輯。
func (g *Game) returnToMainMenu() {
	g.field, g.level, g.town = nil, nil, nil
	g.roster, g.create, g.prompt = nil, nil, nil
	g.cursor, g.castList = nil, nil
	g.members, g.group, g.slot = nil, original.Group{}, 0
	g.party = world.State{}
	g.mazeState = maze.State{}
	g.tombs, g.clanRewarded = nil, false
	g.warnings, g.saveMsg, g.overlay = nil, "", ""
	g.bossFight = false
	g.openMainMenu()
}

// partySummary 列出隊伍編號 + 成員名 + 等級 + 位置(docs/spec/15 §5 的表)。
func (g *Game) partySummary(grp original.Group) string {
	var names []string
	for _, id := range grp.MemberIDs() {
		if id >= 1 && id <= len(g.chars) && g.chars[id-1].Occupied() {
			c := g.chars[id-1]
			names = append(names, fmt.Sprintf("%s(Lv%d)", c.Name, c.Level))
		}
	}
	if len(names) == 0 {
		names = []string{"（沒有成員）"}
	}
	return fmt.Sprintf("%s　位置(%d,%d)", strings.Join(names, "、"), grp.WorldX, grp.WorldY)
}

// shellUpdate 處理外殼接管畫面時的按鍵。只在 g.shell.mode != shellPlaying
// 時被 Update() 呼叫。
func (g *Game) shellUpdate() error {
	switch g.shell.mode {
	case shellTitle: // A1:任意鍵進主選單(docs/spec/15 §3)
		if len(g.pressedKeys()) > 0 {
			g.openMainMenu()
		}

	case shellMainMenu: // A2(docs/spec/15 §4)
		for _, k := range g.pressedKeys() {
			switch k {
			case ebiten.KeyL: // L)oad a Party —— 開局的唯一入口
				if err := g.openPartySelect(); err != nil {
					g.shell.msg = "讀取隊伍失敗:" + err.Error()
				}
			case ebiten.KeyC: // C)har Utilities —— 沿用既有名冊(docs/spec/11 §5)
				g.openRoster()
			case ebiten.KeyP: // P)rogram Notes —— 這裡改成按鍵表(docs/spec/15 §1.1)
				g.overlay = shellKeyHelp
			case ebiten.KeyQ: // Q)uit the Game
				return ebiten.Termination
			}
			break
		}

	case shellPartySelect: // A3(docs/spec/15 §5)
		for _, k := range g.pressedKeys() {
			if k == ebiten.KeyEscape {
				g.openMainMenu()
				break
			}
			if n, ok := digitKey(k); ok {
				g.selectParty(n)
				break
			}
		}

	case shellWipe, shellEnding: // A4/A5:任意鍵回主選單(docs/spec/15 §6/§7)
		if len(g.pressedKeys()) > 0 {
			g.returnToMainMenu()
		}
	}
	return nil
}

// digitKey 把 1–5 的數字鍵翻成整數。逐一列舉,不靠鍵值範圍相減 ——
// ebiten.Key 的常數順序不是本專案的契約。
func digitKey(k ebiten.Key) (int, bool) {
	switch k {
	case ebiten.KeyDigit1:
		return 1, true
	case ebiten.KeyDigit2:
		return 2, true
	case ebiten.KeyDigit3:
		return 3, true
	case ebiten.KeyDigit4:
		return 4, true
	case ebiten.KeyDigit5:
		return 5, true
	}
	return 0, false
}

// shellKeyHelp 是 A6 按鍵表(docs/spec/15 §8)的內容,經既有的敘述覆蓋層
// 機制(g.overlay,見 maze_prompt.go 的 drawOverlay)顯示。
//
// ⚠ **不能用 "\n" 分段** —— drawOverlay 用 ui.Wrap 依欄寬自動折行,
// Wrap 不認得換行字元,會把它當成一個寬度為 1 的普通字元夾在行中間。
// 所以這裡寫成一段連續的敘述,交給 Wrap 自動折。
//
// 手冊 p.51 的 OUTSIDE OPTIONS 與 DOS 版不符(docs/re/139 §4 實跑),
// 這裡照引擎實際行為寫,不照手冊抄。
const shellKeyHelp = "按鍵表：" +
	"主選單　L 載入隊伍、C 角色管理、P 本頁、Q 離開；" +
	"世界地圖／迷宮　方向鍵先轉再走、N 開名冊；" +
	"戰鬥　方向鍵移動或轉身、A 攻擊、C 施法、Enter 結束回合；" +
	"施法選格　I/J/K/M 移動游標、空白鍵施放、ESC 取消；" +
	"S 存檔(世界地圖／營地)。" +
	"（按任意鍵關閉）"

// endingText 是引擎暫用的結局文字。
//
// ⚠ **原版的結局文字沒有盤到**(docs/spec/15 §7):USERLIB 有結局匯出槽
// (docs/re/66),但文字本身沒盤到。這不是原版台詞,只是佔位,
// 標明「未盤到」不要假裝是原版的。
const endingText = "希瑞雅妮（Siriadne）已經倒下,春之碎片重歸完整。"

// drawShell 依 g.shell.mode 畫外殼接管的整個畫布。
func (g *Game) drawShell(dst *ebiten.Image) {
	if g.shell == nil {
		return
	}
	switch g.shell.mode {
	case shellTitle:
		g.drawTitle(dst)
	case shellMainMenu:
		g.drawMainMenu(dst)
	case shellPartySelect:
		g.drawPartySelect(dst)
	case shellWipe:
		g.drawWipe(dst)
	case shellEnding:
		g.drawEnding(dst)
	}
}

// drawTitle 畫 A1 標題畫面。docs/spec/15 §3。
//
// ⚠ STARTUP.BIN(疑似整頁顯示緩衝,16384+8 bytes)還沒有轉出來——
// cmd/convert 目前只轉圖塊與 PICT。用純文字標題,不拿別的圖片冒充
// (同 cmd/convert 對缺圖的處理:用佔位符,不要偷偷補圖)。
//
// ⚠ 音樂:規格說「music.Title 若存在;沒有就靜音」——internal/music
// 只有 Ending 與 Userlib 兩份譜(docs/spec/13),沒有 Title,所以這裡
// 照規格的另一半處理:靜音,不要編一份假的標題曲。
func (g *Game) drawTitle(dst *ebiten.Image) {
	cx := float64(layout.ScreenW) / 2
	y := float64(layout.ScreenH)/2 - 100
	if g.titleFont != nil {
		drawCenter(g.titleFont, dst, "春之石", cx, y)
		y += g.titleFont.LineHeight() * 1.3
		drawCenter(g.titleFont, dst, "Shard of Spring", cx, y)
		y += g.titleFont.LineHeight() * 2
	}
	if g.panel != nil {
		drawCenter(g.panel, dst, "SSI 1986／Digital Illusions MS-DOS 移植／繁體中文 remake", cx, y)
		y += g.panel.LineHeight() * 3
		drawCenter(g.panel, dst, "── 按任意鍵繼續 ──", cx, y)
	}
}

// drawMainMenu 畫 A2 主選單。docs/spec/15 §4:原版六項只做四項——
// R)estore Mazes 與 I)nstall Game 不做(§1.1)。
func (g *Game) drawMainMenu(dst *ebiten.Image) {
	p := g.panel
	if p == nil {
		return
	}
	strokeFrame(dst, layout.View)
	strokeFrame(dst, layout.Prompt)

	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)
	line := func(s string) { p.Draw(dst, s, x, y); y += lh }

	line("主選單")
	y += lh * 0.5
	line("L) 載入隊伍(Load a Party)── 進入遊戲")
	line("C) 角色管理(Char Utilities)── 造角色、組隊")
	line("P) 按鍵表(Program Notes)")
	line("Q) 離開程式(Quit the Game)")
	if g.shell.msg != "" {
		y += lh * 0.5
		line("⚠ " + g.shell.msg)
	}

	py := float64(layout.Prompt.Y + ui.PanelPad)
	p.Draw(dst, "L 載入隊伍　C 角色管理　P 按鍵表　Q 離開",
		float64(layout.Prompt.X+ui.PanelPad), py)
}

// drawPartySelect 畫 A3 隊伍選擇。docs/spec/15 §5 的表:
// 有資料顯示隊伍編號+成員名+等級+位置,未初始化顯示「（空）」且選不下去。
func (g *Game) drawPartySelect(dst *ebiten.Image) {
	p := g.panel
	if p == nil {
		return
	}
	strokeFrame(dst, layout.View)
	strokeFrame(dst, layout.Prompt)

	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)
	line := func(s string) { p.Draw(dst, s, x, y); y += lh }

	line("隊伍選擇")
	y += lh * 0.5
	for i, grp := range g.shell.slots {
		n := i + 1
		if grp.Blank() {
			line(fmt.Sprintf("%d)（空）", n))
			continue
		}
		line(fmt.Sprintf("%d) %s", n, g.partySummary(grp)))
	}
	if g.shell.msg != "" {
		y += lh * 0.5
		line("⚠ " + g.shell.msg)
	}

	py := float64(layout.Prompt.Y + ui.PanelPad)
	p.Draw(dst, "1–5 選隊伍　ESC 回主選單", float64(layout.Prompt.X+ui.PanelPad), py)
}

// drawWipe 畫 A4 全滅畫面。docs/spec/15 §6。
//
// 死亡曲(music.Userlib)已經在 endTurn() 判定 PartyDead 的那一刻放過
// (combat_scene.go)——這裡不重放。
func (g *Game) drawWipe(dst *ebiten.Image) {
	if g.panel == nil {
		return
	}
	cx := float64(layout.ScreenW) / 2
	y := float64(layout.ScreenH)/2 - 40
	drawCenter(g.panel, dst, "全隊陣亡", cx, y)
	y += g.panel.LineHeight() * 2
	drawCenter(g.panel, dst, "── 按任意鍵回主選單 ──", cx, y)
}

// drawEnding 畫 A5 結局畫面。docs/spec/15 §7。
func (g *Game) drawEnding(dst *ebiten.Image) {
	if g.panel == nil {
		return
	}
	cx := float64(layout.ScreenW) / 2
	y := float64(layout.Margin * 2)
	drawCenter(g.panel, dst, "結局", cx, y)
	y += g.panel.LineHeight() * 2
	for _, ln := range ui.Wrap(endingText, 50) {
		drawCenter(g.panel, dst, ln, cx, y)
		y += g.panel.LineHeight()
	}
	y += g.panel.LineHeight()
	drawCenter(g.panel, dst, "⚠ 原版結局文字未盤到,以上是引擎暫用文字", cx, y)
	y += g.panel.LineHeight() * 2
	drawCenter(g.panel, dst, "── 按任意鍵回主選單 ──", cx, y)
}

// drawCenter 把文字置中畫在 (cx, y)。render.Painter 只有左對齊/右對齊,
// 置中靠 Advance() 自己算。
func drawCenter(p *render.Painter, dst *ebiten.Image, s string, cx, y float64) {
	p.Draw(dst, s, cx-p.Advance(s)/2, y)
}

// strokeFrame 畫版面區塊的外框,與 main.go Draw() 裡的同名邏輯一致。
func strokeFrame(dst *ebiten.Image, rc layout.Rect) {
	vector.StrokeRect(dst, float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H),
		2, cgaWhite, false)
}
