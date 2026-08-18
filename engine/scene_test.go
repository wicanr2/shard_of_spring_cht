package main

// T3:每個場景至少一條「按下去有反應」,外加釘住派工表的順序
// (docs/spec/14-remake-worklist.md §8)。
//
// 這一份擋的是**接線**,不是規則:規則層測得再密,某個場景的按鍵沒接上、
// 或優先序被改動,遊戲就是按不動,而所有規則測試照樣綠
// (§8 的 `H)unt` 實例:規則早就寫好且有測試,鍵盤上按不到)。
//
// ⚠ 全部走 `g.Update()`,不直呼 handler —— 直呼測到的是規則,不是接線。

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/ui"
	"shardofspring/internal/world"
)

// TestInputChainOrder 釘住優先序。**這張表就是規則本身** ——
// 順序決定「城鎮裡按 N 會不會開名冊」「技能點畫面開著時城鎮會不會漏接」,
// 而那些行為在任何單點測試裡都看不出來。
//
// ⚠ 改動順序要連這條一起改,而且要在 commit 訊息裡說明為什麼 ——
// 不要因為「測試紅了」就把期望值改成現況。
func TestInputChainOrder(t *testing.T) {
	want := []string{
		"overlay",
		"inspect",
		"cast-cursor", "combat-potion", "cast-sp", "cast-menu", "use-menu", "combat",
		"save-as", "skill-alloc",
		"roster-hotkey",
		"create", "roster", "shell",
		"town", "maze-prompt", "maze", "world",
	}
	g := &Game{}
	var got []string
	for _, s := range g.inputChain() {
		got = append(got, s.Name())
	}
	if len(got) != len(want) {
		t.Fatalf("場景數量 %d,期望 %d:%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 個場景是 %q,期望 %q\n實際順序:%v", i, got[i], want[i], got)
		}
	}
}

// TestDrawOrderCoversEveryDrawer 確認每個「畫得出東西」的場景都在圖層表裡。
//
// ⚠ 判準不是「兩張表要一樣」—— 它們本來就不一樣(輸入是優先序、繪圖是圖層)。
// 判準是**沒有場景被漏掉**:漏掉的那個畫面會整個不見,而輸入照樣進得去,
// 於是「按鍵有反應但看不到」。
func TestDrawOrderCoversEveryDrawer(t *testing.T) {
	// 這三個沒有自己的圖層:游標與藥劑提示畫在別人身上、熱鍵不是畫面、
	// 外殼接管整個畫布(走 Game.Draw 的另一條路)。
	noDraw := map[string]bool{
		"cast-cursor": true, "combat-potion": true, "roster-hotkey": true, "shell": true,
	}
	g := &Game{}
	inDraw := map[string]bool{}
	for _, s := range g.drawOrder() {
		inDraw[s.Name()] = true
	}
	for _, s := range g.inputChain() {
		if noDraw[s.Name()] {
			continue
		}
		if !inDraw[s.Name()] {
			t.Errorf("場景 %q 吃得到按鍵卻不在 drawOrder 裡 —— 畫面會整個不見", s.Name())
		}
	}
}

// newPlayingGame 讀真資產、走完外殼流程,回傳一個正在世界地圖上的遊戲。
func newPlayingGame(t *testing.T) *Game {
	t.Helper()
	g := newIntegrationGame(t, assetsCopy(t))
	press(t, g, ebiten.KeyEnter)  // 標題 → 主選單
	press(t, g, ebiten.KeyL)      // → 匯入入口(還沒有具名存檔)
	press(t, g, ebiten.KeyY)      // 匯入 → 隊伍選擇
	press(t, g, ebiten.KeyDigit5) // 選 PARTY #5 → 遊戲中
	if g.shell.mode != shellPlaying {
		t.Fatalf("沒進到遊戲:%v(msg=%q)", g.shell.mode, g.shell.msg)
	}
	return g
}

