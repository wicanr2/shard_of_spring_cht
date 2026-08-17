package original

import "testing"

// CGA 整頁的掃描線交錯:偶數列在 0x0000 那一區,奇數列在 0x2000。
//
// ⚠ 交錯搞反不會報錯,只會讓畫面碎成斜線(docs/re/20 §1)。
// 這個測試用**只有一列有資料**的合成頁把兩區分開驗 ——
// 真檔驗不出這件事:整張圖交錯錯了仍然是一張圖。
func TestDecodeCGAPageInterleave(t *testing.T) {
	body := make([]byte, CGAPageBytes)
	// 第 0 列(偶數區第 0 列)整列填色 1:0b01010101
	for i := 0; i < cgaRowBytes; i++ {
		body[i] = 0x55
	}
	// 第 1 列(奇數區第 0 列)整列填色 2:0b10101010
	for i := 0; i < cgaRowBytes; i++ {
		body[cgaBankBytes+i] = 0xAA
	}
	// 第 2 列(偶數區第 1 列)整列填色 3
	for i := 0; i < cgaRowBytes; i++ {
		body[cgaRowBytes+i] = 0xFF
	}

	img, err := DecodeCGAPage(wrapBSAVE(body))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Dx(); got != CGAPageW {
		t.Errorf("寬 %d,要 %d", got, CGAPageW)
	}
	if got := img.Bounds().Dy(); got != CGAPageH {
		t.Errorf("高 %d,要 %d", got, CGAPageH)
	}
	for _, tc := range []struct{ y, want int }{{0, 1}, {1, 2}, {2, 3}, {3, 0}} {
		for _, x := range []int{0, 1, 159, 319} {
			if got := int(img.ColorIndexAt(x, tc.y)); got != tc.want {
				t.Errorf("(%d,%d) = %d,要 %d", x, tc.y, got, tc.want)
			}
		}
	}
}

// 高位在左:一個 byte 的 bit 7-6 是最左邊那個像素。
func TestDecodeCGAPagePixelOrder(t *testing.T) {
	body := make([]byte, CGAPageBytes)
	body[0] = 0b11_10_01_00 // 左到右 3,2,1,0
	img, err := DecodeCGAPage(wrapBSAVE(body))
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []int{3, 2, 1, 0} {
		if got := int(img.ColorIndexAt(x, 0)); got != want {
			t.Errorf("x=%d 得到 %d,要 %d", x, got, want)
		}
	}
}

// 資料不足要報錯,不要回一張半截的圖 —— 半截的圖看起來仍然像圖。
func TestDecodeCGAPageTooShort(t *testing.T) {
	if _, err := DecodeCGAPage(wrapBSAVE(make([]byte, CGAPageBytes-1))); err == nil {
		t.Error("少一個 byte 卻沒有報錯")
	}
}

// wrapBSAVE 把裸資料包成 BSAVE 容器(docs/formats/05 §1:7 bytes 標頭 + EOF)。
func wrapBSAVE(body []byte) []byte {
	d := make([]byte, 0, 8+len(body))
	d = append(d, 0xFD, 0x00, 0xB8, 0x00, 0x00, byte(len(body)), byte(len(body)>>8))
	d = append(d, body...)
	return append(d, 0x1A)
}
