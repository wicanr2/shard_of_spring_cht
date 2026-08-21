package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVisualModeRoundTrip(t *testing.T) {
	for _, m := range []visualMode{visualOriginal, visualStorybook} {
		if got := parseVisualMode(m.key()); got != m {
			t.Errorf("%s round-trip 成 %s", m, got)
		}
	}
	if got := parseVisualMode("壞掉的值"); got != visualOriginal {
		t.Errorf("未知模式應保守退回原版，拿到 %s", got)
	}
}

func TestCycleVisualModeIndependentFromMusic(t *testing.T) {
	dir := t.TempDir()
	g := &Game{visualMode: visualOriginal, saveDir: dir}
	before := g.musicMode
	if msg := g.cycleVisualMode(); g.visualMode != visualStorybook || msg == "" {
		t.Fatalf("第一次切換拿到 %s，訊息 %q", g.visualMode, msg)
	}
	if g.musicMode != before {
		t.Fatal("切視覺主題不應改動配樂")
	}
	b, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		t.Fatal(err)
	}
	var c config
	if json.Unmarshal(b, &c) != nil || c.Visual != "storybook" {
		t.Fatalf("視覺模式沒有寫入設定:%s", b)
	}
	g2 := &Game{saveDir: dir}
	g2.loadConfig()
	if g2.visualMode != visualStorybook {
		t.Fatalf("視覺模式讀回成 %s", g2.visualMode)
	}
}
