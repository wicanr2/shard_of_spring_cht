package main

import (
	"bytes"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"shardofspring/internal/music"
)

// 播放層。docs/spec/13-sound.md §3。
//
// ⚠ **音訊裝置開不起來時只記一行警告,遊戲照常跑。**
// 這台開發機沒有音效卡也沒有 DISPLAY,而 `internal/music` 的解析與合成
// 都是純函式 —— 把「算得出聲音」與「放得出聲音」分開,
// 前者測得起來,後者壞掉也不會讓遊戲起不來。

type sound struct {
	ctx *audio.Context
	// warned 讓警告只出現一次 —— 每一格都噴一行會把提示列洗掉。
	warned bool
}

func (g *Game) initSound() {
	// audio.NewContext 同一個行程只能呼叫一次,失敗就整個關掉聲音。
	defer func() {
		if r := recover(); r != nil {
			g.warnings = append(g.warnings, fmt.Sprintf("音訊關閉:%v", r))
			g.sound = nil
		}
	}()
	g.sound = &sound{ctx: audio.NewContext(music.SampleRate)}
}

// play 放一段樂譜。**不阻塞**,也不保證放得出來。
func (g *Game) play(score []string) {
	if g.sound == nil || g.sound.ctx == nil {
		return
	}
	pcm := music.RenderPCM16(music.ParseAll(score), music.SampleRate)
	p, err := g.sound.ctx.NewPlayer(bytes.NewReader(pcm))
	if err != nil {
		if !g.sound.warned {
			g.warnings = append(g.warnings, "音訊播放失敗:"+err.Error())
			g.sound.warned = true
		}
		return
	}
	p.Play()
}
