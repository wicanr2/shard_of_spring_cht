package main

import (
	"bytes"
	"fmt"
	"os"

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
	// off = 玩家自己關掉的(CMBT:52/53 的 `Sound is now on./off.`)。
	// ⚠ 與 `g.sound == nil` **不是同一件事**:nil 是「這台機器放不出聲音」,
	// off 是「玩家不想聽」。合成一個的話,關過聲音之後就分不出
	// 「音訊壞了」與「自己關的」——而前者要警告、後者不要。
	off bool
}

// 音效開關的兩句話,照 `translations/module-text/CMBT.tsv` 第 52/53 列(F3)。
const (
	soundOnMsg  = "音效已開啟。"
	soundOffMsg = "音效已關閉。"
)

// toggleSound 切換音效,回傳要顯示的那一句。
//
// ⚠ 音訊 context 開不起來時**照樣讓玩家切**,只是切了沒有聲音 ——
// 這台機器有沒有音效卡不該改變按鍵的行為。
func (g *Game) toggleSound() string {
	if g.sound == nil {
		g.sound = &sound{} // 只保存開關狀態,沒有 ctx 就不會播
	}
	g.sound.off = !g.sound.off
	if g.sound.off {
		return soundOffMsg
	}
	return soundOnMsg
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
	if g.sound == nil || g.sound.ctx == nil || g.sound.off {
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
