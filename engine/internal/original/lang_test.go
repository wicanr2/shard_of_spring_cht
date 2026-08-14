package original

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"shardofspring/internal/ui"
)

func transDir(t *testing.T) string {
	t.Helper()
	for _, d := range []string{"/translations", "../../../translations"} {
		if _, err := os.Stat(filepath.Join(d, "names")); err == nil {
			return d
		}
	}
	t.Skip("找不到 translations/")
	return ""
}

func readTrans(t *testing.T, rel string) []byte {
	t.Helper()
	d, err := os.ReadFile(filepath.Join(transDir(t), rel))
	if err != nil {
		t.Fatalf("讀 %s:%v", rel, err)
	}
	return d
}

// docs/spec/10 §6 驗收 1 + 2:每一筆都有譯文,而且不破格。
//
// ⚠ 欄寬預算來自**現在的版面**(docs/spec/10 §4),不是原版的限制。
// `MONSTERS.DAT` 的名稱欄是定長 16 bytes,但 remake 不寫回那個檔。
func TestTranslationsFitTheirFields(t *testing.T) {
	for _, c := range []struct {
		file, field string
		count, cols int
		what        string
	}{
		{"names/monsters.tsv", "name", 74, 24, "怪物名(戰鬥的單位列)"},
		{"names/spells.tsv", "1", 33, 24, "法術名(施法選單)"},
		{"names/items.tsv", "1", 57, 20, "道具名"},
	} {
		lang := ParseLangTSV(readTrans(t, c.file))
		missing, over := 0, 0
		for row := 0; row < c.count; row++ {
			s := lang.Get(row, c.field, "")
			if s == "" {
				missing++
				t.Errorf("%s 第 %d 筆沒有譯文", c.what, row)
				continue
			}
			if n := ui.Cols(s); n > c.cols {
				over++
				t.Errorf("%s 第 %d 筆 %q 佔 %d 欄,預算 %d",
					c.what, row, s, n, c.cols)
			}
		}
		if missing == 0 && over == 0 {
			t.Logf("%s:%d 筆全部有譯文且不破格", c.what, c.count)
		}
	}
}

// 驗收 3:87 段地城敘述折行後每行 ≤ 60 欄,最長的一段 ≤ 5 行。
//
// 覆蓋層是 60 欄 × 5 行(docs/spec/04 §3)。原文最長 207 欄,
// 中文譯文只會更短或相近 —— 這條就是要在「某一段譯得太長」時失敗。
func TestDungeonTextFitsOverlay(t *testing.T) {
	const cols, maxLines = 60, 5
	total, worst, worstID := 0, 0, 0
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 51} {
		b, err := os.ReadFile(filepath.Join(transDir(t),
			fmt.Sprintf("dungeon-text/DT%dTEXT.tsv", n)))
		if err != nil {
			continue
		}
		for id, s := range ParseDungeonTextTSV(b) {
			total++
			lines := ui.Wrap(s, cols)
			if len(lines) > worst {
				worst, worstID = len(lines), id
			}
			if len(lines) > maxLines {
				t.Errorf("DT%d 第 %d 段折成 %d 行,覆蓋層只有 %d 行:%q",
					n, id, len(lines), maxLines, s)
			}
			for i, ln := range lines {
				if ui.Cols(ln) > cols+ui.HangTolerance { // 懸掛標點允許超出一個全形字
					t.Errorf("DT%d 第 %d 段第 %d 行 %d 欄:%q", n, id, i, ui.Cols(ln), ln)
				}
			}
		}
	}
	if total != 87 {
		t.Errorf("地城敘述譯文 %d 段,docs/spec/10 §1 記 87", total)
	}
	t.Logf("最長的一段是第 %d 號,折成 %d 行(上限 %d)", worstID, worst, maxLines)
}

// 驗收 4:譯文缺漏時回原文,不是空字串。
func TestLangFallsBackToOriginal(t *testing.T) {
	l := Lang{}
	if got := l.Get(0, "name", "Orc"); got != "Orc" {
		t.Errorf("缺漏時得 %q,應回原文 %q", got, "Orc")
	}
	// 空白的譯文也算缺漏 —— 空字串在畫面上像資料壞了
	l[langKey{1, "name"}] = "   "
	if got := l.Get(1, "name", "Spider"); got != "Spider" {
		t.Errorf("全空白的譯文得 %q,應回原文", got)
	}
	var nilLang Lang
	if got := nilLang.Get(0, "name", "X"); got != "X" {
		t.Error("nil 表也要回原文")
	}
}
