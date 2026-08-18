package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/layout"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
	"shardofspring/internal/town"
	"shardofspring/internal/ui"
)

// 技能點分配畫面。docs/spec/20-skill-allocation.md。
//
// 兩個入口共用同一段程式碼:創角完成之後(create_scene.go 的
// finishCreate)、升級成功之後(town_scene.go 的 trainMember)。
// 版面照 cast_scene.go 的 drawCastMenu——蓋掉主視野,不另開視窗。
//
// 輸入方式照 camp_actions.go 的投入點數輸入(campCastKey 階段 2的做法):
// 打數字、Enter 確認,不是按一下字母就選定——技能編號到 10,單一按鍵放不下
// 兩位數,而 0 另外保留給「離開」(docs/spec/20 §2)。

// skillAllocState 是技能點分配畫面的狀態。非 nil = 畫面開著,蓋在目前的
// 主畫面之上(創角時是名冊,升級時是城鎮的訓練所)。
type skillAllocState struct {
	charID int // g.chars 的 1-based ID——town.LearnSkill 動的是這個角色

	// memberIdx >= 0 時,學到的技能要同時同步進 g.members[memberIdx]
	// (升級入口:這個角色目前在隊伍裡,戰場/營地讀的是 g.members 那一份)。
	// -1 = 不適用(創角入口:角色剛建立,還沒加進任何隊伍)。
	memberIdx int

	input string // 數字輸入緩衝(docs/spec/20 §2 的技能編號)
	// page 是目前這一頁(每頁 SkillPageSize 項)。原版一頁 5 項、`SPACE` 翻頁,
	// 而**一頁 5 項正是「單鍵就能選」的理由** —— 十項擠一頁的話 10 要按兩個鍵。
	page int
	msg  string

	onDone func(g *Game) // 按 0 離開之後要接著做的事;nil = 沒有
}

// openSkillAlloc 開技能點分配畫面。charID 是 g.chars 的 1-based ID。
func (g *Game) openSkillAlloc(charID, memberIdx int, onDone func(g *Game)) {
	g.skillAlloc = &skillAllocState{charID: charID, memberIdx: memberIdx}
	g.skillAlloc.onDone = onDone
}

// skillAllocChar 回傳目前這個角色的目前值——與 town.LearnSkill 共用同一份
// 資料來源(g.chars),不另外維護一份複本。
func (g *Game) skillAllocChar() (original.Character, bool) {
	a := g.skillAlloc
	if a == nil || a.charID < 1 || a.charID > len(g.chars) {
		return original.Character{}, false
	}
	return g.chars[a.charID-1], true
}

// applySkillAllocChar 把學技能之後的結果寫回 g.chars;升級入口
// (memberIdx >= 0)還要同步 g.members——g.members 是 g.chars 的**複本**、
// 不是指標(main.go syncMember 的說明),漏了這一步的話,離開分配畫面
// 之後戰場/營地看到的技能表還是升級前那份。
func (g *Game) applySkillAllocChar(c original.Character) {
	a := g.skillAlloc
	if a == nil {
		return
	}
	if a.memberIdx >= 0 && a.memberIdx < len(g.members) {
		g.members[a.memberIdx] = c
	}
	g.syncMember(c)
}

// skillAllocKey 處理技能點分配畫面的一次按鍵。
func (g *Game) skillAllocKey(k ebiten.Key) {
	a := g.skillAlloc
	if a == nil {
		return
	}
	switch {
	// ⚠ **單鍵選取**:原版一頁 5 項,按 `1`–`5` 直接選那一項,不必按 Enter
	// (右欄寫著 `Enter: # to choose (or remove)`,而畫面上的編號是頁內編號)。
	case k >= ebiten.KeyDigit1 && k <= ebiten.KeyDigit5:
		g.pickSkill(a.page*town.SkillPageSize + int(k-ebiten.KeyDigit1) + 1)
	case k == ebiten.KeyDigit0, k == ebiten.KeyKP0:
		g.closeSkillAlloc()
	// `SPACE` 翻頁(原版 `SPACE for next page`)。+/- 一併收,兩邊都順手。
	case k == ebiten.KeySpace, k == ebiten.KeyEqual, k == ebiten.KeyKPAdd:
		a.page = (a.page + 1) % g.skillPages()
	case k == ebiten.KeyMinus, k == ebiten.KeyKPSubtract:
		a.page = (a.page + g.skillPages() - 1) % g.skillPages()
	case k == ebiten.KeyEscape:
		g.closeSkillAlloc()
	}
}

// pickSkill 處理「選了第 n 項技能」:交給 town.LearnSkill 判(學會 / 取消 / 不足)。
//
// ⚠ **0 隨時能離開,不強迫花完**——SkillPts 存的就是「剩餘」,而且升級會
// 累積(docs/spec/20 §2)。
func (g *Game) pickSkill(n int) {
	a := g.skillAlloc
	c, ok := g.skillAllocChar()
	if !ok {
		g.skillAlloc = nil
		return
	}
	class := rules.Class(c.Class)
	name, hasName := town.SkillName(class, n)
	r := town.LearnSkill(&c, n)
	if r == town.LearnOK {
		g.applySkillAllocChar(c)
		a.msg = fmt.Sprintf("學會了「%s」。", name)
		return
	}
	// 取消也要寫回去 —— 點數退了、旗標清了,不寫回等於白按。
	if r == town.LearnRemoved {
		g.applySkillAllocChar(c)
		a.msg = fmt.Sprintf("取消了「%s」,點數退回。", name)
		return
	}
	if !hasName {
		name = fmt.Sprintf("編號 %d", n)
	}
	a.msg = name + "：" + r.String()
}

