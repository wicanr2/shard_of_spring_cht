package music

import (
	"embed"
	"path"
)

// 曲子的 OGG 資產。由 `tools/music.sh` 從同一份譜產生,**進版控**。
//
// 為什麼定稿之後要落成檔案而不是每次開機現算:
//   - 場景配樂是**循環播放**的,現算要把整首的 PCM 攤在記憶體裡
//     (20 秒的曲子 = 22050 × 4 × 20 ≈ 1.8 MB);OGG 壓完是十分之一以下,
//     而且解碼是串流的
//   - 定稿的東西該是資產,不是每次啟動重算的中間產物
//
// ⚠ **譜仍然是真相** —— OGG 是從 `Remake` 與 `Ending`/`Userlib` 產生的。
// 改了譜要重跑 `tools/music.sh` 並把 `assets/*.ogg` 一起 commit,
// 否則遊戲裡不會變,而且**不會有任何錯誤訊息**。
//
// ⚠ 原版那兩首的譜是原版檔案裡的字串(docs/re/148)。
// 轉成 OGG 只是換容器,**它仍然是原版內容** —— 與 score.go 同一個地位。
//
//go:embed assets/*.ogg
var assetFS embed.FS

// 原版那兩首的檔名。它們不是場景配樂(不循環),由呼叫端在觸發點直接放。
const (
	FileEnding = "original-ending" // 通關曲(docs/re/182)
	FileDeath  = "original-death"  // 死亡曲(USERLIB 的五段)
)

// cueFile 是場景配樂的檔名。**與 Remake 的鍵一一對應** ——
// 少一個的症狀是那個場景安靜,而安靜看起來跟「原版模式」一模一樣。
var cueFile = map[Cue]string{
	CueTitle:  "remake-title",
	CueTown:   "remake-town",
	CueWorld:  "remake-world",
	CueMaze:   "remake-maze",
	CueCombat: "remake-combat",
	CueBoss:   "remake-boss",
}

// CueFile 回傳某個模式下某個場景要循環的曲名;空字串 = 靜音。
//
// ⚠ 與 Loop 的判斷條件一致:只有重製模式有場景配樂。
func CueFile(m Mode, c Cue) string {
	if m != ModeRemake {
		return ""
	}
	return cueFile[c]
}

// OGG 回傳某首曲子的位元組;沒有這個檔就回 nil。
//
// ⚠ **回 nil 不是錯誤**:呼叫端會退回現算(sound.go / bgm.go)。
// 資產掉了還聽得到聲音,比「安靜而且沒人知道為什麼」好。
func OGG(name string) []byte {
	if name == "" {
		return nil
	}
	b, err := assetFS.ReadFile(path.Join("assets", name+".ogg"))
	if err != nil {
		return nil
	}
	return b
}

// Names 列出所有內嵌的曲名(不含副檔名)。給測試用 ——
// 「每個 cue 都要有檔」這件事要驗得出來。
func Names() []string {
	entries, err := assetFS.ReadDir("assets")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		if path.Ext(n) != ".ogg" {
			continue
		}
		out = append(out, n[:len(n)-len(".ogg")])
	}
	return out
}
