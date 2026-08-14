package original

import (
	"image"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// DRAW 巨集解譯器。`*.PIC` 不是點陣圖,是 BASIC 的 `DRAW` 向量巨集
// (docs/formats/07 §4)。
//
// 支援:U D L R E F G H(移動並畫線)、`B` 前綴(移動不畫)、
// `N` 前綴(畫完回原點)、`M±x,±y`(相對)/ `Mx,y`(絕對)、
// `C n`(顏色)、`S n`(縮放,單位 1/4)、`TA n`(旋轉角度)、`P f,b`(填充)。
//
// ⚠ **`P` 只標記起點,不做真正的洪水填充。** 原版的填色會讓封閉圖形變成實心,
// 這裡沒做 —— 所以渲染出來的圖是**輪廓**,拿來辨識主題時要記得這一點。
// (這個限制曾讓我把段 7 誤讀成「不像海岸線」。)
var drawTok = regexp.MustCompile(`(?i)(B|N)?(TA|[UDLREFGHMCSAP])\s*([+\-]?\d+)?(?:\s*,\s*([+\-]?\d+))?`)

var drawDirs = map[byte][2]float64{
	'U': {0, -1}, 'D': {0, 1}, 'L': {-1, 0}, 'R': {1, 0},
	'E': {1, -1}, 'F': {1, 1}, 'G': {-1, 1}, 'H': {-1, -1},
}

type drawCanvas struct {
	w, h int
	px   []uint8
}

func (c *drawCanvas) put(x, y float64, col uint8) {
	xi, yi := int(math.Round(x)), int(math.Round(y))
	if xi >= 0 && xi < c.w && yi >= 0 && yi < c.h {
		c.px[yi*c.w+xi] = col
	}
}

func (c *drawCanvas) line(x0, y0, x1, y1 float64, col uint8) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / math.Max(float64(n), 1)
		c.put(x0+(x1-x0)*t, y0+(y1-y0)*t, col)
	}
}

// RenderDraw 把一段 DRAW 巨集畫進 w×h 的畫布。
//
// ⚠ **起筆點在左下角 (0, h−1),不是正中央。**
// 實測 `WRLDITEM.PIC` 各段相對起點的繪製範圍是 x∈[0,16]、y∈[−16,0] ——
// 全部往右上方畫。從中央起筆會把圖切掉一半,而**畫面上看起來仍像某種圖案**,
// 不會有任何錯誤訊息。
func RenderDraw(macro string, w, h int) *image.Paletted {
	cv := &drawCanvas{w: w, h: h, px: make([]uint8, w*h)}
	x, y := 0.0, float64(h-1)
	color := uint8(3)
	scale, ang := 4.0, 0.0

	for _, m := range drawTok.FindAllStringSubmatch(macro, -1) {
		pre := strings.ToUpper(m[1])
		cmd := strings.ToUpper(m[2])
		n := 1.0
		if m[3] != "" {
			v, _ := strconv.Atoi(m[3])
			n = float64(v)
		}
		switch cmd {
		case "C":
			color = uint8(int(n) & 3)
			continue
		case "S":
			scale = math.Max(n, 1)
			continue
		case "A", "TA":
			ang = n
			continue
		case "P":
			cv.put(x, y, color)
			continue
		}

		sx, sy := x, y
		var nx, ny float64
		switch {
		case cmd == "M":
			if m[3] == "" {
				continue
			}
			dy := 0.0
			if m[4] != "" {
				v, _ := strconv.Atoi(m[4])
				dy = float64(v)
			}
			if m[3][0] == '+' || m[3][0] == '-' {
				nx, ny = x+n, y+dy // 相對
			} else {
				nx, ny = n, dy // 絕對
			}
		default:
			d, ok := drawDirs[cmd[0]]
			if !ok {
				continue
			}
			dist := n * scale / 4
			r := ang * math.Pi / 180
			rx := d[0]*math.Cos(r) - d[1]*math.Sin(r)
			ry := d[0]*math.Sin(r) + d[1]*math.Cos(r)
			nx, ny = x+rx*dist, y+ry*dist
		}

		if pre != "B" {
			cv.line(sx, sy, nx, ny, color)
		}
		if pre != "N" {
			x, y = nx, ny
		}
	}

	img := image.NewPaletted(image.Rect(0, 0, w, h), Palette)
	copy(img.Pix, cv.px)
	return img
}