// closeSkillAlloc 關閉畫面:寫回名冊並存檔(docs/spec/20 §2:離開之後
// 要把角色寫回名冊,否則下次載入時點數與技能都回到舊值),再跑呼叫端
// 接續的事(創角入口:回名冊訊息;升級入口 onDone 是 nil——trainMember
// 的升級訊息在打開這個畫面之前就已經設定好了)。
func (g *Game) closeSkillAlloc() {
	a := g.skillAlloc
	g.skillAlloc = nil
	if err := g.writeChars(); err != nil {
		g.warnings = append(g.warnings, "名冊存檔失敗:"+err.Error())
	}
	if a != nil && a.onDone != nil {
		a.onDone(g)
	}
}

// drawSkillAlloc 畫技能點分配畫面。蓋掉主視野(layout.View),版面照
// docs/spec/20 §3。
func (g *Game) drawSkillAlloc(dst *ebiten.Image) {
	a, p := g.skillAlloc, g.panel
	if a == nil || p == nil {
		return
	}
	c, ok := g.skillAllocChar()
	if !ok {
		return
	}
	// 蓋掉底下的畫面(名冊或城鎮的訓練所)——兩層字疊在一起是「看得到但
	// 讀不出來」,照 cast_scene.go 的 drawCastMenu 同一招。
	rc := layout.View
	vector.DrawFilledRect(dst, float32(rc.X+2), float32(rc.Y+2),
		float32(rc.W-4), float32(rc.H-4), cgaBlack, false)

	lh := p.LineHeight()
	x := float64(layout.View.X + ui.PanelPad)
	y := float64(layout.View.Y + ui.PanelPad)
	line := func(s string) {
		p.Draw(dst, s, x, y)
		y += lh
	}

	// 原版有**兩份**這句話,因為技能點畫面有兩個入口:
	// 創角走 `CHARUTIL`(第 30/32 列 `Skills left:`)、升級走 `TOWN`
	// (第 49/50 列 `You have N points left.`)。引擎的畫面是同一個,
	// 所以照**進入點**挑措辭 —— memberIdx < 0 就是創角那一條路
	// (finishCreate 傳 −1,trainMember 傳隊員索引)。
	if a.memberIdx < 0 {
		line(fmt.Sprintf("%s　剩餘技能點：%d", c.Name, c.SkillPts))
	} else {
		line(fmt.Sprintf("%s　你還剩 %d 點可分配。", c.Name, c.SkillPts))
	}
	y += lh * 0.5

	// ⚠ 一頁 5 項(town.SkillPageSize),照原版 —— 而**一頁 5 項正是
	// 「按一個鍵就選得到」的理由**:編號是頁內的 1–5。
	class := rules.Class(c.Class)
	lo := a.page * town.SkillPageSize
	for i := 0; i < town.SkillPageSize; i++ {
		n := lo + i + 1
		name, has := town.SkillName(class, n)
		if !has {
			continue
		}
		cost, _ := town.SkillCost(class, n)
		mark := " "
		if n <= len(c.Skills) && c.Skills[n-1] == '1' {
			mark = "*" // 原版學過的那一項在名字前面加 `*`
		}
		line(fmt.Sprintf("%d) %s%s (%d)", i+1, mark, ui.PadTo(name, 8), cost))
	}
	if np := g.skillPages(); np > 1 {
		line(fmt.Sprintf("第 %d／%d 頁　空白鍵翻頁", a.page+1, np))
	}
	line(" 0) 結束")
	y += lh * 0.5
	// 兩個入口兩份措辭:創角走 CHARUTIL、升級走 TOWN:51/52
	// 「Enter skill,」+「(0 exits) 」(F3)。
	if a.memberIdx < 0 {
		line("選一項技能,(0離開)")
	} else {
		line("輸入技能,(0離開)")
	}
	// ⚠ 原版右欄寫著 `Enter: # to choose (or remove)` —— **同一個編號按第二次
	// 就取消並退點**。不寫出來的話玩家不會知道自己點錯了還有救。
	line("(標 * 的已經學過,再按一次會取消並退點)")
	if a.msg != "" {
		y += lh * 0.5
		line(a.msg)
	}
}

// skillPages 回傳這個職業的技能要分成幾頁。至少 1,免得取模除以零。
func (g *Game) skillPages() int {
	c, ok := g.skillAllocChar()
	if !ok {
		return 1
	}
	n := 0
	for i := 1; i <= 10; i++ {
		if _, has := town.SkillName(rules.Class(c.Class), i); has {
			n = i
		}
	}
	if n <= 0 {
		return 1
	}
	return (n + town.SkillPageSize - 1) / town.SkillPageSize
}
