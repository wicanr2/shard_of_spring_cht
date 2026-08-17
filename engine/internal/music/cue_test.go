package music

import "testing"

// 場景配樂只在重製模式下有東西。原版模式回 nil **不是「還沒接」** ——
// docs/re/216:原版的世界地圖、地城、城鎮本來就是安靜的。
func TestLoopOnlyInRemakeMode(t *testing.T) {
	for _, c := range []Cue{CueTitle, CueTown, CueWorld, CueMaze, CueCombat, CueBoss} {
		if got := Loop(ModeOriginal, c); got != nil {
			t.Errorf("原版模式的 cue %d 應該靜音,拿到 %d 段", c, len(got))
		}
		if got := Loop(ModeOff, c); got != nil {
			t.Errorf("關閉模式的 cue %d 應該靜音,拿到 %d 段", c, len(got))
		}
		if got := Loop(ModeRemake, c); len(got) == 0 {
			t.Errorf("重製模式缺 cue %d 的譜", c)
		}
	}
	if got := Loop(ModeRemake, CueNone); got != nil {
		t.Errorf("CueNone 應該靜音,拿到 %d 段", len(got))
	}
}

// 每一首都要真的解得出音,而且長度落在「像一段循環」的範圍。
//
// ⚠ 這裡驗的是**結構**不是好不好聽:太短的循環會讓人抓狂,
// 太長的就不是循環而是一首曲子。上下界是設計決定,不是量出來的。
func TestRemakeScoresParse(t *testing.T) {
	const minSec, maxSec = 5.0, 60.0
	for c, score := range Remake {
		notes := ParseAll(score)
		if len(notes) < 8 {
			t.Errorf("cue %d 只有 %d 個音,太短", c, len(notes))
		}
		var total float64
		var sounded int
		for _, n := range notes {
			total += n.Dur
			if n.Freq > 0 {
				sounded++
			}
		}
		if total < minSec || total > maxSec {
			t.Errorf("cue %d 全長 %.1f 秒,不在 %.0f–%.0f 秒之間", c, total, minSec, maxSec)
		}
		// 全部是休止的話上面的長度檢查照樣會過 —— 那是「靜音但看起來正常」。
		if sounded == 0 {
			t.Errorf("cue %d 一個發聲的音都沒有", c)
		}
	}
}

// 模式與設定檔字串要對得回來,而且**壞掉要退回最保守的那一個**。
func TestModeRoundTrip(t *testing.T) {
	for _, m := range []Mode{ModeOriginal, ModeRemake, ModeOff} {
		if got := ParseMode(m.Key()); got != m {
			t.Errorf("%s 轉回來變成 %s", m, got)
		}
	}
	if got := ParseMode("這不是模式"); got != ModeOriginal {
		t.Errorf("認不得的設定值應該退回原版,拿到 %s", got)
	}
	if got := ParseMode(""); got != ModeOriginal {
		t.Errorf("空設定值應該退回原版,拿到 %s", got)
	}
}

// 循環切換要走完三個模式再回到原點。
func TestModeCountCoversAll(t *testing.T) {
	seen := map[Mode]bool{}
	m := ModeOriginal
	for i := 0; i < ModeCount; i++ {
		seen[m] = true
		m = Mode((int(m) + 1) % ModeCount)
	}
	if m != ModeOriginal {
		t.Errorf("轉了 %d 次沒有回到原版,停在 %s", ModeCount, m)
	}
	for _, want := range []Mode{ModeOriginal, ModeRemake, ModeOff} {
		if !seen[want] {
			t.Errorf("循環切換走不到 %s", want)
		}
	}
}
