package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docs/spec/17-scripted-fights.md §5 驗收 1/2:目標 533 / 204 的組成。
// 這裡驗兩件事:資料表本身,以及**表的索引對不對得上真實的 MONSTERS.DAT**
// (最後那一條測試)。引擎有沒有照表走則在 engine/scripted_fight_test.go 驗。

func TestScriptedFightComposition(t *testing.T) {
	cases := []struct {
		target int
		want   []int
	}{
		{204, []int{10}},
		{533, []int{53, 53, 71}},
	}
	for _, c := range cases {
		got, ok := ScriptedFight[c.target]
		if !ok {
			t.Fatalf("目標 %d 應該有腳本清單", c.target)
		}
		if len(got) != len(c.want) {
			t.Fatalf("目標 %d 應有 %d 隻,得到 %d(%v)", c.target, len(c.want), len(got), got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("目標 %d 第 %d 槽應為 %d,得到 %d", c.target, i, c.want[i], got[i])
			}
		}
	}
}

// docs/spec/17 §4:索引是 0-based,與背包道具編號同一慣例。
// 這裡鎖住三個具體值,防止之後有人「順手」±1「修正」。
func TestScriptedFightIndicesAreZeroBased(t *testing.T) {
	if got := ScriptedFight[204][0]; got != 10 {
		t.Errorf("Hill Giant 應該是 0-based 第 10 列,得到 %d", got)
	}
	if got := ScriptedFight[533][2]; got != 71 {
		t.Errorf("Siriadne ! 應該是 0-based 第 71 列,得到 %d", got)
	}
}

// 沒有腳本清單的目標要查不到(呼叫端才知道要退回警告,而不是誤開一場戰鬥)。
func TestScriptedFightUnknownTargetMisses(t *testing.T) {
	if _, ok := ScriptedFight[999]; ok {
		t.Errorf("999 不應該有腳本清單")
	}
}

func TestPriestBlessingAndMarkAreDistinctText(t *testing.T) {
	if PriestBlessing == "" {
		t.Errorf("PriestBlessing 不應該是空字串")
	}
	if PriestEncounterMark == "" {
		t.Errorf("PriestEncounterMark 不應該是空字串")
	}
	if PriestBlessing == PriestEncounterMark {
		t.Errorf("開場的識別字串跟後續的祝福文字不應該是同一句")
	}
}

// 把 ScriptedFight 的索引釘在**真實的 MONSTERS.DAT** 上。
//
// ⚠ **為什麼需要這一條**:`engine/scripted_fight_test.go` 的 fixture 自己把
// 名字種在索引 10/53/71,所以它驗得到「引擎有照表走」,**驗不到「表本身對不對」**。
// 表若哪天被改成 54,那份 fixture 只會看到一個空名字 —— 而這裡會直接說出
// 「第 54 列是 Great Wyvern,不是 Great Dragon」。
//
// 索引與名字的出處:docs/re/180 §3(反組譯)+ docs/re/179(通關紀錄),
// 兩條獨立證據鏈得到同一組。
//
// ⚠ 原版不進版控(CLAUDE.md §1),所以**找不到就 skip** ——
// 與 internal/original/tables_test.go 的 gameDir 同一個做法。
func TestScriptedFightIndicesMatchRealMonstersDat(t *testing.T) {
	var dir string
	for _, d := range []string{"/game/sharspri", "../../../game/sharspri"} {
		if _, err := os.Stat(filepath.Join(d, "MONSTERS.DAT")); err == nil {
			dir = d
			break
		}
	}
	if dir == "" {
		t.Skip("找不到原版 game/sharspri —— 這一項沒有被驗證")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "MONSTERS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	// 定長 36 bytes/筆,前 16 bytes 是名稱(docs/formats/03)。
	// ⚠ 這裡**不經** internal/original 的解析器 —— 直接讀 bytes,
	// 這樣連解析器改壞了也擋得住。
	const recSize, nameLen = 36, 16
	name := func(i int) string {
		return strings.TrimRight(string(raw[i*recSize:i*recSize+nameLen]), " \x00")
	}
	if got := len(raw) / recSize; got != 74 {
		t.Fatalf("怪物數 %d,規格說 74(docs/formats/03)", got)
	}
	for _, c := range []struct {
		target int
		want   []string
	}{
		{204, []string{"Hill Giant"}},
		{533, []string{"Great Dragon", "Great Dragon", "Siriadne !"}},
	} {
		idx, ok := ScriptedFight[c.target]
		if !ok {
			t.Errorf("目標 %d 不在表裡", c.target)
			continue
		}
		if len(idx) != len(c.want) {
			t.Errorf("目標 %d 有 %d 隻,應為 %d 隻", c.target, len(idx), len(c.want))
			continue
		}
		for k, i := range idx {
			if got := name(i); got != c.want[k] {
				t.Errorf("目標 %d 第 %d 隻:索引 %d 在 MONSTERS.DAT 是 %q,應為 %q",
					c.target, k+1, i, got, c.want[k])
			}
		}
	}
}
