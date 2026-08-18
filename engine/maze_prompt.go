package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/maze"
)

// 迷宮裡三個會問問題的機關。規則在 internal/maze(docs/re/155、162),
// 這裡只做「怎麼問、怎麼把答案交回去」。
//
// ⚠ 觸發它們的是**事件目標編號**(518 / 532 / 705,docs/re/161 §3),
// 不是座標也不是格值 —— 所以這裡不做任何座標判斷。

type promptKind int

const (
	promptGem    promptKind = iota // 目標 532:輸入四個寶石顏色
	promptPool                     // 目標 518:選一名隊員治療
	promptRiddle                   // 目標 705:輸入四個名字
)

// mazePrompt 是機關的互動狀態。非 nil = 問題正開著。
type mazePrompt struct {
	kind  promptKind
	head  string    // 題目那一行
	input string    // 目前累積的輸入
	names [4]string // 氏族謎題:已經收下的名字
	idx   int       // 氏族謎題:正在問第幾個(0–3)
	// msg 是**問句還開著**時要一起顯示的回饋(治療池治好一個人之後的那一句)。
	msg string
}

// openPrompt 依事件種類開一個問題;回 false 表示這個種類不問問題。
func (g *Game) openPrompt(t maze.Trigger) bool {
	switch t.Kind {
	case maze.KindGem:
		g.prompt = &mazePrompt{kind: promptGem,
			head: "輸入要碰觸的寶石(B,G,V,R)或按ESC"} // MAZEMOVE:35+36+37
	case maze.KindPool:
		// 先看池子還有沒有水 —— 原版在選人之前就擋掉(docs/re/155 §2.1)。
		if !maze.PoolAvailable(g.group.PoolUses) {
			// 原版的 `This pool is empty!`(MAZEMOVE:83)。房間敘述照樣留著。
			g.overlay = t.Text + "  這座池子已經枯竭!"
			return true
		}
		g.prompt = &mazePrompt{kind: promptPool,
			head: "要治療哪位隊員?(0離開)"} // MAZEMOVE:78
	case maze.KindRiddle:
		// 三態旗標(docs/re/162 §2)。⚠ 原版把它存在哪**未解**,
		// 所以本引擎只留在記憶體裡 —— 存檔不會記得你解過了。
		switch g.clanRiddle() {
		case maze.ClanRewarded:
			g.overlay = g.dungeonText(705, "Eldron 說:你已經不需要我的幫助了。")
			return true
		case maze.ClanUnmet:
			g.overlay = g.dungeonText(707, "去搜 Moonglow 氏族的墓,知道名字再回來。")
			return true
		}
		g.overlay = g.dungeonText(706, "準備好說出氏族的名字了嗎?")
		g.prompt = &mazePrompt{kind: promptRiddle, head: "這幾位兄弟是："} // MAZEMOVE:46
	default:
		return false
	}
	return true
}

// promptRunes 收字元輸入(氏族謎題要打名字)。
func (g *Game) promptRunes(rs []rune) {
	if g.prompt == nil || g.prompt.kind != promptRiddle {
		return
	}
	for _, r := range rs {
		if r >= ' ' && r < 127 && len(g.prompt.input) < 16 {
			g.prompt.input += string(r)
		}
	}
}

// promptKey 處理一次按鍵。
func (g *Game) promptKey(k ebiten.Key) {
	p := g.prompt
	if p == nil {
		return
	}
	if k == ebiten.KeyEscape {
		g.prompt = nil
		return
	}
	switch p.kind {
	case promptGem:
		g.gemKey(k)
	case promptPool:
		g.poolKey(k)
	case promptRiddle:
		g.riddleKey(k)
	}
}

// gemKey:B/G/V/R 四個顏色,滿四個才判一次(docs/re/155 §1)。
func (g *Game) gemKey(k ebiten.Key) {
	p := g.prompt
	var c string
	switch k {
	case ebiten.KeyB:
		c = "B"
	case ebiten.KeyG:
		c = "G"
	case ebiten.KeyV:
		c = "V"
	case ebiten.KeyR:
		c = "R"
	default:
		return
	}
	p.input += c
	if len(p.input) < len(maze.GemAnswer) {
		return
	}
	g.prompt = nil
	if !maze.GemSolved(p.input) {
		// 原版的訊息 545:去 Edrin's Keep 學那首歌。
		g.overlay = g.dungeonText(545,
			"什麼也沒有發生。")
		return
	}
	g.mazeState.Major = maze.GemDestMajor
	g.mazeState.Minor = maze.GemDestMinor
	g.mazeState.Facing = maze.Facing(maze.GemDestFacing)
	g.overlay = g.dungeonText(544, "你的身體變得虛幻,像鬼魂一樣穿了過去。")
}

