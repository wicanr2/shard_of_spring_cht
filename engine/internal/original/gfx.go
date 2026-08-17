package original

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

// 調色盤有**兩組**,由各模組啟動時寫進 `0x3D8` 的那一個 byte 決定
// (docs/re/146 §4:掃全部模組的 `out` 指令):
//
//	WRLDMOVE  0x0A   → 世界地圖:綠 / 紅 / 棕
//	MAZEMOVE  0x0E   → 迷宮
//	CMBT      0x0E   → 戰鬥
//	MENU      0x0E   → 主選單
//	TOWN / CAMP / CHARUTIL 沒有寫 —— 沿用前一個模組設的
//
// ⚠ **先前只有一組**,理由是 `docs/formats/07` §5 讀到 `0x3D8 = 0x0E` ——
// 那是對的,但它只讀了一個模組。**世界地圖寫的是 `0x0A`**,
// 而用錯調色盤畫出來的地圖仍然是一張地圖,只是顏色不對。
//
// `0x3D9`(顏色選擇)從來沒有被任何模組寫過,所以背景色是預設的黑,
// 亮度位元維持 BIOS 設模式時的狀態 —— 取原版實跑截圖裡的低亮度那一組。
var (
	// PaletteWorld 是世界地圖用的(`0x3D8 = 0x0A`,CGA 第 0 組)。
	PaletteWorld = color.Palette{
		color.RGBA{0x00, 0x00, 0x00, 0xff},
		color.RGBA{0x00, 0xaa, 0x00, 0xff}, // 綠
		color.RGBA{0xaa, 0x00, 0x00, 0xff}, // 紅
		color.RGBA{0xaa, 0x55, 0x00, 0xff}, // 棕
	}
	// PaletteMaze 是迷宮 / 戰鬥 / 選單用的(`0x3D8 = 0x0E`)。
	PaletteMaze = color.Palette{
		color.RGBA{0x00, 0x00, 0x00, 0xff},
		color.RGBA{0x00, 0xaa, 0xaa, 0xff}, // 青
		color.RGBA{0xaa, 0x00, 0xaa, 0xff}, // 洋紅
		color.RGBA{0xaa, 0xaa, 0xaa, 0xff}, // 白(低亮度)
	}
	// Palette 是預設的那一組。⚠ 解碼函式本身**不知道**這張圖要畫在哪個畫面,
	// 所以由呼叫端在轉檔時重新指定(cmd/convert)。
	Palette = PaletteMaze
)

// WithPalette 換掉一張已解好的圖的調色盤。**只換色盤不動像素值** ——
// 像素值是原版檔案裡的,換色盤才是畫面差異的來源。
func WithPalette(img *image.Paletted, p color.Palette) *image.Paletted {
	img.Palette = p
	return img
}

// decodeGetArray 解一張 BASIC `GET` 陣列。
//
// docs/formats/07 §1:
//
//	前 2 個 word = 寬(**bit 數**)與 高
//	其後 = 像素資料,2 bit/像素,高位在左
//
// ⚠ **尺寸寫在資料裡** —— 不要對剩餘長度做因數分解(docs/re/19 §3 踩過)。
func decodeGetArray(words []uint16) (*image.Paletted, error) {
	if len(words) < 2 {
		return nil, fmt.Errorf("GET 陣列太短:%d words", len(words))
	}
	wbits, h := int(words[0]), int(words[1])
	w := wbits / 2 // 2 bit/像素
	if w <= 0 || h <= 0 || w > 4096 || h > 4096 {
		return nil, fmt.Errorf("GET 陣列尺寸不合理:%d bits × %d 列", wbits, h)
	}
	rowBytes := (wbits + 7) / 8

	body := make([]byte, 0, (len(words)-2)*2)
	for _, x := range words[2:] {
		body = append(body, byte(x), byte(x>>8))
	}
	if need := rowBytes * h; len(body) < need {
		return nil, fmt.Errorf("像素資料不足:需要 %d bytes,只有 %d", need, len(body))
	}

	img := image.NewPaletted(image.Rect(0, 0, w, h), Palette)
	for y := 0; y < h; y++ {
		row := body[y*rowBytes : (y+1)*rowBytes]
		x := 0
		for _, b := range row {
			for _, sh := range [4]uint{6, 4, 2, 0} {
				if x >= w {
					break
				}
				img.SetColorIndex(x, y, (b>>sh)&3)
				x++
			}
		}
	}
	return img, nil
}

func bodyWords(body []byte) []uint16 {
	out := make([]uint16, len(body)/2)
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(body[i*2:])
	}
	return out
}

