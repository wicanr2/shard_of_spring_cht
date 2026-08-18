package town

import "shardofspring/internal/original"

// 治療所。價目表 docs/re/142,四項服務的效果 docs/re/140 §6(手冊 p.37)。
//
// ⚠ 治療所**不受 `PERSUASIVENESS` 影響**(手冊 p.37 明講),
// 但仍然乘該店的價格倍率 —— 兩件事不一樣,別混。

// HealKind 是治療所的四種服務。
type HealKind int

const (
	HealWounds HealKind = iota
	HealPoison
	HealBind
	HealDeath
)

// docs/spec/19-module-text.md(F1):TOWN:21–24「Healing」/「Unpoison」/
// 「Unbind」/「Ressurect」→「醫療」/「解毒」/「解除束縛」/「復活」。
func (k HealKind) String() string {
	switch k {
	case HealPoison:
		return "解毒"
	case HealBind:
		return "解除束縛"
	case HealDeath:
		return "復活"
	}
	return "醫療"
}

// 狀態碼。定義在 `original`(那是記錄的語意,不是城鎮的),
// 這裡只是別名 —— **同一個數字不要在兩個套件裡各寫一次**。
const (
	StatusOK       = original.StatusOK
	StatusPoisoned = original.StatusPoisoned
	StatusBound    = original.StatusBound
	StatusDead     = original.StatusDead
)

// HealCost 回傳一項服務在某間治療所的價格。
//
// 傷勢按**缺的生命**計價,束縛與復活按**角色等級**計價,解毒是定價。
// 回 0 表示這個人不需要這項服務(不該收錢)。
func HealCost(c original.Character, k HealKind, mult float64) int {
	switch k {
	case HealWounds:
		missing := c.MaxHP - c.HP
		if missing <= 0 {
			return 0
		}
		return Price(HealPerHP*missing, mult)
	case HealPoison:
		if c.Status != StatusPoisoned {
			return 0
		}
		return Price(UnpoisonPrice, mult)
	case HealBind:
		if c.Status != StatusBound {
			return 0
		}
		return Price(UnbindPerLv*c.Level, mult)
	case HealDeath:
		if c.Status != StatusDead && c.HP > 0 {
			return 0
		}
		return Price(ResurrectPerLv*c.Level, mult)
	}
	return 0
}

// Heal 施做一項服務。金幣不足或不需要時回 false,且**不扣錢也不改狀態**。
//
// ⚠ 復活只把人拉回**1 點生命**,不是回滿 —— 2026-08-18 原版實跑量到的
// (狀態 5 的角色付 100 金幣之後生命 = 1,`workplace/dosbox/shots/r3j2-after.png`)。
func Heal(gold *float64, c *original.Character, k HealKind, mult float64) bool {
	cost := HealCost(*c, k, mult)
	if cost == 0 || float64(cost) > *gold {
		return false
	}
	*gold -= float64(cost)
	switch k {
	case HealWounds:
		c.HP = c.MaxHP
	case HealPoison, HealBind:
		c.Status = StatusOK
	case HealDeath:
		c.Status = StatusOK
		c.HP = 1
	}
	return true
}

// ResurrectNote 是給畫面顯示用的說明。原版實跑量過:復活回 **1 點**。
const ResurrectNote = "復活後回 1 點生命(原版實跑量到的)"

// NeededHeal 回傳這個人**現在該做的那一項**服務。
//
// 原版的治療所**沒有服務選單**:選完人直接報價
// (`That will cost N gold, Pay (Y/N)?`),而報的是「這個人現在最該治的那一項」。
// 決定性的一次實跑:生命 5/15 **且**中毒的角色報價 40(解毒價),
// 不是 20+40 —— 付完之後毒解了、**生命還是 5**
// (2026-08-18,`workplace/dosbox/shots/r3i1-combo.png` / `r3i2-after.png`)。
//
// ⚠ 這不需要猜優先序:狀態欄是**單一個值**,同時只會是中毒/束縛/死亡其中之一,
// 所以規則就是「有狀態先治狀態,沒有才治傷」。
func NeededHeal(c original.Character) HealKind {
	switch c.Status {
	case StatusDead:
		return HealDeath
	case StatusBound:
		return HealBind
	case StatusPoisoned:
		return HealPoison
	}
	if c.HP <= 0 {
		return HealDeath // 生命 0 但狀態沒標到 —— 同 HealCost 的判斷
	}
	return HealWounds
}