// poolKey:數字選人,0 離開(docs/re/155 §2)。
func (g *Game) poolKey(k ebiten.Key) {
	if k == ebiten.KeyDigit0 || k == ebiten.KeyKP0 {
		g.prompt = nil
		return
	}
	i := int(k - ebiten.KeyDigit1)
	if i < 0 || i >= len(g.members) {
		return
	}
	m := &g.members[i]
	if !maze.PoolCanHeal(m.Status) {
		g.overlay = m.Name + "：這位角色在這裡沒辦法治療!" // MAZEMOVE:79
		g.prompt = nil
		return
	}
	// 治療量 = round(RND × 5 + 1),值域 1–6(docs/re/185 §2 表列 #5)。
	// ⚠ 不是 g.rand.Roll(maze.PoolRollFaces)(那是 1–5、均勻,少了四捨五入
	// 多出來的上界)。
	roll := maze.PoolRoll(g.rand)
	got := maze.PoolHeal(m.HP, m.MaxHP, roll)
	m.HP += got
	g.group.PoolUses++
	g.syncMember(*m)
	// ⚠ **問句留著繼續問** —— 原版治好一個人之後再問一次
	// `Which party member do you wish to heal? (0 exits)`,要按 0 才離開
	// (2026-08-18 實跑 `workplace/dosbox/shots/q3b-s5.png`)。
	// 先前治完就關掉,治第二個人得重走一次整段路。
	// MAZEMOVE:80+81+82「healed」+「 pts.」+「 is」。
	// ⚠ 英文的繫詞在中文這一句是**體貌標記**不是「是」:`is healed` = 「已治療」。
	g.prompt.msg = fmt.Sprintf("%s 已治療 %d 點。", m.Name, got)
}

// riddleKey:Enter 收下目前這一個名字,四個收滿才判(docs/re/162 §3)。
func (g *Game) riddleKey(k ebiten.Key) {
	p := g.prompt
	switch k {
	case ebiten.KeyBackspace:
		if p.input != "" {
			p.input = p.input[:len(p.input)-1]
		}
		return
	case ebiten.KeyEnter, ebiten.KeyKPEnter:
	default:
		return
	}
	// 原版比的是字串常數 MURTHIN / CERCION / …,全大寫。
	// ⚠ 原版有沒有自己轉大寫**沒有讀到**(docs/re/162 §5),
	// 這裡先轉,理由是不轉的話玩家幾乎不可能過。
	p.names[p.idx] = strings.ToUpper(strings.TrimSpace(p.input))
	p.input = ""
	p.idx++
	if p.idx < len(p.names) {
		return
	}
	g.prompt = nil
	if maze.ClanSolved(p.names) {
		g.clanRewarded = true
		// 那枚戒指是**風暴戒**(道具 29,`MAZEMOVE 0x13042`,docs/re/202)——
		// 先前只印那句話,東西沒有真的進背包。
		g.overlay = g.dungeonText(708, "Eldron 賜給你一枚戒指。") +
			"　" + g.giveMazeItem(maze.RiddleReward)
		return
	}
	g.overlay = g.dungeonText(709, "Eldron 說:再試一次吧。")
}

// clanRiddle 回傳 Eldron 謎題的進度旗標。
//
// ⚠ 旗標本身是原版讀出來的三態(docs/re/162 §2),但**怎麼從 0 走到 1**
// 是本引擎的選擇(maze.RiddleReadyOnAllTombs)。
func (g *Game) clanRiddle() int {
	if g.clanRewarded {
		return maze.ClanRewarded
	}
	if !maze.RiddleReadyOnAllTombs {
		return maze.ClanReady
	}
	for _, n := range maze.TombTargets {
		if !g.tombs[n] {
			return maze.ClanUnmet
		}
	}
	return maze.ClanReady
}

// dungeonText 取這一層的 DT 文字;沒有就用備援字串。
func (g *Game) dungeonText(n int, fallback string) string {
	if g.level != nil {
		if t, ok := g.level.text[n]; ok {
			return t
		}
	}
	return fallback
}

// promptLines 是畫面上要顯示的幾行。
func (p *mazePrompt) lines() []string {
	out := []string{p.head}
	switch p.kind {
	case promptRiddle:
		// MAZEMOVE:47–50 是四段各自獨立的字串(`#1 `…`#4 `),不是格式化出來的。
		slot := [4]string{"#1 ", "#2 ", "#3 ", "#4 "}
		for i := 0; i < p.idx && i < len(slot); i++ {
			out = append(out, slot[i]+p.names[i])
		}
		if p.idx < len(slot) {
			out = append(out, slot[p.idx]+p.input+"_")
		}
	case promptGem:
		out = append(out, p.input+strings.Repeat(".", len(maze.GemAnswer)-len(p.input)))
	}
	// 治療池治好一個人之後,回饋與問句**同時在畫面上** —— 問句沒有關掉。
	if p.msg != "" {
		out = append(out, "", p.msg)
	}
	return out
}
