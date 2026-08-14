package original

import (
	"strings"
	"testing"
)

func TestFastWorldTiles(t *testing.T) {
	tiles, err := DecodeFastWorld(read(t, "FASTWRLD.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tiles) != 9 {
		t.Fatalf("FASTWRLD 解出 %d 張,規格說 9(docs/re/53 §1)", len(tiles))
	}
	// 九張的標頭都必須是 (34 bit, 17 列) = 17×17 像素。
	// docs/re/53 §1:這正是「連續排列」勝過「交錯排列」的判別依據 ——
	// 交錯會讓標頭變成別的數字。
	for i, im := range tiles {
		if w, h := im.Bounds().Dx(), im.Bounds().Dy(); w != 17 || h != 17 {
			t.Errorf("FASTWRLD 第 %d 張是 %d×%d,應為 17×17", i, w, h)
		}
	}
}

func TestWrldItemRows(t *testing.T) {
	rows := SplitPIC(read(t, "WRLDITEM.PIC"))
	// ⚠ 30 行含 7 個空行。空行**不可以被濾掉** —— 行號就是索引
	// (docs/re/54 §2、docs/re/130)。
	if len(rows) != 29 {
		t.Fatalf("WRLDITEM.PIC 有 %d 行,應為 29(去掉檔尾那個空段之後)", len(rows))
	}
	var empty []int
	for i, r := range rows {
		if strings.TrimSpace(r) == "" {
			empty = append(empty, i)
		}
	}
	want := []int{1, 4, 9, 12, 23, 24}
	if len(empty) != len(want) {
		t.Fatalf("空行 %v,應為 %v —— 數量不符表示空行被濾掉了", empty, want)
	}
	for i := range want {
		if empty[i] != want[i] {
			t.Errorf("空行位置 %v,應為 %v", empty, want)
			break
		}
	}
}

// 地形值 → 圖塊來源的對照,包含**已知沒有來源的那幾個**。
// docs/spec/05-world-scene.md §2.2:值 0、10、35–38 沒有來源。
// 把「沒有來源」也寫成測試,是為了讓之後有人補上來源時**測試會失敗**,
// 逼他回來更新規格 —— 而不是安靜地讓一個未解項目消失。
func TestWorldTileOrigin(t *testing.T) {
	for _, c := range []struct {
		v    int
		src  WorldTileSource
		idx  int
		note string
	}{
		{0, SrcNone, 0, "地圖邊界"},
		{1, SrcFastWrld, 0, "草原"},
		{9, SrcFastWrld, 8, "山"},
		{10, SrcWrldItem, 0, "第 0 行"},
		{11, SrcWrldItem, 1, "海洋 —— 對到**空行**,實際走點陣路徑(re/129 §1)"},
		{24, SrcWrldItem, 14, "地城入口"},
		{30, SrcWrldItem, 20, "城鎮"},
		{38, SrcWrldItem, 28, "最後一行"},
	} {
		src, idx := WorldTileOrigin(c.v)
		if src != c.src || (src != SrcNone && idx != c.idx) {
			t.Errorf("地形值 %d(%s):得 src=%v idx=%d,應為 src=%v idx=%d",
				c.v, c.note, src, idx, c.src, c.idx)
		}
	}
}

// 全 12,463 格都要能查到來源(即使是「沒有來源」),而且沒有來源的格數
// 必須正好是規格記的那些值。數字變了表示地圖或映射變了,要回去看。
func TestWorldMapCoverage(t *testing.T) {
	cells, err := DecodeWorldMap(read(t, "WRLDMAP.BIN"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != WorldW*WorldH {
		t.Fatalf("地圖 %d 格,應為 %d×%d", len(cells), WorldW, WorldH)
	}
	byVal := map[int]int{}
	noSrc := 0
	for _, c := range cells {
		if c>>8 != 0 {
			t.Fatalf("高位元組不為 0(值 %#04x)—— docs/formats/05 §2 說全 0", c)
		}
		v := int(c)
		byVal[v]++
		if src, _ := WorldTileOrigin(v); src == SrcNone {
			noSrc++
		}
	}
	// docs/spec/05 §2 的實測表
	for v, want := range map[int]int{
		0: 224, 11: 6933, 12: 8, 13: 1,
		30: 4, 31: 5, 32: 4, // 合計 13 = 城鎮數
		24: 3, 25: 1, 27: 6, 28: 1, // 合計 11 = 地城入口
		35: 3, 36: 10, 37: 9, 38: 14,
	} {
		if byVal[v] != want {
			t.Errorf("地形值 %d 有 %d 格,規格記 %d", v, byVal[v], want)
		}
	}
	if got := byVal[30] + byVal[31] + byVal[32]; got != 13 {
		t.Errorf("城鎮格 %d 個,應為 13(TOWNDATA.DAT 的城鎮數)", got)
	}
	if got := byVal[24] + byVal[25] + byVal[27] + byVal[28]; got != 11 {
		t.Errorf("地城入口 %d 個,應為 11(對上 MAZEDATA.BIN,docs/re/51)", got)
	}
	// 只剩值 0(地圖邊界,224 格)沒有來源。
	if noSrc != 224 {
		t.Errorf("無圖塊來源的格數 %d,應只有值 0 的 224 格", noSrc)
	}
}
