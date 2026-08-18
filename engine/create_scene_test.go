package main

// 創角流程的驗收(docs/re/143 + 2026-08-18 的原版實跑)。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/town"
)

// TestCreateAdjustIsCappedAtThreeRounds —— 重擲只有三輪,ESC 一律用掉一輪。
//
// ⚠ 這是**規則**不是介面:重擲嚴格有利(屬性 2…13 的三角分佈,不滿意再擲,
// 沒有代價),所以無限重擲等於讓玩家把五項刷到 13。
// 先前的寫法是「沒選任何一項的 ESC 才算做完」—— 選了就永遠不會結束。
func TestCreateAdjustIsCappedAtThreeRounds(t *testing.T) {
	g := newShellTestGame(t)
	g.rand = combat.NewRand(1)
	g.rand = combat.NewRand(1)
	g.openCreate()
	g.createKey(ebiten.KeyH) // 人類:唯一可以選職業的種族
	if g.create.step != stepAdjust {
		t.Fatalf("選完種族應該進調整,得 %v", g.create.step)
	}
	for i := 0; i < town.CreateAdjustRounds; i++ {
		if g.create.step != stepAdjust {
			t.Fatalf("第 %d 輪就離開調整了", i+1)
		}
		g.createKey(ebiten.Key1) // 每一輪都選一項 —— 先前這樣會永遠出不去
		g.createKey(ebiten.KeyEscape)
	}
	if g.create.step != stepClass {
		t.Errorf("三輪用完應該進選職業,得 %v(輪數 %d)", g.create.step, g.create.round)
	}
}

// TestCreateAsksToKeepLast —— 「要保留這位角色嗎」問在**最後**。
//
// 原版看得到成品(職業、名字)才問;先前問在只看得到屬性的時候,
// 而且 `N` 的意思是「重擲一組」不是「放棄」—— 沒有放棄的出口。
func TestCreateAsksToKeepLast(t *testing.T) {
	g := newShellTestGame(t)
	g.rand = combat.NewRand(1)
	before := countOccupied(g.chars)
	g.openCreate()
	g.createKey(ebiten.KeyH)
	for i := 0; i < town.CreateAdjustRounds; i++ {
		g.createKey(ebiten.KeyEscape)
	}
	// 職業:原版是 A) Warrior / B) Wizard
	g.createKey(ebiten.KeyA)
	if g.create.step != stepName {
		t.Fatalf("A 應該選戰士並進命名,得 %v", g.create.step)
	}
	for _, r := range "Zed" {
		g.createRunes([]rune{r})
	}
	g.createKey(ebiten.KeyEnter)
	if g.create.step != stepKeep {
		t.Fatalf("命名完應該問要不要保留,得 %v", g.create.step)
	}
	// N = 放棄,名冊人數不變。
	g.createKey(ebiten.KeyN)
	if g.create != nil {
		t.Error("按 N 之後創角畫面該關掉")
	}
	if n := countOccupied(g.chars); n != before {
		t.Errorf("放棄之後名冊人數 %d,原本 %d", n, before)
	}
}

func countOccupied(cs []original.Character) int {
	n := 0
	for _, c := range cs {
		if c.Occupied() {
			n++
		}
	}
	return n
}
