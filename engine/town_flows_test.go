package main

// F3 補回來的三個城鎮子流程,各釘一條。
//
// 三個都是「原版有問一句、引擎直接做掉」的形狀 —— 而少一句問話在畫面上
// 沒有症狀:玩家只會覺得「按下去就扣錢了」,不會知道自己少了一次反悔的機會。

import (
	"fmt"
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

// ── 名冊:F3 補回來的原版四個指令(docs/re/143 的助憶鍵)────────────────

func newRosterGame(t *testing.T) *Game {
	t.Helper()
	g := newPlayingGame(t)
	g.openRoster()
	return g
}

// TestRosterRename 釘住 N)ew Name —— 規則層(town.Rename)一直都在,
// **缺的是接線**,而選單上沒列出來所以「按不到」不像壞掉(19-coverage §2.1)。
func TestRosterRename(t *testing.T) {
	g := newRosterGame(t)
	for i := range g.chars {
		if g.chars[i].Occupied() {
			g.roster.cursor = i
			break
		}
	}
	old := g.chars[g.roster.cursor].Name

	g.testRunes = nil
	press(t, g, ebiten.KeyN)
	if g.roster.rename == nil {
		t.Fatal("N 應該開改名輸入")
	}
	g.rosterRenameRunes([]rune("阿"))
	g.rosterRenameKey(ebiten.KeyEnter)
	if g.roster.rename != nil {
		t.Error("Enter 之後輸入狀態要收掉")
	}
	if got := g.chars[g.roster.cursor].Name; got != "阿" {
		t.Errorf("改名沒生效:%q → %q", old, got)
	}
}

// TestRosterRenameRespectsLimit:上限是記錄欄位的 10,不是原版提示的 9
// (spec/19 §4 的矛盾在裁決之前照記錄走)。
func TestRosterRenameRespectsLimit(t *testing.T) {
	g := newRosterGame(t)
	g.roster.rename = &renameState{slot: 0}
	g.rosterRenameRunes([]rune("一二三四五六七八九十百千"))
	if got := len([]rune(g.roster.rename.text)); got != town.NameMaxRunes {
		t.Errorf("輸入應該夾在 %d 個字,得到 %d", town.NameMaxRunes, got)
	}
}

// TestRosterNKeyNotSwallowedByHotkey:名冊開著時 N 是改名,不是「再開一次名冊」。
//
// ⚠ 這條擋的是派工表的坑:rosterHotkey 排在 rosterScene 前面,
// 只要它對 N 回 true,改名鍵就永遠到不了名冊 —— 而畫面上完全看不出來。
func TestRosterNKeyNotSwallowedByHotkey(t *testing.T) {
	g := newRosterGame(t)
	in := Input{Keys: []ebiten.Key{ebiten.KeyN}}
	if (rosterHotkey{g}).Handles(in) {
		t.Error("名冊開著時 N 不該被熱鍵接手")
	}
}

// TestDisbandRefusesActiveParty:不解散正在玩的那一隊。
func TestDisbandRefusesActiveParty(t *testing.T) {
	g := newRosterGame(t)
	if g.slot < 1 {
		t.Skipf("這個 fixture 沒有目前隊伍(slot=%d)", g.slot)
	}
	before := append([]original.Character(nil), g.chars...)
	g.disbandParty(g.slot)
	if !strings.Contains(g.roster.msg, "不能解散") {
		t.Errorf("解散正在玩的隊伍要擋下來,得到 %q", g.roster.msg)
	}
	for i := range g.chars {
		if g.chars[i].Party != before[i].Party {
			t.Fatalf("被擋下來卻還是動了第 %d 槽的隊伍欄", i+1)
		}
	}
}

// TestDisbandEmptyPartySaysSo:空隊伍要說「沒有隊伍 #N 可解散!」(CHARUTIL:58/59)。
func TestDisbandEmptyPartySaysSo(t *testing.T) {
	g := newRosterGame(t)
	empty := 0
	for n := 1; n <= original.GroupSlots; n++ {
		if n == g.slot {
			continue
		}
		used := false
		for i := range g.chars {
			if p, in := g.chars[i].InParty(); in && p == n {
				used = true
			}
		}
		if !used {
			empty = n
			break
		}
	}
	if empty == 0 {
		t.Skip("這份名冊每一隊都有人")
	}
	g.disbandParty(empty)
	if !strings.Contains(g.roster.msg, "可解散!") {
		t.Errorf("空隊伍要說「沒有隊伍 #N 可解散!」,得到 %q", g.roster.msg)
	}
}

// TestRenamePromptMatchesLimit:提示裡寫死的數字要跟實際上限一致。
//
// ⚠ 這條擋的是「字面對得上原版、行為卻不同」—— 提示寫 9 而程式擋在 10,
// F3 的字串檢查照樣綠燈,玩家卻能打第 10 個字。
func TestRenamePromptMatchesLimit(t *testing.T) {
	want := fmt.Sprintf("最多%d個字元", town.NameMaxRunes)
	if !strings.Contains(renamePrompt, want) {
		t.Errorf("提示 %q 與上限 %d 不一致", renamePrompt, town.NameMaxRunes)
	}
}

// ── 剩下十段接線之後的回歸(docs/re/203)──────────────────────────────

// TestFormationGridIsThreeByThree:R)eorder 畫的是 3×3 的字母格,
// 而九個成員槽就是那九格(CAMP:71/72/73)。
func TestFormationGridIsThreeByThree(t *testing.T) {
	if town.FormationSlots != 9 {
		t.Fatalf("陣型格應該是九格,得到 %d", town.FormationSlots)
	}
	for _, row := range []string{formationRowABC, formationRowDEF, formationRowGHI} {
		if got := strings.Count(row, " ") ; got == 0 {
			t.Errorf("格子圖 %q 少了對齊用的空白", row)
		}
	}
	if !strings.Contains(formationRowABC, "A B C") ||
		!strings.Contains(formationRowGHI, "G H I") {
		t.Error("格子圖的字母要照原版 A–I")
	}
}

// TestRosterHeaderDrivesColumns:資料列的欄位起點是**從表頭量出來的**,
// 不是另外寫一組數字 —— 改表頭就一定連帶改欄位。
func TestRosterHeaderDrivesColumns(t *testing.T) {
	c := rosterColumns(rosterHeader + "   隊伍")
	if len(c) != 7 {
		t.Fatalf("表頭應該量出 7 欄,得到 %d(%v)", len(c), c)
	}
	for i := 1; i < len(c); i++ {
		if c[i] <= c[i-1] {
			t.Errorf("欄位起點要遞增,第 %d 欄 %v <= 第 %d 欄 %v", i, c[i], i-1, c[i-1])
		}
	}
	if c[0] != 0 {
		t.Errorf("第一欄應該從 0 起,得到 %v", c[0])
	}
}

// TestRemoveAsksBeforeDeleting:R)emove 是不可逆的,原版在刪之前問一句
// (CHARUTIL:52)。⚠ 先前引擎按一下 R 就直接刪掉,連確認都沒有。
func TestRemoveAsksBeforeDeleting(t *testing.T) {
	g := newRosterGame(t)
	free := -1
	for i := range g.chars {
		if _, in := g.chars[i].InParty(); g.chars[i].Occupied() && !in {
			free = i
			break
		}
	}
	if free < 0 {
		// 出貨名冊裡每個有人的槽都在隊伍裡 —— 造一個不在隊伍的來測。
		g.chars[0].Party = ' '
		free = 0
	}
	g.roster.cursor = free
	g.testRunes = nil
	press(t, g, ebiten.KeyR)
	if !g.roster.removePick {
		t.Fatal("R 應該先問一句,不是直接刪")
	}
	if !g.chars[g.roster.cursor].Occupied() {
		t.Fatal("還沒回答就已經刪掉了")
	}
	if !strings.Contains(g.roster.msg, rosterRemoveAsk) {
		t.Errorf("要問「%s」,得到 %q", rosterRemoveAsk, g.roster.msg)
	}
	g.rosterRemovePick(ebiten.Key0) // 取消
	if !g.chars[g.roster.cursor].Occupied() {
		t.Error("答 0 不該刪掉")
	}
	press(t, g, ebiten.KeyR)
	g.rosterRemovePick(ebiten.KeyR) // 確認
	if g.chars[g.roster.cursor].Occupied() {
		t.Error("再按一次 R 應該刪掉")
	}
}

// TestTwoExitVariantsAreDistinct:原版的 `(0 exits)` 有兩種空格寫法,
// 分屬移除角色與解散隊伍兩個提問 —— 不要統一成一句。
func TestTwoExitVariantsAreDistinct(t *testing.T) {
	if rosterRemoveAsk == rosterDisbandAsk {
		t.Error("兩個提問的寫法在原版是不同的(CHARUTIL:52 vs 54)")
	}
}

// TestInnMenuSaysHowManyNights —— 旅店選單要講得出「可以住幾晚」。
//
// 按鍵那一支一直收 1–9(TestInnAsksHowManyNights 測的就是它),
// 但選單上寫的是「住一晚」—— 玩家照著字面按,永遠只住一晚,
// 而那不是規則的限制。⚠ **能做到 ≠ 玩家找得到。**
func TestInnMenuSaysHowManyNights(t *testing.T) {
	g := newTownGame(t, original.ShopInn, townInn)
	lines := strings.Join(g.buildingLines(g.town), "\n")
	if strings.Contains(lines, "住一晚") {
		t.Errorf("選單寫「住一晚」會讓玩家以為一次只能住一晚:\n%s", lines)
	}
	if !strings.Contains(lines, "1–9") {
		t.Errorf("選單要標出一次能住幾晚:\n%s", lines)
	}
}

// TestTavernRumorNeedsTalking —— 傳聞要按過 T 才聽得到。
//
// 選單問「你想要:T)與其他冒險者交談…」,而傳聞早就印在下面 ——
// 那句話等於騙人。⚠ **選單列了一個指令,它就得真的有事發生。**
func TestTavernRumorNeedsTalking(t *testing.T) {
	g := newTownGame(t, original.ShopTavern, townTavern)
	g.town.shop.Extra = 1
	g.rumors = map[int]string{1: "有人說北方的礦坑鬧鬼。"}

	before := strings.Join(g.buildingLines(g.town), "\n")
	if strings.Contains(before, "礦坑鬧鬼") {
		t.Errorf("還沒開口就聽到傳聞了:\n%s", before)
	}
	g.townKey(ebiten.KeyT)
	after := strings.Join(g.buildingLines(g.town), "\n")
	if !strings.Contains(after, "礦坑鬧鬼") {
		t.Errorf("按了 T 應該聽到傳聞:\n%s", after)
	}
}