// DecodeTile 解單張圖(98-byte 圖塊群、PICT*.BIN)。
func DecodeTile(d []byte) (*image.Paletted, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	return decodeGetArray(bodyWords(b.Body))
}

// MonstFrames 是 MONST*.BIN 的八張動畫格。
const MonstFrames = 8

// DecodeMonst 解 MONST*.BIN 的八張 17×17。
//
// ⚠ **資料是交錯的,不是連續的**(docs/formats/07 §3):
// 八張子圖被 GET 進同一個二維 BASIC 陣列 `A%(7, n)`,
// 而 BASIC 的二維陣列是**行主序** —— `A%(i, j)` 落在第 `j*8+i` 個元素,
// 所以第 i 張圖的資料是 `words[i::8]`。
//
// 照連續佈局去切**不會報錯**,只會得到八張雜訊 —— 而雜訊與
// 「這個檔本來就沒有圖」長得一樣。解完一定要 dump 出來肉眼看。
func DecodeMonst(d []byte) ([]*image.Paletted, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	words := bodyWords(b.Body)
	out := make([]*image.Paletted, 0, MonstFrames)
	for i := 0; i < MonstFrames; i++ {
		sub := make([]uint16, 0, len(words)/MonstFrames+1)
		for j := i; j < len(words); j += MonstFrames {
			sub = append(sub, words[j])
		}
		img, err := decodeGetArray(sub)
		if err != nil {
			return nil, fmt.Errorf("第 %d 格:%w", i, err)
		}
		out = append(out, img)
	}
	return out, nil
}

// DecodeWorldMap 解 WRLDMAP.BIN。docs/formats/05 §2:
// **東西 121 × 南北 103**,每格 2 bytes,索引 = x × 103 + y。
//
// ⚠ 兩軸的名字改過一次(docs/re/141):先前寫成 103 東西 × 121 南北。
// 格數(12,463)與跨距(103)都沒錯,錯的是**跨距 103 是往東還是往南** ——
// 而轉置過的地圖仍然是一塊連續陸地,畫得出來、看起來也合理。
const (
	WorldW = 121 // 東西
	WorldH = 103 // 南北,同時是索引的跨距
)

func DecodeWorldMap(d []byte) ([]uint16, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	cells := bodyWords(b.Body)
	if len(cells) < WorldW*WorldH {
		return nil, fmt.Errorf("世界地圖只有 %d 格,需要 %d", len(cells), WorldW*WorldH)
	}
	return cells[:WorldW*WorldH], nil
}

// CGA 整頁的尺寸([`docs/re/20`](../../../docs/re/20-cga-layout.md) §1)。
//
// ⚠ 這是**整頁顯示緩衝**的佈局,與 `GET` 陣列(圖塊、PICT)**不一樣**:
// 那些是連續存放並自帶尺寸,這裡是硬體的掃描線交錯、尺寸寫死。
// re/20 §2 記過同一件事:「不要假設其他素材也是整頁佈局」。
const (
	CGAPageW     = 320
	CGAPageH     = 200
	cgaRowBytes  = CGAPageW / 4 // 4 像素/byte
	cgaBankBytes = 0x2000       // 奇數列那一區的起點
	// CGAPageBytes 是一整頁:兩區各 8 KB。
	CGAPageBytes = cgaBankBytes * 2
)

// DecodeCGAPage 解一張 CGA 整頁畫面(`STARTUP.BIN`)。
//
// 佈局(docs/re/20 §1):
//
//	每列 80 bytes × 4 像素 = 320 像素,2 bit/像素,高位在左
//	偶數列 offset 0x0000 + (列÷2) × 80
//	奇數列 offset 0x2000 + (列÷2) × 80
//
// ⚠ **交錯區搞反或每列 bytes 數算錯不會報錯**,只會讓畫面碎成斜線 ——
// 而斜線與「這個檔本來就沒有圖」在程式看來沒有差別。解完一定要 dump 出來看
// (re/20 §1 的第 3 條證據就是靠肉眼確認框線是直的)。
func DecodeCGAPage(d []byte) (*image.Paletted, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	if len(b.Body) < CGAPageBytes {
		return nil, fmt.Errorf("CGA 整頁需要 %d bytes,只有 %d", CGAPageBytes, len(b.Body))
	}
	img := image.NewPaletted(image.Rect(0, 0, CGAPageW, CGAPageH), Palette)
	for y := 0; y < CGAPageH; y++ {
		off := (y/2)*cgaRowBytes + (y%2)*cgaBankBytes
		row := b.Body[off : off+cgaRowBytes]
		x := 0
		for _, by := range row {
			for _, sh := range [4]uint{6, 4, 2, 0} {
				img.SetColorIndex(x, y, (by>>sh)&3)
				x++
			}
		}
	}
	return img, nil
}
