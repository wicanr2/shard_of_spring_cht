package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
	"shardofspring/internal/ui"
)

// 戰場上的單位檢視面板。字面照 `translations/module-text/CMBT.tsv` 第 179–192 列:
//
//	Status:   Speed:    Skill:    Strength:   Magical:   Armor rating:
//	Attacks with:    Seeks>    YES / no    (ESC, ↑↓ scrolls)
//
// ⚠ **只顯示,不改任何狀態** —— 這一整支是唯讀的。
//
// ⚠ **開這個面板的按鍵是引擎自己挑的。** `CMBT` 的指令鏈裡有 `?` 與 `/`
// (docs/re/70),但**哪一個對到這個面板沒有讀到** —— 這裡用 `?`,
// 並在提示列寫出來。⛔ 不要把這個選擇寫成「原版就是這個鍵」。

type inspectState struct {
	idx int // 目前看的是第幾個單位(field.Units 的索引)
}

// inspectFields 是面板上的欄位標籤,順序照原版字串在表裡的排法。
const (
	inspectStatus = "狀態： "     // 179
	inspectYes    = "是"          // 181
	inspectNo     = "否"          // 182
	inspectSpeed  = "速度： "     // 183
	inspectSkill  = "技巧： "     // 184
	inspectStr    = "力量： "     // 185
	inspectMagic  = "魔法： "     // 186
	inspectArmor  = "護甲等級："  // 187
	inspectWeapon = "攻擊方式："  // 188
	inspectNone   = "無"          // 9 `None`
	inspectScroll = "(ESC,捲動)" // 191 + 192
)

// openInspect 打開檢視面板,停在第一個場上的單位。回 false 表示沒東西可看。
func (g *Game) openInspect() bool {
	if g.field == nil {
		return false
	}
	for i := range g.field.Units {
		if g.field.Units[i].Name != "" {
			g.inspect = &inspectState{idx: i}
			return true
		}
	}
	return false
}

// inspectStep 往前/往後換一個單位。**跳過沒有名字的空槽** ——
// 空槽在畫面上是一整頁空白,看起來像壞掉。
func (g *Game) inspectStep(d int) {
	st, f := g.inspect, g.field
	if st == nil || f == nil {
		return
	}
	n := len(f.Units)
	for k := 1; k <= n; k++ {
		j := ((st.idx+d*k)%n + n) % n
		if f.Units[j].Name != "" {
			st.idx = j
			return
		}
	}
}

// inspectKey 處理檢視面板的按鍵。回 true 表示這一格被吃掉了。
func (g *Game) inspectKey(k ebiten.Key) bool {
	switch k {
	case ebiten.KeyEscape:
		g.inspect = nil
	case ebiten.KeyUp, ebiten.KeyI:
		g.inspectStep(-1)
	case ebiten.KeyDown, ebiten.KeyM:
		g.inspectStep(1)
	default:
		return false
	}
	return true
}

// inspectLines 是面板上的每一行。
func (g *Game) inspectLines() []string {
	st, f := g.inspect, g.field
	if st == nil || f == nil || st.idx < 0 || st.idx >= len(f.Units) {
		return nil
	}
	u := f.Units[st.idx]
	status := u.StatusText()
	// 防護 = 防具的欄4 + 護甲技能(傷害公式減的就是這兩項,docs/spec/01 §5)。
	armor := f.ArmorRating(st.idx)
	out := []string{
		fmt.Sprintf("%s　%s%s", u.Name, inspectStatus, status),
		fmt.Sprintf("%s%-4d%s%-4d", inspectSpeed, u.Speed, inspectSkill, u.ToHit),
		fmt.Sprintf("%s%-4d%s%s", inspectStr, u.Str, inspectMagic, yesNo(u.SP > 0)),
		fmt.Sprintf("%s%d", inspectArmor, armor),
		inspectWeapon + f.WeaponName(st.idx),
	}
	// 鎖定對象只在隊上有人會「策略」時才看得到(手冊 p.35,docs/re/186 §2)。
	if line := g.tacticsLine(u); line != "" {
		out = append(out, line)
	}
	return append(out, inspectScroll)
}

func yesNo(b bool) string {
	if b {
		return inspectYes
	}
	return inspectNo
}

// drawInspect 把面板畫在訊息列那一塊(與施法/道具選單同一塊)。
func (g *Game) drawInspect(dst *ebiten.Image) {
	if g.inspect == nil || g.panel == nil {
		return
	}
	clearMessage(dst) // message.go
	p := g.panel
	lh := p.LineHeight()
	x := float64(layout.Message.X + ui.PanelPad)
	y := float64(layout.Message.Y + ui.PanelPad)
	for _, ln := range g.inspectLines() {
		p.Draw(dst, ln, x, y)
		y += lh
	}
}
