package combat

// 攻擊附帶中毒。規則出自 docs/re/191。

// PoisonWeapon 是「有毒的自然武器」那個哨兵值(屬性 4 == 62)。
//
// `MONSTERS.DAT` 欄5 只有 0–13 與 62 兩群,62 那一群正好是
// 眼鏡蛇 / 死亡蝮蛇 / 食屍鬼 / 坑蝮蛇 / 響尾蛇(docs/re/191 §4)。
//
// ⚠ 角色取不到這個值 —— `ITEMS.DAT` 只有 57 筆。所以中毒是**怪物咬人專用**。
const PoisonWeapon = 62

// PoisonChance 是中毒機率。`ds:9802` 的 DGROUP 初值 = MBF **0.15**。
const PoisonChance = 0.15

// StatusPoisoned 是狀態欄的中毒編號(= `SPELLS.DAT` 的系別 1)。
const StatusPoisoned = 1

// poison 在傷害套用之後判定中毒;中了就改狀態並回**接在傷害那句後面**的
// 尾巴,沒中回空字串。
//
// ⚠ 原版這一句印在另一行(`LOCATE 13h, 17h`)而且**不帶名字** —— 它靠上一行
// 剛講過誰被打中。引擎把一次攻擊寫成一行,所以直接串在後面,
// 讀起來的語序與原版相同。
//
// 三個條件同時成立才算(docs/re/191 §2):
//
//	攻擊者屬性 4 == 62、RND < 0.15、**防禦者狀態 == 0**
//
// 外加兩道位置上的閘門(§1):
//
//	防禦者是隊員(索引 ≥ 9)、而且這一擊沒把他打死
//
// ⚠ **兩道位置閘門在擲骰之前、三個條件在擲骰之後。** 原版把武器、擲骰、
// 狀態三項各算成一個布林再 AND —— 也就是說武器不是 62 的時候**那一次 RND
// 照樣消耗掉**。順序搬錯會讓同一顆種子跑出不同的戰鬥
// (docs/spec/07 §8 驗收 6),所以這裡照抄兩段的分界。
func (f *Field) poison(atk, def int) string {
	if !f.Units[def].Alive() || def < PartyBase {
		return ""
	}
	weapon := f.Units[atk].Weapon == PoisonWeapon
	rolled := f.Rand.Float01() < PoisonChance
	normal := f.Units[def].Status == 0
	if !weapon || !rolled || !normal {
		return ""
	}
	f.Units[def].Status = StatusPoisoned
	return msgPoisoned
}
