package town

import (
	"testing"

	"shardofspring/internal/combat"
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
	// 巫師就算旗標是 1 也不行 —— 兩張技能表不同,第 9 格對巫師是 Monster lore
	mage := wizard("000000001 0"[:10])
	if g := CanHunt(mage, true); g != SkillNoSkill {
		t.Errorf("巫師不能打獵:得 %v", g)
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

// 撿道具:由隊尾往隊首找空格,而且標成未辨識(docs/re/168 §1/§2)。
func TestPickUpFillsFromTheBack(t *testing.T) {
	mk := func() []original.Character {
		p := make([]original.Character, 3)
		for i := range p {
			for s := range p[i].Pack {
				p[i].Pack[s] = original.NotEquipped
			}
			p[i].Identified = "1111111111" // 先全設成已辨識,才看得出被清掉
		}
		return p
	}
	p := mk()
	who, slot := PickUp(p, 7)
	// ⚠ 方向是刻意的:原版的外層迴圈遞減,最後一個隊員先拿。
	if who != 2 || slot != 0 {
		t.Errorf("第一件應進最後一個隊員的第 0 格,得 (%d,%d)", who, slot)
	}
	if p[2].Pack[0] != 7 {
		t.Errorf("道具沒放進去:%v", p[2].Pack)
	}
	if IsIdentified(p[2], 0) {
		t.Error("撿來的東西應該是未辨識的")
	}
	// 旁邊那一格不能被動到 —— 原版差一格寫的就是這裡(docs/re/168 §4.1)
	if !IsIdentified(p[2], 1) {
		t.Error("只能清自己那一格的旗標,不能清下一格")
	}

	// 全滿時回 (-1,-1),不能靜默塞掉
	full := mk()
	for i := range full {
		for s := range full[i].Pack {
			full[i].Pack[s] = 1
		}
	}
	if w, s := PickUp(full, 7); w != -1 || s != -1 {
		t.Errorf("背包全滿應回 (-1,-1),得 (%d,%d)", w, s)
	}
}

// 打獵的收穫:兩個邊界各自能獨立弄錯,所以分開驗(docs/re/177 §4)。
func TestHuntYieldEdges(t *testing.T) {
	cases := []struct {
		roll int // ScriptRand 的 Roll(16) 回傳值(1…16)
		want int
	}{
		{1, 0},  // INT(RND×16)=0  → −6 → 夾成 0
		{7, 0},  // =6  → 0        ← ★ 最後一個失敗,`0` 不被夾但也是失敗
		{8, 1},  // =7  → 1        ← ★ 第一個成功
		{16, 9}, // =15 → 9        ← 上界,實跑量到的最大值
	}
	for _, c := range cases {
		r := &combat.ScriptRand{Values: []int{c.roll}}
		if got := HuntYield(r); got != c.want {
			t.Errorf("Roll=%d:收穫應為 %d,得 %d", c.roll, c.want, got)
		}
		if r.Faces[0] != HuntFaces {
			t.Errorf("面數應為 %d,得 %d", HuntFaces, r.Faces[0])
		}
	}
}

// 失敗率是 7/16 不是 6/16 —— 差別在「收穫 0 也是失敗」。
func TestHuntFailureRateIsSevenSixteenths(t *testing.T) {
	fail := 0
	for roll := 1; roll <= HuntFaces; roll++ {
		if HuntYield(&combat.ScriptRand{Values: []int{roll}}) == 0 {
			fail++
		}
	}
	if fail != 7 {
		t.Errorf("16 個擲出裡應有 7 個失敗,得 %d", fail)
	}
}

// TestIdentifyThreshold 釘住鑑定的成功門檻(docs/re/189)。
//
// ⚠ 這條擋的是「先前必定成功」那個佔位回來 —— 必定成功在畫面上完全正常,
// 玩家只會覺得這個技能很好用。
func TestIdentifyThreshold(t *testing.T) {
	if IdentifyFactor != 4.5 {
		t.Errorf("乘數 %v,原版 ds:72D2 是 4.5", IdentifyFactor)
	}
	if got := IdentifyThreshold(20); got != 90 {
		t.Errorf("智能 20 的門檻應該是 90,得到 %v", got)
	}
	if got := IdentifyThreshold(10); got != 45 {
		t.Errorf("智能 10 的門檻應該是 45,得到 %v", got)
	}
	// 智能 0 → 門檻 0,d100 最小 1 → 必定失敗。
	c := original.Character{Int: 0}
	if IdentifySucceeds(c, fixedRoll(1)) {
		t.Error("智能 0 不該鑑定成功")
	}
	// 智能 30(超過上限,防呆)→ 門檻 135 → 必定成功。
	if !IdentifySucceeds(original.Character{Int: 30}, fixedRoll(100)) {
		t.Error("門檻遠高於 d100 時應該必定成功")
	}
}

// fixedRoll 是永遠回同一個值的擲骰來源。
type fixedRoll int

func (r fixedRoll) Roll(int) int { return int(r) }

// 打獵之後的補給品夾在 255(docs/re/218 §1)。
//
// ⚠ 這個夾子**只在打獵那一條路徑上** —— 原版的 TOWN 一次都沒碰 ds:6F10,
// 所以買補給品不經過它。測試只驗夾子本身,不要推廣到買賣。
func TestCapProvisions(t *testing.T) {
	for _, tc := range []struct{ in, want int }{
		{0, 0},
		{254, 254},
		{ProvisionCap, ProvisionCap},
		{ProvisionCap + 1, ProvisionCap},
		{9999, ProvisionCap},
	} {
		if got := CapProvisions(tc.in); got != tc.want {
			t.Errorf("CapProvisions(%d) = %d,要 %d", tc.in, got, tc.want)
		}
	}
}
