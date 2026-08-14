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

// docs/re/138 §4:傳聞 10 段、酒館索引 11 個 —— **差額本身是結論**。
//
// ⚠ 這條測試鎖住的是那個差額。哪天第 11 段被定位到,它會失敗,
// 逼人回來更新 docs/re/138 §4 與實作端的「未定位」訊息。
func TestRumorCountVsTavernIndices(t *testing.T) {
	var town []byte
	var err error
	for _, dir := range []string{"/game/sharspri", "../../../game/sharspri"} {
		if town, err = os.ReadFile(filepath.Join(dir, "TOWN.EXE")); err == nil {
			break
		}
	}
	if err != nil {
		t.Skip(err)
	}
	rumors := ExtractRumors(town)
	if len(rumors) != 10 {
		t.Errorf("抽出 %d 段傳聞,docs/re/138 §4 記 10 —— "+
			"若第 11 段被定位到了,請更新那一節與實作端的訊息", len(rumors))
	}

	shops, err := ParseShops(read(t, "TOWNDATA.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	idx := map[int]bool{}
	for _, s := range shops {
		if s.Kind == ShopTavern {
			idx[s.Extra] = true
		}
	}
	if len(idx) != 11 {
		t.Errorf("酒館的位移 36 有 %d 個相異值,docs/re/138 §4 記 11", len(idx))
	}
	missing := 0
	for i := range idx {
		if _, ok := rumors[i]; !ok {
			missing++
		}
	}
	if missing != 1 {
		t.Errorf("查不到文字的酒館索引有 %d 個,應為 1(那個差額就是未解的部分)", missing)
	}
}

// docs/re/138 §1:商店賣的是**編號範圍內**的道具,不是全部 57 件。
//
// ⚠ 檢定用「店名對得上內容」—— 名字與內容都由作者填寫,
// 而作者不會讓劍舖賣藥水。
func TestShopStockMatchesName(t *testing.T) {
	shops, err := ParseShops(read(t, "TOWNDATA.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := ParseItems(read(t, "ITEMS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	goods, wide := 0, 0
	for _, s := range shops {
		if s.Kind != ShopGoods {
			continue
		}
		goods++
		if s.First < 0 || s.Last >= len(items) || s.First > s.Last {
			t.Errorf("%s 的範圍 %d–%d 超出 0–%d", s.Name, s.First, s.Last, len(items)-1)
		}
		if s.Last-s.First+1 > 20 {
			wide++
		}
	}
	if goods != 26 {
		t.Errorf("賣道具的商店 %d 間,docs/re/138 §2 記 26", goods)
	}
	// 沒有一間店賣超過 20 件 —— 賣 57 件的那個實作會在這裡失敗
	if wide != 0 {
		t.Errorf("有 %d 間店的品項超過 20 件 —— 範圍可能沒套上", wide)
	}
	// 四種特殊建築的數量
	for k, want := range map[ShopKind]int{
		ShopHealer: 8, ShopTavern: 11, ShopInn: 10, ShopTrainer: 6,
	} {
		n := 0
		for _, s := range shops {
			if s.Kind == k {
				n++
			}
		}
		if n != want {
			t.Errorf("%v 有 %d 間,docs/re/138 §2 記 %d", k, n, want)
		}
	}
}
