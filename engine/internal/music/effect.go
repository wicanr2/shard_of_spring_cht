package music

// 音效。docs/re/228、docs/spec/13 §8。
//
// 原版的音效走 BASIC 的 `SOUND freq, ticks`,與樂曲的 `PLAY` 是兩個不同的敘述 ——
// 這也是為什麼掃樂譜(docs/re/216)掃不到它們。發聲的硬體是同一個 PC 喇叭,
// 所以這裡產出的是同樣的 `[]Note`,交給同一個 Render。
//
// ⚠ 表裡的頻率與拍數是**原版讀出來的值,照抄不改**,地位與 score.go 相同。

// Tick 是 BASIC 的時鐘節拍長度(秒)。`SOUND` 的第二個參數用的就是它。
const Tick = 1.0 / 18.2

// tone 是一個 `SOUND` 敘述:頻率(Hz)與長度(節拍)。
type tone struct {
	freq  float64
	ticks float64
}

// 音效代碼。原版傳的就是這兩個字母的字串。
const (
	FxHit   = "HT" // 一般命中
	FxHack  = "HK" // 狂暴劈砍
	FxBreak = "BR" // CMBT 裡沒有呼叫端
	FxDie   = "DD" // 單位死亡
	FxSpell = "PS" // 施法
	FxFlee  = "FL" // 只有一個呼叫端,語意沒讀
	FxBeep  = "BP" // BASIC 的 BEEP;九個呼叫端沒有逐條讀
)

// 迴圈次數與掃頻範圍。**從 `cmp`／`jle` 讀出來的**,不是數 `SOUND` 的個數
// (docs/re/228 §3:迴圈本體是三個音,跑幾圈寫在別的地方)。
const (
	spellLoops = 7    // PS:`cmp ax, 7 / jle`
	fleeLoops  = 3    // FL:`cmp ax, 3 / jle`
	sweepFrom  = 1100 // FL 之後的下掃起點(`mov ax, 44Ch`)
	sweepStep  = 25   // `add ax, 0FFE7h`
	sweepTo    = 400  // `cmp ax, 190h / jge`
)

// beepFreq/beepTicks 是 BASIC `BEEP` 的規格(800 Hz、四分之一秒)。
//
// ⚠ 這兩個值**不是從 CMBT 讀出來的** —— 原版走 `INT 3E:2B`(BEEP),
// 參數在直譯器裡不在遊戲裡。BEEP 的語意是公開規格,所以照規格填;
// 而 `BP` **本來就不接觸發點**,填錯也不會有人聽到。
const (
	beepFreq  = 800
	beepTicks = 0.25 / Tick
)

// 施法那一串三音。PS 與 FL 共用(docs/re/228 §3)。
var spellTriple = []tone{{1200, 0.25}, {1175, 0.25}, {1150, 0.25}}

// effects 是七個代碼的波形。**七個全部在這裡**,即使只有四個接了觸發點 ——
// 它們是讀出來的事實,少寫一個下次就得重讀一次。
var effects = map[string][]tone{
	FxHit:   {{120, 8}},
	FxHack:  {{90, 6}},
	FxBreak: {{80, 10}},
	FxDie:   {{45, 12}},
	FxBeep:  {{beepFreq, beepTicks}},
	FxSpell: repeat(spellTriple, spellLoops),
	FxFlee:  append(repeat(spellTriple, fleeLoops), sweep()...),
}

func repeat(t []tone, n int) []tone {
	out := make([]tone, 0, len(t)*n)
	for i := 0; i < n; i++ {
		out = append(out, t...)
	}
	return out
}

// sweep 是 FL 尾巴的下掃:1100 起、每次減 25、到 400 為止(含)。
func sweep() []tone {
	var out []tone
	for f := sweepFrom; f >= sweepTo; f -= sweepStep {
		out = append(out, tone{float64(f), 0.1})
	}
	return out
}

// Effect 回傳一個音效的音符;認不得的代碼回 nil。
//
// ⚠ 回 nil **不是錯誤**:原版比對不到代碼時也只是發一個提示音就結束。
// 呼叫端不必分辨,沒有音就不播。
func Effect(code string) []Note {
	ts, ok := effects[code]
	if !ok {
		return nil
	}
	out := make([]Note, 0, len(ts))
	for _, t := range ts {
		// Gate 1:`SOUND` 沒有圓滑奏／斷奏的概念,整段都在發聲。
		out = append(out, Note{Freq: t.freq, Dur: t.ticks * Tick, Gate: 1})
	}
	return out
}

// Codes 列出所有代碼。給測試用 —— 「表有沒有掉一個」要驗得出來。
func Codes() []string {
	return []string{FxHit, FxHack, FxBreak, FxDie, FxSpell, FxFlee, FxBeep}
}
