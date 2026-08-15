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

func (k HealKind) String() string {
	switch k {
	case HealPoison:
		return "解毒"
	case HealBind:
		return "解除束縛"
	case HealDeath:
		return "復活"
	}
	return "治療傷勢"
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
// ⚠ 復活只把人拉回**1 點生命**,不是回滿 —— 拉回滿血等於白送一次治療傷勢,
// 而畫面上看不出來(復活完滿血很像「本來就該這樣」)。
// ⚠ 這一條沒有原版證據,是**具名的保守選擇**(見 ResurrectAssumption)。
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

// ResurrectAssumption 是給畫面顯示用的說明。
const ResurrectAssumption = "⚠ 復活後回幾點生命未解 —— 本引擎回 1 點"
