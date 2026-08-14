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
