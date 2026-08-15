// Package music 解 BASIC 的 `PLAY` 巨集,並算出方波取樣。
//
// 來源:docs/re/148 —— 原版有 15 段 `PLAY` 字串,PC 喇叭的樂譜。
// 語法是 MS BASIC 的公開規格,這裡只實作原版用到的子集。
package music

import (
	"math"
	"strconv"
	"strings"
)

// Note 是一個音。Freq 為 0 表示休止。
type Note struct {
	Freq float64 // Hz
	Dur  float64 // 秒
	Gate float64 // 實際發聲的比例:一般 7/8、`ML` 時 1
}

// State 是 `PLAY` 的執行狀態。**它跨呼叫殘留** ——
// 原版最後兩段沒有 `T`/`MB`,靠的就是前面留下的狀態(docs/re/148 §3)。
type State struct {
	Tempo    int  // T:每分鐘幾個四分音符
	Octave   int  // O:0–6
	Length   int  // L:預設音長的分母
	Legato   bool // ML
	Staccato bool // MS
}

// NewState 回傳 BASIC 的預設值(`T120 O4 L4`,一般發聲 7/8)。
func NewState() *State { return &State{Tempo: 120, Octave: 4, Length: 4} }

// 半音相對 C 的位置。
var semitone = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// Freq 回傳某八度某音名的頻率。
//
// BASIC 的 `O4` 是中央八度,`O4 A` = 440 Hz。
func Freq(octave int, name byte, accidental int) float64 {
	n, ok := semitone[upper(name)]
	if !ok {
		return 0
	}
	// 以 O4 A(440 Hz)為基準:半音距離 = (八度差 × 12) + (音名差 − 9)
	steps := (octave-4)*12 + (n + accidental - 9)
	return 440 * math.Pow(2, float64(steps)/12)
}

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

// Parse 解一段 `PLAY` 字串,把音附加到 out,並就地更新狀態。
//
// ⚠ **狀態要傳進來**,不要每段重建 —— 原版的十段是一首曲子的續播,
// 後面幾段沒有 `T`/`O`,重建狀態會讓它們用預設速度播出來,
// 而那聽起來仍然像一首曲子,只是不對。
func (s *State) Parse(macro string, out []Note) []Note {
	i := 0
	for i < len(macro) {
		c := upper(macro[i])
		switch {
		case c == ' ':
			i++
		case c == 'M':
			i++
			if i < len(macro) {
				switch upper(macro[i]) {
				case 'L':
					s.Legato, s.Staccato = true, false
				case 'S':
					s.Legato, s.Staccato = false, true
				case 'N':
					s.Legato, s.Staccato = false, false
				}
				i++ // 'B'/'F' 是前景/背景,對音符沒有影響
			}
		case c == 'T' || c == 'O' || c == 'L':
			n, next := number(macro, i+1)
			switch c {
			case 'T':
				if n > 0 {
					s.Tempo = n
				}
			case 'O':
				s.Octave = n
			case 'L':
				if n > 0 {
					s.Length = n
				}
			}
			i = next
		case c == 'P': // 休止
			n, next := number(macro, i+1)
			if n == 0 {
				n = s.Length
			}
			out = append(out, Note{Dur: s.beat(n)})
			i = next
		case c >= 'A' && c <= 'G':
			i++
			acc := 0
			for i < len(macro) && (macro[i] == '#' || macro[i] == '+' || macro[i] == '-') {
				if macro[i] == '-' {
					acc--
				} else {
					acc++
				}
				i++
			}
			n, next := number(macro, i)
			i = next
			if n == 0 {
				n = s.Length
			}
			out = append(out, Note{Freq: Freq(s.Octave, c, acc), Dur: s.beat(n), Gate: s.gate()})
		default:
			i++ // 不認得的記號略過,**不要當成錯誤** —— 原版的字串很寬鬆
		}
	}
	return out
}

// beat 回傳「分母為 n 的音符」有多長(秒)。
func (s *State) beat(n int) float64 {
	if n <= 0 {
		n = 4
	}
	return 240.0 / float64(s.Tempo) / float64(n)
}

func (s *State) gate() float64 {
	switch {
	case s.Legato:
		return 1
	case s.Staccato:
		return 0.75
	}
	return 0.875 // BASIC 的一般模式:7/8
}

// number 讀一串十進位數字,回傳值與下一個位置。沒有數字時回 0。
func number(s string, i int) (int, int) {
	j := i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j == i {
		return 0, i
	}
	n, _ := strconv.Atoi(s[i:j])
	return n, j
}

// ParseAll 依序解多段 `PLAY`,共用同一份狀態。
func ParseAll(macros []string) []Note {
	st := NewState()
	var out []Note
	for _, m := range macros {
		out = st.Parse(strings.TrimSpace(m), out)
	}
	return out
}

// Render 把音符算成 8-bit 單聲道方波取樣(PC 喇叭就是方波)。
//
// ⚠ 回傳的是 **PCM 資料**,不碰任何音訊裝置 —— 這樣它可以在沒有音效卡、
// 沒有 DISPLAY 的環境裡測試。
func Render(notes []Note, sampleRate int) []byte {
	var out []byte
	for _, n := range notes {
		total := int(n.Dur * float64(sampleRate))
		on := int(float64(total) * n.Gate)
		for i := 0; i < total; i++ {
			v := byte(128)
			if i < on && n.Freq > 0 {
				period := float64(sampleRate) / n.Freq
				if math.Mod(float64(i), period) < period/2 {
					v = 200
				} else {
					v = 56
				}
			}
			out = append(out, v)
		}
	}
	return out
}

// SampleRate 是本引擎的取樣率。Ebitengine 的音訊需要 16-bit LE 立體聲。
const SampleRate = 22050

// RenderPCM16 把音符算成 **16-bit LE 立體聲**,也就是 Ebitengine 吃的格式。
//
// ⚠ 與 Render 分開是為了讓測試能驗轉換本身:
// 8-bit 單聲道是 PC 喇叭的形狀,16-bit 立體聲是播放層的要求,
// **把兩者混在一起會讓「音錯了」與「格式錯了」分不開**。
func RenderPCM16(notes []Note, sampleRate int) []byte {
	mono := Render(notes, sampleRate)
	out := make([]byte, 0, len(mono)*4)
	for _, v := range mono {
		// 0–255 的無號 → −32768…32767 的有號
		s := int16((int(v) - 128) * 256)
		lo, hi := byte(s&0xff), byte(s>>8)
		out = append(out, lo, hi, lo, hi) // 左右聲道相同
	}
	return out
}
