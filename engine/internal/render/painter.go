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
//
// 兩種後端:向量字(`Face`)與點陣字(`bmp`)。**場景層看不出差別** ——
// 選哪一種在 `engine/font_ttf.go` / `font_eten.go`(build tag)決定。
type Painter struct {
	Face *text.GoTextFace // 非 nil = 向量字後端
	Fill color.Color

	bmp   *BitmapFont // 非 nil = 點陣字後端
	scale float64     // 點陣字的整數倍放大
}

// NewPainter 用指定的字型來源與字級建一個向量字 Painter。
func NewPainter(src *text.GoTextFaceSource, size float64, c color.Color) *Painter {
	return &Painter{Face: &text.GoTextFace{Source: src, Size: size}, Fill: c}
}

// NewBitmapPainter 建一個點陣字 Painter。
//
// ⚠ scale 只能是整數 —— 點陣字非整數倍放大會糊掉,那正是
// docs/spec/04 §1 對美術定的同一條規則。
func NewBitmapPainter(f *BitmapFont, scale int, c color.Color) *Painter {
	if scale < 1 {
		scale = 1
	}
	return &Painter{bmp: f, scale: float64(scale), Fill: c}
}

// Size 回傳實際字級(像素)。點陣字是「全形高 × 放大倍率」。
func (p *Painter) Size() float64 {
	if p.bmp != nil {
		return float64(p.bmp.FullH) * p.scale
	}
	return p.Face.Size
}

// LineHeight 回傳建議行高。
func (p *Painter) LineHeight() float64 { return p.Size() * 1.3 }

// Advance 回傳這段字的實際像素寬度。
//
// ⚠ 對齊要用**實際寬度**,不能用空白補位 —— 向量字是比例字,
// 補空白只在等寬終端機裡有效。Cols() 只用來算**容量**,不用來排版。
func (p *Painter) Advance(s string) float64 {
	if p.bmp != nil {
		w := 0
		for _, r := range s {
			w += p.bmp.Advance(r)
		}
		return float64(w) * p.scale
	}
	return text.Advance(s, p.Face)
}

// Draw 在 (x, y) 畫一段字,y 是**基線上緣**(左上角對齊)。
func (p *Painter) Draw(dst *ebiten.Image, s string, x, y float64) {
	if p.bmp != nil {
		p.drawBitmap(dst, s, x, y)
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(p.Fill)
	text.Draw(dst, s, p.Face, op)
}

// drawBitmap 逐字把點陣影像貼上去。
//
// ⚠ 字型裡沒有的字**跳過但仍然前進**(Advance 給的是全形寬)——
// 少畫一個字只是缺一格,少前進一格會讓整行後面全部錯位。
func (p *Painter) drawBitmap(dst *ebiten.Image, s string, x, y float64) {
	for _, r := range s {
		if img := p.bmp.Glyph(r); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(p.scale, p.scale)
			op.GeoM.Translate(x, y)
			op.Filter = ebiten.FilterNearest // 點陣字不插值
			op.ColorScale.ScaleWithColor(p.Fill)
			dst.DrawImage(img, op)
		}
		x += float64(p.bmp.Advance(r)) * p.scale
	}
}

// DrawRight 讓字串的**右緣**落在 x。
func (p *Painter) DrawRight(dst *ebiten.Image, s string, x, y float64) {
	p.Draw(dst, s, x-p.Advance(s), y)
}