// newFightingGame 回傳一個正在打仗的遊戲(隊員面前站著一隻怪物)。
func newFightingGame(t *testing.T) *Game {
	t.Helper()
	g := newKillingBlowGame(t)
	g.field.Units[combat.MonsterBase].HP = 99 // 別讓第一擊就結束
	return g
}

// sceneCase 是一個場景的一條驗收:擺好狀態 → 按一個鍵 → 有反應。
type sceneCase struct {
	scene string // 對應 inputChain 的 Name(),也是「哪個場景該接手」
	setup func(t *testing.T) *Game
	keys  []ebiten.Key
	// want 回傳空字串表示通過,否則是失敗訊息。
	want func(g *Game) string
}

func TestT3EveryScenePressReacts(t *testing.T) {
	cases := []sceneCase{
		{
			scene: "overlay",
			setup: func(t *testing.T) *Game {
				return &Game{overlay: "測試用敘述", noSrc: map[int]bool{}}
			},
			keys: []ebiten.Key{ebiten.KeySpace},
			want: func(g *Game) string {
				if g.overlay != "" {
					return "按鍵之後覆蓋層應該關掉"
				}
				return ""
			},
		},
		{
			scene: "cast-cursor",
			setup: func(t *testing.T) *Game {
				g := newFightingGame(t)
				g.cursor = &castCursor{x: 5, y: 5}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyK}, // 手冊 p.34 的菱形配置:K = 右
			want: func(g *Game) string {
				if g.cursor == nil {
					return "游標不該被關掉"
				}
				if g.cursor.x != 6 {
					return "K 應該把游標往右移一格"
				}
				return ""
			},
		},
		{
			// CMBT:179–192 的單位檢視面板。⚠ **唯讀** —— 這條驗的是
			// 「↓ 真的換了一個單位」,不是「按了有事發生」。
			scene: "inspect",
			setup: func(t *testing.T) *Game {
				g := newFightingGame(t)
				if !g.openInspect() {
					t.Fatal("戰場上應該有單位可以看")
				}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyDown},
			want: func(g *Game) string {
				if g.inspect == nil {
					return "面板不該被關掉"
				}
				if g.inspect.idx == 0 {
					return "↓ 應該換到下一個有名字的單位"
				}
				return ""
			},
		},
		{
			// CMBT:101「 # SP ? 」—— 投入幾點法力那一步。
			// ⚠ 這一步先前不存在(固定投一級),而投入量會改變威力與狀態強度。
			scene: "cast-sp",
			setup: func(t *testing.T) *Game {
				g := newFightingGame(t)
				g.castUnit = combat.PartyBase
				g.castSP = &castSPState{spell: original.Spell{Name: "FIREBALL", UnitCost: 2}}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyDigit4},
			want: func(g *Game) string {
				if g.castSP == nil {
					return "投入點數那一步不該被關掉"
				}
				if g.castSP.input != "4" {
					return "按 4 應該把 4 收進輸入緩衝"
				}
				return ""
			},
		},
		{
			// DOS 版自己的字串是「Use arrow keys to position cursor.」(CMBT:110/111),
			// 手冊 p.34 的 I/J/K/M 是 Apple II 版 —— **兩組都要能動**。
			// ⚠ 這一條是 F3 對字串時翻出來的:引擎原本只收 I/J/K/M,
			// 而畫面上看不出「方向鍵沒反應」是漏接還是設計。
			scene: "cast-cursor",
			setup: func(t *testing.T) *Game {
				g := newFightingGame(t)
				g.cursor = &castCursor{x: 5, y: 5}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyRight},
			want: func(g *Game) string {
				if g.cursor == nil {
					return "游標不該被關掉"
				}
				if g.cursor.x != 6 {
					return "方向鍵右應該把游標往右移一格"
				}
				return ""
			},
		},
		{
			scene: "combat-potion",
			setup: func(t *testing.T) *Game {
				g := newFightingGame(t)
				g.useUnit = g.actor
				g.combatPotion = &potionPrompt{slot: 0, stage: 1}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyT}, // T)oss:丟給隊友 → 進選目標
			want: func(g *Game) string {
				if g.combatPotion == nil {
					return "T 不該直接關掉這個子流程"
				}
				if g.combatPotion.stage != 2 {
					return "T 應該進到「選目標」那一步"
				}
				return ""
			},
		},
		{
			scene: "cast-menu",
			setup: func(t *testing.T) *Game {
				g := newFightingGame(t)
				g.castList = []original.Spell{{Index: 0, Name: "火球"}}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyEscape},
			want: func(g *Game) string {
				if len(g.castList) != 0 {
					return "ESC 應該關掉施法選單"
				}
				return ""
			},
		},
		{
			scene: "use-menu",
			setup: func(t *testing.T) *Game {
				g := newFightingGame(t)
				g.useList = []useEntry{{slot: 0, name: "藥水"}}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyEscape},
			want: func(g *Game) string {
				if len(g.useList) != 0 {
					return "ESC 應該關掉道具選單"
				}
				return ""
			},
		},
		{
			scene: "combat",
			setup: func(t *testing.T) *Game { return newFightingGame(t) },
			keys:  []ebiten.Key{ebiten.KeyEnter}, // 結束這一輪
			want: func(g *Game) string {
				if g.field == nil {
					return "戰鬥不該消失"
				}
				if g.field.Round < 1 {
					return "Enter 應該結束這一輪、開下一回合"
				}
				return ""
			},
		},
		{
			scene: "save-as",
			setup: func(t *testing.T) *Game {
				g := newPlayingGame(t)
				g.openSaveAs()
				return g
			},
			keys: []ebiten.Key{ebiten.KeyEscape},
			want: func(g *Game) string {
				if g.saveAs != nil {
					return "ESC 應該關掉另存新檔畫面"
				}
				return ""
			},
		},
		{
			scene: "skill-alloc",
			setup: func(t *testing.T) *Game {
				g := newPlayingGame(t)
				g.openSkillAlloc(1, 0, nil)
				return g
			},
			keys: []ebiten.Key{ebiten.KeyDigit7},
			want: func(g *Game) string {
				if g.skillAlloc == nil {
					return "數字鍵不該關掉技能點畫面"
				}
				if g.skillAlloc.input != "7" {
					return "數字鍵應該進到輸入緩衝"
				}
				return ""
			},
		},
		{
			scene: "roster-hotkey",
			setup: func(t *testing.T) *Game { return newPlayingGame(t) },
			keys:  []ebiten.Key{ebiten.KeyN},
			want: func(g *Game) string {
				if g.roster == nil || !g.roster.open {
					return "遊戲中按 N 應該開名冊"
				}
				return ""
			},
		},
		{
			scene: "create",
			setup: func(t *testing.T) *Game {
				g := newPlayingGame(t)
				g.openCreate()
				return g
			},
			keys: []ebiten.Key{ebiten.KeyEscape},
			want: func(g *Game) string {
				if g.create != nil {
					return "選種族那一步按 ESC 應該離開創角"
				}
				return ""
			},
		},
		{
			scene: "roster",
			setup: func(t *testing.T) *Game {
				g := newPlayingGame(t)
				g.openRoster()
				return g
			},
			keys: []ebiten.Key{ebiten.KeyDown},
			want: func(g *Game) string {
				if g.roster == nil || !g.roster.open {
					return "方向鍵不該關掉名冊"
				}
				if g.roster.cursor == 0 {
					return "方向鍵應該移動名冊游標"
				}
				return ""
			},
		},
		{
			scene: "shell",
			setup: func(t *testing.T) *Game {
				g := &Game{noSrc: map[int]bool{}}
				g.openTitle()
				return g
			},
			keys: []ebiten.Key{ebiten.KeyEnter},
			want: func(g *Game) string {
				if g.shell.mode != shellMainMenu {
					return "標題按任意鍵應該進主選單"
				}
				return ""
			},
		},
		{
			scene: "town",
			setup: func(t *testing.T) *Game {
				g := newPlayingGame(t)
				s := g.townSites[0]
				if !g.enterTown(s.X, s.Y) {
					t.Fatalf("進不了城鎮 (%d,%d)", s.X, s.Y)
				}
				return g
			},
			// ⚠ 營地是 Z 不是 C —— 建築清單的字母 A 起算,C 已經被
			// 第三間建築佔走(town_scene.go 的說明)。
			keys: []ebiten.Key{ebiten.KeyZ},
			want: func(g *Game) string {
				if g.town == nil {
					return "城鎮不該關掉"
				}
				if g.town.mode != townCamp {
					return "Z 應該開營地"
				}
				return ""
			},
		},
		{
			scene: "maze-prompt",
			setup: func(t *testing.T) *Game {
				g := newPlayingGame(t)
				if !g.openPrompt(maze.Trigger{Kind: maze.KindGem}) {
					t.Fatal("寶石謎題開不起來")
				}
				return g
			},
			keys: []ebiten.Key{ebiten.KeyB}, // 四色之一(docs/re/155 §1)
			want: func(g *Game) string {
				if g.prompt == nil {
					return "顏色鍵不該關掉謎題"
				}
				if g.prompt.input == "" {
					return "顏色鍵應該進到答案緩衝"
				}
				return ""
			},
		},
		{
			scene: "maze",
			setup: func(t *testing.T) *Game {
				g := newPlayingGame(t)
				e := g.mazeData[0]
				if !g.enterMaze(e.WorldX, e.WorldY) {
					t.Fatalf("進不了迷宮 (%d,%d)", e.WorldX, e.WorldY)
				}
				g.overlay, g.prompt = "", nil // 進場敘述會蓋在上面,先收掉
				return g
			},
			// ⛔ **ESC 不再離開地城。** 原版的唯一出路是走到格陣列界外
			// (docs/re/147),ESC 曾經是本引擎多開的第二條出路 —— 一道作弊門。
			// 現在這一格測的是 `C` 紮營(原版在地城裡也能紮營,
			// docs/spec/14 §12-B:`Making Camp..` 在 MAZEMOVE.EXE 裡)。
			keys: []ebiten.Key{ebiten.KeyC},
			want: func(g *Game) string {
				if g.level == nil {
					return "C 不該離開地城"
				}
				if g.town == nil || g.town.mode != townCamp {
					return "地城裡按 C 應該紮營"
				}
				if g.town.wild {
					return "地城裡紮營算室內,wild 應該是 false(打不到獵)"
				}
				return ""
			},
		},
		{
			scene: "world",
			setup: func(t *testing.T) *Game { return newPlayingGame(t) },
			keys:  []ebiten.Key{ebiten.KeyUp},
			want: func(g *Game) string {
				// 朝向不同時只轉身(docs/spec/05 §6),所以兩者其一有變就算有反應。
				if g.party.Facing != world.North {
					return "方向鍵應該至少讓隊伍轉向"
				}
				return ""
			},
		},
	}

	// 每個場景都要有一條 —— 少一條就是有場景沒人看著。
	covered := map[string]bool{}
	for _, c := range cases {
		covered[c.scene] = true
	}
	for _, s := range (&Game{}).inputChain() {
		if !covered[s.Name()] {
			t.Errorf("場景 %q 沒有任何「按下去有反應」的驗收", s.Name())
		}
	}

	for _, c := range cases {
		t.Run(c.scene, func(t *testing.T) {
			g := c.setup(t)
			in := Input{Keys: c.keys}
			// 先確認**真的是這個場景接手** —— 否則測到的是別人的反應。
			var taker string
			for _, s := range g.inputChain() {
				if s.Handles(in) {
					taker = s.Name()
					break
				}
			}
			if taker != c.scene {
				t.Fatalf("這一格由 %q 接手,不是 %q —— 擺設狀態或優先序有問題", taker, c.scene)
			}
			press(t, g, c.keys...)
			if msg := c.want(g); msg != "" {
				t.Error(msg)
			}
		})
	}
}

