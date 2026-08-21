package main

import (
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"image"
	_ "image/png"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
)

// 現代主題是本重製版自己的可散布資產，與玩家轉出的原版 assets/ 分開內嵌。
// 這樣三平台包不會誤帶原版資料，玩家也不必重新轉檔才能看到現代 UI。

//go:embed assets/modern/title.png
var modernTitlePNG []byte

//go:embed assets/modern/world/*.png assets/modern/walk/*.png assets/modern/maze/*.png assets/modern/monst/*.png assets/modern/combat/*.png
var modernFS embed.FS

func loadModernTitle() *ebiten.Image {
	im, _, err := image.Decode(bytes.NewReader(modernTitlePNG))
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(im)
}

func loadModernSet(dir, prefix string, first, last int) map[int]*ebiten.Image {
	out := map[int]*ebiten.Image{}
	for n := first; n <= last; n++ {
		name := fmt.Sprintf("assets/modern/%s/%s%d.png", dir, prefix, n)
		b, err := fs.ReadFile(modernFS, name)
		if err != nil {
			continue
		}
		im, _, err := image.Decode(bytes.NewReader(b))
		if err == nil {
			out[n] = ebiten.NewImageFromImage(im)
		}
	}
	return out
}

func loadModernWorld() map[int]*ebiten.Image {
	out := map[int]*ebiten.Image{}
	for n := 1; n <= 38; n++ {
		b, err := fs.ReadFile(modernFS, fmt.Sprintf("assets/modern/world/t%02d.png", n))
		if err != nil {
			continue
		}
		im, _, err := image.Decode(bytes.NewReader(b))
		if err == nil {
			out[n] = ebiten.NewImageFromImage(im)
		}
	}
	return out
}

func loadModernWalk() walkArt { return walkArt(loadModernSet("walk", "w", 0, 9)) }

func loadModernMaze() map[int]*ebiten.Image {
	out := map[int]*ebiten.Image{}
	for n := 1; n <= 32; n++ {
		b, err := fs.ReadFile(modernFS, fmt.Sprintf("assets/modern/maze/t%02d.png", n))
		if err != nil {
			continue
		}
		im, _, err := image.Decode(bytes.NewReader(b))
		if err == nil {
			out[n] = ebiten.NewImageFromImage(im)
		}
	}
	return out
}

func loadModernMonst() monstArt {
	out := monstArt{}
	for n := 1; n <= 22; n++ {
		b, err := fs.ReadFile(modernFS, fmt.Sprintf("assets/modern/monst/monst%02d.png", n))
		if err != nil {
			continue
		}
		im, _, err := image.Decode(bytes.NewReader(b))
		if err == nil {
			out[n] = ebiten.NewImageFromImage(im)
		}
	}
	return out
}

func loadModernCombat() combatArt {
	out := combatArt{}
	for n := 0; n <= 8; n++ {
		b, err := fs.ReadFile(modernFS, fmt.Sprintf("assets/modern/combat/c%d.png", n))
		if err != nil {
			continue
		}
		im, _, err := image.Decode(bytes.NewReader(b))
		if err == nil {
			out[n] = ebiten.NewImageFromImage(im)
		}
	}
	return out
}
