package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/rules"
)

// 腳本戰鬥(A5 結局的最後一塊,docs/spec/17-scripted-fights.md)的測試。
// §5 六條驗收,逐條對應下面的 Test:
//
//	1 TestScriptedFight533ComposesByName
//	2 TestScriptedFight204IsOneHillGiant
//	3 TestRandomEncounterStillWorks + TestFireTriggerFallsBackForUnscriptedTarget
//	4 TestScriptedBossFightReachesEndingScreen
//	5 TestSecondFightDoesNotInheritScriptedMonsters
//	6(全部不開視窗)靠 g.testKeys,見 shell_test.go 開頭的說明,這裡沿用同一套。

// newScriptedFightTestGame 建一個只給腳本戰鬥測試用的最小 Game——
// 不碰存檔、不碰完整資產載入,只給 M4/M10 戰鬥系統需要的欄位(docs/spec/07)。
//
// 三隻怪物的索引與名字照 docs/re/180 §3:10 = Hill Giant、53 = Great Dragon、
// 71 = Siriadne !。其餘索引留空(零值,Tier 0)—— 只用來讓隨機遭遇測試
// 有候選可挑,名字用不到。
func newScriptedFightTestGame(t *testing.T) *Game {
	t.Helper()
	monsters := make([]original.Monster, 72)
	monsters[10] = original.Monster{Index: 10, Name: "Hill Giant", Speed: 8, HPDie: 30, ToHit: 8, Armor: 2, Tier: 3}
	monsters[53] = original.Monster{Index: 53, Name: "Great Dragon", Speed: 15, HPDie: 60, ToHit: 15, Armor: 4, Tier: 10}
	monsters[71] = original.Monster{Index: 71, Name: "Siriadne !", Speed: 18, HPDie: 80, ToHit: 18, Armor: 5, Tier: 13}

	g := &Game{
		monsters: monsters,
		members: []original.Character{
			{Party: '1', Name: "灰燼", ID: 1, Class: '1', Race: 'H',
				Speed: 10, Str: 10, ToHit: 10, MaxHP: 50, HP: 50, Level: 5,
				Weapon: original.NotEquipped, Armor: original.NotEquipped},
		},
		items: map[int]combat.Item{},
		rand:  combat.NewRand(1),
		noSrc: map[int]bool{},
	}
	return g
}

// killAllMonsters 直接把場上怪物打到 0 血。測試不驗傷害公式(那是
// internal/combat 自己的測試範圍),只驗「打贏之後」的流程,所以不必真的
// 推進回合去磨血。
func killAllMonsters(f *combat.Field) {
	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		f.Units[i].HP = 0
	}
}

// monsterNames 收集場上還有名字的怪物槽(Build() 只填到怪物數,
// 其餘槽留零值,Name == "")。
//
// ⚠ 隨機遭遇的候選裡有不少 fixture 佔位怪(Tier 0、Name ""),
// 這個函式看不到它們 —— 數怪物隻數要用 monsterCount。
func monsterNames(f *combat.Field) []string {
	var names []string
	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		if f.Units[i].Name != "" {
			names = append(names, f.Units[i].Name)
		}
	}
	return names
}

// monsterCount 數場上活著的怪物槽,不看名字(Build() 一定會給存活怪物
// HP ≥ 1,見 internal/combat/build.go 的 r.Roll(max1(m.HPDie)))——
// 用來驗證「隻數」時不會被空名字的 fixture 佔位怪騙到。
func monsterCount(f *combat.Field) int {
	n := 0
	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		if f.Units[i].Alive() {
			n++
		}
	}
	return n
}

// ── 驗收 1:目標 533 → 3 隻,兩隻 Great Dragon 一隻 Siriadne !────────────
//
// ⚠ 用**名字**比對,不是用索引數量 —— 索引寫錯 ±1 也湊得出「3 隻」
// (docs/spec/17 §5 驗收 1 的但書)。

func TestScriptedFight533ComposesByName(t *testing.T) {
	g := newScriptedFightTestGame(t)
	if !g.startScriptedCombat(maze.TargetFinalBoss) {
		t.Fatalf("533 應該有腳本清單")
	}
	names := monsterNames(g.field)
	if len(names) != 3 {
		t.Fatalf("應該有 3 隻,得到 %d(%v)", len(names), names)
	}
	dragons, siriadne := 0, 0
	for _, n := range names {
		switch n {
		case "Great Dragon":
			dragons++
		case "Siriadne !":
			siriadne++
		default:
			t.Errorf("出現了預期外的名字 %q", n)
		}
	}
	if dragons != 2 {
		t.Errorf("Great Dragon 應該 2 隻,得到 %d(%v)", dragons, names)
	}
	if siriadne != 1 {
		t.Errorf("Siriadne ! 應該 1 隻,得到 %d(%v)", siriadne, names)
	}
	if !g.bossFight {
		t.Errorf("目標 533 應該設 g.bossFight = true(docs/spec/15 §7)")
	}
}

// ── 驗收 2:目標 204 → 1 隻 Hill Giant ──────────────────────────────────

