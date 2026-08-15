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
// 資格由**戰場上的單位**決定,不是由名冊決定(docs/re/150 §2.1):
// 朝向 > 0(還在場上)且 狀態 < 5(未陣亡)。
//
// ⚠ **逃走的人拿不到。** 逃跑在原版是「把朝向清成 0」(docs/re/103),
// 而逃走的人**生命值仍然大於 0** —— 用生命值判斷會多發給他們,
// 而畫面上完全看不出來。
//
// 回傳分給每個人的數量與說明文字。
func (g *Game) awardExp(units []combat.Unit) (int, string) {
	total := combat.TotalExp(units)
	if total == 0 {
		return 0, ""
	}
	var idx []int // 有資格的成員在 g.members 裡的索引
	for u := combat.PartyBase; u < combat.PartyBase+combat.PartyMax; u++ {
		i := u - combat.PartyBase
		if u < len(units) && i < len(g.members) && units[u].EarnsExp() {
			idx = append(idx, i)
		}
	}
	share := combat.ExpShare(total, len(idx))
	for _, i := range idx {
		g.members[i].Exp += float64(share)
		g.syncMember(g.members[i])
	}
	return share, fmt.Sprintf("獲得經驗 %d,每人 %d(%d 人分)", total, share, len(idx))
}