// ── C3:提示列與訊息面板的收斂(docs/spec/14 §4)────────────────────────

// TestEveryScenePrompt 確認每個場景都說得出自己的指令提示,而且**放得進提示列**。
//
// ⚠ 畫到框外的字**照樣印得出來**,只是壓在別的面板上 —— 沒有錯誤訊息,
// 也沒有測試會紅,只有一團看不懂的畫面。戰鬥那一行就這樣壓著隊伍面板很久,
// 是拍截圖才看見的。
func TestEveryScenePrompt(t *testing.T) {
	// 這兩個沒有自己的提示列:熱鍵不是畫面;外殼的提示由各自的畫面直接畫
	// (shell 接管整個畫布,走 Game.Draw 的另一條路)。
	noPrompt := map[string]bool{"roster-hotkey": true, "shell": true}

	// 動態提示要在**有狀態**時量,不然量到的是預設分支。
	g := newFightingGame(t)
	g.combatPotion = &potionPrompt{slot: 0, stage: 2}
	g.create = &createState{step: stepAdjust, picked: map[int]bool{}}
	g.prompt = &mazePrompt{kind: promptGem}

	for _, s := range g.inputChain() {
		p := s.Prompt()
		if noPrompt[s.Name()] {
			if p != "" {
				t.Errorf("場景 %q 不該有提示列,得到 %q", s.Name(), p)
			}
			continue
		}
		if p == "" {
			t.Errorf("場景 %q 沒有指令提示 —— 玩家不知道這個畫面能按什麼", s.Name())
			continue
		}
		if n := ui.Cols(p); n > layout.PromptCols {
			t.Errorf("場景 %q 的提示列 %d 欄,超過 %d —— 會畫到框外壓住別的面板:%q",
				s.Name(), n, layout.PromptCols, p)
		}
	}
}

