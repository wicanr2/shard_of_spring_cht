package rules

// ScriptedFight 記著哪些迷宮事件目標照清單開戰,不用 re/169 的隨機挑怪規則。
// docs/spec/17-scripted-fights.md §3、docs/re/180 §3。
//
// 原版在 CMBT 讀到 `ds:372C` 起最多 8 槽的清單時(哨兵 99)就照清單擺,
// 不再走隨機挑怪迴圈(docs/re/180 §1)。這裡只記「解出來的組成」,
// 不模擬那個 8 槽陣列本身 —— 陣列只是「清單怎麼搬進 CMBT」的實作細節,
// 依 CLAUDE.md §1.2 不用挖;組成才是規則。
//
// ⚠ 索引是 **0-based**(MONSTERS.DAT,與背包道具編號同一慣例)——
// ⛔ 不要在這裡 ±1「修正」。
//
// ⚠ **其餘 13 處寫入點還沒盤**(docs/re/180 §6)。之後解出第三個事件時,
// 應該只需要在這張表裡加一列 —— ⛔ 不要為個別事件寫死 if 分支
// (呼叫端 combat_scene.go 的 startScriptedCombat 就是照這個假設寫的)。
var ScriptedFight = map[int][]int{
	204: {10},         // 山丘巨人挾持祭司(maze.TargetPriest)—— 1 隻 Hill Giant(第 10 列)
	533: {53, 53, 71}, // 最終首領(maze.TargetFinalBoss)—— 2 隻 Great Dragon(第 53 列)+ 1 隻 Siriadne !(第 71 列)
}

// PriestBlessing 是打贏 204(山丘巨人挾持祭司)之後顯示的文字,
// 譯自 MAZEMOVE 字串常數區 0x14A48 的原文:
//
//	'The priest thanks you for freeing him from his giant captor and blesses the party.'
//
// (docs/re/161 §4)。
//
// ⚠ **祝福的實際效果沒有讀到**(docs/re/161 §4 只讀到這句字串,
// `0x131D0` 起那一長串是結局文字的排版,不是屬性設定)——
// 這裡只顯示文字,⛔ 不附加任何屬性加成。
const PriestBlessing = "祭司感謝你們把他從山丘巨人手中救出,並為隊伍獻上祝福。"

// PriestEncounterMark 是腳本戰鬥 204 開場寫進 combat.Field.Log 第一行的
// 固定字串。
//
// ⚠ **這是本引擎的實作手段,不是原版行為**:combat_scene.go 的 endTurn()
// 要在戰鬥結束的那一刻判斷「這場贏的是不是祭司事件」才能顯示 PriestBlessing
// (204 與隨機遭遇都可能只出 1 隻 Hill Giant,composition 本身分不出來),
// 但 Game(main.go)與 combat.Field(internal/combat/field.go)都不能為了
// 這一件事新增欄位 —— CLAUDE.md 這一輪的邊界只開放 maze_scene.go、
// combat_scene.go 與 internal/rules 的新檔。於是借 Log 本身當狀態:
// Log 是這場戰鬥自己的紀錄,天生就帶著「這一場是什麼」的答案,
// 不需要另一個變數重複記一次。
const PriestEncounterMark = "遭遇：山丘巨人挾持祭司!"
