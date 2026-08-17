package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"shardofspring/internal/combat"
	"shardofspring/internal/music"
)

// 場景 → 配樂點。⚠ 這張表就是規格(docs/spec/13 §7),
// 順序有意義:戰鬥蓋在城鎮與迷宮之上。
func TestCurrentCueFollowsScene(t *testing.T) {
	tests := []struct {
		name string
		set  func(*Game)
		want music.Cue
	}{
		{"標題", func(g *Game) { g.shell = &shellState{mode: shellTitle} }, music.CueTitle},
		{"主選單", func(g *Game) { g.shell = &shellState{mode: shellMainMenu} }, music.CueTitle},
		{"結局讓給通關曲", func(g *Game) { g.shell = &shellState{mode: shellEnding} }, music.CueNone},
		{"全滅讓給死亡曲", func(g *Game) { g.shell = &shellState{mode: shellWipe} }, music.CueNone},
		{"世界地圖", func(g *Game) { g.shell = &shellState{mode: shellPlaying} }, music.CueWorld},
		{"城鎮", func(g *Game) {
			g.shell = &shellState{mode: shellPlaying}
			g.town = &townState{}
		}, music.CueTown},
		{"迷宮", func(g *Game) {
			g.shell = &shellState{mode: shellPlaying}
			g.level = &mazeLevel{}
		}, music.CueMaze},
		{"戰鬥蓋過城鎮", func(g *Game) {
			g.shell = &shellState{mode: shellPlaying}
			g.town = &townState{}
			g.field = &combat.Field{}
		}, music.CueCombat},
		{"最終戰", func(g *Game) {
			g.shell = &shellState{mode: shellPlaying}
			g.field = &combat.Field{}
			g.bossFight = true
		}, music.CueBoss},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Game{}
			tt.set(g)
			if got := g.currentCue(); got != tt.want {
				t.Errorf("拿到 cue %d,要 %d", got, tt.want)
			}
		})
	}
}

// 沒有音訊裝置時 updateBGM 什麼都不做,而且**不會爆** ——
// 容器與 CI 就是這個狀態(SHARD_NOSOUND)。
func TestUpdateBGMWithoutAudio(t *testing.T) {
	g := &Game{musicMode: music.ModeRemake, shell: &shellState{mode: shellPlaying}}
	g.updateBGM()
	if g.bgm.player != nil {
		t.Error("沒有音訊 context 卻建出了 player")
	}
	if !g.bgm.set || g.bgm.cue != music.CueWorld {
		t.Errorf("cue 沒記下來:set=%v cue=%d", g.bgm.set, g.bgm.cue)
	}
	// 場景沒換就不該重跑(這裡驗的是旗標,不是效能)。
	g.bgm.cue = music.CueMaze
	g.updateBGM()
	if g.bgm.cue != music.CueWorld {
		t.Error("場景換了卻沒有重新對 cue")
	}
}

// 切換模式要循環、要寫進設定檔、下次讀得回來。
func TestCycleMusicModePersists(t *testing.T) {
	dir := t.TempDir()
	g := &Game{saveDir: dir, shell: &shellState{mode: shellPlaying}}

	if g.musicMode != music.ModeOriginal {
		t.Fatalf("預設應該是原版,拿到 %s", g.musicMode)
	}
	msg := g.cycleMusicMode()
	if g.musicMode != music.ModeRemake {
		t.Errorf("切一次應該到重製,拿到 %s", g.musicMode)
	}
	if msg == "" {
		t.Error("切換沒有回訊息")
	}
	if g.bgm.set {
		t.Error("模式換了,set 應該清掉讓下一格重算")
	}

	b, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		t.Fatalf("設定檔沒寫出來:%v", err)
	}
	var c config
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatalf("設定檔不是合法 JSON:%v", err)
	}
	if c.Music != music.ModeRemake.Key() {
		t.Errorf("設定檔寫的是 %q", c.Music)
	}

	// 讀回來
	g2 := &Game{saveDir: dir}
	g2.loadConfig()
	if g2.musicMode != music.ModeRemake {
		t.Errorf("讀回來變成 %s", g2.musicMode)
	}

	// 走完一圈回到原版
	g.cycleMusicMode()
	g.cycleMusicMode()
	if g.musicMode != music.ModeOriginal {
		t.Errorf("轉三次沒有回到原版,停在 %s", g.musicMode)
	}
}

// 設定檔壞掉不該擋住遊戲,也不該把模式換成別的。
func TestLoadConfigTolerateGarbage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte("{壞掉的"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Game{saveDir: dir, musicMode: music.ModeRemake}
	g.loadConfig()
	if g.musicMode != music.ModeRemake {
		t.Errorf("壞掉的設定檔改動了模式:%s", g.musicMode)
	}
}
