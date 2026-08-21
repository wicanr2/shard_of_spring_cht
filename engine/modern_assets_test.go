package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"io/fs"
	"testing"
)

// alphaExtent 量實際可見像素，不把固定的 68×68 畫布誤當成角色尺寸。
func alphaExtent(im image.Image) (int, int) {
	b := im.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := im.At(x, y).RGBA()
			if a != 0 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x+1 > maxX {
					maxX = x + 1
				}
				if y+1 > maxY {
					maxY = y + 1
				}
			}
		}
	}
	return maxX - minX, maxY - minY
}

func modernPNGExtent(t *testing.T, name string) (int, int) {
	t.Helper()
	b, err := fs.ReadFile(modernFS, name)
	if err != nil {
		t.Fatal(err)
	}
	im, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return alphaExtent(im)
}

func TestModernSpritesUseTheirTileHeight(t *testing.T) {
	for n := 1; n <= 22; n++ {
		name := fmt.Sprintf("assets/modern/monst/monst%02d.png", n)
		w, h := modernPNGExtent(t, name)
		if w < 56 && h < 56 {
			t.Errorf("%s 可見範圍只有 %d×%d，怪物沒有充分利用方格", name, w, h)
		}
	}
	for n := 1; n <= 8; n++ {
		name := fmt.Sprintf("assets/modern/walk/w%d.png", n)
		_, h := modernPNGExtent(t, name)
		if h < 56 {
			t.Errorf("%s 可見高度只有 %d，主角比例或去背可能已退化", name, h)
		}
	}
}

func TestModernWorldAutoAtlasesAreComplete(t *testing.T) {
	a := loadModernWorldAuto()
	for i, im := range append(a.grass[:], a.ocean[:]...) {
		if im == nil || im.Bounds().Dx() != 272 || im.Bounds().Dy() != 272 {
			t.Fatalf("自然地形材質頁 %d 尺寸 = %v，want 272×272", i, im)
		}
	}
	for mask := 0; mask < 16; mask++ {
		for kind, im := range map[string]image.Image{
			"forest": a.forest[mask], "coast": a.coast[mask],
		} {
			if im == nil || im.Bounds().Dx() != 272 || im.Bounds().Dy() != 272 {
				t.Fatalf("%s mask %02d 材質頁不完整", kind, mask)
			}
		}
	}
}
