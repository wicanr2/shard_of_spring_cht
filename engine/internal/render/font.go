package render

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// FontCandidates 是預設會去找的字型檔,依序嘗試。
//
// 全部是**可再散布**的授權(OFL / MOE 開放使用)——
// docs/spec/04 §4:「選字型時先看授權,不要等打包才發現不能附」。
//
// ⚠ M3 是**從檔案載入**,不是 embed。spec/04 說的 embed 需要先做子集化
// (整份 Noto Sans CJK 是 19 MB),那排在 M7 中文化上線。
var FontCandidates = []string{
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/moe/MoeStandardSong.ttf",
}

// LoadFont 載入字型;path 為空時依序試 FontCandidates。
//
// `.ttc` 是字型集合,一個檔裡有 JP/KR/SC/TC/HK 五套。**必須挑 TC** ——
// 挑錯的話大部分字仍然畫得出來(共用碼位),只有少數字形是簡體或日文寫法,
// **而那種錯誤要逐字比對才看得出來**。
func LoadFont(path string) (*text.GoTextFaceSource, string, error) {
	paths := FontCandidates
	if path != "" {
		paths = []string{path}
	}
	var tried []string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			tried = append(tried, fmt.Sprintf("%s(%v)", p, err))
			continue
		}
		srcs, err := text.NewGoTextFaceSourcesFromCollection(bytes.NewReader(b))
		if err != nil || len(srcs) == 0 {
			// 不是集合就當單一字型再試一次
			s, err2 := text.NewGoTextFaceSource(bytes.NewReader(b))
			if err2 != nil {
				tried = append(tried, fmt.Sprintf("%s(%v)", p, err))
				continue
			}
			return s, p, nil
		}
		if s := pickTC(srcs); s != nil {
			return s, p, nil
		}
		return srcs[0], p, nil
	}
	return nil, "", fmt.Errorf("找不到可用的中文字型,試過:%s", strings.Join(tried, "、"))
}

// pickTC 從集合裡挑繁體中文那一套。挑不到回 nil,由呼叫端決定。
func pickTC(srcs []*text.GoTextFaceSource) *text.GoTextFaceSource {
	for _, s := range srcs {
		f := s.Metadata().Family
		if strings.Contains(f, "TC") || strings.Contains(f, "HK") ||
			strings.Contains(f, "Traditional") {
			return s
		}
	}
	return nil
}
