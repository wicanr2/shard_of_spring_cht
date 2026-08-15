package save

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"shardofspring/internal/original"
)

// 驗收 1(docs/spec/18 §7):存檔往返,存 → 讀 → 全部欄位相同,含 Progress。
func TestSaveRoundTrip(t *testing.T) {
	s := &Save{
		Version: CurrentVersion,
		Active:  3,
		Progress: Progress{
			DisabledEvents: map[string][]int{"2": {204}, "51": {532, 533}},
			Tombs:          []int{701, 702},
			ClanRewarded:   true,
			MazeFile:       "51",
			MazeFacing:     3,
		},
	}
	s.Chars[0] = original.Character{
		Party: '1', Name: "灰燼", ID: 1, Race: 'H', Class: '1',
		Speed: 11, Str: 10, Int: 9, End: 9, ToHit: 0,
		MaxHP: 9, HP: 7, MaxSP: 0, SP: 0,
		Weapon: original.NotEquipped, Armor: original.NotEquipped,
		Status: 0, Level: 1, Skills: "0000000000",
		Identified: "1000000000", StatMag: 0, SkillUsed: true,
		SkillPts: 1, Exp: 1500,
	}
	s.Groups[0] = original.Group{
		Gold: 63447, Provisions: 20, Encounter: 54,
		Month: 1, Day: 1, Hour: 4, Sub: 2,
		WorldX: 8, WorldY: 8, Facing: 3,
		LightTurns: 0, VisLit: 3, VisDark: 2,
		PoolUses: 2, MazeX: 46, MazeY: 37, LightPick: original.NoLight, Fled: 0,
	}
	s.Groups[0].Members[0] = 1
	for i := 1; i < original.MemberSlots; i++ {
		s.Groups[0].Members[i] = 99
	}

	path := Path(t.TempDir(), "roundtrip")
	if err := Write(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, got) {
		t.Errorf("存檔往返後不同:\n寫入 %+v\n讀出 %+v", s, got)
	}
}

// 驗收 2:未知 version 拒絕載入,不嘗試解析,而且訊息要講清楚。
func TestReadRejectsUnknownVersion(t *testing.T) {
	path := Path(t.TempDir(), "future")
	future := &Save{Version: CurrentVersion + 1}
	if err := Write(path, future); err != nil {
		t.Fatal(err)
	}
	_, err := Read(path)
	if err == nil {
		t.Fatal("未知版本應該回錯誤,卻讀成功了")
	}
	if !strings.Contains(err.Error(), "版本") {
		t.Errorf("錯誤訊息應該講清楚是版本問題,得到:%v", err)
	}
}

// 讀不存在的檔案:呼叫端要能用 os.IsNotExist 判斷「還沒存過」。
func TestReadMissingFileIsNotExist(t *testing.T) {
	_, err := Read(Path(t.TempDir(), "nope"))
	if !os.IsNotExist(err) {
		t.Errorf("讀不存在的存檔應該回可以用 os.IsNotExist 判斷的錯誤,得到:%v", err)
	}
}

// 驗收 5:兩份存檔互不干擾 —— 存 A、存 B、讀回 A,得到 A 的狀態。
func TestTwoSavesDoNotInterfere(t *testing.T) {
	dir := t.TempDir()
	a := &Save{Version: CurrentVersion, Active: 1, Progress: Progress{MazeFile: "2"}}
	b := &Save{Version: CurrentVersion, Active: 5, Progress: Progress{MazeFile: "51"}}
	if err := Write(Path(dir, "a"), a); err != nil {
		t.Fatal(err)
	}
	if err := Write(Path(dir, "b"), b); err != nil {
		t.Fatal(err)
	}
	got, err := Read(Path(dir, "a"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Active != 1 || got.Progress.MazeFile != "2" {
		t.Errorf("讀回 a 得到 Active=%d MazeFile=%q,應該是存 a 時的值(1, \"2\"),"+
			"沒有被 b 污染", got.Active, got.Progress.MazeFile)
	}
}

// List:列出目錄下的 *.json,不含副檔名,已排序,忽略非 .json 檔。
func TestList(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"zeta.json", "alpha.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List() = %v,應為 %v(已排序、不含 .txt、不含副檔名)", got, want)
	}
}

