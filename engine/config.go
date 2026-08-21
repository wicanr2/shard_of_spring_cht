package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"shardofspring/internal/music"
)

// 玩家偏好。**與存檔分開**(docs/spec/18 §2 的存檔是進度,這裡是偏好):
// 配樂模式不該綁在某一份存檔上,換存檔不該把音樂設定換掉;
// 而且存檔有版本檢查,為了一個偏好去動它的 schema 不划算。

// configFile 放在存檔目錄旁邊,與 saves/*.json 同一層。
const configFile = "config.json"

// config 是設定檔的內容。欄位用字串不用數字 ——
// 玩家看得懂也改得動(與存檔選 JSON 同一個理由)。
type config struct {
	Music  string `json:"music"`
	Visual string `json:"visual"`
}

func (g *Game) configPath() string {
	return filepath.Join(g.effectiveSaveDir(), configFile)
}

// loadConfig 讀設定。**讀不到、壞掉都不是錯誤** —— 用預設值繼續,
// 不要因為一個偏好檔擋住遊戲。
func (g *Game) loadConfig() {
	b, err := os.ReadFile(g.configPath())
	if err != nil {
		return
	}
	var c config
	if json.Unmarshal(b, &c) != nil {
		return
	}
	g.musicMode = music.ParseMode(c.Music)
	g.visualMode = parseVisualMode(c.Visual)
	g.bgm.set = false
}

// saveConfig 寫回設定。寫失敗只記一行警告 —— 偏好存不下來不影響遊玩。
func (g *Game) saveConfig() {
	dir := g.effectiveSaveDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		g.warnings = append(g.warnings, "設定存不下來:"+err.Error())
		return
	}
	b, err := json.MarshalIndent(config{Music: g.musicMode.Key(), Visual: g.visualMode.key()}, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(g.configPath(), append(b, '\n'), 0o644); err != nil {
		g.warnings = append(g.warnings, "設定存不下來:"+err.Error())
	}
}
