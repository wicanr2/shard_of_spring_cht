package original

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

// Palette 是 CGA 第三調色盤:黑 / 青 / 紅 / 白。
// docs/formats/07 §5:`0x3D8 = 0x0E`(繪圖模式 + 「黑白」位元);
// `0x3D9` 從來沒有被任何模組或 BRUN30 寫過,所以背景色是預設的黑。
var Palette = color.Palette{
	color.RGBA{0x00, 0x00, 0x00, 0xff},
	color.RGBA{0x55, 0xff, 0xff, 0xff},
	color.RGBA{0xff, 0x55, 0x55, 0xff},
	color.RGBA{0xff, 0xff, 0xff, 0xff},
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
// 103 × 121 格,每格 2 bytes,索引 = y × 103 + x。
const (
	WorldW = 103
	WorldH = 121
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