func TestScriptedFight204IsOneHillGiant(t *testing.T) {
	g := newScriptedFightTestGame(t)
	if !g.startScriptedCombat(maze.TargetPriest) {
		t.Fatalf("204 應該有腳本清單")
	}
	names := monsterNames(g.field)
	if len(names) != 1 || names[0] != "Hill Giant" {
		t.Errorf("204 應該是 1 隻 Hill Giant,得到 %v", names)
	}
	if g.bossFight {
		t.Errorf("204 不是最終戰,不該動到 g.bossFight")
	}
}

// ── 驗收 3:沒有腳本清單 → 呼叫端退回隨機遭遇,原本那條路不能被弄壞 ───────

func TestStartScriptedCombatFalseForUnknownTarget(t *testing.T) {
	g := newScriptedFightTestGame(t)
	if g.startScriptedCombat(999) {
		t.Errorf("沒有腳本清單的目標應該回 false")
	}
	if g.field != nil {
		t.Errorf("回 false 時不該留下半成品的 g.field")
	}
}

func TestFireTriggerFallsBackForUnscriptedTarget(t *testing.T) {
	g := newScriptedFightTestGame(t)
	// 目前 maze.special() 只會把 204/533 標成 KindScript,999 不會真的
	// 從迷宮事件表產生 —— 這裡直接建一個 Trigger 來驗 fireTrigger 那段
	// 查表失敗的分支本身是不是「資料驅動」(之後加第三個目標只需要在
	// rules.ScriptedFight 加一列,不必動這段程式碼)。
	g.fireTrigger(maze.Trigger{Kind: maze.KindScript, Number: 999, Text: "測試用文字"})
	if g.field != nil {
		t.Errorf("沒有腳本清單時不該開戰")
	}
	if len(g.warnings) == 0 {
		t.Errorf("應該留下一則警告,明講這個目標沒有腳本怪物清單")
	}
}

// 對照組:既有的隨機遭遇路徑(docs/re/169)不能被這次改動動到。
func TestRandomEncounterStillWorks(t *testing.T) {
	g := newScriptedFightTestGame(t)
	g.party.X, g.party.Y = 10, 10 // rules.WorldZone → 1(西北起始區)
	if !g.startCombat() {
		t.Fatalf("隨機遭遇應該照舊觸發")
	}
	if g.bossFight {
		t.Errorf("隨機遭遇不該動到 g.bossFight")
	}
	if n := monsterCount(g.field); n != 1 {
		t.Errorf("隨機遭遇一律只挑 1 隻(startCombat 既有行為),得到 %d", n)
	}
}

// ── 驗收 4:打贏 533 → 結局畫面 → 回主選單(這條路徑現在真的走得到)──────

func TestScriptedBossFightReachesEndingScreen(t *testing.T) {
	g := newScriptedFightTestGame(t)
	g.shell = &shellState{mode: shellPlaying}

	g.fireTrigger(maze.Trigger{Kind: maze.KindScript, Number: maze.TargetFinalBoss, Text: "最終決戰的描述"})
	if g.overlay == "" {
		t.Fatalf("前提不成立:DT 文字應該先進 g.overlay(docs/re/161 §3 先印文字再做事)")
	}
	if g.field == nil {
		t.Fatalf("前提不成立:533 應該立刻建好這場戰鬥")
	}
	if !g.bossFight {
		t.Fatalf("前提不成立:g.bossFight 應該被設成 true")
	}

	// 關掉 DT 文字的敘述覆蓋層。
	g.testKeys = []ebiten.Key{ebiten.KeyEnter}
	mustUpdate(t, g)
	if g.overlay != "" {
		t.Fatalf("前提不成立:敘述覆蓋層應該已經關閉")
	}
	if g.field == nil {
		t.Fatalf("前提不成立:關掉覆蓋層後戰鬥應該還在")
	}

	killAllMonsters(g.field)
	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.field != nil {
		t.Errorf("戰鬥結束後 g.field 應該清空")
	}
	if g.shell.mode != shellEnding {
		t.Fatalf("打贏 533 應該進結局畫面,得到 %v", g.shell.mode)
	}
	if g.bossFight {
		t.Errorf("bossFight 應該在轉場後重置")
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEnter}
	mustUpdate(t, g)
	if g.shell.mode != shellMainMenu {
		t.Errorf("按鍵後應回主選單,得到 %v", g.shell.mode)
	}
}

// ── 驗收 5:連續兩場戰鬥,第一場是腳本的,第二場沒有殘留 ─────────────────
//
// ⚠ 這條是針對 docs/spec/17 §4 最後一列那個刻意差異的:原版的 `ds:372C`
// 沒找到重置點,引擎選擇「每次開戰前都是全新配置」而不是沿用/清空一個
// 共用陣列 —— 光靠「第一場開得出 3 隻」測不出這個決定有沒有真的生效,
// 一定要接著開第二場才驗得出來。

