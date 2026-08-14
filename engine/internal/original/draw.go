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

// SplitPIC 把 .PIC 切成一段一段的 DRAW 巨集(CRLF 分隔,0x1A 結尾)。
func SplitPIC(d []byte) []string {
	s := strings.ReplaceAll(string(d), "\x1a", "")
	var out []string
	for _, l := range strings.Split(s, "\r\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
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
	SrcNone     WorldTileSource = iota // 沒有來源 —— 值 0、10、35–38
	SrcFastWrld                        // FASTWRLD.BIN 第 (值−1) 張
	SrcWrldItem                        // WRLDITEM.PIC 段索引 (值−14)
)

// WrldItemBias 是地形值與 WRLDITEM.PIC 段索引之間的差:段 = 值 − 14。
//
// ⚠ **這個數字被推翻過一次。** 第一版用「地形值 11 是海洋(55.6%),
// 而段 1 是實心磚牆 —— 整片海洋不可能是磚牆」推出 −11。
// 那個推理只排除了一個候選,**沒有正面確認任何一個**。
//
// 正解用**資料側已知語意的圖塊**去對,7/7 全中:
//
//	值 30/31/32 = 城鎮(13 處)  → 段 16/17/18 = 白色街區平面圖
//	值 24 = 地城入口            → 段 10 = 紅色塔樓帶門
//	值 25 = 地城入口            → 段 11 = 紅色城堡帶門
//	值 27 = 地城入口            → 段 13 = 青色洞穴拱門
//	值 28 = 地城入口            → 段 14 = 山體下方一道小門
//
// 相同測試下 −11 只中 2/7(城鎮全部畫成丘陵)。
//
// ⚠ 連帶結果:**海洋(值 11)沒有 WRLDITEM 來源**,它的圖從哪來**未解**
// (docs/spec/05 §2.2)。
const WrldItemBias = 14

// WorldTileOrigin 回傳地形值的圖塊來源與索引。
func WorldTileOrigin(v int) (WorldTileSource, int) {
	switch {
	case v >= 1 && v <= 9:
		return SrcFastWrld, v - 1
	case v >= WrldItemBias && v <= WrldItemBias+22:
		return SrcWrldItem, v - WrldItemBias
	default:
		return SrcNone, 0
	}
}
