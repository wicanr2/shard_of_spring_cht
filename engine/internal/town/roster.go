package town

import (
	"strings"

	"shardofspring/internal/original"
)

// 角色名冊。docs/spec/11 §5。

// NameMaxRunes 是角色名稱的長度上限。
// 來自 `CHARUTIL.EXE` 的 `'Please enter the new name (9 char max)'`。
//
// ⚠ 記錄的名稱欄是 **10 bytes**(位移 2–11),但原版只讓玩家輸入 9 個 ——
// 兩個數字不同,照抄小的那個。差的那一格用途未解。
const NameMaxRunes = 9

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

// CreateUnresolved 是角色創造規則的未解說明(docs/spec/11 §5)。
//
// ⛔ 屬性怎麼決定沒有讀出來。**不要因為「看起來像 3d6」就寫成 3d6** ——
// 出貨五人的屬性是 3–16,3d6 的範圍是 3–18,兩者相容但不等價。
const CreateUnresolved = "角色創造的屬性規則未解,暫用出貨五人的分佈"

// ShippedAttrRange 回傳出貨五個角色的屬性範圍,給創造畫面當暫用依據。
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