// TestPromptFollowsActiveScene 釘住「提示列跟著實際接手者走」。
//
// ⚠ 這是 C3 的重點:提示列先前自己寫了一份 switch 去猜現在是哪個畫面,
// 與 Update() 的派工是**兩份各自演化的判斷** —— 戰鬥那一行因此停在
// 「空白鍵:推進一回合」,而那個鍵在 M10 就移除了。
func TestPromptFollowsActiveScene(t *testing.T) {
	g := newPlayingGame(t)
	if got := g.activeScene().Name(); got != "world" {
		t.Fatalf("世界地圖上應該是 world 接手,得到 %q", got)
	}
	s := g.townSites[0]
	if !g.enterTown(s.X, s.Y) {
		t.Fatalf("進不了城鎮 (%d,%d)", s.X, s.Y)
	}
	if got := g.activeScene().Name(); got != "town" {
		t.Fatalf("進城之後應該是 town 接手,得到 %q", got)
	}
	if g.activeScene().Prompt() != (townScene{g}).Prompt() {
		t.Error("提示列沒有跟著接手的場景走")
	}
	// 覆蓋層蓋上來時,提示列也要跟著換 —— 它吃掉所有按鍵。
	g.overlay = "測試用敘述"
	if got := g.activeScene().Name(); got != "overlay" {
		t.Errorf("覆蓋層開著時應該由 overlay 接手,得到 %q", got)
	}
}