func TestSecondFightDoesNotInheritScriptedMonsters(t *testing.T) {
	g := newScriptedFightTestGame(t)
	g.shell = &shellState{mode: shellPlaying}

	// 第一場:腳本戰鬥 533,3 隻怪。
	if !g.startScriptedCombat(maze.TargetFinalBoss) {
		t.Fatalf("前提不成立:533 應該開戰")
	}
	if len(monsterNames(g.field)) != 3 {
		t.Fatalf("前提不成立:第一場應該有 3 隻")
	}
	killAllMonsters(g.field)
	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.field != nil {
		t.Fatalf("前提不成立:第一場結束後 g.field 應該清空")
	}

	// 第二場:腳本戰鬥 204,只有 1 隻。
	if !g.startScriptedCombat(maze.TargetPriest) {
		t.Fatalf("204 應該開戰")
	}
	names := monsterNames(g.field)
	if len(names) != 1 || names[0] != "Hill Giant" {
		t.Errorf("第二場應該只有 1 隻 Hill Giant,不該殘留第一場的怪物,得到 %v", names)
	}
}

// 對照組:第二場走隨機遭遇時同樣不該混進第一場腳本戰鬥的怪物。
func TestSecondRandomFightDoesNotInheritScriptedMonsters(t *testing.T) {
	g := newScriptedFightTestGame(t)
	g.shell = &shellState{mode: shellPlaying}

	if !g.startScriptedCombat(maze.TargetFinalBoss) {
		t.Fatalf("前提不成立:533 應該開戰")
	}
	killAllMonsters(g.field)
	g.testKeys = []ebiten.Key{ebiten.KeyEscape}
	mustUpdate(t, g)
	if g.field != nil {
		t.Fatalf("前提不成立:第一場結束後 g.field 應該清空")
	}

	g.party.X, g.party.Y = 10, 10 // 西北起始區,index 10/53/71 的階級在這裡都不合格
	if !g.startCombat() {
		t.Fatalf("隨機遭遇應該照舊觸發")
	}
	if n := monsterCount(g.field); n != 1 {
		t.Fatalf("前提不成立:隨機遭遇應該只有 1 隻,得到 %d", n)
	}
	for i := combat.MonsterBase; i < combat.MonsterBase+combat.MonsterMax; i++ {
		if n := g.field.Units[i].Name; n == "Great Dragon" || n == "Siriadne !" || n == "Hill Giant" {
			t.Errorf("隨機遭遇不該挑到只有腳本戰鬥才會出現的怪物,得到 %q"+
				"(索引 10/53/71 的階級在區域 1 都超出 ±1 容忍,見 rules.ZoneAccepts)", n)
		}
	}
}

// ── 打贏 204 顯示祭司的後續文字(docs/spec/17 §3)───────────────────────
//
// ⚠ 祝福的實際效果沒有讀到(docs/re/161 §4 只讀到這句字串)——
// 這裡只驗「文字有沒有出現」,不驗任何屬性加成。

func TestPriestFightVictoryShowsBlessingText(t *testing.T) {
	g := newScriptedFightTestGame(t)
	if !g.startScriptedCombat(maze.TargetPriest) {
		t.Fatalf("前提不成立:204 應該開戰")
	}
	killAllMonsters(g.field)
	g.endTurn()

	found := false
	for _, line := range g.field.Log {
		if line == rules.PriestBlessing {
			found = true
		}
	}
	if !found {
		t.Errorf("打贏 204 之後,戰鬥紀錄裡應該有祭司的祝福文字,得到 %v", g.field.Log)
	}
}

// 對照組:打贏 533 不該混進祭司的文字(204 與 533 不能互相污染)。
func TestBossFightVictoryDoesNotShowPriestText(t *testing.T) {
	g := newScriptedFightTestGame(t)
	if !g.startScriptedCombat(maze.TargetFinalBoss) {
		t.Fatalf("前提不成立:533 應該開戰")
	}
	killAllMonsters(g.field)
	g.endTurn()

	for _, line := range g.field.Log {
		if line == rules.PriestBlessing {
			t.Errorf("533 的勝利不該混進祭司的祝福文字,得到 %v", g.field.Log)
		}
	}
}

// 對照組:一般隨機遭遇的勝利也不該混進祭司的文字
// (composition 相同時 —— 隨機遇到剛好也是 1 隻 Hill Giant —— 更要驗)。
func TestRandomVictoryDoesNotShowPriestTextEvenWithHillGiant(t *testing.T) {
	g := newScriptedFightTestGame(t)
	g.field = combat.Build(g.members, []original.Monster{g.monsters[10]}, g.items, g.rand)
	g.field.Place()
	g.field.ResetPoints(&g.points)
	g.actor = g.firstActor()
	g.field.Log = append(g.field.Log, "遭遇 Hill Giant！") // startCombat() 既有的格式,不是 rules.PriestEncounterMark

	killAllMonsters(g.field)
	g.endTurn()

	for _, line := range g.field.Log {
		if line == rules.PriestBlessing {
			t.Errorf("隨機遇到的 Hill Giant 不該被誤判成祭司事件,得到 %v", g.field.Log)
		}
	}
}
