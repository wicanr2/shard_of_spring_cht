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
// 成長量是**現骰後夾上限**(見 LevelGain)。
//
// 經驗值**不扣**(手冊沒說要扣,而累計欄的語意是「累計到多少」)。
func Train(c *original.Character, exp, guildExtra int, r Roller) TrainResult {
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
	gainHP := LevelGain(r, c.End, rules.MaxHPGain(c.End, wizard))
	c.MaxHP += gainHP
	c.HP += gainHP // 升級時把新增的部分也補滿,舊傷不補

	if wizard {
		gainSP := LevelGain(r, c.Int, rules.MaxSPGain(c.Int))
		c.MaxSP += gainSP
		c.SP += gainSP
	}
	return TrainOK
}

// LevelGain 回傳升一級實際加多少點:**擲骰,超過上限就用上限**。
//
//	成長 = min(擲骰(屬性), 上限)
//
// 上限是手冊 p.48/p.49 兩張表(`MAX … GAIN PER LEVEL`)—— 欄名寫著「最多」,
// 所以那兩張表給的不是成長量本身,是成長量的**天花板**。
//
// **「現骰」是確定的** —— 專案負責人實測原版,同一個角色反覆升級成長量**會變**
// (2026-08-15,第 1 級證據)。
//
// ⚠ **但骰面是具名假設**:裁定沒有指定骰幾面,原版那段程式碼也沒有讀到。
// 這裡取**屬性本身**(生命骰體能、法力骰智能),理由有二:
//
//   - 只有骰面**可能超過上限**,「超過就用上限」才有作用 ——
//     若骰 1…上限,那句規則永遠不會生效;
//   - 全遊戲的擲骰成語都是 `INT(RND × N) + 1`(docs/re/152 §3),
//     而這裡手邊唯一的 N 就是那個屬性。
//
// ⛔ **這不是讀出來的。** 要裁決得去讀升級那段程式碼,或在原版裡
// 同一個角色反覆升級看成長量會不會變。
//
// 效果:低屬性的人幾乎每次都吃滿(上限本來就低),高屬性的人**平均拿不到上限** ——
// 體能 20 的戰士上限 14,骰 1…20 夾完平均只有 9.45。
func LevelGain(r Roller, attr, cap int) int {
	if cap <= 0 {
		return 0
	}
	if attr < 1 {
		attr = 1
	}
	if v := r.Roll(attr); v < cap {
		return v
	}
	return cap
}

// GrowthNote 是訓練所畫面的說明。
const GrowthNote = "升級成長:擲骰(屬性)夾在手冊 p.48/49 的上限。⚠ 骰面是假設,原版那段沒讀到"

// NeedExp 回傳這個等級升下一級所需的累計經驗;已達頂級回 0。
func NeedExp(level int) int {
	if level < 1 || level >= rules.MaxLevel {
		return 0
	}
	return rules.ExpForLevel[level]
}
