package town

import (
	"testing"

	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

func hero(level, end int) original.Character {
	return original.Character{
		Name: "測試", Class: byte(rules.ClassHero), Level: level, End: end,
		MaxHP: 10, HP: 4,
	}
}

func TestTrainNeedsTheRightGuild(t *testing.T) {
	c := hero(1, 10)
	if got := Train(&c, 999_999, 1); got != TrainWrongGuild {
		t.Fatalf("戰士進魔法訓練所應被擋下,得 %v", got)
	}
	if c.Level != 1 {
		t.Error("被擋下卻升級了")
	}
}

func TestTrainNeedsExperience(t *testing.T) {
	c := hero(1, 10)
	if got := Train(&c, 299, 0); got != TrainNotEnoughExp {
		t.Fatalf("299 經驗應不足,得 %v", got)
	}
	if got := Train(&c, 300, 0); got != TrainOK {
		t.Fatalf("300 經驗應可升級,得 %v", got)
	}
	if c.Level != 2 {
		t.Fatalf("等級應為 2,得 %d", c.Level)
	}
}

// 升級加的是上限與當前值,但**不補舊傷**。
func TestTrainAddsGrowthWithoutHealingOldWounds(t *testing.T) {
	c := hero(1, 10) // 體質 10 → 戰士成長上限 7(手冊 p.49)
	Train(&c, 300, 0)
	if c.MaxHP != 17 {
		t.Errorf("最大生命應為 10+7=17,得 %d", c.MaxHP)
	}
	if c.HP != 11 {
		t.Errorf("當前生命應為 4+7=11(舊傷不補),得 %d", c.HP)
	}
}

// 法師才有法力成長。這條擋的是「兩張表用錯欄」——
// 用戰士欄給法師加血,數字仍然合理,畫面上看不出來。
func TestWizardGrowsSPAndUsesWizardColumn(t *testing.T) {
	c := original.Character{
		Name: "法", Class: byte(rules.ClassWizard), Level: 1,
		End: 10, Int: 12, MaxHP: 8, HP: 8, MaxSP: 5, SP: 5,
	}
	Train(&c, 300, 1)
	if c.MaxHP != 8+5 { // 體質 10 的**法師**欄是 5,不是戰士的 7
		t.Errorf("法師最大生命應為 8+5=13,得 %d", c.MaxHP)
	}
	if c.MaxSP != 5+8 { // 智力 12 → 8
		t.Errorf("最大法力應為 5+8=13,得 %d", c.MaxSP)
	}
}

func TestTrainIsFreeOfCharge(t *testing.T) {
	// 手冊 p.37:訓練完全免費。介面上沒有金幣參數,這裡確認升級不需要它。
	c := hero(1, 10)
	if Train(&c, 300, 0) != TrainOK {
		t.Error("升級不該需要金幣")
	}
}

func TestGuildTeaches(t *testing.T) {
	if GuildTeaches(0) != byte(rules.ClassHero) {
		t.Error("位移 36 = 0 應是武術(戰士)")
	}
	if GuildTeaches(1) != byte(rules.ClassWizard) {
		t.Error("位移 36 = 1 應是魔法(法師)")
	}
}
