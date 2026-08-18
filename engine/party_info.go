package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// `P)` 隊伍資訊畫面。原版在世界地圖按 `P` 開一個框:
//
//	Hour: 4  Day: 1
//	In the Month of Spirit
//	Visibility = 3
//	Player's Clock: …
//	[Press a key]
//
// (2026-08-18 對照原版量到的,docs/spec/14 §12-C。)
// **這是遊戲裡唯一看得到日、月與能見度數值的地方。**
//
// ⚠ 原版最後兩行是**現實世界**的時鐘與日期(`Player's Clock`)——
// 那是 1986 年的貼心設計(提醒玩家玩多久了)。引擎不做:
// 它與遊戲狀態無關,而且在視窗化的現代桌面上系統列就有。

// monthNames 是十二個月份名,讀自 `MENU.EXE` 檔案位移 9888 的 `DATA` 敘述:
//
//	 Spirit,Dragon,Rose,Sword,Unicorn,Metal,Lotus,Axe,Panther,Ice,Mandrake,Aurora
//
// ⚠ **只有 12 個,而時鐘的月份範圍是 1–21**(docs/formats/02 的進位表)。
// 第 13 個月之後原版印什麼**未查** —— 所以這裡回空字串,由呼叫端只印編號,
// ⛔ 不要拿 `(m-1)%12` 繞回去湊一個名字:那會產生一個看起來合理的假答案。
var monthNames = [12]string{
	"靈月(Spirit)", "龍月(Dragon)", "薔薇月(Rose)", "劍月(Sword)",
	"獨角獸月(Unicorn)", "金月(Metal)", "蓮月(Lotus)", "斧月(Axe)",
	"豹月(Panther)", "冰月(Ice)", "曼德拉草月(Mandrake)", "極光月(Aurora)",
}

// monthName 回傳月份名;超出 12 回空字串(見 monthNames 的說明)。
func monthName(m int) string {
	if m < 1 || m > len(monthNames) {
		return ""
	}
	return monthNames[m-1]
}

// partyInfoLines 組出 `P)` 那一頁的內容。
func (g *Game) partyInfoLines() []string {
	c := g.party.Clock
	month := monthName(c.Month)
	if month == "" {
		// ⚠ 沒有名字就只印編號 —— 原版第 13 個月印什麼未查。
		month = fmt.Sprintf("第 %d 月", c.Month)
	}
	return []string{
		fmt.Sprintf("第 %d 時　第 %d 日", c.Hour, c.Day),
		month,
		fmt.Sprintf("能見度 = %d", g.party.Visibility),
		"",
		"（按任意鍵）",
	}
}

// openPartyInfo 把資訊頁放進覆蓋層。⚠ 用既有的覆蓋層機制,
// 不新開一個場景 —— 它的行為(任意鍵關掉)正好就是原版的 `[Press a key]`。
func (g *Game) openPartyInfo() {
	g.overlay = joinLines(g.partyInfoLines())
}

// joinLines 用換行接起來。⚠ 覆蓋層自己會折行,這裡只保留刻意的斷行。
func joinLines(ls []string) string {
	out := ""
	for i, l := range ls {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

// partyInfoKey 是開資訊頁的按鍵。原版是 `P`。
var partyInfoKey = ebiten.KeyP
