// Package layout 定義畫面版面。
//
// 全部數字出自 docs/spec/04-display-layout.md §2,**不要在這裡即興調整** ——
// 要改先改規格。
package layout

// 畫布與美術縮放。docs/spec/04 §1。
const (
	ScreenW = 1024
	ScreenH = 768

	// TileSrc 是原版圖塊的邊長(docs/formats/07 §1:17×17)。
	TileSrc = 17
	// ArtScale 必須是整數 —— 非整數倍會讓像素藝術糊掉(docs/spec/04 §1)。
	ArtScale = 4
	// TileDst 是放大後的圖塊邊長。
	TileDst = TileSrc * ArtScale

	// ViewTiles 是主視野的邊長(格)。153 = 9 × 17,原版圖畫區正好 9×9 格
	// (docs/formats/07:PICT*.BIN 是 153×153)。
	ViewTiles = 9

	Margin = 24
)

// Rect 是版面區塊。用 int 而非 float —— 版面全部落在整數像素上,
// 浮點會在縮放時引入半像素邊緣。
type Rect struct{ X, Y, W, H int }

func (r Rect) Right() int  { return r.X + r.W }
func (r Rect) Bottom() int { return r.Y + r.H }

// 五個區塊。docs/spec/04 §2 的表格。
var (
	// View 是主視野 / 清單區。612 = 9 × 17 × 4。
	View = Rect{Margin, Margin, ViewTiles * TileDst, ViewTiles * TileDst}

	// Party 是隊伍狀態:5 名角色 × 名稱/HP/SP + 金幣/食糧。
	Party = Rect{View.Right() + Margin, Margin, 340, 300}

	// Message 是短訊息與即時戰鬥訊息。長敘述走 Overlay。
	Message = Rect{Party.X, Party.Bottom() + Margin, Party.W, 288}
	// MsgCols 是訊息面板一行放得下的欄數(全形算 2,ui.Cols 的定義)。
	// 340 px 扣兩側內距 16 → 內寬 308 px,20 px 字 → 30 欄
	// (docs/spec/04 §5)。**折行不截斷** —— 截掉的字看不出來,折行看得出來。
	MsgCols = 30

	// Prompt 是底部的按鍵提示列。原版就有(手冊 p.31 截圖右下),
	// 本專案把它從側欄移到底部整條,因為中文的按鍵說明比英文長。
	Prompt = Rect{Margin, View.Bottom() + Margin, ScreenW - 2*Margin, 84}
	// PromptCols 是提示列一行放得下的欄數。992 px 扣兩側內距 16 → 960 px,
	// 20 px 的字半形約 10 px → 96 欄。
	//
	// ⚠ 這是**上限不是目標**:超過就會畫到框外,而畫到框外的字**照樣印得出來**,
	// 只是壓在別的面板上 —— 沒有任何錯誤,只有一團看不懂的畫面
	// (戰鬥那一行就這樣壓了右邊的隊伍面板很久,拍截圖才看見)。
	PromptCols = 96
	// ViewCols 是主視野一行放得下的欄數:612 px 扣內距 → 580 px ÷ 10 = 58 欄。
	ViewCols = 58

	// Overlay 是敘述覆蓋層,置中於主視野之上。
	// 800×260 內距 32 → 24px 字每行 30 個全形字 × 5 行 = 300 欄容量,
	// 而最長的一段地城敘述是 207 欄 —— 一頁放得下,不必分頁(docs/spec/04 §3)。
	Overlay = Rect{(ScreenW - 800) / 2, (ScreenH - 260) / 2, 800, 260}
)

// OverlayPad 是覆蓋層的內距(docs/spec/04 §3)。
const OverlayPad = 32
