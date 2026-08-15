// convert 把玩家自備的原版資料檔轉成引擎讀的 JSON 與 PNG。
//
// ⚠ **轉出的東西不進版控。** docs/spec/03-engine-plan.md §3:
// 數值本身是原版資料,依 CLAUDE.md §1「不散布原版資料檔」,
// 由玩家用自備的原版跑這支程式產生。
//
//	convert -in game/sharspri -out assets
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"shardofspring/internal/original"
)

func main() {
	in := flag.String("in", "game/sharspri", "原版資料夾(唯讀)")
	out := flag.String("out", "assets", "輸出資料夾")
	trans := flag.String("translations", "/translations", "譯文資料夾(docs/spec/10)")
	flag.Parse()

	if err := run(*in, *out, *trans); err != nil {
		fmt.Fprintln(os.Stderr, "轉換失敗:", err)
		os.Exit(1)
	}
}

func run(in, out, transDir string) error {
	for _, d := range []string{
		filepath.Join(out, "data"),
		filepath.Join(out, "gfx", "tiles"),
		filepath.Join(out, "gfx", "pict"),
		filepath.Join(out, "gfx", "monst"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	var report []string
	step := func(name string, n int, err error) error {
		if err != nil {
			return fmt.Errorf("%s:%w", name, err)
		}
		report = append(report, fmt.Sprintf("  %-14s %4d", name, n))
		return nil
	}

	// --- 數值表 ---------------------------------------------------------
	monsters, err := original.ParseMonsters(mustRead(in, "MONSTERS.DAT"))
	// 譯文(docs/spec/10):在**轉檔期**併進資料,執行期就沒有查表分支。
	// ⚠ 缺漏時保留原文,不要變成空字串。
	lang := loadLang(transDir)
	for i := range monsters {
		monsters[i].Name = lang.monsters.Get(i, "name", monsters[i].Name)
	}
	if err := step("monsters", len(monsters), err); err != nil {
		return err
	}
	spells, err := original.ParseSpells(mustRead(in, "SPELLS.DAT"))
	for i := range spells {
		spells[i].Name = lang.spells.Get(i, "1", spells[i].Name)
		spells[i].HitMsg = lang.spells.Get(i, "6", spells[i].HitMsg)
	}
	if err := step("spells", len(spells), err); err != nil {
		return err
	}
	items, err := original.ParseItems(mustRead(in, "ITEMS.DAT"))
	for i := range items {
		items[i].Name = lang.items.Get(i, "1", items[i].Name)
		items[i].Alias = lang.items.Get(i, "2", items[i].Alias)
	}
	if err := step("items", len(items), err); err != nil {
		return err
	}
	shops, err := original.ParseShops(mustRead(in, "TOWNDATA.DAT"))
	for i := range shops {
		if zh, ok := lang.places[shops[i].Name]; ok {
			shops[i].Name = zh
		}
		if zh, ok := lang.places[shops[i].Town]; ok {
			shops[i].Town = zh
		}
	}
	sites, serr := original.ParseTownSites(mustRead(in, "TOWNDATA.BIN"))
	if err := step("town sites", len(sites), serr); err != nil {
		return err
	}
	if err := step("shops", len(shops), err); err != nil {
		return err
	}

	for _, w := range []struct {
		name string
		v    any
	}{
		{"monsters", monsters}, {"spells", spells}, {"items", items},
		{"shops", shops}, {"towns", original.Towns(shops)}, {"townsites", sites},
	} {
		if err := writeJSON(filepath.Join(out, "data", w.name+".json"), w.v); err != nil {
			return err
		}
	}

	// --- 圖形 -----------------------------------------------------------
	// 98-byte 圖塊群(docs/formats/07 §1)。清單寫死,不用萬用字元 ——
	// 萬用字元會把之後新增的檔安靜地吃進來,而錯的檔會解成雜訊不會報錯。
	tiles := []string{
		"BORDER", "DOOR", "EXITSPOT", "FIRESTRM", "HAILSTRM",
		"LAVA", "MAZEWALL", "WATER", "WINDSTRM",
	}
	nTile := 0
	for _, t := range tiles {
		img, err := original.DecodeTile(mustRead(in, t+".BIN"))
		if err != nil {
			return fmt.Errorf("圖塊 %s:%w", t, err)
		}
		if err := writePNG(filepath.Join(out, "gfx", "tiles", strings.ToLower(t)+".png"), img); err != nil {
			return err
		}
		nTile++
	}
	if err := step("tiles", nTile, nil); err != nil {
		return err
	}

	// PICT*.BIN —— ⚠ 編號跳號(3/4/5 缺)是原始封裝就少,不是漏檔
	// (docs/formats/07 §2)。所以清單寫死。
	nPict := 0
	for _, n := range []int{1, 2, 6, 7, 8} {
		img, err := original.DecodeTile(mustRead(in, fmt.Sprintf("PICT%d.BIN", n)))
		if err != nil {
			return fmt.Errorf("PICT%d:%w", n, err)
		}
		if err := writePNG(filepath.Join(out, "gfx", "pict", fmt.Sprintf("pict%d.png", n)), img); err != nil {
			return err
		}
		nPict++
	}
	if err := step("pict", nPict, nil); err != nil {
		return err
	}

	// MONST*.BIN —— 每檔八張交錯的 17×17,輸出成一張橫向 sprite sheet。
	nMonst := 0
	for i := 1; i <= 22; i++ {
		frames, err := original.DecodeMonst(mustRead(in, fmt.Sprintf("MONST%d.BIN", i)))
		if err != nil {
			return fmt.Errorf("MONST%d:%w", i, err)
		}
		if err := writePNG(filepath.Join(out, "gfx", "monst", fmt.Sprintf("monst%02d.png", i)),
			hstrip(frames)); err != nil {
			return err
		}
		nMonst++
	}
	if err := step("monst", nMonst, nil); err != nil {
		return err
	}

	// 世界地形圖塊:依 docs/spec/05 §4 的來源表,每個有來源的地形值輸出一張
	// 17×17 PNG。⚠ **沒有來源的值(0、10、35–38)不輸出** ——
	// 引擎在執行期畫成刺眼的佔位符,而不是在這裡偷偷補一張圖。
	fast, err := original.DecodeFastWorld(mustRead(in, "FASTWRLD.BIN"))
	if err != nil {
		return fmt.Errorf("FASTWRLD:%w", err)
	}
	segs := original.SplitPIC(mustRead(in, "WRLDITEM.PIC"))
	if err := os.MkdirAll(filepath.Join(out, "gfx", "world"), 0o755); err != nil {
		return err
	}
	nWorld := 0
	for v := 0; v <= 38; v++ {
		src, idx := original.WorldTileOrigin(v)
		var img image.Image
		switch src {
		case original.SrcFastWrld:
			img = fast[idx]
		case original.SrcWrldItem:
			// ⚠ 空行代表該值不走向量路徑(docs/re/54 §2)。不輸出,
			// 讓執行期畫佔位符 —— 不要拿空巨集渲染出一張全黑的圖冒充。
			if idx >= len(segs) || strings.TrimSpace(segs[idx]) == "" {
				continue
			}
			img = original.RenderDraw(segs[idx], 17, 17)
		default:
			continue // 無來源,交給執行期畫佔位符
		}
		if err := writePNG(filepath.Join(out, "gfx", "world", fmt.Sprintf("t%02d.png", v)), img); err != nil {
			return err
		}
		nWorld++
	}
	if err := step("world tiles", nWorld, nil); err != nil {
		return err
	}

	// MAZEITEM.PIC:第 k 行 = 圖塊值 k(**偏移 0**,CONTEXT.md §6 的訂正)。
	// 空行 = 該值「用到但不繪製」—— 行 0/18/19 是空的,正好對上程式碼的清單
	// (docs/re/56 §2)。**不要把空行濾掉**,行號就是索引。
	if err := os.MkdirAll(filepath.Join(out, "gfx", "maze"), 0o755); err != nil {
		return err
	}
	mi := original.SplitPIC(mustRead(in, "MAZEITEM.PIC"))
	nMazeTile := 0
	for k, row := range mi {
		if strings.TrimSpace(row) == "" {
			continue
		}
		img := original.RenderDraw(row, 17, 17)
		if err := writePNG(filepath.Join(out, "gfx", "maze",
			fmt.Sprintf("t%02d.png", k)), img); err != nil {
			return err
		}
		nMazeTile++
	}
	if err := step("maze tiles", nMazeTile, nil); err != nil {
		return err
	}

	// 迷宮:MAZEDATA + 六個 .SQZ + 事件表 + 房間文字,全部轉成 JSON。
	entries, err := original.ParseMazeData(mustRead(in, "MAZEDATA.BIN"))
	if err := step("mazedata", len(entries), err); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(out, "data", "mazedata.json"), entries); err != nil {
		return err
	}
	nMaze, nEvent, nText := 0, 0, 0
	seenMaze, seenText := map[int]bool{}, map[int]bool{}
	for _, e := range entries {
		// ⚠ **不要跳過 (0,0) 的第 13 筆。** 它在世界地圖上沒有入口,
		// 但指向 DG51 / DT51 —— 那是地城 5 的下半層,只能從樓梯進去
		// (docs/re/60 §3)。跳過它會讓跨關卡的目的地不存在,
		// 而畫面上只會看到「走樓梯沒反應」。
		if !seenMaze[e.MazeFile] {
			seenMaze[e.MazeFile] = true
			m, err := original.DecodeSQZ(mustRead(in, fmt.Sprintf("DG%dMAZE.SQZ", e.MazeFile)))
			if err != nil {
				return fmt.Errorf("DG%d:%w", e.MazeFile, err)
			}
			if err := writeJSON(filepath.Join(out, "data",
				fmt.Sprintf("maze%d.json", e.MazeFile)), m); err != nil {
				return err
			}
			nMaze++
		}
		if !seenText[e.TextFile] {
			seenText[e.TextFile] = true
			evs, err := original.ParseEvents(mustRead(in, fmt.Sprintf("DE%dEFF.BIN", e.TextFile)))
			if err != nil {
				return fmt.Errorf("DE%d:%w", e.TextFile, err)
			}
			if err := writeJSON(filepath.Join(out, "data",
				fmt.Sprintf("events%d.json", e.TextFile)), evs); err != nil {
				return err
			}
			nEvent++
			txt := original.ParseDungeonText(mustRead(in, fmt.Sprintf("DT%dTEXT.DAT", e.TextFile)))
			// 地城敘述的譯文以 id 對應
			if zh := lang.dungeon[e.TextFile]; zh != nil {
				for id, t := range zh {
					if _, ok := txt[id]; ok {
						txt[id] = t
					}
				}
			}
			if err := writeJSON(filepath.Join(out, "data",
				fmt.Sprintf("dtext%d.json", e.TextFile)), txt); err != nil {
				return err
			}
			nText += len(txt)
		}
	}
	if err := step("mazes", nMaze, nil); err != nil {
		return err
	}
	if err := step("event tables", nEvent, nil); err != nil {
		return err
	}
	if err := step("dungeon text", nText, nil); err != nil {
		return err
	}

	// 酒館傳聞。docs/re/138 §4:TOWN.EXE 0x032C9–0x03A40 的長敘述。
	// ⚠ **找到 10 段而索引有 11 個** —— 第 11 段未定位。
	// 這裡按出現順序編 1…10,⛔ 不補第 11 段。
	rumors := original.ExtractRumors(mustRead(in, "TOWN.EXE"))
	for id, zh := range lang.rumors {
		if _, ok := rumors[id]; ok {
			rumors[id] = zh
		}
	}
	if err := writeJSON(filepath.Join(out, "data", "rumors.json"), rumors); err != nil {
		return err
	}
	if err := step("rumors", len(rumors), nil); err != nil {
		return err
	}

	// 存檔的初始複本。**原版目錄唯讀**(CLAUDE.md §8),而引擎要寫存檔,
	// 所以把 CHARS.DAT / GROUPS.DAT 複製到可寫的資產目錄。
	// ⚠ 只在目標不存在時複製 —— 重跑轉檔不該把玩家的進度洗掉。
	if err := os.MkdirAll(filepath.Join(out, "save"), 0o755); err != nil {
		return err
	}
	nSave := 0
	for _, n := range []string{"CHARS.DAT", "GROUPS.DAT"} {
		dst := filepath.Join(out, "save", n)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.WriteFile(dst, mustRead(in, n), 0o644); err != nil {
			return err
		}
		nSave++
	}
	if err := step("save seed", nSave, nil); err != nil {
		return err
	}

	// 世界地圖:轉成灰階 PNG 供目視,數值另存 JSON。
	cells, err := original.DecodeWorldMap(mustRead(in, "WRLDMAP.BIN"))
	if err != nil {
		return fmt.Errorf("WRLDMAP:%w", err)
	}
	if err := writeJSON(filepath.Join(out, "data", "worldmap.json"), map[string]any{
		"w": original.WorldW, "h": original.WorldH, "cells": cells,
	}); err != nil {
		return err
	}
	if err := step("worldmap", len(cells), nil); err != nil {
		return err
	}

	fmt.Println("轉換完成 →", out)
	for _, l := range report {
		fmt.Println(l)
	}
	fmt.Println("\n⚠ 這些是原版資料,不要進版控(CLAUDE.md §1)。")
	return nil
}

// hstrip 把 n 張同尺寸的圖橫向接成一張。
func hstrip(fr []*image.Paletted) *image.Paletted {
	if len(fr) == 0 {
		return image.NewPaletted(image.Rect(0, 0, 1, 1), original.Palette)
	}
	w, h := fr[0].Bounds().Dx(), fr[0].Bounds().Dy()
	dst := image.NewPaletted(image.Rect(0, 0, w*len(fr), h), original.Palette)
	for i, f := range fr {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetColorIndex(i*w+x, y, f.ColorIndexAt(x, y))
			}
		}
	}
	return dst
}

