package town

import (
	"strings"

	"shardofspring/internal/original"
)

// 角色名冊。docs/spec/11 §5。

// NameMaxRunes 是角色名稱的長度上限。
//
// ⚠ **原版自己不一致**(docs/re/143 §4):創造時提示
// `Enter your character's name (10 char)`,改名時提示 `(9 char max)`。
// 記錄的名稱欄是 **10 bytes**(位移 2–11)—— 所以 10 才是欄位真正的容量,
// 改名那個 9 是原版少算一格。這裡取 10。
//
// ⚠ 先前這裡寫 9,並把差的那一格記成「用途未解」。**那不是未解,
// 是兩個提示打架** —— 只讀到其中一個提示時,它看起來像個乾淨的事實。
const NameMaxRunes = 10

// JoinParty 把一位角色編進第 n 隊,同時維護**兩個地方**。
//
// ⚠ docs/spec/06 §1:隊伍成員記在角色的位移 1 **與** GROUPS.DAT 的成員槽。
// 只寫一邊會讓 M3 的一致性檢查噴警告 —— 那個檢查已經做好了,這裡只要不弄壞它。
//
// 回 false 表示沒有空的成員槽。
func JoinParty(c *original.Character, g *original.Group, party int) bool {
	if party < 1 || party > original.GroupSlots {
		return false
	}
	// 先找空槽:值不在 1–32 的就是空的(docs/re/135 §3)
	slot := -1
	for i, v := range g.Members {
		if v < 1 || v > 32 {
			slot = i
			break
		}
		if v == c.ID {
			slot = i // 已經在隊裡,原地更新
			break
		}
	}
	if slot < 0 {
		return false
	}
	// 人數上限 5(docs/re/135 §3 的 `cmp ds:34F8h, 5`)
	if len(g.MemberIDs()) >= original.PartySlots && g.Members[slot] != c.ID {
		return false
	}
	g.Members[slot] = c.ID
	c.Party = byte('0' + party)
	return true
}

// LeaveParty 把一位角色移出隊伍,同樣要動兩個地方。
func LeaveParty(c *original.Character, g *original.Group) {
	for i, v := range g.Members {
		if v == c.ID {
			g.Members[i] = 99 // 出貨資料用 99 表示空槽(docs/re/135 §2)
		}
	}
	c.Party = original.NoParty
}

// Rename 改名。超過 9 個字截斷 —— 這是**輸入端**的限制,不是顯示端。
func Rename(c *original.Character, name string) {
	rs := []rune(strings.TrimSpace(name))
	if len(rs) > NameMaxRunes {
		rs = rs[:NameMaxRunes]
	}
	c.Name = string(rs)
}

// Delete 刪除一位角色。
//
// ⚠ 位移 1 寫 `'*'`(`CHARUTIL` 的 `'Is deleted !'`),名稱清空 ——
// 而**佔用判定看的是名稱**(docs/spec/06 §2),所以名稱一定要清。
func Delete(c *original.Character) {
	c.Name = ""
	c.Party = original.NoParty
}

// ShippedAttrRange 回傳出貨五個角色的屬性範圍。
//
// 現在只給測試當**健全性檢查**用:擲骰公式的支撐集(2…13)加上種族修正
// 之後,要涵蓋得住出貨五人實際擁有的值(docs/re/178 §4)。
//
// ⚠ 這個檢查**不能拿來裁決常數** —— 舊的 A=5/B=4 與新的 A=6/B=2 都涵蓋得住。
// 涵蓋是必要條件不是充分條件。
func ShippedAttrRange(chars []original.Character) (lo, hi int) {
	lo, hi = 99, 0
	for _, c := range chars {
		if !c.Occupied() {
			continue
		}
		for _, v := range []int{c.Speed, c.Str, c.Int, c.End, c.ToHit} {
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
	}
	if lo > hi {
		return 3, 18
	}
	return lo, hi
}
