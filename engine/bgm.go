package main

import (
	"github.com/hajimehoshi/ebiten/v2/audio"

	"shardofspring/internal/music"
)

// 場景配樂的播放層。設計與約束見 internal/music/cue.go 的檔頭。
//
// ⚠ 與 sound.go 的 `play` 分工:
//   - `play` 放**一次性**的曲子(原版那兩首),放完就結束
//   - 這裡放**循環**的場景配樂,直到場景換掉
//
// 兩者不能同時響 —— `play` 會先把場景配樂停掉(sound.go)。

// bgmState 是目前正在循環的那一首。
type bgmState struct {
	cue music.Cue
	// set = 這個 cue 已經處理過了(可能是播放中,也可能是「這個場景本來就靜音」)。
	// ⚠ 與 `player != nil` **不是同一件事**:靜音的 cue 也算處理過,
	// 少了這個旗標,靜音場景會每一格都重跑一次判斷。
	set    bool
	player *audio.Player
}

// currentCue 依目前所在的場景決定要放哪一首。
//
// ⚠ 順序就是優先序,與 scene.go 的 `inputChain` 同一個道理:
// 戰鬥蓋在城鎮與迷宮之上(在城鎮裡被襲擊時要聽到戰鬥曲)。
func (g *Game) currentCue() music.Cue {
	if g.shell != nil && g.shell.mode != shellPlaying {
		switch g.shell.mode {
		case shellEnding, shellWipe:
			// 通關曲與死亡曲是**一次性**的,由 scene.go / combat_scene.go
			// 直接 play。這裡讓開,不要蓋過去。
			return music.CueNone
		default:
			return music.CueTitle
		}
	}
	switch {
	case g.field != nil:
		if g.bossFight {
			return music.CueBoss
		}
		return music.CueCombat
	case g.town != nil:
		return music.CueTown
	case g.level != nil:
		return music.CueMaze
	}
	return music.CueWorld
}

// updateBGM 每一格呼叫一次。場景沒換就什麼都不做。
func (g *Game) updateBGM() {
	c := g.currentCue()
	if g.bgm.set && g.bgm.cue == c {
		return
	}
	g.stopBGM()
	g.bgm.cue, g.bgm.set = c, true

	score := music.Loop(g.musicMode, c)
	if score == nil || g.sound == nil || g.sound.ctx == nil || g.sound.off {
		return
	}
	src, length, err := g.source(music.CueFile(g.musicMode, c), score)
	if err != nil || length == 0 {
		return
	}
	// ⚠ 循環要交給 audio.NewInfiniteLoop,不要自己在 Update 裡看播完沒有 ——
	// 後者會在每一圈之間留下一格的靜音,而那個縫在慢的曲子上聽得出來。
	p, err := g.sound.ctx.NewPlayer(audio.NewInfiniteLoop(src, length))
	if err != nil {
		g.warnOnce("配樂播放失敗:" + err.Error())
		return
	}
	p.Play()
	g.bgm.player = p
}

// stopBGM 停掉目前循環的那一首。**不清 `set`** ——
// 呼叫端要的是「停下來」,不是「下一格重放」。
func (g *Game) stopBGM() {
	if g.bgm.player != nil {
		_ = g.bgm.player.Close()
		g.bgm.player = nil
	}
}

// cycleMusicMode 循環切換配樂模式,回傳要顯示的那一句。
//
// ⚠ 切完要把 `set` 清掉再讓下一格重算 —— 模式換了,同一個場景的答案也換了。
func (g *Game) cycleMusicMode() string {
	g.musicMode = music.Mode((int(g.musicMode) + 1) % music.ModeCount)
	g.bgm.set = false
	g.saveConfig()
	switch g.musicMode {
	case music.ModeRemake:
		return "配樂:重製(場景配樂是後加的,原版沒有)"
	case music.ModeOff:
		return "配樂:關閉"
	default:
		return "配樂:原版(只有通關曲與死亡曲)"
	}
}
