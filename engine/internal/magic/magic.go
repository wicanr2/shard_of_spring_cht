// Package magic 是法術與魔法道具的規則。
//
// 規則出自 docs/spec/02-magic.md 與 docs/spec/09-magic-items.md;
// 每一條在下面註明章節。
package magic

import (
	"fmt"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
)

// SkillBase 是技能旗標在角色記錄裡的起點:MID$(記錄, 系別 + 41, 1)
// (docs/spec/09 §1)。
//
// ⚠ **索引是「系別」不是「法術編號」。** 系別只有 1–5(五種 runes),
// 33 個法術共用這五個旗標。寫成法術編號會讓大部分法術查到界外,
// 而界外在 Go 裡是 panic、在原版裡是讀到別的欄位 —— 兩邊都不是靜默,
// 但錯的方向不同,所以這裡明寫。
const SkillBase = 41

// WizardClass 是可以施法的職業碼(docs/formats/01 位移 15)。
const WizardClass = '2'

// MagicItemMin 是「魔法道具」的物品編號門檻:**大於**它才走道具發動
// (docs/spec/02 §4)。
const MagicItemMin = 26

// Fail 說明為什麼施不出來。
type Fail int

const (
	OK Fail = iota
	FailNotWizard
	FailNoSkill
	FailNoPoints
	FailBelowOneLevel
)

func (f Fail) String() string {
	switch f {
	case FailNotWizard:
		return "不是法師"
	case FailNoSkill:
		return "沒有這一系的符文技能"
	case FailNoPoints:
		return "法力不足"
	case FailBelowOneLevel:
		return "投入的點數不足一級"
	}
	return ""
}

// CanCast 檢查三個閘門(docs/spec/09 §1)。
func CanCast(c original.Character, s original.Spell, invest int) Fail {
	if c.Class != WizardClass {
		return FailNotWizard
	}
	i := s.School + SkillBase - 1 // Skills 是 0-based 切片,記錄位移是 1-based
	if s.School < 1 || i-SkillBase < 0 || i-SkillBase >= len(c.Skills) {
		return FailNoSkill
	}
	if c.Skills[s.School-1] == '0' {
		return FailNoSkill
	}
	if invest > c.SP {
		return FailNoPoints
	}
	if Level(s, invest) < 1 {
		return FailBelowOneLevel
	}
	return OK
}

// Level 回傳這次施法的等級:INT(投入 ÷ 每級單價)(docs/spec/09 §2)。
func Level(s original.Spell, invest int) int {
	if s.UnitCost <= 0 {
		return invest // 單價 0 的法術:投多少算多少級
	}
	return invest / s.UnitCost
}

// Power 回傳威力:每點威力 × **投入點數**(不是等級,docs/spec/09 §2)。
//
// ⚠ 兩者只在單價 = 1 時相同。寫成「× 等級」不會報錯,
// 只會讓單價 > 1 的法術全部變弱。
func Power(s original.Spell, invest int) int { return s.Power * invest }

// StatusMagnitude 回傳狀態強度:每級單價 ÷ 投入(docs/spec/09 §4)。
//
// ⚠ **投得越多、值越小。** 而狀態強度在傷害公式裡是減項
// (docs/spec/01 §5),所以值小 = 被打得越痛。這個方向反直覺,是照抄的。
func StatusMagnitude(s original.Spell, invest int) int {
	if invest <= 0 {
		return 0
	}
	return s.UnitCost / invest
}

// Result 是一次施法的結果。
type Result struct {
	Fail       Fail
	Message    string
	Unresolved bool // 效果未解 —— 訊息會標出來,而且不套用任何數值
}

// 效果類別。docs/formats/04。
const (
	EffGroupDamage  = 1
	EffSingleDamage = 2
	EffAttr3        = 3 // ⚠ 影響哪個屬性**未解**
	EffStrength     = 4
	EffHitPoints    = 5
	EffSpeed        = 6
	EffProtect      = 7
	EffRaise        = 8
	EffCure         = 9
	EffUnbind       = 10
	EffBind         = 11
	EffUtility      = 12
	EffTransference = 13 // ⚠ 僅一例,**未解**
)

