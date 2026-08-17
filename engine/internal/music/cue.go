package music

// 場景配樂。**這一層不是原版的東西。**
//
// 原版全遊戲只有兩首曲子(docs/re/216:通關曲與死亡曲),
// 世界地圖、地城、城鎮是**安靜的** —— 那不是缺口,是 1986 年 PC 喇叭遊戲的常態。
// 所以場景配樂是 remake 的增補,設計上有三條約束:
//
//  1. **預設不開。** 玩家先聽到的是原版的樣子。
//  2. **切得掉,而且切換點要講清楚現在是哪一種** ——
//     否則玩家聽到的是「這裡本來就有音樂」,不會知道是後加的
//     (docs/spec/13 §5 的 ⛔ 講的就是這件事)。
//  3. **用同一套 `PLAY` 語法與同一個方波合成器。** 不引進取樣音檔:
//     一來聽起來與原版那兩首同源,二來不必處理任何第三方音樂的授權。
//
// ⚠ 與 [score.go] 的分界:那邊是**原版檔案裡的字串,照抄不改**;
// 這邊是本專案自己寫的譜,要調整聽感就直接改。

// Mode 是配樂模式。玩家可以在遊戲中切換(F5)。
type Mode int

const (
	ModeOriginal Mode = iota // 只有原版那兩首;場景安靜
	ModeRemake               // 原版兩首 + 場景配樂
	ModeOff                  // 全部靜音
)

// ModeCount 是循環切換用的模式總數。
const ModeCount = 3

// String 是顯示給玩家看的名字。
func (m Mode) String() string {
	switch m {
	case ModeRemake:
		return "重製"
	case ModeOff:
		return "關閉"
	default:
		return "原版"
	}
}

// ParseMode 把設定檔裡的字串轉回模式。認不得就回原版 ——
// ⚠ 設定檔壞掉時要退回**最保守的那一個**,不是最華麗的那一個。
func ParseMode(s string) Mode {
	switch s {
	case "remake":
		return ModeRemake
	case "off":
		return ModeOff
	default:
		return ModeOriginal
	}
}

// Key 是寫進設定檔的字串。
func (m Mode) Key() string {
	switch m {
	case ModeRemake:
		return "remake"
	case ModeOff:
		return "off"
	default:
		return "original"
	}
}

// Cue 是一個配樂點。**對應的是玩家所在的場景,不是事件** ——
// 事件配樂(通關、全滅)走原版那兩首,由呼叫端直接 `play`。
type Cue int

const (
	CueNone   Cue = iota // 靜音(標題以外的外殼畫面、結局、全滅)
	CueTitle             // 標題與主選單
	CueTown              // 城鎮
	CueWorld             // 世界地圖
	CueMaze              // 地城
	CueCombat            // 戰鬥
	CueBoss              // 最終戰
)

// Remake 是本專案寫的場景配樂,語法與原版的譜相同(docs/spec/13 §2)。
//
// 每一首都設計成**可以直接接回開頭循環**:結束落在主音上、
// 最後一小節不留懸空的和聲。⚠ 循環播放的曲子與一次性的曲子寫法不同 ——
// 一次性的曲子可以在屬音上收尾,循環的不行,那會每一圈都聽到一次沒解決的懸置。
//
// 調性照場景分:城鎮與世界地圖是大調,地城與戰鬥是小調,
// 最終戰在小調上再加半音進行。
var Remake = map[Cue][]string{
	// 標題:慢、莊嚴。D 小調。
	CueTitle: {
		"MB T92 O3 L4 MN",
		"D A F A", "L2 D", "L4 E F",
		"A O4 C O3 A F", "L2 G", "L4 A O4 C",
		"D C O3 B- A", "L2 G", "L4 F E",
		"D F E D", "L1 D",
	},
	// 城鎮:明亮、輕快。F 大調。
	CueTown: {
		"MB T112 O4 L8 MN",
		"F G A O5 C", "O4 A F L4 G",
		"L8 A O5 C O4 A F", "L4 G L2 F",
		"L8 A O5 C D C", "O4 B- G L4 A",
		"L8 G F E F", "L4 G L2 F",
	},
	// 世界地圖:行進感,不搶戲。D 多利安調(小三度但大六度,聽起來不陰暗)。
	CueWorld: {
		"MB T104 O3 L4 MN",
		"D F G A", "L2 A", "L4 G F",
		"L2 D", "L4 D F",
		"G A O4 C O3 A", "L2 G",
		"L4 F G A F", "L1 D",
	},
	// 地城:慢、稀疏、留白。D 小調,用休止撐出空間。
	CueMaze: {
		"MB T76 O3 L2 ML",
		"D", "P4 L4 F", "L2 E-",
		"P4 L4 D", "L2 C",
		"P4 L4 E-", "L2 D", "P2",
	},
	// 戰鬥:急促、斷奏。D 小調。
	CueCombat: {
		"MB T144 O3 L8 MS",
		"D D A D", "B- A G F",
		"D D A D", "L4 E L8 F E",
		"D D A D", "O4 C O3 B- A G",
		"F G A B-", "L4 O4 D P4",
	},
	// 最終戰:更快,加半音進行(D–E♭)製造不安定。
	CueBoss: {
		"MB T160 O3 L8 MS",
		"D E- D E-", "D E- D E-",
		"A- G F E", "L4 D L8 D D",
		"O4 D O3 B A- G", "A- G F E",
		"L4 D O4 D", "L2 D",
	},
}

// Loop 回傳某個模式下某個場景要**循環播放**的譜;nil = 靜音。
//
// ⚠ 只有 ModeRemake 有場景配樂。原版模式回 nil 不是「還沒接」,
// 是**原版就是安靜的**。
func Loop(m Mode, c Cue) []string {
	if m != ModeRemake {
		return nil
	}
	return Remake[c]
}
