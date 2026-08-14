// Shard of Spring remake — M0 骨架。
//
// M0 的範圍(docs/spec/03-engine-plan.md §7):開得起視窗、把版面五個區塊
// 畫成框驗證座標。**不碰遊戲邏輯、不讀原版資料、不處理輸入。**
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/layout"
)

// 原版 CGA 第三調色盤:黑 / 青 / 紅 / 白(docs/formats/07 §5,`0x3D8 = 0x0E`)。
// M0 只用來把區塊框畫出來;正式的美術上色在 M1 之後。
var (
	cgaBlack = color.RGBA{0x00, 0x00, 0x00, 0xff}
	cgaCyan  = color.RGBA{0x55, 0xff, 0xff, 0xff}
	cgaRed   = color.RGBA{0xff, 0x55, 0x55, 0xff}
	cgaWhite = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

type Game struct{}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(cgaBlack)

	frame := func(r layout.Rect, c color.Color) {
		vector.StrokeRect(screen,
			float32(r.X), float32(r.Y), float32(r.W), float32(r.H),
			2, c, false)
	}

	frame(layout.View, cgaCyan)
	frame(layout.Party, cgaWhite)
	frame(layout.Message, cgaWhite)
	frame(layout.Prompt, cgaWhite)

	// 主視野的 9×9 格線 —— 用來目視確認圖塊格對得上邊框,
	// 而不是只在測試裡算數字。
	for i := 1; i < layout.ViewTiles; i++ {
		off := float32(i * layout.TileDst)
		vector.StrokeLine(screen,
			float32(layout.View.X)+off, float32(layout.View.Y),
			float32(layout.View.X)+off, float32(layout.View.Bottom()),
			1, cgaCyan, false)
		vector.StrokeLine(screen,
			float32(layout.View.X), float32(layout.View.Y)+off,
			float32(layout.View.Right()), float32(layout.View.Y)+off,
			1, cgaCyan, false)
	}

	// 敘述覆蓋層,用紅色框以便和底層區塊區分。
	frame(layout.Overlay, cgaRed)
}

// Layout 回傳邏輯畫布尺寸。⚠ 這是 Ebitengine 的介面方法,
// 與 internal/layout 套件同名純屬巧合,不要把兩者混為一談。
func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	return layout.ScreenW, layout.ScreenH
}

func main() {
	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	ebiten.SetWindowTitle("春之石 Shard of Spring")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
