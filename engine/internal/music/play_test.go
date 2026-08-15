package music

import (
	"math"
	"testing"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// BASIC 的 `O4 A` 是 440 Hz。
func TestFreqAnchor(t *testing.T) {
	if got := Freq(4, 'A', 0); !near(got, 440, 0.01) {
		t.Errorf("O4 A = %.3f Hz,應為 440", got)
	}
	// 中央 C(O4 C)= 261.626 Hz
	if got := Freq(4, 'C', 0); !near(got, 261.626, 0.01) {
		t.Errorf("O4 C = %.3f Hz,應為 261.626", got)
	}
	// 升半音是 ×2^(1/12)
	if got, want := Freq(3, 'F', 1), Freq(3, 'F', 0)*math.Pow(2, 1.0/12); !near(got, want, 0.001) {
		t.Errorf("F# = %.3f,應為 F × 2^(1/12) = %.3f", got, want)
	}
	// 八度差是兩倍
	if !near(Freq(4, 'A', 0), Freq(3, 'A', 0)*2, 0.001) {
		t.Error("O4 A 應為 O3 A 的兩倍")
	}
}

// `L8` = 八分音符;`T108` 下一個四分音符是 60/108 秒。
func TestTempoAndLength(t *testing.T) {
	s := NewState()
	s.Tempo = 108
	if got, want := s.beat(4), 60.0/108; !near(got, want, 1e-9) {
		t.Errorf("四分音符 %.4f 秒,應為 %.4f", got, want)
	}
	if got, want := s.beat(8), 30.0/108; !near(got, want, 1e-9) {
		t.Errorf("八分音符 %.4f 秒,應為 %.4f", got, want)
	}
}

// 原版的十段是**一首曲子的續播** —— 後兩段沒有 T/O,靠前面留下的狀態。
//
// ⚠ 如果每段都重建 State,後兩段會用預設的 T120 播出來,
// 而那聽起來仍然像一首曲子,只是不對。
func TestEndingSharesStateAcrossMacros(t *testing.T) {
	st := NewState()
	var notes []Note
	for _, m := range Ending {
		notes = st.Parse(m, notes)
	}
	if st.Tempo != 108 {
		t.Errorf("最後的速度是 %d,應該還是 108(最後兩段沒有 T)", st.Tempo)
	}
	if len(notes) == 0 {
		t.Fatal("一個音都沒解出來")
	}
	// 最後一段是 `O3 L3 G` —— 三分音符的 G3
	last := notes[len(notes)-1]
	if !near(last.Freq, Freq(3, 'G', 0), 0.001) {
		t.Errorf("最後一個音 %.2f Hz,應為 O3 G = %.2f", last.Freq, Freq(3, 'G', 0))
	}
	if want := 240.0 / 108 / 3; !near(last.Dur, want, 1e-9) {
		t.Errorf("最後一個音 %.4f 秒,應為 %.4f", last.Dur, want)
	}
}

// `MB` 是背景播放、`ML` 是圓滑奏 —— 兩者都**不是音符**。
func TestModifiersAreNotNotes(t *testing.T) {
	st := NewState()
	n := st.Parse("MB T50 O3 L8 ML DEFE", nil)
	if len(n) != 4 {
		t.Fatalf("`DEFE` 應為 4 個音,得 %d", len(n))
	}
	if !st.Legato {
		t.Error("ML 應該打開圓滑奏")
	}
	for i, x := range n {
		if x.Gate != 1 {
			t.Errorf("第 %d 個音的 Gate = %.3f,圓滑奏應為 1", i+1, x.Gate)
		}
	}
}

// 升記號要吃掉,不能當成下一個音。
func TestSharpIsNotANote(t *testing.T) {
	st := NewState()
	n := st.Parse("O3 L8 E F#GD", nil)
	if len(n) != 4 {
		t.Fatalf("`E F#G D` 應為 4 個音,得 %d", len(n))
	}
	if !near(n[1].Freq, Freq(3, 'F', 1), 0.001) {
		t.Errorf("第 2 個音應為 F#,得 %.2f Hz", n[1].Freq)
	}
}

// 音符後面的數字覆蓋預設長度。
func TestPerNoteLengthOverrides(t *testing.T) {
	st := NewState()
	st.Tempo, st.Length = 120, 4
	n := st.Parse("C C8", nil)
	if !near(n[0].Dur, 0.5, 1e-9) {
		t.Errorf("預設四分音符應為 0.5 秒,得 %.4f", n[0].Dur)
	}
	if !near(n[1].Dur, 0.25, 1e-9) {
		t.Errorf("`C8` 應為 0.25 秒,得 %.4f", n[1].Dur)
	}
}

// Render 出來的長度要對得上總時值,而且不碰任何音訊裝置。
func TestRenderLength(t *testing.T) {
	notes := ParseAll(Ending)
	const sr = 22050
	pcm := Render(notes, sr)
	var total float64
	for _, n := range notes {
		total += n.Dur
	}
	want := int(total * sr)
	if diff := len(pcm) - want; diff < -len(notes) || diff > len(notes) {
		t.Errorf("PCM 長度 %d,總時值換算是 %d(差 %d)", len(pcm), want, diff)
	}
	if len(pcm) == 0 {
		t.Fatal("沒有取樣")
	}
}

func TestUserlibScoreParses(t *testing.T) {
	n := ParseAll(Userlib)
	if len(n) != 12 {
		t.Errorf("USERLIB 五段合計應為 12 個音,得 %d", len(n))
	}
}
