package town

import (
	"testing"

	"shardofspring/internal/original"
)

func hurt(hp, maxhp, level, status int) original.Character {
	return original.Character{Name: "傷", HP: hp, MaxHP: maxhp, Level: level, Status: status}
}

// 價目表 docs/re/142:治療 2、解毒 40、解束縛 94/lvl、復活 100/lvl,倍率 1.0。
func TestHealCostsMatchTheOriginalPriceList(t *testing.T) {
	c := hurt(4, 20, 3, StatusOK) // 缺 16 點
	if got := HealCost(c, HealWounds, 1.0); got != 32 {
		t.Errorf("缺 16 點應收 32,得 %d", got)
	}
	p := hurt(10, 10, 3, StatusPoisoned)
	if got := HealCost(p, HealPoison, 1.0); got != 40 {
		t.Errorf("解毒應收 40,得 %d", got)
	}
	b := hurt(10, 10, 3, StatusBound)
	if got := HealCost(b, HealBind, 1.0); got != 94*3 {
		t.Errorf("等級 3 解束縛應收 282,得 %d", got)
	}
	d := hurt(0, 10, 4, StatusDead)
	if got := HealCost(d, HealDeath, 1.0); got != 400 {
		t.Errorf("等級 4 復活應收 400,得 %d", got)
	}
}

// 不需要的服務不能收錢 —— 滿血的人按「治療」不該扣 2 塊。
func TestHealChargesNothingWhenNotNeeded(t *testing.T) {
	c := hurt(20, 20, 3, StatusOK)
	if got := HealCost(c, HealWounds, 1.0); got != 0 {
		t.Errorf("滿血不該收費,得 %d", got)
	}
	if got := HealCost(c, HealPoison, 1.0); got != 0 {
		t.Errorf("沒中毒不該收解毒費,得 %d", got)
	}
	gold := 100.0
	if Heal(&gold, &c, HealWounds, 1.0) {
		t.Error("滿血不該能治療")
	}
	if gold != 100 {
		t.Errorf("失敗時不該扣錢,剩 %g", gold)
	}
}

func TestHealAppliesShopMultiplier(t *testing.T) {
	c := hurt(10, 20, 1, StatusOK) // 缺 10 → 基準 20
	if got := HealCost(c, HealWounds, 1.25); got != 25 {
		t.Errorf("倍率 1.25 應為 INT(20×1.25)=25,得 %d", got)
	}
}

func TestHealBlockedWhenGoldShort(t *testing.T) {
	c := hurt(1, 20, 1, StatusOK)
	gold := 10.0 // 需要 38
	if Heal(&gold, &c, HealWounds, 1.0) {
		t.Error("錢不夠不該治療")
	}
	if gold != 10 || c.HP != 1 {
		t.Errorf("失敗時狀態不該變,金 %g 血 %d", gold, c.HP)
	}
}

// 復活回 1 點,不是回滿 —— 回滿等於白送一次治療傷勢,而畫面上看不出來。
func TestResurrectReturnsOneHP(t *testing.T) {
	c := hurt(0, 30, 2, StatusDead)
	gold := 1000.0
	if !Heal(&gold, &c, HealDeath, 1.0) {
		t.Fatal("應該復活得了")
	}
	if c.HP != 1 {
		t.Errorf("復活後應為 1 點,得 %d", c.HP)
	}
	if c.Status != StatusOK {
		t.Errorf("復活後狀態應為正常,得 %d", c.Status)
	}
	if gold != 800 {
		t.Errorf("等級 2 復活應扣 200,剩 %g", gold)
	}
}
