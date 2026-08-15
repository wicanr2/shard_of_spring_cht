package combat

// SeededRand 是固定種子的擲骰來源(docs/spec/07 §2)。
//
// 用 xorshift64*,理由是**它的狀態完全由種子決定,而且沒有全域狀態** ——
// `math/rand` 的全域來源會被別處的呼叫打亂,那會讓「同一顆種子跑兩次結果相同」
// 這條驗收在真實程式裡失效,而**單元測試看不出來**(測試裡沒有別處)。
//
// ⚠ **這不是原版的亂數產生器。** `INT 3D:03` 的演算法沒有被解出來,
// 而「同種子重現同一場戰鬥」與「重現 1986 年那一場」是兩件事。
// 規格要的是前者(docs/spec/07 §2)。
type SeededRand struct{ s uint64 }

// NewRand 用種子建一個來源。種子 0 會被換成一個非零值 —— xorshift 的
// 全零狀態是不動點,會讓每一次擲骰都回同一個數。
func NewRand(seed uint64) *SeededRand {
	if seed == 0 {
		seed = 0x9E3779B97F4A7C15
	}
	return &SeededRand{s: seed}
}

func (r *SeededRand) next() uint64 {
	r.s ^= r.s >> 12
	r.s ^= r.s << 25
	r.s ^= r.s >> 27
	return r.s * 2685821657736338717
}

// Roll 回 1..faces。faces ≤ 0 一律回 1(不 panic —— 戰鬥中途 panic
// 會讓整場結果不可重現,而回 1 是可觀察的)。
func (r *SeededRand) Roll(faces int) int {
	if faces <= 0 {
		return 1
	}
	return int(r.next()>>33)%faces + 1
}

// ScriptRand 是測試用的腳本化來源:照給定的序列回值,用完循環。
//
// 有了它,公式的每一項都可以獨立驗 —— 不必去猜某顆種子會擲出什麼。
type ScriptRand struct {
	Values []int
	// Faces 記下每次被要求的面數 —— 傷害公式的「面數是哪一項」
	// 本身就是規則的一部分(武器傷害 / 力量 / 命中能力 − 5,docs/re/153),
	// 而只看回傳值分不出這三條分支。
	Faces []int
	i     int
}

func (r *ScriptRand) Roll(faces int) int {
	r.Faces = append(r.Faces, faces)
	if len(r.Values) == 0 {
		return 1
	}
	v := r.Values[r.i%len(r.Values)]
	r.i++
	return v
}
