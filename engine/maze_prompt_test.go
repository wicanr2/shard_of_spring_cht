package main

// 地城機關問答的接線測試。

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
)

// TestPoolKeepsAskingAfterHealing —— 治療池治好一個人之後**問句留著**。
//
// 原版再問一次 `Which party member do you wish to heal? (0 exits)`,
// 要按 0 才離開(2026-08-18 實跑 `q3b-s5.png`)。
// 先前治完就把問句收掉 —— 治第二個人得重走一次整段路,
// 而那條路在兩軸修好之前根本走不通。
func TestPoolKeepsAskingAfterHealing(t *testing.T) {
	g := &Game{
		rand:    combat.NewRand(3),
		members: []original.Character{{Name: "凱恩", HP: 3, MaxHP: 20}},
		chars:   []original.Character{{Name: "凱恩", HP: 3, MaxHP: 20}},
	}
	g.prompt = &mazePrompt{kind: promptPool, head: "要治療哪一位?(0離開)"}

	g.poolKey(ebiten.KeyDigit1)
	if g.prompt == nil {
		t.Fatal("治好一個人之後問句不該關掉")
	}
	if g.members[0].HP <= 3 {
		t.Errorf("沒有治到:生命 %d", g.members[0].HP)
	}
	if !strings.Contains(strings.Join(g.prompt.lines(), "\n"), "已治療") {
		t.Errorf("回饋要跟問句一起顯示:%v", g.prompt.lines())
	}
	// 0 才離開。
	g.poolKey(ebiten.KeyDigit0)
	if g.prompt != nil {
		t.Error("按 0 應該離開")
	}
	_ = maze.PoolRollFaces // 規則常數在 internal/maze,這裡只測接線
}
