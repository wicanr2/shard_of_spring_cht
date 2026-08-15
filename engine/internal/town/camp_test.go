package town

import (
	"testing"

	"shardofspring/internal/original"
)

func warrior(skills string) original.Character {
	return original.Character{Name: "阿", Class: '1', Skills: skills}
}
func wizard(skills string) original.Character {
	return original.Character{Name: "巫", Class: '2', Skills: skills}
}

// H)unt 的閘門順序:室內 → 技能 → 今天用過 → 狀態(docs/re/166 §2)。
// 順序會改變玩家看到哪一句話,所以逐條測。
func TestCanHuntGateOrder(t *testing.T) {
	hunter := warrior("000000001 0"[:10])
	if !hasSkill(hunter, SkillHunting) {
		t.Fatal("測試資料壞了:第 9 格應該是 1")
	}
	if g := CanHunt(hunter, false); g != SkillIndoors {
		t.Errorf("室內應先擋:得 %v", g)
	}
	if g := CanHunt(warrior("0000000000"), true); g != SkillNoSkill {
		t.Errorf("沒有 Hunting 應擋:得 %v", g)
	}
	// 法師就算旗標是 1 也不行 —— 兩張技能表不同,第 9 格對法師是 Monster lore
	mage := wizard("000000001 0"[:10])
	if g := CanHunt(mage, true); g != SkillNoSkill {
		t.Errorf("法師不能打獵:得 %v", g)
	}
	spent := hunter
	spent.SkillUsed = true
	if g := CanHunt(spent, true); g != SkillSpent {
		t.Errorf("今天用過應擋:得 %v", g)
	}
	bound := hunter
	bound.Status = original.StatusBound
	if g := CanHunt(bound, true); g != SkillDisabled {
		t.Errorf("束縛應擋:得 %v", g)
	}
	// 中毒(1)仍然可以 —— 門檻是 > 1
	poisoned := hunter
	poisoned.Status = original.StatusPoisoned
	if g := CanHunt(poisoned, true); g != SkillOK {
		t.Errorf("中毒仍可打獵(門檻是 > 1):得 %v", g)
	}
}

// 三個 lore 的分界是讀到的常數(docs/re/166 §3)。
func TestLoreBands(t *testing.T) {
	for _, c := range []struct {
		item, want int
	}{
		{0, SkillWeaponLore}, {20, SkillWeaponLore},
		{21, SkillPotionLore}, {56, SkillPotionLore},
		{57, SkillItemLore}, {98, SkillItemLore},
		{original.NotEquipped, 0}, // 99 = 空格
	} {
		if got := LoreFor(c.item); got != c.want {
			t.Errorf("道具 %d → 技能 %d,應為 %d", c.item, got, c.want)
		}
	}
}

func TestCanIdentify(t *testing.T) {
	// 第 6 格 = Weapon lore
	sage := wizard("000001 0000"[:10])
	if !hasSkill(sage, SkillWeaponLore) {
		t.Fatal("測試資料壞了:第 6 格應該是 1")
	}
	if g := CanIdentify(warrior("1111111111"), 1); g != SkillNotWizard {
		t.Errorf("戰士不能辨識:得 %v", g)
	}
	if g := CanIdentify(sage, 1); g != SkillOK {
		t.Errorf("有 Weapon lore 應可辨識武器:得 %v", g)
	}
	// 同一個人辨識藥劑段的道具就不行 —— 要 Potion lore
	if g := CanIdentify(sage, 30); g != SkillNoSkill {
		t.Errorf("沒有 Potion lore 應擋:得 %v", g)
	}
	spent := sage
	spent.SkillUsed = true
	if g := CanIdentify(spent, 1); g != SkillSpent {
		t.Errorf("今天用過應擋:得 %v", g)
	}
}

// lore 分界的邊界值。0-based 編號:20 是最後一件護甲、21 是第一瓶藥水
// (docs/re/167 §3/§4)——這兩格寫錯不會有症狀,所以釘住。
func TestLoreBoundaryIsAtTheFirstPotion(t *testing.T) {
	if LoreFor(20) != SkillWeaponLore {
		t.Error("編號 20(ITEMS.DAT 第 21 列 Plate +2)應該算武器護甲")
	}
	if LoreFor(21) != SkillPotionLore {
		t.Error("編號 21(第 22 列 Heal potion)應該算藥水")
	}
	// ⚠ 第三段接不到任何真實道具(資料只到編號 56)——
	// 這不是 bug,是原版就這樣(docs/re/167 §4)。
	if LoreFor(57) != SkillItemLore {
		t.Error("編號 > 56 走 Item lore")
	}
}
