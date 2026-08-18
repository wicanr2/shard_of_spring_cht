package main

import "shardofspring/internal/world"

// 朝向的顯示。原版側欄寫 `Facing ↓`(2026-08-18 對照原版的截圖)。
//
// ⚠ 編碼是 **1 北 2 東 3 南 4 西**(docs/re/71),與戰鬥、`MAZEDATA` 同一套。

// facingArrow 把朝向編號轉成箭頭 + 中文。
func facingArrow(f int) string {
	switch f {
	case 1:
		return "↑ 北"
	case 2:
		return "→ 東"
	case 3:
		return "↓ 南"
	case 4:
		return "← 西"
	}
	return "－"
}

// facingNow 回傳**目前這個場景**的朝向。
//
// ⚠ 世界地圖與地城各有一份朝向(`world.State.Facing` 與 `maze.State.Facing`),
// 不是同一個變數 —— 側欄要顯示哪一個看玩家在哪裡。
func (g *Game) facingNow() int {
	if g.level != nil {
		return int(g.mazeState.Facing)
	}
	return int(world.Facing(g.party.Facing))
}
