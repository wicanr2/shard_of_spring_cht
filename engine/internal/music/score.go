package music

// 原版的樂譜。docs/re/148 —— 逐字抄自兩支 EXE 的字串表。
//
// ⚠ 這些是**原版檔案裡的字串**,照抄不改。要調整聽感請改 Render,
// 不要改譜。

// Ending 是 `MAZEMOVE.EXE` `0x04CBC`–`0x04DB0` 的十段。
//
// ⚠ 後兩段**沒有 `T`/`MB`** —— 它們沿用前面的狀態,
// 所以這十段要當成**一首曲子的續播**,用同一份 State 依序解。
//
// ⚠ 「這是通關曲」是**位置上的推測**(緊鄰結局文字),不是讀到呼叫端。
var Ending = []string{
	"MB T108 O3 L8a",
	"MB T108 O3 L16",
	"MB T108 O3 L8 E F#GD",
	"MB T108 O3 L2 G ",
	"MB T108 O4 L8 CC",
	"MB T108 O4 L4 EC",
	"MB T108 O4 L8 DC",
	"MB T108 O3 L8 BA",
	"O3 L8 G ",
	"O3 L3 G ",
}

// Userlib 是 `USERLIB.EXE` `0x0180E`–`0x01868` 的五段。
//
// ⚠ `T50` 比 Ending 慢一半;`USERLIB` 的匯出槽裡有死亡與結局兩個,
// 所以它**可能**是死亡音效 —— 同樣沒有讀到呼叫端。
var Userlib = []string{
	"MB T50 O3 L8 ML DEFE",
	"MB T50 O3 L8 ML FE",
	"MB T50 O3 L4 ML DE",
	"MB T50 O3 L4 ML FF",
	"MB T50 O3 L8 ML AG",
}