// TestCombatPromptIsNotStale 直接擋住那一行舊文字回來。
func TestCombatPromptIsNotStale(t *testing.T) {
	g := newFightingGame(t)
	p := (combatScene{g}).Prompt()
	for _, gone := range []string{"空白鍵", "固定投一級"} {
		if strings.Contains(p, gone) {
			t.Errorf("戰鬥提示列還寫著已經移除的操作 %q:%q", gone, p)
		}
	}
	for _, want := range []string{"A：攻擊", "Enter"} {
		if !strings.Contains(p, want) {
			t.Errorf("戰鬥提示列缺了 %q:%q", want, p)
		}
	}
}

// TestArchwayNeedsPositionAndFacing:拱門敘述要**站對格子而且朝南**
// (docs/re/198)。⚠ 少了朝向那一項,從南邊走上來也會看到「你面前是……」,
// 而那時玩家背對著拱門 —— 那個錯誤不會顯示成錯誤。
func TestArchwayNeedsPositionAndFacing(t *testing.T) {
	g := newPlayingGame(t)
	for _, c := range []struct {
		x, y int
		f    world.Facing
		want bool
	}{
		{archwayX, archwayY, world.South, true},
		{archwayX, archwayY, world.North, false},
		{archwayX, archwayY, world.East, false},
		{archwayX + 1, archwayY, world.South, false},
		{archwayX, archwayY - 1, world.South, false},
	} {
		g.overlay = ""
		g.party.X, g.party.Y, g.party.Facing = c.x, c.y, c.f
		g.archwayCheck()
		if got := g.overlay == archwayText; got != c.want {
			t.Errorf("(%d,%d) 朝向 %d:印了 %v,期望 %v", c.x, c.y, c.f, got, c.want)
		}
	}
}

