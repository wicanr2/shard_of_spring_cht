package main

// 野外與地城的紮營入口。
//
// 原版**在世界地圖與地城都能按 `C` 紮營**:`Making Camp..` 這個字串在
// `WRLDMOVE.EXE` 與 `MAZEMOVE.EXE` 各出現一次、`TOWN.EXE` **零次**
// (2026-08-18 對照原版量到的,docs/spec/14 §12-B)。
//
// ⚠ 這不只是少一個入口 —— 營地的 11 個指令(睡覺回血、打獵補糧、鑑定、
// 換裝備、調隊形)在原版是**野外隨時可用**的。只能在城鎮紮營等於強迫玩家
// 走回城才能休息,遠征地城的節奏完全不同。
//
// ⚠ 城鎮那一邊仍然用 `Z` 而不是 `C` —— 建築清單的字母從 A 起算,
// 而 `C) Hamlet Hospital` 已經佔掉 C(town_scene.go)。**同一個鍵在不同畫面
// 有不同意思是原版就有的事**,不是這裡引進的。
func (g *Game) makeCamp(wild bool) {
	if g.town != nil {
		return // 已經在城鎮/營地裡
	}
	g.town = &townState{
		mode: townCamp,
		wild: wild,
		msg:  "紮營中……", // WRLDMOVE:43 / MAZEMOVE:89
	}
}

// campInPlace 回傳「這個營地紮在地圖上」——野外或地城,不是城鎮裡。
//
// 原版紮營時**地圖仍然畫著**,隊伍那一格換成帳篷,選單開在右下角那個框
// (2026-08-18 對照原版截圖 `workplace/qa/k0-camp.png`)。城鎮裡的營地
// 沒有地圖可留,走的是原本那條「整個視野換成選單」的路。
func (g *Game) campInPlace() bool {
	return g.town != nil && g.town.mode == townCamp && g.town.name == ""
}

