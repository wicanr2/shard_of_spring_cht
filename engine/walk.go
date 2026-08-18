package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
)

// 隊伍的人形圖示。來源是 `WALKDRAW.PIC`(10 段 BASIC `DRAW` 巨集),
// 由 `cmd/convert` 渲染成 `gfx/walk/`(世界地圖色盤)與 `gfx/walk-maze/`(地城色盤)。
//
// ⚠ **這不是地形圖塊**,是畫在地形**上面**的角色 —— 世界地圖與地城共用同一份
// (`MENU.EXE` 載入後放進 COMMON,只有它含這個檔名)。
//
// 段的語意(docs/re/219,用原版四個朝向的截圖逐像素比對出來的):
//
//	段 0        紮營的帳篷
//	段 1,2      朝向 1(北)的兩個步態
//	段 3,4      朝向 2(東)
//	段 5,6      朝向 3(南)
//	段 7,8      朝向 4(西)
//	段 9        未定(不是人形)
//
// 也就是 `段 = (朝向−1)×2 + 1 + 步態`。

// WalkSegs 是人形那八段的數量(不含帳篷與段 9)。
const WalkSegs = 8

// WalkTent 是紮營時取代人形的那一段。
const WalkTent = 0

// walkArt 是一組已載入的人形圖(索引 = 段號)。
type walkArt map[int]*ebiten.Image

// loadWalk 讀 <assets>/gfx/<dir>/w0.png … w9.png。
//
// ⚠ **讀不到就回 nil**,呼叫端退回白框(不拿別的圖冒充,同 docs/spec/15 §3)。
func loadWalk(assets, dir string) walkArt {
	out := walkArt{}
	for k := 0; k <= 9; k++ {
		f, err := os.Open(filepath.Join(assets, "gfx", dir, fmt.Sprintf("w%d.png", k)))
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err == nil {
			out[k] = ebiten.NewImageFromImage(img)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// seg 回傳某個朝向、某個步態要畫哪一段;0 = 沒有對應的圖。
//
// ⚠ 朝向的編碼是 **1 北 2 東 3 南 4 西**(docs/re/71)。超出範圍就回段 1 ——
// 畫錯朝向比不畫好判斷,而不畫會讓玩家以為隊伍不見了。
func (w walkArt) seg(facing, gait int) *ebiten.Image {
	if w == nil {
		return nil
	}
	if facing < 1 || facing > 4 {
		facing = 1
	}
	return w[(facing-1)*2+1+gait%2]
}

// tent 回傳紮營用的那一段。
func (w walkArt) tent() *ebiten.Image {
	if w == nil {
		return nil
	}
	return w[WalkTent]
}

// drawWalk 在指定位置畫一張人形(或帳篷)。回傳畫了沒有 ——
// 沒畫的話呼叫端要退回白框。
func drawWalk(dst *ebiten.Image, img *ebiten.Image, px, py float64) bool {
	if img == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(layout.ArtScale, layout.ArtScale)
	op.GeoM.Translate(px, py)
	// 整數倍放大不該有插值(docs/spec/04 §1)。
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(img, op)
	return true
}
