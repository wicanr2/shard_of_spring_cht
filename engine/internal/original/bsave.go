// Package original 解析原版《Shard of Spring》的資料檔。
//
// ⚠ 這個套件**只在資產轉換階段用**(engine/cmd/convert),遊戲本體不碰原版檔。
// docs/spec/03-engine-plan.md §3:原版 .DAT 只讀一次,轉成 JSON 進引擎。
//
// 每個解析器的欄位佈局都出自 docs/formats/,並在註解裡指明出處。
// **規格改了才改這裡**,不要反過來。
package original

import (
	"encoding/binary"
	"fmt"
)

// BSAVE 是 BASIC 的 BSAVE 容器,所有 .BIN 共用。
// docs/formats/05 §1:
//
//	0xFD  segment(2)  offset(2)  length(2)   ← 7 bytes 標頭
//	資料 …
//	0x1A                                     ← EOF
type BSAVE struct {
	Segment uint16
	Offset  uint16 // ⚠ 作者端的位址,不是載入位址。不要拿它當結構資訊。
	Body    []byte
}

// ParseBSAVE 檢查容器並取出資料段。
//
// ⚠ 標頭是 **7 bytes,奇數**(docs/formats/05 §1)。不扣標頭就切 2-byte
// 交錯資料,奇偶性會整個翻掉 —— 而錯位的結果看起來仍像「某種圖」,
// 不會當場報錯。這是本容器最容易出的錯。
func ParseBSAVE(d []byte) (*BSAVE, error) {
	if len(d) < 8 {
		return nil, fmt.Errorf("BSAVE 容器太短:%d bytes", len(d))
	}
	if d[0] != 0xFD {
		return nil, fmt.Errorf("BSAVE 標記錯誤:got 0x%02X, want 0xFD", d[0])
	}
	if d[len(d)-1] != 0x1A {
		return nil, fmt.Errorf("BSAVE 缺 EOF 0x1A,結尾是 0x%02X", d[len(d)-1])
	}
	ln := int(binary.LittleEndian.Uint16(d[5:7]))
	if 7+ln > len(d) {
		return nil, fmt.Errorf("BSAVE 標頭宣告 %d bytes,檔案只有 %d", ln, len(d)-8)
	}
	return &BSAVE{
		Segment: binary.LittleEndian.Uint16(d[1:3]),
		Offset:  binary.LittleEndian.Uint16(d[3:5]),
		Body:    d[7 : 7+ln],
	}, nil
}

// MBF 解讀 Microsoft Binary Format 單精度浮點(4 bytes,little-endian)。
//
//	b[0..2] = 尾數(b[2] 的最高位是符號)
//	b[3]    = 指數,偏移 129;為 0 代表值就是 0
//
// 用在 TOWNDATA.DAT 的商店價格倍率(docs/re/126)與 TOWNDATA.BIN 的座標。
//
// ⚠ MBF 表示不出精確的 1.3(最接近的是 1.2999999523),
// 而那個誤差是**有觀測後果的**:售價 = INT(基準價 × 倍率) 會因此少 1。
// 不要為了「看起來整齊」把結果四捨五入 —— 原版就是截斷。
func MBF(b []byte) float64 {
	if len(b) < 4 || b[3] == 0 {
		return 0
	}
	frac := uint32(b[2]&0x7F)<<16 | uint32(b[1])<<8 | uint32(b[0])
	v := (1 + float64(frac)/(1<<23))
	if b[2]&0x80 != 0 {
		v = -v
	}
	// 指數偏移 129。用逐次乘除而非 math.Pow —— 指數是小整數,
	// 而 math.Pow 在邊界會引入浮點誤差,這裡的誤差是有後果的(見上)。
	e := int(b[3]) - 129
	for ; e > 0; e-- {
		v *= 2
	}
	for ; e < 0; e++ {
		v /= 2
	}
	return v
}

// u16 讀 1-based 位移的 2-byte 整數。
//
// ⚠ docs/formats/ 的位移一律是 **1-based**(BASIC MID$ 慣例),
// 而 Go 的切片是 0-based。全部經過這個函式,不要在呼叫端自己減一 ——
// 兩種慣例混在同一份程式碼裡是本專案最容易出的差一錯。
func u16(rec []byte, pos1 int) int {
	return int(int16(binary.LittleEndian.Uint16(rec[pos1-1 : pos1+1])))
}

// str 讀 1-based 位移起、長 n 的字串,去掉尾端空白。
func str(rec []byte, pos1, n int) string {
	s := rec[pos1-1 : pos1-1+n]
	end := len(s)
	for end > 0 && (s[end-1] == ' ' || s[end-1] == 0) {
		end--
	}
	return string(s[:end])
}
