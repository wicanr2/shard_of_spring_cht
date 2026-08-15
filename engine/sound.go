package main

import (
	"os"
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

// NoSoundEnv 設了(任何非空值)就完全不建音訊 context。
//
// ⚠ **這不是「方便」,是必要的**:在沒有音訊裝置的環境(容器、CI、
// headless 截圖),oto 開 ALSA 失敗的錯誤是**由 ebiten 的遊戲迴圈丟回來的**,
// `RunGame` 會直接以錯誤結束 —— 遊戲跑不到第二格。
//
// docs/spec/13 說「沒有音訊裝置時只記警告」,那是指 initSound 的 recover;
// **播放期的錯誤是另一條路,recover 攔不到,它會中斷整個迴圈。**
const NoSoundEnv = "SHARD_NOSOUND"

func (g *Game) initSound() {
	if os.Getenv(NoSoundEnv) != "" {
		g.sound = nil
		return
	}
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
