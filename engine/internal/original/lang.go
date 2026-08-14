package original

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
)

// 譯文表。docs/spec/10-localization.md。
//
// TSV 欄位:row \t field \t original \t orig_bytes \t translation \t trans_bytes \t fits \t note
// 其中 `row` 是記錄編號、`field` 是欄位識別(數字或名稱)。

// Lang 是「記錄編號 + 欄位」→ 譯文。
type Lang map[langKey]string

type langKey struct {
	Row   int
	Field string
}

// Get 回傳譯文;沒有就回 fallback。
//
// ⚠ **缺漏時回原文,不回空字串**(docs/spec/10 §2)。空字串在畫面上是
// 「這一格沒東西」,看起來像資料壞了,不像沒翻譯。
func (l Lang) Get(row int, field, fallback string) string {
	if l == nil {
		return fallback
	}
	if v, ok := l[langKey{row, field}]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

// ParseLangTSV 解析 translations/names/*.tsv。
func ParseLangTSV(d []byte) Lang {
	out := Lang{}
	sc := bufio.NewScanner(bytes.NewReader(d))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	first := true
	for sc.Scan() {
		ln := sc.Text()
		if first { // 標題列
			first = false
			continue
		}
		f := strings.Split(ln, "\t")
		if len(f) < 5 {
			continue
		}
		row, err := strconv.Atoi(strings.TrimSpace(f[0]))
		if err != nil {
			continue
		}
		out[langKey{row, strings.TrimSpace(f[1])}] = strings.TrimSpace(f[4])
	}
	return out
}

// ParseDungeonTextTSV 解析 translations/dungeon-text/*.tsv:id \t original \t translation \t note
func ParseDungeonTextTSV(d []byte) map[int]string {
	out := map[int]string{}
	sc := bufio.NewScanner(bytes.NewReader(d))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		f := strings.Split(sc.Text(), "\t")
		if len(f) < 3 {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(f[0]))
		if err != nil {
			continue
		}
		if t := strings.TrimSpace(f[2]); t != "" {
			out[id] = t
		}
	}
	return out
}

// ParsePlaceTSV 解析 translations/source/towndata.tsv:
// kind \t town \t original \t orig_bytes \t translation \t …
//
// 回「原文 → 譯文」。城鎮名與商店名混在同一個檔裡,用 `original` 當鍵 ——
// ⚠ **不用列號**,因為那個檔的列序與 TOWNDATA.DAT 的記錄序不保證一致。
func ParsePlaceTSV(d []byte) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(d))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		f := strings.Split(sc.Text(), "\t")
		if len(f) < 5 {
			continue
		}
		orig, zh := strings.TrimSpace(f[2]), strings.TrimSpace(f[4])
		if orig != "" && zh != "" {
			out[orig] = zh
		}
	}
	return out
}

// ExtractRumors 從 TOWN.EXE 抽出酒館傳聞。docs/re/138 §4。
//
// 位址範圍與長度門檻都是**觀察到的**,不是從程式碼讀出來的 ——
// 所以這支函式回傳的段數要當成證據看:
//
//	10 段找到、11 個索引(酒館的 TOWNDATA 位移 36 是 1–11)
//
// ⚠ **差的那一段不補。** 呼叫端查不到索引時要明講,不要拿別段頂替。
func ExtractRumors(town []byte) map[int]string {
	const lo, hi, minLen = 0x032C0, 0x03A40, 55
	out := map[int]string{}
	n := 0
	start := -1
	flush := func(end int) {
		if start < 0 || end-start < minLen {
			start = -1
			return
		}
		s := strings.TrimSpace(string(town[start:end]))
		// 去掉開頭的描述子殘骸(非字母、非引號的前幾個位元組)
		for len(s) > 0 && !(s[0] >= 'A' && s[0] <= 'Z') && s[0] != '"' {
			s = s[1:]
		}
		if len(s) >= minLen {
			n++
			out[n] = s
		}
		start = -1
	}
	end := hi
	if end > len(town) {
		end = len(town)
	}
	for i := lo; i < end; i++ {
		if town[i] >= 0x20 && town[i] < 0x7F {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(end)
	return out
}
