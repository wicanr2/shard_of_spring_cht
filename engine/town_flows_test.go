package main

// F3 補回來的三個城鎮子流程,各釘一條。
//
// 三個都是「原版有問一句、引擎直接做掉」的形狀 —— 而少一句問話在畫面上
// 沒有症狀:玩家只會覺得「按下去就扣錢了」,不會知道自己少了一次反悔的機會。

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/original"
	"shardofspring/internal/town"
)

// newTownGame 擺一個站在指定建築裡的遊戲。
func newTownGame(t *testing.T, kind original.ShopKind, mode townMode) *Game {
	t.Helper()
	c := original.Character{ID: 1, Name: "凱恩", Class: '1', Race: 'H',
		Level: 1, HP: 1, MaxHP: 20, SP: 0, MaxSP: 0,
		Weapon: original.NotEquipped, Armor: original.NotEquipped}
	for i := range c.Pack {
		c.Pack[i] = original.NotEquipped
	}
	g := &Game{
		members: []original.Character{c},
		chars:   []original.Character{c},
		group:   original.Group{Gold: 1000, Provisions: 5},
	}
	g.town = &townState{
		mode: mode,
		shop: original.Shop{Name: "測試", Kind: kind, PriceMult: 1},
	}
	return g
}

// TestInnAsksHowManyNights:R 之後要先問幾晚,不是直接住一晚(TOWN:30/31)。
func TestInnAsksHowManyNights(t *testing.T) {
	g := newTownGame(t, original.ShopInn, townInn)
	gold0, hp0 := g.group.Gold, g.members[0].HP

	g.townKey(ebiten.KeyR)
	if g.group.Gold != gold0 {
		t.Errorf("問「住幾晚」的時候不該先扣錢:%.0f → %.0f", gold0, g.group.Gold)
	}
	if !strings.Contains(g.town.msg, "要住幾晚?") {
		t.Errorf("R 應該問住幾晚,得到 %q", g.town.msg)
	}

	g.townKey(ebiten.KeyDigit3)
	each := town.Price(town.TownInnPrice, 1)
	if want := gold0 - float64(each*3); g.group.Gold != want {
		t.Errorf("住三晚應該扣 %d×3,得到 %.0f(期望 %.0f)", each, g.group.Gold, want)
	}
	if g.members[0].HP <= hp0 {
		t.Errorf("住三晚應該回三次生命,%d → %d", hp0, g.members[0].HP)
	}
	if g.townCount != 0 {
		t.Error("答完之後「問幾晚」的狀態要清掉")
	}
}

// TestInnZeroExits:0 是離開,不是住 0 晚。
func TestInnZeroExits(t *testing.T) {
	g := newTownGame(t, original.ShopInn, townInn)
	gold0 := g.group.Gold
	g.townKey(ebiten.KeyR)
	g.townKey(ebiten.KeyDigit0)
	if g.group.Gold != gold0 {
		t.Errorf("按 0 應該離開,不該扣錢:%.0f → %.0f", gold0, g.group.Gold)
	}
	if g.townCount != 0 {
		t.Error("按 0 之後狀態要清掉")
	}
}

// TestTavernAsksHowManyRations:B 之後要先問幾份(TOWN:58/59)。
func TestTavernAsksHowManyRations(t *testing.T) {
	g := newTownGame(t, original.ShopTavern, townTavern)
	before := g.group.Provisions

	g.townKey(ebiten.KeyB)
	if g.group.Provisions != before {
		t.Errorf("問「幾份」的時候不該先買:%d → %d", before, g.group.Provisions)
	}
	if !strings.Contains(g.town.msg, "要買幾份口糧?") {
		t.Errorf("B 應該問買幾份,得到 %q", g.town.msg)
	}

	g.townKey(ebiten.KeyDigit4)
	if g.group.Provisions != before+4 {
		t.Errorf("買四份應該 %d → %d,得到 %d", before, before+4, g.group.Provisions)
	}
}

// TestHealerAsksBeforeCharging:治療所先報價再收錢(TOWN:27/28)。
func TestHealerAsksBeforeCharging(t *testing.T) {
	g := newTownGame(t, original.ShopHealer, townHealer)
	gold0, hp0 := g.group.Gold, g.members[0].HP

	g.townKey(ebiten.KeyDigit1) // 選第一位
	g.townKey(ebiten.KeyW)      // 醫療
	if g.healPay == nil {
		t.Fatal("選完服務應該先問要不要付款")
	}
	if g.group.Gold != gold0 || g.members[0].HP != hp0 {
		t.Error("還沒回答就已經扣錢或治好了")
	}
	if !strings.Contains(g.town.msg, "付款嗎?(Y/N)") {
		t.Errorf("要問「付款嗎?(Y/N)」,得到 %q", g.town.msg)
	}

	g.townKey(ebiten.KeyN) // 不付
	if g.healPay != nil {
		t.Error("答 N 之後問句要收掉")
	}
	if g.group.Gold != gold0 || g.members[0].HP != hp0 {
		t.Errorf("答 N 不該扣錢也不該治療:金幣 %.0f、生命 %d", g.group.Gold, g.members[0].HP)
	}

	g.townKey(ebiten.KeyDigit1)
	g.townKey(ebiten.KeyW)
	g.townKey(ebiten.KeyY) // 付
	if g.members[0].HP <= hp0 {
		t.Errorf("答 Y 應該治療,%d → %d", hp0, g.members[0].HP)
	}
	if g.group.Gold >= gold0 {
		t.Errorf("答 Y 應該扣錢,%.0f → %.0f", gold0, g.group.Gold)
	}
}

// TestPartyAllDeadIsBlocked:全滅的隊伍不能選(MENU:99)。
// ⚠ 判準是**狀態 5**,不是生命值 —— 生命值 0 的人可能只是還沒被判死。
func TestPartyAllDeadIsBlocked(t *testing.T) {
	dead := original.Character{ID: 1, Name: "凱恩", Status: original.StatusDead, HP: 0}
	alive := original.Character{ID: 2, Name: "葛羅德", Status: 0, HP: 0}
	g := &Game{chars: []original.Character{dead, alive}}

	var grp original.Group
	grp.Members[0] = 1
	if !g.partyAllDead(grp) {
		t.Error("成員全部狀態 5 應該判成全滅")
	}
	grp.Members[1] = 2
	if g.partyAllDead(grp) {
		t.Error("還有人狀態不是 5(即使生命值 0)就不算全滅")
	}
	var empty original.Group
	if g.partyAllDead(empty) {
		t.Error("一個成員都沒有的隊伍不算全滅 —— 那是「不存在」,另一句話")
	}
}