// 天色會縮世界地圖的視野(docs/re/213):半徑 = 生效能見度。
//
// ⚠ 這一條測的是**規則存在**,不是像素:深夜(能見度 1)只該畫 3×3,
// 而 9×9 的四角必定落在半徑外。
func TestWorldViewShrinksAtNight(t *testing.T) {
	const half = layout.ViewTiles / 2
	for _, tc := range []struct{ hour, want int }{
		{10, 4}, // 白天 → 9×9 全開
		{17, 2}, // 傍晚 → 5×5
		{20, 1}, // 深夜 → 3×3
	} {
		if got := world.Daylight(tc.hour); got != tc.want {
			t.Fatalf("時 %d 的天色是 %d,想要 %d", tc.hour, got, tc.want)
		}
		// 視野邊長 = 2×半徑 + 1,而畫布是 9×9 —— 半徑 4 才畫得滿
		if side := 2*tc.want + 1; side > layout.ViewTiles {
			t.Errorf("時 %d 的視野 %d 超過畫布 %d", tc.hour, side, layout.ViewTiles)
		}
	}
	// 半徑 1 時,9×9 的角落(0,0)一定在範圍外 —— 迴圈裡的裁切條件用的就是這個
	if !(0 < half-1) {
		t.Error("半徑 1 時左上角應該被裁掉,裁切條件寫錯了")
	}
}

// P) 隊伍資訊:唯一看得到日、月與能見度的地方(docs/spec/14 §12-C)。
func TestPartyInfoScreen(t *testing.T) {
	g := newPlayingGame(t)
	press(t, g, partyInfoKey)
	if g.overlay == "" {
		t.Fatal("按 P 沒有開資訊頁")
	}
	for _, want := range []string{"時", "日", "能見度"} {
		if !strings.Contains(g.overlay, want) {
			t.Errorf("資訊頁少了 %q:%q", want, g.overlay)
		}
	}
}

// 月份名只有 12 個,而時鐘的月份範圍是 1–21。
//
// ⚠ 超出 12 要**回空字串**,不要用 (m-1)%12 繞回去湊一個名字 ——
// 那會產生一個看起來合理的假答案,而原版第 13 個月印什麼**未查**。
func TestMonthNameStopsAtTwelve(t *testing.T) {
	if monthName(1) == "" || monthName(12) == "" {
		t.Error("1–12 月都該有名字")
	}
	for _, m := range []int{0, 13, 21, 99, -1} {
		if got := monthName(m); got != "" {
			t.Errorf("第 %d 月不該有名字,拿到 %q", m, got)
		}
	}
}
