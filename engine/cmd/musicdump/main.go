// musicdump 把每一首曲子倒成 WAV,拿來試聽。
//
// **這支不進發行包**(tools/release.sh 只建引擎與轉換器)——
// 場景配樂是自己寫的譜(internal/music/cue.go),而譜好不好聽沒有測試驗得出來,
// 只能聽。把渲染結果落成檔案是唯一能讓人判斷的形式。
//
//	go run ./cmd/musicdump -out /tmp/music
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"shardofspring/internal/music"
)

func main() {
	out := flag.String("out", "music", "輸出資料夾")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "失敗:", err)
		os.Exit(1)
	}
}

// 曲名對譜。原版那兩首也倒出來 —— 要比較「同源不同源」就得放在一起聽。
func scores() map[string][]string {
	m := map[string][]string{
		"original-ending": music.Ending,
		"original-death":  music.Userlib,
	}
	for cue, name := range map[music.Cue]string{
		music.CueTitle:  "remake-title",
		music.CueTown:   "remake-town",
		music.CueWorld:  "remake-world",
		music.CueMaze:   "remake-maze",
		music.CueCombat: "remake-combat",
		music.CueBoss:   "remake-boss",
	} {
		m[name] = music.Remake[cue]
	}
	return m
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, score := range scores() {
		notes := music.ParseAll(score)
		pcm := music.RenderPCM16(notes, music.SampleRate)
		var total float64
		for _, n := range notes {
			total += n.Dur
		}
		p := filepath.Join(dir, name+".wav")
		if err := writeWAV(p, pcm); err != nil {
			return err
		}
		fmt.Printf("%-18s %3d 音  %5.1f 秒  %s\n", name, len(notes), total, p)
	}
	return nil
}

// writeWAV 包一個 44 byte 的 RIFF 標頭。
// PCM 已經是 16-bit LE 立體聲(music.RenderPCM16),這裡只補標頭。
func writeWAV(path string, pcm []byte) error {
	const (
		channels = 2
		bits     = 16
	)
	byteRate := music.SampleRate * channels * bits / 8
	blockAlign := channels * bits / 8

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := func(v ...any) error {
		for _, x := range v {
			if err := binary.Write(f, binary.LittleEndian, x); err != nil {
				return err
			}
		}
		return nil
	}
	if _, err := f.WriteString("RIFF"); err != nil {
		return err
	}
	if err := w(uint32(36 + len(pcm))); err != nil {
		return err
	}
	if _, err := f.WriteString("WAVEfmt "); err != nil {
		return err
	}
	if err := w(uint32(16), uint16(1), uint16(channels),
		uint32(music.SampleRate), uint32(byteRate),
		uint16(blockAlign), uint16(bits)); err != nil {
		return err
	}
	if _, err := f.WriteString("data"); err != nil {
		return err
	}
	if err := w(uint32(len(pcm))); err != nil {
		return err
	}
	_, err = f.Write(pcm)
	return err
}
