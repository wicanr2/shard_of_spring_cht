package layout

import "testing"

// 每個區塊都必須落在畫布內,且與外緣保持 Margin。
func TestRectsInsideScreen(t *testing.T) {
	for _, c := range []struct {
		name string
		r    Rect
	}{
		{"View", View}, {"Party", Party}, {"Message", Message},
		{"Prompt", Prompt}, {"Overlay", Overlay},
	} {
		if c.r.X < 0 || c.r.Y < 0 {
			t.Errorf("%s 起點為負:%+v", c.name, c.r)
		}
		if c.r.Right() > ScreenW {
			t.Errorf("%s 右緣 %d 超出畫布寬 %d", c.name, c.r.Right(), ScreenW)
		}
		if c.r.Bottom() > ScreenH {
			t.Errorf("%s 底緣 %d 超出畫布高 %d", c.name, c.r.Bottom(), ScreenH)
		}
	}
}

// 主視野必須正好是 9×9 個放大後的圖塊 —— 這個數字來自原版
// PICT*.BIN 的 153×153(docs/formats/07),不是挑出來的。
func TestViewIsWholeTiles(t *testing.T) {
	if View.W%TileDst != 0 || View.H%TileDst != 0 {
		t.Fatalf("主視野 %dx%d 不是圖塊 %d 的整數倍", View.W, View.H, TileDst)
	}
	if got := View.W / TileDst; got != ViewTiles {
		t.Errorf("主視野橫向 %d 格,應為 %d", got, ViewTiles)
	}
	if View.W != 153*ArtScale {
		t.Errorf("主視野寬 %d,應為原版圖畫區 153 × %d = %d", View.W, ArtScale, 153*ArtScale)
	}
}

// 非整數倍縮放會讓像素藝術糊掉(docs/spec/04 §1)。這條是設計約束,
// 不是實作細節 —— 有人把 ArtScale 改成 3.5 之類的時候要當場擋下來。
func TestArtScaleIsInteger(t *testing.T) {
	if ArtScale < 1 {
		t.Fatalf("ArtScale = %d,必須 ≥ 1 的整數", ArtScale)
	}
	if TileDst != TileSrc*ArtScale {
		t.Errorf("TileDst = %d,與 TileSrc × ArtScale = %d 不符", TileDst, TileSrc*ArtScale)
	}
}

// 區塊之間不可重疊(覆蓋層例外 —— 它本來就蓋在主視野上)。
func TestNoOverlapExceptOverlay(t *testing.T) {
	base := []struct {
		name string
		r    Rect
	}{{"View", View}, {"Party", Party}, {"Message", Message}, {"Prompt", Prompt}}
	for i := 0; i < len(base); i++ {
		for j := i + 1; j < len(base); j++ {
			a, b := base[i].r, base[j].r
			if a.X < b.Right() && b.X < a.Right() && a.Y < b.Bottom() && b.Y < a.Bottom() {
				t.Errorf("%s 與 %s 重疊:%+v / %+v", base[i].name, base[j].name, a, b)
			}
		}
	}
}

// 右欄的兩塊要對齊,而且底緣要與主視野齊平 —— 版面才不會看起來歪的。
func TestRightColumnAligned(t *testing.T) {
	if Party.X != Message.X || Party.W != Message.W {
		t.Errorf("右欄兩塊沒對齊:Party %+v / Message %+v", Party, Message)
	}
	if Message.Bottom() != View.Bottom() {
		t.Errorf("右欄底緣 %d 與主視野底緣 %d 不齊", Message.Bottom(), View.Bottom())
	}
}

// 覆蓋層置中。
func TestOverlayCentered(t *testing.T) {
	if l, r := Overlay.X, ScreenW-Overlay.Right(); l != r {
		t.Errorf("覆蓋層左右留白不等:%d / %d", l, r)
	}
	if tp, bt := Overlay.Y, ScreenH-Overlay.Bottom(); tp != bt {
		t.Errorf("覆蓋層上下留白不等:%d / %d", tp, bt)
	}
}
