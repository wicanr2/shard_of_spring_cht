package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
	"shardofspring/internal/ui"
)

// 另存新檔。docs/spec/18-save-format.md §2:「多存檔 = 多個檔」,但原本
// main.go 的 S 鍵只會寫回 effectiveSaveName() 目前那一份 —— 玩家沒有任何
// 介面可以另外取名、開一份新的存檔,「多存檔」在玩家那一側等於不存在。
// 這裡補一個文字輸入畫面(g.saveAs 非 nil = 開著),蓋在遊戲畫面之上,
// 玩家打完名字按 Enter 就切到那個名字繼續存(main.go 的 effectiveSaveName)。
//
// ⚠ 輸入迴圈**故意不沿用** create_scene.go 的 createRunes/createKey 那一套
// (原始的 inpututil.AppendJustPressedKeys/ebiten.AppendInputChars)——那一套
// 靠 ebiten.RunGame 的事件迴圈填,沒有真正視窗系統的環境永遠讀不到任何輸入
// (main.go 的 g.testKeys 說明),而 docs/spec/15 §9 要求每個新畫面都要有
// 不開視窗的測試。這裡改用 g.pressedKeys()/g.inputRunes(),兩者在正式執行
// (g.testKeys 恆為 nil)時與原始路徑完全等價。

// saveNameMaxRunes 是另存新檔的檔名上限。⚠ 這是本引擎自己的選擇,不是 RE
// 結論 —— 原版沒有多存檔,自然也沒有檔名長度可抄。20 字對繁中檔名夠用,
// 也不會把 saves/ 目錄撐得難以瀏覽。
const saveNameMaxRunes = 20

// saveAsState 是另存新檔畫面的狀態。
type saveAsState struct {
	name string
	msg  string
}

// openSaveAs 開啟另存新檔畫面。
//
// ⚠ **故意不預填目前的存檔名**——預填的話,玩家沒注意到就按 Enter 會直接
// 覆寫原本那份存檔,「另存」的「另」字就沒了。要覆寫現有存檔,打一樣的
// 名字進去就是了,這裡不代勞。
func (g *Game) openSaveAs() {
	g.saveAs = &saveAsState{}
}

// saveAsRunes 收玩家打的字元。
func (g *Game) saveAsRunes(rs []rune) {
	s := g.saveAs
	if s == nil {
		return
	}
	for _, r := range rs {
		if r < ' ' {
			continue
		}
		if len([]rune(s.name)) >= saveNameMaxRunes {
			return
		}
		s.name += string(r)
	}
}

// saveAsKey 處理另存新檔畫面的按鍵。
func (g *Game) saveAsKey(k ebiten.Key) {
	s := g.saveAs
	if s == nil {
		return
	}
	switch k {
	case ebiten.KeyBackspace:
		if s.name != "" {
			r := []rune(s.name)
			s.name = string(r[:len(r)-1])
		}
	case ebiten.KeyEscape:
		g.saveAs = nil
	case ebiten.KeyEnter, ebiten.KeyKPEnter:
		g.finishSaveAs()
	}
}

// validSaveName 只擋檔案系統會出事的字元(路徑分隔號、Windows 保留字元、
// "."/".." 這兩個特殊名稱)以及空白名稱——存檔名最終會被 save.Path 接成
// "<dir>/<name>.json",不擋的話玩家打個 "../x" 就能寫到 saves/ 之外。
func validSaveName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\\:*?\"<>|")
}

// finishSaveAs 驗證檔名並存檔(切到這個名字之後呼叫既有的 g.save())。
func (g *Game) finishSaveAs() {
	s := g.saveAs
	name := strings.TrimSpace(s.name)
	if !validSaveName(name) {
		s.msg = "名稱不能是空白,也不能包含 / \\ : * ? \" < > |"
		return
	}
	g.saveName = name
	// docs/re/149:存檔本身也推進時鐘一格,與 main.go 既有的 S 鍵一致 ——
	// 這裡不是新規則,只是「另存」也要遵守同一條,不能因為改了個名字
	// 就漏掉。
	g.party.Tick()
	if err := g.save(); err != nil {
		g.saveMsg = "存檔失敗:" + err.Error()
	} else {
		g.saveMsg = fmt.Sprintf("已另存新檔:%s", name)
	}
	g.saveAs = nil
}

// drawSaveAs 畫另存新檔的文字輸入,蓋在世界地圖/迷宮之上。
func (g *Game) drawSaveAs(dst *ebiten.Image) {
	s, p := g.saveAs, g.panel
	if s == nil || p == nil {
		return
	}
	g.drawPanelFrame(dst, layout.View)
	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)
	line := func(str string) { p.Draw(dst, str, x, y); y += lh }

	line("另存新檔")
	y += lh * 0.5
	line(fmt.Sprintf("名稱(最多 %d 字,不含 / \\ : * ? \" < > |):", saveNameMaxRunes))
	line(s.name + "_")
	y += lh * 0.3
	line("Enter 確定　ESC 取消")
	if s.msg != "" {
		y += lh * 0.5
		line("⚠ " + s.msg)
	}
}
