package combat

import "shardofspring/internal/original"

// Build 依 docs/spec/01-combat.md §1 的來源欄把隊伍與怪物填進單位陣列。
//
// ⚠ **索引配置不可以改**(docs/spec/07 §1):怪物 0–8、隊伍 9–13。
// ⚠ `r` 是 FloatRand 不是 Rand —— Field.Rand 要能給 Float01()
// (docs/re/185:命中／狂暴擲骰要用它)。
func Build(party []original.Character, monsters []original.Monster,
	items map[int]Item, r FloatRand) *Field {

	// ⚠ **回合從 1 起算,不是 0。** 原版的 `ds:9314` 初始化成 0,
	// 但它在**每回合開頭** `inc`(docs/re/195 §1)—— 所以打第一回合時
	// 它的值是 1。引擎的第一回合不遞增(玩家直接行動),`Round` 要自己從 1 開始。
	//
	// 從 0 起算的後果不是「差一」而是**多擋一回合**:群體傷害的閘門
	// 「回合 == 1」寫成「≤ 1」才擋得住第一回合,於是第二回合也被擋掉,
	// 而畫面上第一回合顯示「第 0 回合」。
	f := &Field{Rand: r, Items: items, Round: 1}

	// 隊上只要有一個人會「策略」,畫面就顯示怪物在追誰(手冊 p.35)。
	// ⚠ 這是**顯示**能力,不影響任何規則 —— 怪物的鎖定照樣進行。
	for _, c := range party {
		if skill(c, TacticsSkill) > 0 {
			f.Tactics = true
			break
		}
	}

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
			Weapon: bareIfZero(m.Weapon), Armor: m.Armor, WeaponKnown: true,
			Str: m.Str, SP: m.SP, ToHit: m.ToHit,
			Status: 0,     // 常數 0
			Facing: South, // 常數 3 —— 出場時全部面南(docs/re/96)
			Kind:   m.Class, Tier: m.Tier, Exp: m.Exp,
			// 屬性 14 對怪物而言是**法術系別 1–5**(docs/re/170),
			// 原版擲 `INT(RND × ds:94B8) + 1`,而 `ds:94B8` 的初值就是 **5**
			// (docs/re/178 §2 從 DGROUP 讀出來的,先前是從行為推的)。
			//
			// ⚠ 系別 1 與 ActionFighter 同值。**這不會誤觸空手道閘門** ——
			// 那個閘門的第二層要技能旗標,而怪物沒有技能旗標(Karate 恆為 0)。
			Action: r.Roll(MonsterActionFaces),
		}
	}

	for i, c := range party {
		if i >= PartyMax {
			break
		}
		// ⚠ 記錄的位移 34/36 是**背包格號**不是物品編號(docs/re/75 §1)——
		// 中間隔一層查表。直接把格號當編號用,拿到的會是背包第 n 件的
		// 傷害值,而**畫面上一切正常**:編號 0–9 全都是合法的道具。
		weapon, known := c.EquippedItem(c.Weapon, BareHandMin)
		armor, _ := c.EquippedItem(c.Armor, NoArmor)
		f.Units[PartyBase+i] = Unit{
			Name:  c.Name,
			Speed: c.Speed, HP: c.HP,
			Weapon: weapon, Armor: armor, WeaponKnown: known,
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
