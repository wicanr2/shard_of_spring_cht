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
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "轉換失敗:", err)
		os.Exit(1)
	}
}

func run(in, out string) error {
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
	if err := step("monsters", len(monsters), err); err != nil {
		return err
	}
	spells, err := original.ParseSpells(mustRead(in, "SPELLS.DAT"))
	if err := step("spells", len(spells), err); err != nil {
		return err
	}
	items, err := original.ParseItems(mustRead(in, "ITEMS.DAT"))
	if err := step("items", len(items), err); err != nil {
		return err
	}
	shops, err := original.ParseShops(mustRead(in, "TOWNDATA.DAT"))
	if err := step("shops", len(shops), err); err != nil {
		return err
	}

	for _, w := range []struct {
		name string
		v    any
	}{
		{"monsters", monsters}, {"spells", spells}, {"items", items},
		{"shops", shops}, {"towns", original.Towns(shops)},
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