// Apply 把一個法術套到目標身上,回傳訊息。
//
// ⚠ **沒有 default 分支吞掉未知類別**(docs/spec/09 §7 驗收 8)——
// 未知的類別要走到 unresolved 那一條,在訊息上看得見。
func Apply(s original.Spell, invest int, caster *combat.Unit,
	targets []*combat.Unit) Result {

	p := Power(s, invest)
	name := s.Name
	switch s.Effect {
	case EffGroupDamage:
		for _, t := range targets {
			combat.Apply(t, p)
		}
		return Result{Message: fmt.Sprintf("%s 對全體造成 %d 點傷害", name, p)}

	case EffSingleDamage:
		if len(targets) > 0 {
			combat.Apply(targets[0], p)
		}
		return Result{Message: fmt.Sprintf("%s 造成 %d 點傷害", name, p)}

	case EffAttr3:
		// ⚠ 影響哪個屬性未解(docs/spec/09 §3)。**不猜** ——
		// 猜一個屬性會讓未解項變成看不出來的錯誤。
		return Result{Unresolved: true,
			Message: fmt.Sprintf("%s 發動,但效果類別 3 影響哪個屬性未解", name)}

	case EffStrength, EffHitPoints, EffSpeed:
		for _, t := range targets {
			switch s.Effect {
			case EffStrength:
				t.Str += p
			case EffHitPoints:
				t.HP += p
				if t.HP < 0 {
					t.HP = 0
				}
			case EffSpeed:
				t.Speed += p
			}
		}
		attr := map[int]string{
			EffStrength: "力量", EffHitPoints: "生命值", EffSpeed: "速度",
		}[s.Effect]
		return Result{Message: fmt.Sprintf("%s：%s %+d", name, attr, p)}

	case EffProtect:
		for _, t := range targets {
			t.ArmSkin += p
		}
		return Result{Message: fmt.Sprintf("%s 提供 %d 點防護", name, p)}

	case EffRaise:
		for _, t := range targets {
			if !t.Alive() {
				t.HP = 1
				t.Status = 0
			}
		}
		return Result{Message: name + " 讓倒下的同伴復活"}

	case EffCure:
		for _, t := range targets {
			t.Status = 0
			t.StatMag = 0
		}
		return Result{Message: name + " 解除了不良狀態"}

	case EffUnbind:
		for _, t := range targets {
			if t.Status == 2 { // Bound
				t.Status = 0
			}
		}
		return Result{Message: name + " 解除了束縛"}

	case EffBind:
		for _, t := range targets {
			t.Status = s.School
			t.StatMag = StatusMagnitude(s, invest)
		}
		return Result{Message: fmt.Sprintf("%s %s", name, s.HitMsg)}

	case EffUtility:
		return Result{Message: name + "(非戰鬥效用)"}

	case EffTransference:
		// ⚠ 僅一例,群內無對照,結構上無法再分(docs/spec/02「未解」)。
		return Result{Unresolved: true,
			Message: name + " 發動,但效果未解(類別 13,全遊戲僅此一個)"}
	}

	// 走到這裡表示資料裡有規格沒列的類別 —— 讓它看得見,不要靜默。
	return Result{Unresolved: true,
		Message: fmt.Sprintf("%s 的效果類別 %d 不在規格裡", name, s.Effect)}
}

// ItemTriggers 判定一件魔法道具這次有沒有發動(docs/spec/09 §5)。
//
// ⚠ **分母未解。** 先前用 `combat.DamageFaces`(26),理由是「與傷害公式的骰面
// 同一個數」—— 而那個 26 是幻影:它來自把 `mov bx, 1Ah` 讀成面數,
// 0x1A 其實是浮點累加器的位址(docs/re/153 §9)。
//
// 這裡改用 `combat.ToHitFaces`(原版 ds:977Eh)—— 那是**原版唯一的一顆骰**:
// 命中、狂暴門檻、`CMBT 0x1505A` 都擲它。**但這一條的原始碼還沒讀到**,
// 所以它是一個有理由的選擇,不是讀出來的規則。
func ItemTriggers(itemIndex, successRate int, r combat.Rand) bool {
	if itemIndex <= MagicItemMin {
		return false // 不是魔法道具
	}
	return r.Roll(combat.ToHitFaces) <= successRate
}
