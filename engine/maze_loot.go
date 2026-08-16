package main

import (
	"shardofspring/internal/original"
	"shardofspring/internal/town"
)

// 迷宮的定點道具(docs/re/202)。與戰鬥掉落是**兩支不同的常式**,
// 掃背包的方向也相反,兩邊各自照抄(docs/re/168 §1)。

// 訊息字面照 `translations/module-text/MAZEMOVE.tsv`。
const (
	mazeFoundItem = "找到了一件道具。"                     // 84 `found an item.`
	mazePackFull  = "隊伍的道具已經滿了,請在營地丟棄一些。" // 85
)

// giveMazeItem 把一件道具塞給隊伍,回傳要顯示的那一句。
//
// ⚠ **由隊尾往隊首找**(`MAZEMOVE 0x13946` 的 `dec` 迴圈,docs/re/168 §1)——
// 與 `T)rade`／`E)quip` 的由前往後不同,也與戰鬥掉落的由前往後不同。
// ⚠ 撿來的東西標成**未鑑定**(docs/re/168 §2)。
func (g *Game) giveMazeItem(item int) string {
	for i := len(g.members) - 1; i >= 0; i-- {
		for slot := 0; slot < original.PackSlots; slot++ {
			if g.members[i].Pack[slot] != original.NotEquipped {
				continue
			}
			g.members[i].Pack[slot] = item
			town.SetIdentified(&g.members[i], slot, false)
			g.syncMember(g.members[i])
			return g.members[i].Name + " " + mazeFoundItem
		}
	}
	return mazePackFull
}
