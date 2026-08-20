package music

import "testing"

// 驗收 1(docs/spec/13 §8.3):七個代碼的頻率與拍數逐項對得上 docs/re/228 §3。
//
// ⚠ 這張表是**手抄自反組譯**的第二份 —— 與 effects 對照,
// 抄錯一個數字兩邊就不一樣。同一份資料寫兩次是這裡刻意要的。
func TestEffectWaveformsMatchTheDisassembly(t *testing.T) {
	for _, c := range []struct {
		code       string
		notes      int
		firstFreq  float64
		firstTicks float64
	}{
		{FxHit, 1, 120, 8},
		{FxHack, 1, 90, 6},
		{FxBreak, 1, 80, 10},
		{FxDie, 1, 45, 12},
		{FxSpell, 21, 1200, 0.25},    // 7 圈 × 3 音
		{FxFlee, 9 + 29, 1200, 0.25}, // 3 圈 × 3 音 + 1100…400 每次 −25
	} {
		got := Effect(c.code)
		if len(got) != c.notes {
			t.Errorf("%s 有 %d 個音,應為 %d", c.code, len(got), c.notes)
			continue
		}
		if got[0].Freq != c.firstFreq {
			t.Errorf("%s 第一個音 %g Hz,應為 %g", c.code, got[0].Freq, c.firstFreq)
		}
		if want := c.firstTicks * Tick; got[0].Dur != want {
			t.Errorf("%s 第一個音 %g 秒,應為 %g(%g 拍)", c.code, got[0].Dur, want, c.firstTicks)
		}
	}
}

// FL 的下掃要真的掃到底:最後一個音是 400 Hz。
//
// ⚠ 邊界是 `cmp ax, 190h / jge` —— **含 400**。寫成 `>` 會少一個音,
// 而少一個音聽不出來(docs/re/228 §3)。
func TestFleeSweepEndsAtFourHundred(t *testing.T) {
	n := Effect(FxFlee)
	last := n[len(n)-1]
	if last.Freq != sweepTo {
		t.Errorf("下掃的最後一個音 %g Hz,應為 %d", last.Freq, sweepTo)
	}
	// 倒數第二個應該高 25 Hz
	if prev := n[len(n)-2]; prev.Freq-last.Freq != sweepStep {
		t.Errorf("下掃的間距 %g Hz,應為 %d", prev.Freq-last.Freq, sweepStep)
	}
}

// 音效整段都在發聲:`SOUND` 沒有圓滑奏／斷奏的概念。
//
// ⚠ 沿用 `PLAY` 的 7/8 會讓每個音尾巴斷一截 —— 對 `PS` 那種 13 毫秒的音
// 是**聽得出來的**,而且聽起來像雜訊不像「短了一點」。
func TestEffectNotesAreFullyGated(t *testing.T) {
	for _, code := range Codes() {
		for i, n := range Effect(code) {
			if n.Gate != 1 {
				t.Errorf("%s 第 %d 個音 Gate=%g,應為 1", code, i, n.Gate)
			}
		}
	}
}

// 認不得的代碼回 nil,不是 panic —— 原版比對不到也只是發個提示音。
func TestUnknownEffectCodeIsSilent(t *testing.T) {
	if got := Effect("ZZ"); got != nil {
		t.Errorf("認不得的代碼應回 nil,得 %v", got)
	}
}

// Codes() 要涵蓋 effects 表的全部,少一個測試就掃不到它。
func TestCodesCoversEveryEffect(t *testing.T) {
	if len(Codes()) != len(effects) {
		t.Errorf("Codes() 有 %d 個,effects 表有 %d 個", len(Codes()), len(effects))
	}
	for _, c := range Codes() {
		if Effect(c) == nil {
			t.Errorf("Codes() 列了 %s,但 effects 表裡沒有", c)
		}
	}
}
