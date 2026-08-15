package main

import (
	"encoding/json"
	"fmt"
	"os"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
)

// 經驗值的存放。docs/re/140 §9。
//
// ⚠ **經驗值在 `CHARS.DAT` 裡的位移未解。** 94 bytes 中沒定語意的只剩
// 13、52–53、86–94,而 86–94 在出貨資料裡前三個仍是 `0x20`(從未被寫過),
// docs/re/89 的 39 個位移聯集也沒有一個落在那裡。
//
// 所以本引擎把經驗值放在**自己的旁掛檔** `<assets>/save/exp.json`,
// 不動原版記錄的任何一個 byte。定位出來之後再搬進去 ——
// 那時舊存檔仍然讀得回來,因為旁掛檔是額外的、不是替代的。
//
// ⛔ 不要「先找個空位寫下去」。寫錯位移不會報錯:原版仍然開得起來,
// 只是某個欄位悄悄變了值,而我們的往返測試只比對自己寫的那幾格。

// loadExp 讀旁掛的經驗值檔。檔案不存在就當全部是 0(不是錯誤)。
func (g *Game) loadExp() {
	b, err := os.ReadFile(g.expPath)
	if err != nil {
		return
	}
	var m map[string]int
	if err := json.Unmarshal(b, &m); err != nil {
		g.warnings = append(g.warnings, "經驗值檔讀不動:"+err.Error())
		return
	}
	for k, v := range m {
		var id int
		if _, err := fmt.Sscanf(k, "%d", &id); err != nil {
			continue
		}
		if id >= 1 && id <= original.CharSlots {
			g.exp[id-1] = v
		}
	}
}

// saveExp 寫回旁掛的經驗值檔。
func (g *Game) saveExp() error {
	if g.expPath == "" {
		return nil
	}
	m := make(map[string]int, original.CharSlots)
	for i, v := range g.exp {
		if v > 0 {
			m[fmt.Sprint(i+1)] = v
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(g.expPath, b, 0o644)
}

// charExp 回傳目前隊伍中第 i 位成員的經驗值。
//
// ⚠ 成員在名冊裡的槽號要用 `ID` 而不是隊伍裡的順序 ——
// 隊伍順序會因為離隊而改變,經驗值不會跟著搬。
func (g *Game) charExp(c original.Character) int {
	if c.ID >= 1 && c.ID <= original.CharSlots {
		return g.exp[c.ID-1]
	}
	return 0
}

// awardExp 把一場戰鬥的經驗值分給生還者。docs/spec/07。
//
// 回傳分給每個人的數量與說明文字。
func (g *Game) awardExp(units []combat.Unit) (int, string) {
	total := combat.TotalExp(units)
	if total == 0 {
		return 0, ""
	}
	var alive []int // 生還者在名冊裡的槽號
	for _, c := range g.members {
		if c.HP > 0 && c.ID >= 1 && c.ID <= original.CharSlots {
			alive = append(alive, c.ID)
		}
	}
	share := combat.ExpShare(total, len(alive))
	for _, id := range alive {
		g.exp[id-1] += share
	}
	if err := g.saveExp(); err != nil {
		g.warnings = append(g.warnings, "經驗值存檔失敗:"+err.Error())
	}
	return share, fmt.Sprintf("獲得經驗 %d,每人 %d(%s)",
		total, share, combat.ExpSplitAssumption)
}
