// Package render 是需要 Ebitengine 的繪製與字型載入。
//
// ⚠ **與 internal/ui 分開是刻意的。** ebiten 的 package init 需要 DISPLAY,
// 匯入它的套件在 headless 容器裡連測試都跑不起來 ——
// 所以排版與量測(可測)放 internal/ui,繪製(不可測)放這裡。
package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Painter 把字畫到畫面上,並提供量測。
type Painter struct {
	Face *text.GoTextFace
	Fill color.Color
}

// NewPainter 用指定的字型來源與字級建一個 Painter。
func NewPainter(src *text.GoTextFaceSource, size float64, c color.Color) *Painter {
	return &Painter{Face: &text.GoTextFace{Source: src, Size: size}, Fill: c}
}

// LineHeight 回傳建議行高。
func (p *Painter) LineHeight() float64 { return p.Face.Size * 1.3 }

// Advance 回傳這段字的實際像素寬度。
//
// ⚠ 對齊要用**實際寬度**,不能用空白補位 —— 這裡的字型是比例字,
// 補空白只在等寬終端機裡有效。Cols() 只用來算**容量**,不用來排版。
func (p *Painter) Advance(s string) float64 { return text.Advance(s, p.Face) }

// Draw 在 (x, y) 畫一段字,y 是**基線上緣**(左上角對齊)。
func (p *Painter) Draw(dst *ebiten.Image, s string, x, y float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(p.Fill)
	text.Draw(dst, s, p.Face, op)
}

// DrawRight 讓字串的**右緣**落在 x。
func (p *Painter) DrawRight(dst *ebiten.Image, s string, x, y float64) {
	p.Draw(dst, s, x-p.Advance(s), y)
}
