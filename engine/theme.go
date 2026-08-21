package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/layout"
)

// visualMode 與配樂模式分開。原版是零值，設定損壞時也會失敗即關閉地
// 回到原版，不會讓重製增補悄悄變成預設行為。
type visualMode int

const (
	visualOriginal visualMode = iota
	visualStorybook
	visualModeCount
)

func parseVisualMode(s string) visualMode {
	if s == "storybook" {
		return visualStorybook
	}
	return visualOriginal
}

func (m visualMode) key() string {
	if m == visualStorybook {
		return "storybook"
	}
	return "original"
}

func (m visualMode) String() string {
	if m == visualStorybook {
		return "手繪冒險書"
	}
	return "原版"
}

var storybook = struct {
	back, panel, view, ink, rule, accent color.RGBA
}{
	back:   color.RGBA{0x1d, 0x16, 0x10, 0xff},
	panel:  color.RGBA{0xc4, 0xad, 0x7a, 0xff},
	view:   color.RGBA{0xb3, 0x9a, 0x68, 0xff},
	ink:    color.RGBA{0x2b, 0x1c, 0x12, 0xff},
	rule:   color.RGBA{0x69, 0x45, 0x28, 0xff},
	accent: color.RGBA{0x6f, 0x2f, 0x4d, 0xff},
}

func (g *Game) cycleVisualMode() string {
	g.visualMode = visualMode((int(g.visualMode) + 1) % int(visualModeCount))
	g.saveConfig()
	return "視覺主題:" + g.visualMode.String()
}

func (g *Game) screenBackground() color.Color {
	if g.visualMode == visualStorybook {
		return storybook.back
	}
	return cgaBlack
}

func (g *Game) applyThemeText() {
	if g.panel == nil {
		return
	}
	c := color.Color(cgaWhite)
	if g.visualMode == visualStorybook {
		c = storybook.ink
	}
	g.panel.Fill = c
	if g.overlayFont != nil {
		g.overlayFont.Fill = c
	}
	if g.titleFont != nil {
		g.titleFont.Fill = c
	}
}

func (g *Game) drawPanelBase(dst *ebiten.Image, rc layout.Rect, view bool) {
	if g.visualMode != visualStorybook {
		return
	}
	fill := storybook.panel
	if view {
		fill = storybook.view
	}
	vector.DrawFilledRect(dst, float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H), fill, false)
}

func (g *Game) drawPanelFrame(dst *ebiten.Image, rc layout.Rect) {
	if g.visualMode != visualStorybook {
		vector.StrokeRect(dst, float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H),
			2, cgaWhite, false)
		return
	}
	vector.StrokeRect(dst, float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H),
		4, storybook.ink, false)
	vector.StrokeRect(dst, float32(rc.X+7), float32(rc.Y+7), float32(rc.W-14), float32(rc.H-14),
		2, storybook.rule, false)
	// 四角的小菱形像冒險書的壓印，不使用貼圖即可在所有平台穩定重現。
	for _, p := range [][2]float32{{float32(rc.X + 7), float32(rc.Y + 7)},
		{float32(rc.Right() - 7), float32(rc.Y + 7)},
		{float32(rc.X + 7), float32(rc.Bottom() - 7)},
		{float32(rc.Right() - 7), float32(rc.Bottom() - 7)}} {
		vector.DrawFilledCircle(dst, p[0], p[1], 3, storybook.accent, false)
	}
}
