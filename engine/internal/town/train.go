package town

import (
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// 訓練所。手冊 p.37(docs/re/140 §6):
//
//   - 升級**完全免費**,只看經驗夠不夠
//   - 戰士的訓練所只收戰士,法師的只收法師(TOWNDATA.DAT 位移 36:0 = 武術、1 = 魔法)

// TrainResult 說明一次訓練的結果。
type TrainResult int

const (
	TrainOK TrainResult = iota
	TrainWrongGuild
	TrainNotEnoughExp
	TrainMaxLevel
)

func (r TrainResult) String() string {
	switch r {
	case TrainWrongGuild:
		return "這間訓練所不收這個職業"
	case TrainNotEnoughExp:
		return "經驗還不夠"
	case TrainMaxLevel:
		return "已經是最高等級"
	}
	return ""
}

// GuildTeaches 回傳這間訓練所收哪個職業。
// extra 是 TOWNDATA.DAT 位移 36:0 = 武術(戰士)、1 = 魔法(法師)。
func GuildTeaches(extra int) byte {
	if extra == 1 {
		return byte(rules.ClassWizard)
	}
	return byte(rules.ClassHero)
}

// Train 讓一位角色在訓練所升一級。
//
// 成長量取手冊 p.48/p.49 兩張表的**上限**(`MAX … GAIN PER LEVEL`)。
// ⚠ 表的欄名是「最多」,所以原版很可能是**在 1 到上限之間現骰**;
// 沒有讀到那段程式碼,這裡取上限是一個**具名的假設**(見 GrowthAssumption),
// 換成擲骰只要改這個函式。
//
// 經驗值**不扣**(手冊沒說要扣,而累計欄的語意是「累計到多少」)。
func Train(c *original.Character, exp, guildExtra int) TrainResult {
	if c.Class != GuildTeaches(guildExtra) {
		return TrainWrongGuild
	}
	if c.Level >= rules.MaxLevel {
		return TrainMaxLevel
	}
	if !rules.CanLevelUp(c.Level, exp) {
		return TrainNotEnoughExp
	}
	wizard := c.Class == byte(rules.ClassWizard)

	c.Level++
	gainHP := rules.MaxHPGain(c.End, wizard)
	c.MaxHP += gainHP
	c.HP += gainHP // 升級時把新增的部分也補滿,舊傷不補

	if wizard {
		gainSP := rules.MaxSPGain(c.Int)
		c.MaxSP += gainSP
		c.SP += gainSP
	}
	return TrainOK
}

// GrowthAssumption 是給畫面顯示用的說明。
const GrowthAssumption = "⚠ 手冊給的是成長**上限**,原版是否現骰未解 —— 本引擎取上限"

// NeedExp 回傳這個等級升下一級所需的累計經驗;已達頂級回 0。
func NeedExp(level int) int {
	if level < 1 || level >= rules.MaxLevel {
		return 0
	}
	return rules.ExpForLevel[level]
}
