package main

import (
	"testing"

	"shardofspring/internal/combat"
	"shardofspring/internal/music"
)

// 驗收 5(docs/spec/13 §8.3):施法出 PS —— **成功與失敗都出**,
// 因為響的是施法這個動作本身(cast_scene.go 的具名決定)。
func TestCastQueuesTheSpellSound(t *testing.T) {
	for _, c := range []struct {
		name   string
		fizzle bool
	}{{"發動", false}, {"失敗", true}} {
		g, s := groupSpellGame(t, 2, 14, 12)
		invest := 30
		if c.fizzle {
			g.field.Rand = alwaysHighRand{}
			invest = 1
		}
		g.field.Sounds = nil
		g.castAt(s, invest, 13, 12)
		if !sliceHas(g.field.Sounds, music.FxSpell) {
			t.Errorf("%s:音效 %v,應含 %s", c.name, g.field.Sounds, music.FxSpell)
		}
	}
}

// pumpEffects 一定要把 Sounds 清掉,**即使這台機器放不出聲音**。
//
// ⚠ 不清的症狀是「記憶體慢慢長」,不是「沒有聲音」—— 兩者長得完全不一樣,
// 所以要單獨釘住(docs/spec/13 §8.3 驗收 6/7)。
func TestPumpEffectsDrainsWithoutAudio(t *testing.T) {
	g := &Game{field: &combat.Field{}}
	g.field.Sounds = []string{music.FxHit, music.FxDie}
	g.sound = nil // 沒有音訊裝置
	g.pumpEffects()
	if len(g.field.Sounds) != 0 {
		t.Errorf("沒有音訊裝置時也要清空佇列,還剩 %v", g.field.Sounds)
	}

	// 玩家關掉聲音:規則層照樣填,播放層照樣清,而且不積壓。
	g.sound = &sound{off: true}
	g.field.Sounds = []string{music.FxHit}
	g.pumpEffects()
	if len(g.field.Sounds) != 0 || len(g.sound.fx) != 0 {
		t.Errorf("關掉聲音之後不該積壓:Sounds=%v fx=%v", g.field.Sounds, g.sound.fx)
	}
}

// 佇列有上限,滿了丟最舊的 —— 積壓會讓玩家聽到上一回合的聲音。
func TestEffectQueueIsCapped(t *testing.T) {
	g := &Game{field: &combat.Field{}, sound: &sound{}}
	for i := 0; i < fxQueueMax*3; i++ {
		g.field.Sounds = append(g.field.Sounds, music.FxHit)
	}
	g.pumpEffects()
	if len(g.sound.fx) > fxQueueMax {
		t.Errorf("佇列 %d 個,上限是 %d", len(g.sound.fx), fxQueueMax)
	}
}

// fxFrames 要對得上音長:HT 是 8 拍 ≈ 0.44 秒 ≈ 26 格。
//
// ⚠ 算錯的症狀是音效**互相蓋掉**或**間隔變長**,兩者都不像 bug。
func TestFxFramesMatchesTheDuration(t *testing.T) {
	n := music.Effect(music.FxHit)
	if got := fxFrames(n); got < 20 || got > 35 {
		t.Errorf("HT 佔 %d 格,8 拍 ÷ 18.2 × 60 應在 26 附近", got)
	}
	if got := fxFrames(nil); got != 1 {
		t.Errorf("空的音串應回 1 格(不能回 0,會變成每一格都試著播),得 %d", got)
	}
}

func sliceHas(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
