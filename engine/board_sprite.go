package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
)

// 戰場單位的圖。來源是 `MONST<N>.BIN`,`cmd/convert` 轉成八格橫向 sprite sheet
// (`gfx/monst/monstNN.png`)。對應規則見 docs/re/220 與 `combat.SpriteFile`。
//
// ⚠ **只取第 0 格。** 八格是同一個生物的動畫,而「哪一格對哪個狀態」沒有讀到 ——
// 挑錯格是「看起來完全正常」的錯誤,所以先固定用第一格。

// monstArt 是已載入的單位圖(索引 = MONST 檔號)。
type monstArt map[int]*ebiten.Image

// loadMonst 讀 <assets>/gfx/monst/monstNN.png,取每張的第 0 格(17×17)。
func loadMonst(assets string) monstArt {
	out := monstArt{}
	for n := 1; n <= combat.MonstFiles; n++ {
		f, err := os.Open(filepath.Join(assets, "gfx", "monst", fmt.Sprintf("monst%02d.png", n)))
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			continue
		}
		sub, ok := img.(interface{ SubImage(image.Rectangle) image.Image })
		if !ok {
			continue
		}
		b := img.Bounds()
		w := b.Dy() // 正方形一格:寬 = 高
		if w > b.Dx() {
			w = b.Dx()
		}
		out[n] = ebiten.NewImageFromImage(
			sub.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Min.X+w, b.Min.Y+w)))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// unit 回傳某個單位要畫的圖;nil = 沒有對應的圖(呼叫端退回文字)。
func (m monstArt) unit(u combat.Unit) *ebiten.Image {
	if m == nil {
		return nil
	}
	return m[combat.SpriteFile(u)]
}
