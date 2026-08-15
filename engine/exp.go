package main

import (
	"fmt"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
)

// 經驗值。**存在原版記錄裡**:`CHARS.DAT` 位移 90–93,MBF 單精度。
// docs/re/150、docs/formats/01。
//
// 五處程式碼、三支模組都讀這個位移,其中兩處緊接在載入 `Exp.  ` 標籤之後 ——
// 是**讀端**給出的答案,不是從資料側猜的。
//
// ⚠ 本引擎一度把經驗值放在自己的旁掛檔 `save/exp.json`,理由是「位移未解、
// 不要亂寫進原版記錄」。那個理由在位移解出來之後就不成立了 ——
// 留著旁掛檔會讓**同一個量有兩個真相**,而兩邊不同步時沒有任何一邊會報錯。

// charExp 回傳這個角色的經驗值(取整數,畫面只印整數)。
func (g *Game) charExp(c original.Character) int { return int(c.Exp) }

// awardExp 把一場戰鬥的經驗值分給有資格的成員。
//
// 資格是原版結算迴圈的兩個條件 `and` 起來(docs/re/150 §2):
//
//	當前生命值 > 0   且   狀態 < 5(未陣亡)
//
// ⚠ **不是「還活著的人」而已** —— 中毒、束縛、凝滯、冰封的人照分,
// 只有狀態 5(死亡)不分。把它寫成 `HP > 0` 在出貨資料上看不出差別,
// 因為死人的 HP 也是 0;要到「HP 0 但狀態不是 5」那種狀態才會分岔。
//
// 回傳分給每個人的數量與說明文字。
func (g *Game) awardExp(units []combat.Unit) (int, string) {
	total := combat.TotalExp(units)
	if total == 0 {
		return 0, ""
	}
	var idx []int // 有資格的成員在 g.members 裡的索引
	for i, c := range g.members {
		if c.EarnsExp() {
			idx = append(idx, i)
		}
	}
	share := combat.ExpShare(total, len(idx))
	for _, i := range idx {
		g.members[i].Exp += float64(share)
		g.syncMember(g.members[i])
	}
	return share, fmt.Sprintf("獲得經驗 %d,每人 %d(%s)",
		total, share, combat.ExpSplitAssumption)
}
