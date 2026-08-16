package render

// 點陣字型後端。docs/spec/04 §4:本機版用倚天中文系統 3.53 的 24×24 明體,
// 發行版用開源 TTF —— 兩者共用同一個 Painter 介面,場景層不知道差別。
//
// 資產由 `tools/eten_font.py` 產生(格式說明在那支腳本的 docstring)。
// ⚠ **那份資產是倚天的著作物**:不進版控、不隨發行版散布。

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// BitmapFont 是一套等寬點陣字:全形一種寬度、半形一種寬度,高度相同。
type BitmapFont struct {
	FullW, FullH int
	HalfW, HalfH int

	recs map[rune]glyphRec
	blob []byte

	// cache 是已經轉成 ebiten 影像的字。**用到才轉** ——
	// 一萬三千字全部預先轉成材質要 1.3 億像素,而一場遊戲用不到 2%。
	cache map[rune]*ebiten.Image
}

type glyphRec struct {
	off      uint32
	width    uint8
	rowBytes uint8
}

const bitmapMagic = "ETEN24\x00\x00"

// LoadBitmapFont 讀 tools/eten_font.py 產生的資產。
func LoadBitmapFont(b []byte) (*BitmapFont, error) {
	const headerLen = 8 + 6*4
	if len(b) < headerLen || string(b[:8]) != bitmapMagic {
		return nil, fmt.Errorf("不是點陣字型資產(magic 不符)")
	}
	u32 := func(i int) int { return int(binary.LittleEndian.Uint32(b[i:])) }
	if v := u32(8); v != 1 {
		return nil, fmt.Errorf("點陣字型版本 %d,本引擎只認得 1", v)
	}
	f := &BitmapFont{
		FullW: u32(12), FullH: u32(16), HalfW: u32(20), HalfH: u32(24),
		recs: map[rune]glyphRec{}, cache: map[rune]*ebiten.Image{},
	}
	count := u32(28)
	idxEnd := headerLen + count*12
	if idxEnd > len(b) {
		return nil, fmt.Errorf("索引表宣稱 %d 字,檔案放不下", count)
	}
	f.blob = b[idxEnd:]
	for i := 0; i < count; i++ {
		p := headerLen + i*12
		r := rune(binary.LittleEndian.Uint32(b[p:]))
		f.recs[r] = glyphRec{
			off:      binary.LittleEndian.Uint32(b[p+4:]),
			width:    b[p+8],
			rowBytes: b[p+9],
		}
	}
	return f, nil
}

// substitute 把字型裡沒有的字換成同義的 Big5 字。
//
// 倚天字型是 **Big5** 的,而引擎介面用了幾個 Unicode 裝飾符號 ——
// 它們不在 Big5 裡,畫出來會是空白。⚠ **空白不會有人發現**:
// 「※ 金幣公式未解」少了那個記號,整句話看起來仍然通順。
//
// ⚠ 這是**字型層的代換,不是改文字**:原始字串仍然是 ⚠,
// 開源字型那一版照樣畫得出來(docs/spec/21 §4)。
var substitute = map[rune]rune{
	'⚠': '※', // 注意記號
	'✓': '○', // 打勾
	'✚': '＋', // 施法游標
	'▶': '▲', // 指標
	'～': '〜', // 波浪(全形)
	'‰': '％', // 千分號 —— Big5 沒有,退成百分號(只出現在一處說明文字)
}

// glyphFor 把字對應到字型裡真的有的那個字。
func (f *BitmapFont) glyphFor(r rune) (rune, bool) {
	if _, ok := f.recs[r]; ok {
		return r, true
	}
	if alt, ok := substitute[r]; ok {
		if _, ok := f.recs[alt]; ok {
			return alt, true
		}
	}
	return r, false
}

// Advance 回傳一個字的寬度(未縮放)。字型裡沒有的字回全形寬 ——
// 畫出來是一個空格,**寬度要對**,不然整行後面的字全部位移。
func (f *BitmapFont) Advance(r rune) int {
	if g, ok := f.recs[r]; ok {
		return int(g.width)
	}
	if alt, ok := f.glyphFor(r); ok {
		return int(f.recs[alt].width)
	}
	return f.FullW
}

// Glyph 回傳一個字的影像(白色 + alpha),沒有就回 nil。
func (f *BitmapFont) Glyph(r rune) *ebiten.Image {
	if img, ok := f.cache[r]; ok {
		return img
	}
	key, ok := f.glyphFor(r)
	if !ok {
		f.cache[r] = nil
		return nil
	}
	g := f.recs[key]
	w, h := int(g.width), f.FullH
	// ⚠ **每列的位元組數由資產明寫**,不從寬度推:全形 24 位元剛好 3 B,
	// 半形卻是 12 位元存在 2 B 裡(右邊 4 位是墊的)—— 同一條公式湊不出兩者,
	// 而推錯的症狀是**漢字變成雜訊、ASCII 完全正常**(半形那條剛好推對)。
	rowBytes := int(g.rowBytes)
	if rowBytes == 0 {
		f.cache[r] = nil
		return nil
	}
	need := rowBytes * h
	if int(g.off)+need > len(f.blob) {
		f.cache[r] = nil
		return nil
	}
	src := f.blob[g.off : int(g.off)+need]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			bit := src[y*rowBytes+x/8] >> (7 - uint(x%8)) & 1
			if bit == 1 {
				img.Set(x, y, color.White)
			}
		}
	}
	out := ebiten.NewImageFromImage(img)
	f.cache[r] = out
	return out
}
