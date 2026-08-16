//go:build !eten

package main

// 預設(發行版)的字型:開源向量字。docs/spec/04 §4 —— 授權要可再散布。
//
// 另一個版本在 font_eten.go(build tag `eten`),用倚天中文系統的點陣字。
// ⚠ **兩個版本的畫面不一樣**:字級與欄寬預算各有一套
// (internal/layout 的欄寬常數同樣分兩個檔),截圖也要分開拍。

import (
	"fmt"

	"shardofspring/internal/render"
)

// fontVariant 印在啟動訊息裡,讓「現在跑的是哪一套字」看得見。
const fontVariant = "開源向量字"

// newPainters 建三個字級的 Painter:側欄/訊息 20px、敘述覆蓋層 24px、標題 32px。
func newPainters(fontPath string) (panel, overlay, title *render.Painter, name string, err error) {
	src, path, err := render.LoadFont(fontPath)
	if err != nil {
		return nil, nil, nil, "", err
	}
	return render.NewPainter(src, 20, cgaWhite),
		render.NewPainter(src, 24, cgaWhite),
		render.NewPainter(src, 32, cgaWhite),
		fmt.Sprintf("%s(%s)", fontVariant, path), nil
}
