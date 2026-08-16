//go:build eten

package layout

// 欄寬預算(本機版:倚天 24×24 點陣字,半形固定 12 px)。docs/spec/21。
//
// 點陣字是**等寬**的,所以這幾個數字是除法的結果,沒有估計成分 ——
// 但也因此**比向量字版少兩成**:同一個面板放得下的字變少了。
// 清單放不放得下要照這一組重新確認,不能沿用 cols_ttf.go 的數字。
const (
	// MsgCols:訊息面板 308 px ÷ 12 = 25 欄。
	MsgCols = 25
	// ViewCols:主視野 580 px ÷ 12 = 48 欄。
	ViewCols = 48
	// PromptCols:提示列 960 px ÷ 12 = 80 欄。
	PromptCols = 80
)
