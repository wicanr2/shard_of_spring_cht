package combat

import "shardofspring/internal/original"

// Build 依 docs/spec/01-combat.md §1 的來源欄把隊伍與怪物填進單位陣列。
//
// ⚠ **索引配置不可以改**(docs/spec/07 §1):怪物 0–8、隊伍 9–13。
func Build(party []original.Character, monsters []original.Monster,
	items map[int]Item, r Rand) *Field {

	f := &Field{Rand: r, Items: items}

	for i, m := range monsters {
		if i >= MonsterMax {
			break
		}
		f.Units[MonsterBase+i] = Unit{
			Name: m.Name, IsMonster: true,
			// 屬性 2/3 的怪物來源是「欄 × 亂數」(docs/spec/01 §1)。
			// ⚠ **乘的是什麼亂數未解** —— 這裡用 1…速度 / 1…生命值,
			// 是把「欄值當上限」的讀法。解出來時改這兩行。
			Speed: r.Roll(max1(m.Speed)),
			HP:    r.Roll(max1(m.HPDie)),
			// 武器 0 → 60(赤手空拳的哨兵,docs/formats/03)
			Weapon: bareIfZero(m.Weapon), Armor: m.Armor,
			Str: m.Str, SP: m.SP, ToHit: m.ToHit,
			Status: 0,     // 常數 0
			Facing: South, // 常數 3 —— 出場時全部面南(docs/re/96)
			Kind:   m.Class, Tier: m.Tier, Exp: m.Exp,
			// ⚠ 屬性 14(行動類型)在原版是擲出來的:`INT(RND × ds:94B8) + 1`,
			// 而 **`ds:94B8` 未解**(docs/re/163 §2)。這裡留 0 ——
			// 目前只有空手道閘門讀它,而怪物沒有技能旗標,所以留 0 不影響任何數字。
			// ⛔ 怪物 AI 要用到它的時候**先把面數解出來**,不要在這裡填一個數。
		}
	}

	for i, c := range party {
		if i >= PartyMax {
			break
		}
		f.Units[PartyBase+i] = Unit{
			Name:  c.Name,
			Speed: c.Speed, HP: c.HP,
			Weapon: c.Weapon, Armor: c.Armor,
			Str: c.Str, SP: c.SP, Status: c.Status, ToHit: c.ToHit,
			Facing: North,
			Tier:   99, // 角色固定 99(docs/spec/01 §1)
			// 屬性 14 / 11 由職業字元一起決定(docs/re/163 §1)
			Action:  PartyAction(c.Class),
			Kind:    partyKind(c.Class),
			StatMag: c.StatMag,
			// 屬性 16/17 只有 Hero 有(位移 49/48 的技能旗標)
			Berserk: skill(c, 8), ArmSkin: skill(c, 7), Karate: skill(c, KarateSkill),
		}
	}
	f.Sort()
	return f
}

// partyKind 是隊員的圖組(屬性 11)。原版與屬性 14 在同一個分支裡設
// (docs/re/163 §1),所以兩者一定同號。
func partyKind(class byte) int {
	if PartyAction(class) == ActionFighter {
		return KindFighter
	}
	return KindWizard
}

// skill 讀第 n 個技能旗標(1-based,docs/formats/01 位移 41+n)。
// ⚠ **同一格在兩張表裡是不同的技能** —— Wizard 沒有 Berserking / Armored skin,
// 所以這裡先看職業。
func skill(c original.Character, n int) int {
	if c.Class != '1' { // 只有 Hero
		return 0
	}
	if n < 1 || n > len(c.Skills) {
		return 0
	}
	if c.Skills[n-1] == '1' {
		return 1
	}
	return 0
}

// bareIfZero:怪物欄5 的 0 代表沒有武器,而傷害公式的門檻是 ≥ 60。
func bareIfZero(w int) int {
	if w == 0 {
		return BareHandMin
	}
	return w
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}