// List 對不存在的目錄回空清單,不是錯誤 —— 那只表示還沒存過任何東西。
func TestListMissingDirIsEmpty(t *testing.T) {
	got, err := List(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("目錄不存在不該是錯誤,得到:%v", err)
	}
	if len(got) != 0 {
		t.Errorf("應該回空清單,得到 %v", got)
	}
}

// 驗收 6:匯入原版 .DAT → 角色/隊伍逐欄與現行讀檔路徑(ParseChars/ParseGroups)相同。
func TestImportMatchesDirectParse(t *testing.T) {
	dir := t.TempDir()

	chars := make([]original.Character, original.CharSlots)
	chars[0] = original.Character{Party: '1', Name: "測試角色", ID: 1, Level: 3,
		Weapon: original.NotEquipped, Armor: original.NotEquipped}
	var charBytes []byte
	for _, c := range chars {
		charBytes = append(charBytes, c.Bytes()...)
	}
	charsPath := filepath.Join(dir, "CHARS.DAT")
	if err := os.WriteFile(charsPath, charBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	groups := make([]original.Group, original.GroupSlots)
	groups[0] = original.Group{WorldX: 8, WorldY: 8, Facing: 3, Gold: 75, Provisions: 20}
	groups[0].Members[0] = 1
	for i := 1; i < original.MemberSlots; i++ {
		groups[0].Members[i] = 99
	}
	var groupBytes []byte
	for i := range groups {
		if i == 0 {
			groupBytes = append(groupBytes, groups[0].Bytes()...)
			continue
		}
		blank := make([]byte, original.GroupRecLen)
		for k := range blank {
			blank[k] = ' '
		}
		groupBytes = append(groupBytes, blank...)
	}
	groupsPath := filepath.Join(dir, "GROUPS.DAT")
	if err := os.WriteFile(groupsPath, groupBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Import(charsPath, groupsPath)
	if err != nil {
		t.Fatal(err)
	}

	wantChars, err := original.ParseChars(charBytes)
	if err != nil {
		t.Fatal(err)
	}
	wantGroups, err := original.ParseGroups(groupBytes)
	if err != nil {
		t.Fatal(err)
	}
	for i, w := range wantChars {
		if !reflect.DeepEqual(got.Chars[i], w) {
			t.Fatalf("第 %d 個角色與直接 ParseChars 的結果不同:\n%+v\n%+v", i, got.Chars[i], w)
		}
	}
	for i, w := range wantGroups {
		if !reflect.DeepEqual(got.Groups[i], w) {
			t.Fatalf("第 %d 支隊伍與直接 ParseGroups 的結果不同:\n%+v\n%+v", i, got.Groups[i], w)
		}
	}
	// Progress 全部取零值:匯入等於「一次性事件都還沒觸發」(docs/spec/18 §4)。
	if !reflect.DeepEqual(got.Progress, Progress{}) {
		t.Errorf("匯入的 Progress 應該是零值,得到 %+v", got.Progress)
	}
	if got.Active != 0 {
		t.Errorf("匯入不預設 Active,應為 0,得到 %d", got.Active)
	}
	if got.Version != CurrentVersion {
		t.Errorf("匯入的存檔版本應該是 %d,得到 %d", CurrentVersion, got.Version)
	}
}

// Import 只讀,不寫 —— 來源檔案內容應該原封不動。
func TestImportDoesNotModifySourceFiles(t *testing.T) {
	dir := t.TempDir()
	charsPath := filepath.Join(dir, "CHARS.DAT")
	groupsPath := filepath.Join(dir, "GROUPS.DAT")

	chars := make([]byte, original.CharRecLen*original.CharSlots)
	for i := range chars {
		chars[i] = ' '
	}
	groups := make([]byte, original.GroupRecLen*original.GroupSlots)
	for i := range groups {
		groups[i] = ' '
	}
	if err := os.WriteFile(charsPath, chars, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(groupsPath, groups, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Import(charsPath, groupsPath); err != nil {
		t.Fatal(err)
	}

	gotChars, err := os.ReadFile(charsPath)
	if err != nil {
		t.Fatal(err)
	}
	gotGroups, err := os.ReadFile(groupsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotChars, chars) {
		t.Error("Import 之後 CHARS.DAT 的內容變了")
	}
	if !reflect.DeepEqual(gotGroups, groups) {
		t.Error("Import 之後 GROUPS.DAT 的內容變了")
	}
}