func mustRead(dir, name string) []byte {
	d, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		fmt.Fprintln(os.Stderr, "讀不到原版檔:", err)
		fmt.Fprintln(os.Stderr, "請用 -in 指定自備原版的 sharspri 資料夾。")
		os.Exit(1)
	}
	return d
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	return enc.Encode(v)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// langTables 是所有譯文表。docs/spec/10 §2。
type langTables struct {
	monsters, spells, items original.Lang
	dungeon                 map[int]map[int]string // DT 檔號 → id → 譯文
	places                  map[string]string      // 城鎮／商店名:原文 → 譯文
	rumors                  map[int]string         // 酒館傳聞(docs/re/138 §4)
}

// loadLang 讀 translations/ 底下的 TSV。
//
// ⚠ **讀不到就回空表,不是錯誤** —— 譯文缺漏時 Lang.Get 會保留原文,
// 所以沒有譯文的環境仍然跑得起來(英文版)。
func loadLang(dir string) langTables {
	lt := langTables{dungeon: map[int]map[int]string{}}
	read := func(rel string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return nil
		}
		return b
	}
	lt.places = original.ParsePlaceTSV(read("source/towndata.tsv"))
	lt.rumors = original.ParseDungeonTextTSV(read("module-text/TOWN-rumors.tsv"))
	lt.monsters = original.ParseLangTSV(read("names/monsters.tsv"))
	lt.spells = original.ParseLangTSV(read("names/spells.tsv"))
	lt.items = original.ParseLangTSV(read("names/items.tsv"))
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 51} {
		if b := read(fmt.Sprintf("dungeon-text/DT%dTEXT.tsv", n)); b != nil {
			lt.dungeon[n] = original.ParseDungeonTextTSV(b)
		}
	}
	return lt
}