// SplitPIC 把 .PIC 切成一行一行的 DRAW 巨集(CRLF 分隔,0x1A 結尾)。
//
// ⚠ **空行一定要保留。** `WRLDITEM.PIC` 有 29 行、其中 6 行是空的,
// 而**行號就是索引**(見 WorldTileOrigin)。把空行濾掉會讓 29 變 23、
// 索引整個位移 —— 而位移後的圖仍然是「某種圖」,畫面上看不出錯。
//
// 空行本身有意義:它代表**那個圖塊值不走向量路徑**
// (值 11 海洋原版根本不畫,docs/re/132 §1)。
func SplitPIC(d []byte) []string {
	s := strings.ReplaceAll(string(d), "\x1a", "")
	rows := strings.Split(s, "\r\n")
	// 只去掉檔尾那一個空段(最後一行的 CRLF 造成的),其餘空行保留
	if len(rows) > 0 && strings.TrimSpace(rows[len(rows)-1]) == "" {
		rows = rows[:len(rows)-1]
	}
	return rows
}

// ---------------------------------------------------------------------------
// FASTWRLD.BIN — 9 張地形圖塊,各 92 bytes,**連續排列不交錯**
// (docs/re/53 §1:兩種都試過,只有連續讓九組的標頭都是 (34,17))
// ---------------------------------------------------------------------------

const (
	fastTileCount = 9
	fastTileLen   = 92
)

func DecodeFastWorld(d []byte) ([]*image.Paletted, error) {
	b, err := ParseBSAVE(d)
	if err != nil {
		return nil, err
	}
	out := make([]*image.Paletted, 0, fastTileCount)
	for i := 0; i < fastTileCount; i++ {
		g := b.Body[i*fastTileLen : (i+1)*fastTileLen]
		img, err := decodeGetArray(bodyWords(g))
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, nil
}

// WorldTileSource 說明某個地形值的圖從哪來。
// docs/spec/05-world-scene.md §4。
type WorldTileSource int

const (
	SrcNone     WorldTileSource = iota // 不畫任何東西 —— 只有值 0(地圖邊界)
	SrcFastWrld                        // FASTWRLD.BIN 第 (值−1) 張
	SrcWrldItem                        // WRLDITEM.PIC 第 (值−10) 行,0-based
	SrcBackdrop                        // 值 11(海洋)—— **原版刻意不畫**,見下
)

// WrldItemBias 是地形值與 `WRLDITEM.PIC` 行號之間的差:行 = 值 − 10(0-based)。
//
// 出處 docs/re/54 §2,判別依據是**空行**:該檔 29 行裡有 6 行是空的,
// 而偏移 10 是唯一讓兩邊同時成立的值 ——
//
//	6/6   每個空行對到的圖塊值,要嘛不畫(值 11 海洋),要嘛地圖上從未出現
//	20/20 地圖上用到的每個向量圖塊值,都對到非空行
//
// 其餘偏移(11、14)兩邊各有 4–7 筆違規。
//
// ⚠ **我曾把這條規則推翻兩次,兩次都是錯的**(docs/re/130)。
// 根因是 SplitPIC 當時把空行濾掉,29 行變 23,索引位移 ——
// 然後我去調偏移來補償。**行號就是索引的資料,空行不能濾。**
//
// docs/re/132 補上了程式碼側的直接證據:載入迴圈是 `FOR I = 10 TO 38`,
// 每讀一行就存進 `0xC980 + 4×I`,而繪製時**用地形值直接當索引**。
const WrldItemBias = 10

// WrldItemLast 是載入迴圈的上界(docs/re/132 §2:`cmp ax,26h / jle`)。
// 檔案剛好 29 行,對到值 10–38。
const WrldItemLast = 38

// OceanTile 是海洋的地形值。**原版的派工鏈明確跳過它,一個像素都不畫**
// (docs/re/132 §1),所以它顯示的就是底色。全圖 55.63%。
const OceanTile = 11

// WorldTileOrigin 回傳地形值的圖塊來源與行號。
//
// ⚠ 呼叫端還要檢查那一行是不是**空的**。空行有兩種成因,不要混:
// 值 11 是原版刻意跳過(SrcBackdrop),其餘空行(值 14/19/22/33/34)
// 則是**地圖上一次都沒出現**的值 —— 兩者在檔案裡長得一樣。
func WorldTileOrigin(v int) (WorldTileSource, int) {
	switch {
	case v == OceanTile:
		return SrcBackdrop, 0
	case v >= 1 && v <= 9:
		return SrcFastWrld, v - 1
	case v >= WrldItemBias && v <= WrldItemLast:
		return SrcWrldItem, v - WrldItemBias
	default:
		return SrcNone, 0
	}
}
