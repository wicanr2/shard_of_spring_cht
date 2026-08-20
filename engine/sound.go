package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

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
	// fx 是等著播的音效代碼,fxWait 是還要等幾格才輪到下一個。
	//
	// ⚠ **要排隊,不能同一格全部播出去。** 一次怪物回合可以打三次、
	// 死兩隻,五個音同時響會疊成一團噪音 —— 而它疊起來仍然像音效,
	// 不像壞掉,所以沒有人會回報。原版是同步發聲(`SOUND` 會等),
	// 這裡用「播一個、等它播完」把那個順序還原。
	fx     []string
	fxWait int
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
	// 場景配樂跟著總開關走。⚠ 關的時候要**真的停掉**(循環曲不會自己結束),
	// 開的時候把 `set` 清掉讓下一格重算 —— 場景沒換,但答案換了。
	g.stopBGM()
	g.bgm.set = false
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

// play 放一首曲子,`name` 是內嵌的 OGG 名、`score` 是同一首的譜。
// **不阻塞**,也不保證放得出來。
//
// ⚠ 一次性的曲子(通關、全滅)要**壓過**循環的場景配樂 ——
// 兩者同時響會疊成噪音,而且疊起來還是聽得出旋律,
// 所以不會有人把它當成壞掉,只會覺得音樂很奇怪(bgm.go)。
func (g *Game) play(name string, score []string) {
	g.stopBGM()
	if g.sound == nil || g.sound.ctx == nil || g.sound.off {
		return
	}
	src, _, err := g.source(name, score)
	if err != nil {
		g.warnOnce("音訊播放失敗:" + err.Error())
		return
	}
	p, err := g.sound.ctx.NewPlayer(src)
	if err != nil {
		g.warnOnce("音訊播放失敗:" + err.Error())
		return
	}
	p.Play()
}

// source 回傳一首曲子的取樣來源與長度(位元組)。
//
// **先試內嵌的 OGG,失敗才現算。** 兩條路的輸出格式相同
// (16-bit LE 立體聲 @ music.SampleRate),所以呼叫端不必分辨用了哪一條。
//
// ⚠ 退路存在的理由不是「以防萬一」,是**故障時的可觀察性**:
// 資產掉了或解碼壞了還聽得到聲音,比「安靜而且沒人知道為什麼」好 ——
// 而安靜看起來跟「玩家把音樂關掉了」一模一樣。
func (g *Game) source(name string, score []string) (io.ReadSeeker, int64, error) {
	if data := music.OGG(name); data != nil {
		// ⚠ 用 `DecodeWithSampleRate` 不是 `DecodeF32`:前者輸出
		// **16-bit LE 立體聲**,與 `RenderPCM16` 和 `ctx.NewPlayer` 對得上。
		// `DecodeF32` 出來的是 float32,要配 `NewPlayerF32` ——
		// 混用不會編譯失敗,會**放出噪音**。
		s, err := vorbis.DecodeWithSampleRate(music.SampleRate, bytes.NewReader(data))
		if err == nil {
			return s, s.Length(), nil
		}
		g.warnOnce("配樂解碼失敗,改用現算:" + err.Error())
	}
	if score == nil {
		return nil, 0, errNoSource
	}
	pcm := music.RenderPCM16(music.ParseAll(score), music.SampleRate)
	return bytes.NewReader(pcm), int64(len(pcm)), nil
}

var errNoSource = errors.New("這首曲子既沒有 OGG 也沒有譜")

// warnOnce 讓同一類警告只出現一次 —— 每一格都噴一行會把提示列洗掉。
func (g *Game) warnOnce(msg string) {
	if g.sound == nil || g.sound.warned {
		return
	}
	g.warnings = append(g.warnings, msg)
	g.sound.warned = true
}

// fxQueueMax 是音效佇列的上限。滿了就丟掉最舊的 ——
// ⚠ 積壓比漏掉更糟:玩家會聽到上一回合的聲音配這一回合的畫面。
const fxQueueMax = 6

// pumpEffects 把規則層排的音效(`combat.Field.Sounds`)搬進播放佇列,
// 每一格最多放一個。
//
// ⚠ **不論播不播得出來都要清空 `Sounds`** —— 沒有音效卡、玩家關了聲音、
// 佇列滿了,規則層都照樣在填。不清的話它會一路長到戰鬥結束。
func (g *Game) pumpEffects() {
	if g.field != nil && len(g.field.Sounds) > 0 {
		if g.sound != nil {
			g.sound.fx = append(g.sound.fx, g.field.Sounds...)
			if n := len(g.sound.fx) - fxQueueMax; n > 0 {
				g.sound.fx = g.sound.fx[n:]
			}
		}
		g.field.Sounds = g.field.Sounds[:0]
	}
	s := g.sound
	if s == nil || s.ctx == nil || s.off {
		// ⚠ 關著聲音的時候把佇列也倒掉,否則開回來會一次全響。
		if s != nil {
			s.fx = nil
		}
		return
	}
	if s.fxWait > 0 {
		s.fxWait--
		return
	}
	if len(s.fx) == 0 {
		return
	}
	code := s.fx[0]
	s.fx = s.fx[1:]
	notes := music.Effect(code)
	if len(notes) == 0 {
		return
	}
	pcm := music.RenderPCM16(notes, music.SampleRate)
	p, err := s.ctx.NewPlayer(bytes.NewReader(pcm))
	if err != nil {
		g.warnOnce("音效播放失敗:" + err.Error())
		return
	}
	p.Play()
	s.fxWait = fxFrames(notes)
}

// fxFrames 是這一串音要佔幾格(60 fps)。
func fxFrames(notes []music.Note) int {
	var sec float64
	for _, n := range notes {
		sec += n.Dur
	}
	f := int(sec * float64(ebiten.TPS()))
	if f < 1 {
		f = 1
	}
	return f
}
