package main

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/save"
)

// 存檔格式(docs/spec/18-save-format.md)的測試。§7 九條驗收:
//
//	1 TestSaveJSONRoundTripPreservesEverything(internal/save 已經測過格式本身,
//	  這裡另外測 engine 端組出來的 *save.Save 也一樣往返)
//	2 internal/save/save_test.go 的 TestReadRejectsUnknownVersion
//	3 TestOneTimeEventDoesNotComeBack(三段:觸發 → 離開再回來 → 存檔讀回)
//	4 TestSaveInMazeResumesAtSameCell
//	5 internal/save/save_test.go 的 TestTwoSavesDoNotInterfere
//	6 internal/save/save_test.go 的 TestImportMatchesDirectParse
//	7 全部靠 `go test ./...` 整套跑,internal/original 的往返測試沒有被動過
//	8 TestGameDirNeverWritten
//	9(全部不開視窗)沿用 shell_test.go / scripted_fight_test.go 的 g.testKeys 機制

// newSaveTestGame 建一個給存檔測試用的 Game:一支隊員(灰燼)在世界座標
// (1,1),那一格是 DG2 的入口,DG2 只有一格事件 —— 目標 204(山丘巨人
// 挾持祭司),與 rules.ScriptedFight[204] = {10}(Hill Giant)對應
// (docs/re/180 §3)。assets/save 兩份 .DAT 與 assets/data 三份迷宮 JSON
// 都寫進 t.TempDir(),不碰真正的 game/sharspri/。
func newSaveTestGame(t *testing.T) (*Game, string) {
	t.Helper()
	dir := t.TempDir()
	saveWorkDir := filepath.Join(dir, "save")
	dataDir := filepath.Join(dir, "data")
	for _, d := range []string{saveWorkDir, dataDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	chars := make([]original.Character, original.CharSlots)
	chars[0] = original.Character{
		Party: '1', Name: "灰燼", ID: 1, Class: '1', Race: 'H',
		Speed: 10, Str: 10, ToHit: 10, MaxHP: 50, HP: 50, Level: 5,
		Weapon: original.NotEquipped, Armor: original.NotEquipped,
	}
	var charBytes []byte
	for _, c := range chars {
		charBytes = append(charBytes, c.Bytes()...)
	}
	if err := os.WriteFile(filepath.Join(saveWorkDir, "CHARS.DAT"), charBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	parsedChars, err := original.ParseChars(charBytes)
	if err != nil {
		t.Fatal(err)
	}

	occupied := original.Group{
		WorldX: 1, WorldY: 1, Facing: 3,
		Gold: 75, Provisions: 20, Encounter: 54,
		Month: 1, Day: 1, Hour: 4, Sub: 2,
		VisLit: 3, VisDark: 2, LightPick: original.NoLight,
	}
	occupied.Members[0] = 1
	for i := 1; i < len(occupied.Members); i++ {
		occupied.Members[i] = 99
	}
	blank := make([]byte, original.GroupRecLen)
	for i := range blank {
		blank[i] = ' '
	}
	var groupBytes []byte
	groupBytes = append(groupBytes, occupied.Bytes()...)
	for i := 0; i < original.GroupSlots-1; i++ {
		groupBytes = append(groupBytes, blank...)
	}
	savePath := filepath.Join(saveWorkDir, "GROUPS.DAT")
	if err := os.WriteFile(savePath, groupBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// DG2 的入口(docs/formats/06):世界座標 (1,1),進去落在 Major5/Minor5,
	// 事件檔是 DE2EFF(TextFile=2)。
	writeFixtureJSON(t, filepath.Join(dataDir, "maze2.json"),
		original.Maze{Majors: 10, Cells: make([]int, 10*original.MazeRows)})
	writeFixtureJSON(t, filepath.Join(dataDir, "events2.json"),
		[]original.Event{{Major: 5, Minor: 5, Dir: 0, Target: maze.TargetPriest, DestMinor: 0}})
	writeFixtureJSON(t, filepath.Join(dataDir, "dtext2.json"),
		map[int]string{maze.TargetPriest: "'Thank the Gods you have come to rescue me!'"})

	monsters := make([]original.Monster, 11)
	monsters[10] = original.Monster{Index: 10, Name: "Hill Giant", Speed: 8, HPDie: 30, ToHit: 8, Armor: 2, Tier: 3}

	g := &Game{
		assets:    dir,
		savePath:  savePath,
		chars:     parsedChars,
		noSrc:     map[int]bool{},
		monsters:  monsters,
		items:     map[int]combat.Item{},
		rand:      combat.NewRand(1),
		mazeData:  []original.MazeEntry{{WorldX: 1, WorldY: 1, StartMajor: 5, StartMinor: 5, Facing: 1, TextFile: 2, MazeFile: 2}},
		mazeTiles: map[int]*ebiten.Image{},
	}
	g.openTitle()
	return g, dir
}

func writeFixtureJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── 驗收 3:一次性事件不再復活(三段全部要驗)──────────────────────────

func TestOneTimeEventDoesNotComeBack(t *testing.T) {
	g, _ := newSaveTestGame(t)
	if err := g.loadParty(1); err != nil {
		t.Fatal(err)
	}
	g.shell.mode = shellPlaying

	// ── 段 1:觸發 ──────────────────────────────────────────────────
	if !g.enterMaze(1, 1) {
		t.Fatalf("前提不成立:(1,1) 應該是 DG2 的入口")
	}
	trig := maze.Scan(g.level.events, g.mazeState, g.level.text)
	if trig.Kind != maze.KindScript || trig.Number != maze.TargetPriest {
		t.Fatalf("前提不成立:一進迷宮就該站在目標 204 上,得到 %+v", trig)
	}
	g.fireTrigger(trig)
	if g.overlay == "" {
		t.Fatalf("前提不成立:DT 文字應該先進 g.overlay(docs/re/161 §3)")
	}
	if g.field == nil {
		t.Fatalf("前提不成立:204 應該立刻建好腳本戰鬥")
	}

	g.testKeys = []ebiten.Key{ebiten.KeyEnter} // 關掉敘述覆蓋層
	mustUpdate(t, g)
	if g.field == nil {
		t.Fatalf("前提不成立:關掉覆蓋層後戰鬥應該還在")
	}

	killAllMonsters(g.field)
	g.testKeys = []ebiten.Key{ebiten.KeyEscape} // 戰鬥已分勝負,ESC 離開
	mustUpdate(t, g)
	if g.field != nil {
		t.Fatalf("前提不成立:戰鬥結束後 g.field 應該清空")
	}
	if got := g.level.events[0].Major; got != maze.DisabledCoord {
		t.Errorf("打贏之後這個目標應該立刻作廢(Major=%d),得到 %d", maze.DisabledCoord, got)
	}
	if len(g.disabledEvents["2"]) != 1 || g.disabledEvents["2"][0] != maze.TargetPriest {
		t.Errorf("g.disabledEvents[\"2\"] 應該記著 204,得到 %v", g.disabledEvents)
	}

	// ── 段 2:離開迷宮 → 走回來 → 不再觸發 ─────────────────────────────
	g.level = nil // 離開(main.go 的 ESC-in-maze / Left 結果做的就是這件事)
	if !g.enterMaze(1, 1) {
		t.Fatalf("重新進入 DG2 應該成功")
	}
	if got := g.level.events[0].Major; got != maze.DisabledCoord {
		t.Errorf("重新載入後這個目標應該仍然是作廢狀態(loadLevel 要重套 g.disabledEvents),"+
			"Major=%d,應為 %d", got, maze.DisabledCoord)
	}
	trig2 := maze.Scan(g.level.events, g.mazeState, g.level.text)
	if trig2.Kind != maze.KindNone {
		t.Errorf("走回原本觸發的格子不該再觸發,得到 %+v", trig2)
	}

	// ── 段 3:存檔 → 讀回 → 仍然不觸發(同時驗收 4:回到迷宮同一格)──────
	g.mazeState.Major, g.mazeState.Minor = 6, 5 // 模擬「已經往東走了一格」
	if err := g.save(); err != nil {
		t.Fatalf("存檔失敗:%v", err)
	}

	g2, _ := newSaveTestGame(t) // 全新的 Game,模擬重開程式
	g2.assets = g.assets
	g2.savePath = g.savePath
	g2.hydrateFromSave() // 讀 g.save() 剛寫的 saves/party.json
	if err := g2.loadParty(1); err != nil {
		t.Fatalf("重新載入第 1 隊失敗:%v", err)
	}
	g2.resumeMaze(1)

	if g2.level == nil {
		t.Fatalf("驗收 4:讀回存檔後應該回到迷宮裡,g2.level 卻是 nil")
	}
	if g2.mazeState.Major != 6 || g2.mazeState.Minor != 5 {
		t.Errorf("驗收 4:應該回到存檔時的同一格 (6,5),得到 (%d,%d)",
			g2.mazeState.Major, g2.mazeState.Minor)
	}
	if got := g2.level.events[0].Major; got != maze.DisabledCoord {
		t.Errorf("驗收 3 段 3:存檔讀回後這個目標仍然應該是作廢狀態,Major=%d,應為 %d",
			got, maze.DisabledCoord)
	}
	origSpot := maze.State{Major: 5, Minor: 5, Facing: 1}
	if trig3 := maze.Scan(g2.level.events, origSpot, g2.level.text); trig3.Kind != maze.KindNone {
		t.Errorf("驗收 3 段 3:存檔讀回後原本觸發的格子不該再觸發,得到 %+v", trig3)
	}
}

// ── 補充:墓與氏族謎題的進度也要跟著存檔往返(docs/spec/18 §3.2)──────────

func TestTombsAndClanRewardedRoundTrip(t *testing.T) {
	g, _ := newSaveTestGame(t)
	if err := g.loadParty(1); err != nil {
		t.Fatal(err)
	}
	g.tombs = map[int]bool{701: true, 703: true}
	g.clanRewarded = true
	if err := g.save(); err != nil {
		t.Fatal(err)
	}

	g2, _ := newSaveTestGame(t)
	g2.assets, g2.savePath = g.assets, g.savePath
	g2.hydrateFromSave()

	if !g2.tombs[701] || !g2.tombs[703] || len(g2.tombs) != 2 {
		t.Errorf("墓的進度應該往返成功,得到 %v", g2.tombs)
	}
	if !g2.clanRewarded {
		t.Error("clanRewarded 應該往返成功,得到 false")
	}
}

// ── 驗收 2 的 engine 端對照:未知版本的存檔要讓 hydrateFromSave 留下警告,
// 不是安靜地套用一份解析失敗的資料。────────────────────────────────────

func TestHydrateFromSaveWarnsOnUnknownVersion(t *testing.T) {
	g, dir := newSaveTestGame(t)
	saveDir := filepath.Join(dir, "saves")
	future := &save.Save{Version: save.CurrentVersion + 1, Active: 1}
	if err := save.Write(save.Path(saveDir, save.DefaultName), future); err != nil {
		t.Fatal(err)
	}
	beforeChars := append([]original.Character(nil), g.chars...)

	g.hydrateFromSave()

	if len(g.warnings) == 0 {
		t.Error("未知版本的存檔應該留下警告")
	}
	for i, c := range g.chars {
		if !reflect.DeepEqual(c, beforeChars[i]) {
			t.Fatalf("拒絕載入的存檔不該動到 g.chars,第 %d 筆變了", i)
		}
	}
}

// ── 驗收 8:game/sharspri/ 底下一個 byte 都沒被改 ────────────────────────
//
// ⚠ 這條測試**要有原版檔案才驗得了**——沒有的話用 t.Skip 明講跳過,
// 不能讓「沒跑到」看起來像「通過」(同 internal/original/tables_test.go
// 的 gameDir 慣例)。這裡驗的是唯一會經手真正 game/sharspri/ 路徑的新程式碼:
// save.Import(呼叫端傳真正的路徑進去)。

func saveTestGameDir(t *testing.T) string {
	t.Helper()
	for _, d := range []string{"/game/sharspri", "../game/sharspri"} {
		if _, err := os.Stat(filepath.Join(d, "CHARS.DAT")); err == nil {
			return d
		}
	}
	t.Skip("找不到原版 game/sharspri —— 這一項沒有被驗證")
	return ""
}

func hashDir(t *testing.T, dir string) map[string][32]byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][32]byte{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = sha256.Sum256(b)
	}
	return out
}

func TestGameDirNeverWritten(t *testing.T) {
	dir := saveTestGameDir(t)
	before := hashDir(t, dir)

	// 唯一會經手真正 game/sharspri/ 路徑的新程式碼:save.Import 讀
	// CHARS.DAT/GROUPS.DAT(唯讀)。順便走一次 hydrateFromSave/save() 的
	// 整條路徑,但目標目錄一律是 t.TempDir(),不是 dir。
	s, err := save.Import(filepath.Join(dir, "CHARS.DAT"), filepath.Join(dir, "GROUPS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := save.Write(save.Path(tmp, "imported"), s); err != nil {
		t.Fatal(err)
	}
	if _, err := save.Read(save.Path(tmp, "imported")); err != nil {
		t.Fatal(err)
	}

	after := hashDir(t, dir)
	if len(before) != len(after) {
		t.Fatalf("game/sharspri/ 底下的檔案數變了:%d → %d", len(before), len(after))
	}
	for name, want := range before {
		if got := after[name]; got != want {
			t.Errorf("game/sharspri/%s 的內容變了 —— CLAUDE.md §8 規定唯讀", name)
		}
	}
}
