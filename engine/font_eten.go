//go:build eten

package main

// 本機版的字型:倚天中文系統 3.53 的 24×24 明體點陣字(docs/spec/21)。
//
// ⛔ **這個版本不能發行。** 倚天字型是 1993 年的商業軟體,
// `CLAUDE.md` §1 與 docs/spec/04 §4 要求公開產出只有引擎程式碼與翻譯文本 ——
// 打包給別人的版本要用預設的 `font_ttf.go`(不加 `-tags eten`)。
//
// 字型資產由 `tools/eten_font.py` 產生,放在 engine/assets/font/eten24.bin
// (gitignore)。沒有那個檔就編不起來 —— **這是刻意的**:
// 編得起來但沒有字,會變成一個畫面全空、原因看不出來的執行檔。

import (
	_ "embed"
	"fmt"

	"shardofspring/internal/render"
)

//go:embed assets/font/eten24.bin
var etenFont []byte

const fontVariant = "倚天 24×24 明體"

// newPainters 建三個 Painter。**點陣字只有一種字身**,所以字級靠整數倍放大:
// 側欄/訊息與敘述覆蓋層都是 1×(24px),標題 2×(48px)。
//
// ⚠ 側欄先前是 20px 向量字,換成 24px 點陣之後**欄寬預算跟著變**
// (internal/layout 的 cols_eten.go)—— 那不是可以事後再調的細節,
// 清單放不放得下直接取決於它。
func newPainters(_ string) (panel, overlay, title *render.Painter, name string, err error) {
	f, err := render.LoadBitmapFont(etenFont)
	if err != nil {
		return nil, nil, nil, "", err
	}
	return render.NewBitmapPainter(f, 1, cgaWhite),
		render.NewBitmapPainter(f, 1, cgaWhite),
		render.NewBitmapPainter(f, 2, cgaWhite),
		fmt.Sprintf("%s(%d 字)", fontVariant, len(etenFont)), nil
}
