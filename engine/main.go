// Shard of Spring remake。
//
// M2:世界地圖場景(docs/spec/05-world-scene.md)
// M3:隊伍、角色與存檔(docs/spec/06-party-and-save.md)。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"shardofspring/internal/combat"
	"shardofspring/internal/layout"
	"shardofspring/internal/maze"
	"shardofspring/internal/original"
	"shardofspring/internal/render"
	"shardofspring/internal/save"
	"shardofspring/internal/world"
)

var (
	cgaBlack = color.RGBA{0x00, 0x00, 0x00, 0xff}
	cgaCyan  = color.RGBA{0x55, 0xff, 0xff, 0xff}
	cgaWhite = color.RGBA{0xff, 0xff, 0xff, 0xff}
	// 未解地形的佔位色。docs/spec/05 §4:**未解的東西在畫面上要刺眼** ——
	// 用黑色或海洋填會讓那 66 格偽裝成正常地形。
	missing = color.RGBA{0xff, 0x00, 0xff, 0xff}
)

type Game struct {
	savePath string // <assets>/save/GROUPS.DAT
	slot     int

	world *world.Map
	party world.State
	tiles map[int]*ebiten.Image // 地形值 → 圖;沒有來源的值不在裡面
	// noSrc 記下畫面上出現過的未解地形值,顯示在提示列。
	// 讓未解項目在**執行時**也看得見,不是只在文件裡。
	noSrc map[int]bool

	// M3:隊伍與存檔(docs/spec/06)
	members []original.Character // 依 GROUPS.DAT 的成員槽順序
	group   original.Group
	panel   *render.Painter
	// 載入時發現的不一致,畫在提示列。⚠ 不自行修正(docs/spec/06 §1)。
	warnings []string
	// 最近一次動作的結果,畫在提示列(存檔、離開城鎮之類)。
	// ⚠ 名字留著是因為它一開始只給存檔用;現在是通用的一行狀態訊息,
	// 由**世界地圖那一層**的動作寫入 —— 城鎮/戰鬥各自有自己的訊息欄。
	saveMsg string

	// M4:戰鬥(docs/spec/07)
	monsters []original.Monster
	items    map[int]combat.Item
	// rand 是遊戲全域的擲骰來源。⚠ 型別是**介面**不是 `*combat.SeededRand` ——
	// 測試要能換成固定值的擲骰(發動判定會擋掉大多數低效力的法術,
	// 而「測發動之後的行為」就得先把發動判定拿掉,docs/re/201 §3)。
	rand     combat.FloatRand
	field    *combat.Field // nil = 不在戰鬥中
	// M10:戰場(docs/spec/12)
	points combat.Points
	actor  int // 目前輪到的隊員索引;−1 = 沒有人能動
	// settled = 這一場已經結算過(發過經驗與金幣)。結算有三個入口:
	// 玩家的最後一擊、怪物回合結束、stepCombat —— 全部走 g.settle()
	// (combat_scene.go),靠這個旗標保證同一場只發一次。
	// 建新戰場時歸零,不是每回合。
	settled bool

	// M5:迷宮(docs/spec/08)
	assets      string
	mazeData    []original.MazeEntry
	mazeTiles   map[int]*ebiten.Image
	level       *mazeLevel // nil = 不在迷宮中
	mazeState   maze.State
	overlay     string // 非空 = 敘述覆蓋層開著
	overlayFont *render.Painter
	// 迷宮機關的互動狀態(docs/re/161 §3 的五個目標編號)。
	prompt *mazePrompt
	// tombs 記著踩過哪幾座墓(事件 701–704),餵 Eldron 謎題的進度旗標。
	tombs map[int]bool
	// clanRewarded:謎題解過了。⚠ **原版把這個旗標存在哪未解**
	// (docs/re/162 §5)—— 所以它只活在記憶體裡,存檔不會記得。
	clanRewarded bool

	// M8:城鎮與名冊(docs/spec/11)
	chars  []original.Character
	roster *rosterState
	create *createState // 非 nil = 建立角色的畫面蓋在名冊上

	// 技能點分配畫面(docs/spec/20-skill-allocation.md,skill_alloc_scene.go)。
	// 非 nil = 畫面開著,蓋在目前的主畫面之上——創角完成之後蓋在名冊上,
	// 升級成功之後蓋在城鎮的訓練所上,兩個入口共用同一份狀態與程式碼。
	skillAlloc *skillAllocState

	shops     []original.Shop
	townSites []original.TownSite // TOWNDATA.BIN 的座標表(docs/re/53 §2)
	itemList  []original.Item
	town      *townState
	rumors    map[int]string // 酒館傳聞:位移 36 → 文字(docs/re/138 §4)

	// M11:聲音(docs/spec/13)。nil = 音訊關閉,遊戲照常跑。
	sound *sound

	// M6:法術(docs/spec/09)
	spells   []original.Spell
	castUnit int
	castList []original.Spell
	// cursor 非 nil = 施法的**選格階段**(手冊 p.34 的 I/J/K/M + 空白鍵)
	cursor *castCursor
	// castSP 非 nil = 施法的**投入點數**那一步(CMBT:101「 # SP ? 」)。
	// ⚠ 這一步先前被跳掉(固定投一級),而投入量會改變威力與狀態強度 ——
	// 少了它,`SPELLS.DAT` 的單價欄在遊戲裡等於沒有作用。
	castSP *castSPState
	// castPage 是法術清單的分頁(CMBT:113 的 PgDn)。
	castPage int

	// inspect 非 nil = 戰場的單位檢視面板開著(CMBT:179–192,inspect_scene.go)。
	inspect *inspectState

	// townCount 非 0 = 城鎮正在問「要幾個」(1–9,0 離開):
	// 'R' 住幾晚(TOWN:30/31)、'B' 買幾份口糧(TOWN:58/59)。
	// ⚠ 放在 Game 而不是 townState —— townState 是唯讀邊界(use_item.go 有說明),
	// 子流程狀態一律放這裡,與 campPotion 同一種做法。
	townCount byte

	// healPay 非 nil = 治療所正在問「這將花費 N 金幣,付款嗎?(Y/N)」(TOWN:27/28)。
	healPay *healPayState

	// pendingGold > 0 = 戰後撿到的金幣還沒回答「要撿嗎?」(CMBT:60–63)。
	// ⚠ 沒答就是不撿 —— 見 exp.go 的 takeGold。
	pendingGold int
	// pendingLoot 非 nil = 戰後掉落的道具還沒回答「要撿嗎?」(CMBT:61–63)。
	// ⚠ 與金幣同一條規矩:沒答就是不撿(docs/re/200 §1.2)。
	pendingLoot *pendingLoot

	// dispelled 記著這一場戰鬥裡誰已經用過 D)ispell(角色編號 → 用過了)。
	// 建新戰場時清空,不進存檔 —— 見 combat_scene.go 的 dispell()。
	dispelled map[int]bool

	// 戰鬥中的 U)se an item + 藥劑「自己/丟給別人」(docs/spec/12-combat-board.md
	// §5.3、docs/spec/19-coverage.md §2-1)。use_item.go。
	useUnit      int           // 開道具選單的人(field.Units 索引,同 castUnit 的用法)
	useList      []useEntry    // 選單開著時的可選道具清單
	combatPotion *potionPrompt // 非 nil = 正在問「自己/丟給別人」或選目標(戰鬥)

	// 營地 U)se an item 的同一段子流程(docs/spec/16-camp-actions.md §3、
	// docs/spec/19-coverage.md §2-1)。townState(town_scene.go)是唯讀邊界、
	// 不能新增欄位,子流程狀態放在這裡,由 use_item.go 的 campPotionKey /
	// campPotionLines 讀寫。
	campPotion *potionPrompt

	// M12:遊戲外殼(docs/spec/15-game-shell.md)。標題/主選單/隊伍選擇/全滅/結局。
	shell     *shellState
	titleFont *render.Painter // 標題用 32px(docs/spec/04 §4)

	// bossFight:目前這場戰鬥是不是迷宮事件目標 533(maze.TargetFinalBoss,
	// 最終首領 Siriadne)引發的劇情戰鬥。
	//
	// 由 combat_scene.go 的 startScriptedCombat 設定(docs/spec/17)——
	// 迷宮事件目標 533 的怪物組成已經解出來了(docs/re/180:2 隻 Great Dragon
	// + Siriadne !)。打贏且這個旗標為真 → 結局畫面(docs/spec/15 §7)。
	bossFight bool

	// M13:引擎自己的存檔格式(docs/spec/18-save-format.md)。取代 M3 把
	// GROUPS.DAT/CHARS.DAT 當主存檔路徑的做法 —— 那兩個檔仍然是 roster/create
	// (engine/roster_scene.go、create_scene.go,邊界不准動)在用的**工作副本**,
	// JSON 存檔(saves/party.json)才是玩家實際看到的「存檔」(docs/spec/18 §5)。
	saveDir string // saves/*.json 所在目錄。預設在資產目錄旁邊,-save 可覆寫

	// saveName 是目前選定的具名存檔(docs/spec/18 §2 的多存檔;B2 存檔選擇 /
	// B3 匯入 / 另存新檔畫面,shell_scene.go 與 save_ui.go)。
	//
	// ⚠ 空字串 = 還沒經過那些畫面選過(或測試用 struct literal 直接建構
	// *Game,例如既有測試 fixture)——effectiveSaveName() 這時候退回
	// save.DefaultName,與 B2/B3/另存新檔加入之前「寫死 save.DefaultName」
	// 的行為完全一致。
	saveName string

	// saveAs 非 nil = 另存新檔的文字輸入畫面開著,蓋在遊戲畫面之上
	// (save_ui.go)。docs/spec/18 §2:多存檔 = 多個檔,玩家要能替目前進度
	// 另外取一個檔名,不然「多存檔」在玩家那一側等於不存在。
	saveAs *saveAsState

	// disabledEvents 是已作廢的一次性事件(docs/spec/18 §3.2 的
	// Progress.DisabledEvents):key = 事件檔編號(original.MazeEntry.TextFile,
	// 即 DE<N>EFF.BIN 的 N)→ 目標編號集合。
	//
	// ⚠ 這是修 docs/re/181 那個真實缺口的關鍵狀態:maze.DisableTarget 只改
	// 記憶體裡當時載入的 level.events,而 loadLevel 每次進迷宮都重新讀檔 ——
	// 沒有這份記錄,走出迷宮再走回來,一次性事件就復活了。
	// loadLevel(maze_scene.go)在讀完事件表之後,把這裡記著的目標重新作廢一次。
	disabledEvents map[string][]int

	// pendingActive / pendingMazeFile / pendingMazeFacing 是 hydrateFromSave
	// 從 JSON 存檔讀到、還沒套用的「上次玩到第幾隊、在哪座迷宮的哪個朝向」。
	// resumeMaze(shell_scene.go 的 selectParty 呼叫)套用之後會清掉這三個欄位。
	//
	// ⚠ 只在選到的隊伍剛好是存檔的 Active 那一隊時才套用 —— MazeFile 是
	// 單一一組欄位(不像 Group.MazeX/MazeY 每隊各有一份),選別隊時沒有對應
	// 的意義,這是本引擎的簡化,不是 RE 結論(docs/spec/18 沒有規定這件事)。
	pendingActive     int
	pendingMazeFile   string
	pendingMazeFacing int

	// testKeys 是測試用的假輸入佇列(docs/spec/15 §9)。
	// nil = 正式執行,Update() 照舊呼叫 inpututil 讀真正的鍵盤;
	// 非 nil(即使是空切片)= 測試模式,底下改用這個切片當「這一格剛按下的鍵」。
	//
	// ⚠ 只為了這件事才加這層:ebitengine 在沒有 X11 display 的環境
	// 連套件初始化都會 panic(glfw.Init 需要 display),而 inpututil 的
	// 「剛按下」狀態本來就是靠 RunGame 的事件迴圈在更新 —— 不跑 RunGame,
	// 不管有沒有 display,永遠讀不到任何鍵。沒有這層接縫,
	// docs/spec/15 §9 要求的「不開視窗、直接呼叫 Update()」測試做不到。
	testKeys []ebiten.Key

	// testRunes 同 testKeys,但給文字輸入用(目前只有另存新檔畫面,
	// save_ui.go 在用)——ebiten.AppendInputChars 跟 inpututil 一樣靠
	// RunGame 的事件迴圈填,不跑 RunGame 永遠是空的,道理與 testKeys 相同。
	testRunes []rune
}

// pressedKeys / pressed 包一層 inpututil,讓 g.testKeys 非 nil 時可以
// 用假輸入取代真正的鍵盤(見上面 testKeys 的說明)。
func (g *Game) pressedKeys() []ebiten.Key {
	if g.testKeys != nil {
		return g.testKeys
	}
	return inpututil.AppendJustPressedKeys(nil)
}

func (g *Game) pressed(k ebiten.Key) bool {
	if g.testKeys != nil {
		for _, tk := range g.testKeys {
			if tk == k {
				return true
			}
		}
		return false
	}
	return inpututil.IsKeyJustPressed(k)
}

// inputRunes 包一層 ebiten.AppendInputChars,讓 g.testKeys 非 nil(測試模式)
// 時可以用 g.testRunes 取代真正的文字輸入 —— 沿用 g.testKeys 當「是不是
// 測試模式」的旗標,不必另開一個布林:測試一律會先設 g.testKeys(即使是
// 空切片)才呼叫 Update(),見 pressedKeys() 的同一條說明。
func (g *Game) inputRunes() []rune {
	if g.testKeys != nil {
		return g.testRunes
	}
	return ebiten.AppendInputChars(nil)
}

func (g *Game) Update() error {
	// 輸入**整格只收一次**,再往下傳(scene.go 的 Input)——
	// 先前每個分支各自呼叫 inpututil,其中一半繞過了 g.testKeys 接縫。
	in := Input{Keys: g.pressedKeys(), Runes: g.inputRunes()}

	// 派工是一張表,順序就是優先序(scene.go 的 inputChain)。
	// 第一個 Handles 為真的場景吃掉這一格 —— 唯一會「不吃就往下傳」的是
	// N)ames 熱鍵,它的 Handles 只在真的按下 N 時為真。
	for _, sc := range g.inputChain() {
		if !sc.Handles(in) {
			continue
		}
		if sc.Update(in) == TransitionQuit {
			return ebiten.Termination
		}
		return nil
	}
	return nil
}

// save 把目前狀態寫回 GROUPS.DAT。docs/spec/06 §6。
//
// ⚠ 寫的是 <assets>/save/ 的複本,**不碰 game/sharspri/**(CLAUDE.md §8)。
// ⚠ 只覆寫已解的欄位;未解的位置由 Group.Bytes() 從 Raw 原樣保留。
// syncMember 把一位隊伍成員的變化寫回名冊。
//
// ⚠ `g.members` 是 `g.chars` 的**複本**,不是指標 ——
// 戰鬥扣的血、拿到的經驗全部只改到複本。少了這一步,
// 存檔會把「打完仗之前」的名冊寫回去,而且**不會報錯**:
// 檔案長度對、欄位合法、原版也開得起來,只是進度沒了。
func (g *Game) syncMember(c original.Character) {
	if c.ID >= 1 && c.ID <= len(g.chars) {
		g.chars[c.ID-1] = c
	}
}

func (g *Game) syncMembers() {
	for _, c := range g.members {
		g.syncMember(c)
	}
}

// save 寫回存檔。**兩個檔一起寫,再寫一份 JSON。**
//
// 原版的存檔常式(`USERLIB` 槽 34)在同一段裡開了兩個檔:
// `#1` = `CHARS.DAT`(記錄 94)、`#2` = `GROUPS.DAT`(記錄 90),docs/re/80 §1。
// 只寫其中一個會讓兩份資料對不上 —— 隊伍位置前進了,成員的血量卻回到上一次。
//
// ⚠ docs/spec/18 §5:**JSON 存檔才是玩家看到的「存檔」**,上面兩個 .DAT
// 是工作副本 —— roster_scene.go / create_scene.go(邊界不准動)直接讀寫
// 它們,拿掉會讓那兩支檔案的存檔功能失效。這裡兩層都寫,JSON 是額外多做的,
// 不是取代(取代做不到,見 docs/spec/18 §5 的邊界:不能動那兩支檔案)。
func (g *Game) save() error {
	g.syncMembers()
	if err := g.writeChars(); err != nil {
		return err
	}
	b, err := os.ReadFile(g.savePath)
	if err != nil {
		return err
	}
	groups, err := original.ParseGroups(b)
	if err != nil {
		return err
	}
	grp := g.group
	grp.WorldX, grp.WorldY = g.party.X, g.party.Y
	grp.Facing = int(g.party.Facing)
	grp.Month, grp.Day = g.party.Clock.Month, g.party.Clock.Day
	grp.Hour, grp.Sub = g.party.Clock.Hour, g.party.Clock.Sub
	grp.Encounter = g.party.Encounter
	// 光源三欄(位移 45/59/61)。⚠ 生效能見度不寫回 —— 記錄裡沒有那一欄
	// (docs/re/204 §2),寫回去會蓋掉 59 或 61 其中一個。
	grp.LightTurns = g.party.LightTurns
	grp.VisLit, grp.VisDark = g.party.VisLit, g.party.VisDark
	// 位移 83:在不在迷宮、是哪一座(docs/re/204 §1)。原版靠它決定
	// 讀檔後要 CHAIN 到 WRLDMOVE 還是 MAZEMOVE。
	grp.MazeNum = g.party.MazeNum

	// docs/spec/18 §3.2 MazeFile + 驗收 4:在迷宮裡存檔要記住是哪一座、
	// 在哪一格,讀回才能回到迷宮裡而不是世界地圖。
	//
	// ⚠ 朝向**不能**寫回 grp.Facing —— GROUPS.DAT 位移 41 只有一格,
	// 存的是世界地圖朝向,離開迷宮回到世界地圖時還要用它;寫進迷宮朝向
	// 會讓「下次真的走出迷宮」時面向錯的方向。迷宮朝向另外存進
	// save.Progress.MazeFacing(save.go 的說明)。
	mazeFile, mazeFacing := "", 0
	if g.level != nil {
		grp.MazeX, grp.MazeY = g.mazeState.Major, g.mazeState.Minor
		mazeFile = strconv.Itoa(g.level.entry.MazeFile)
		mazeFacing = int(g.mazeState.Facing)
	}
	groups[g.slot-1] = grp

	out := make([]byte, 0, len(b))
	for _, x := range groups {
		out = append(out, x.Bytes()...)
	}
	if len(out) != len(b) {
		return fmt.Errorf("寫出 %d bytes,原檔 %d", len(out), len(b))
	}
	if err := os.WriteFile(g.savePath, out, 0o644); err != nil {
		return err
	}
	return g.writeSaveFile(groups, mazeFile, mazeFacing)
}

// writeSaveFile 組一份 save.Save 並寫進 saves/<effectiveSaveName()>.json。
// docs/spec/18 §5:這是玩家實際看到的存檔。
func (g *Game) writeSaveFile(groups []original.Group, mazeFile string, mazeFacing int) error {
	s := &save.Save{Version: save.CurrentVersion, Active: g.slot}
	copy(s.Chars[:], g.chars)
	copy(s.Groups[:], groups)
	s.Progress = save.Progress{
		DisabledEvents: g.disabledEvents,
		ClanRewarded:   g.clanRewarded,
		MazeFile:       mazeFile,
		MazeFacing:     mazeFacing,
	}
	for n := range g.tombs {
		s.Progress.Tombs = append(s.Progress.Tombs, n)
	}
	sort.Ints(s.Progress.Tombs) // 穩定輸出 —— map 沒有固定順序,不排序的話每次存檔 JSON 都會不一樣
	return save.Write(save.Path(g.effectiveSaveDir(), g.effectiveSaveName()), s)
}

// effectiveSaveName 回傳目前選定的具名存檔。docs/spec/18 §2 的多存檔 ——
// B2 存檔選擇 / B3 匯入(shell_scene.go)與另存新檔(save_ui.go)都靠這裡
// 才能讓 hydrateFromSave/writeSaveFile 讀寫到「玩家挑的那一份」,而不是
// 永遠寫死同一個檔名。
//
// ⚠ g.saveName 是空字串時(還沒經過那些畫面,或測試用 struct literal
// 直接建構 *Game)退回 save.DefaultName —— 與這一輪之前「寫死
// save.DefaultName」的行為完全一致,internal/save/save_test.go 與
// engine/save_test.go 的既有測試都沒有設過 g.saveName,靠這一點才繼續通過。
func (g *Game) effectiveSaveName() string {
	if g.saveName != "" {
		return g.saveName
	}
	return save.DefaultName
}

// effectiveSaveDir 回傳實際要用的存檔目錄。docs/spec/18 §2:預設在資產目錄
// 旁邊,-save 可覆寫(main() 把覆寫值放進 g.saveDir,loadStatic 把預設值
// 放進 g.saveDir——正式執行時這裡一定會直接回傳 g.saveDir)。
//
// ⚠ **沒有經過 loadStatic 設定過的 *Game(測試直接用struct literal 建構)
// 不能落回目前工作目錄** —— filepath.Join("", "party.json") 會解成
// "party.json",相對於 `go test` 的工作目錄(也就是 engine/ 原始碼目錄),
// 這裡曾經真的把測試 fixture 的存檔寫進 engine/party.json,污染了
// 下一個測試讀到的資料(TestJoinBlankSlotFromMainMenuAppliesNewPartyDefaults
// 曾經因此失敗)。改成跟著 g.assets 走(测试 fixture 幾乎都會設 assets),
// 放進 assets **底下**而不是旁邊,確保落在呼叫端已經準備好、會被清掉的
// 目錄裡(例如 t.TempDir())。g.assets 也是空字串時才退回相對路徑
// "saves"——目前沒有任何會呼叫到存讀檔的測試會落到這一支。
func (g *Game) effectiveSaveDir() string {
	if g.saveDir != "" {
		return g.saveDir
	}
	if g.assets != "" {
		return filepath.Join(g.assets, "saves")
	}
	return "saves"
}

// hydrateFromSave 讀 saves/<effectiveSaveName()>.json,若存在就覆寫工作用的
// <assets>/save/{CHARS,GROUPS}.DAT,並套用 Progress(docs/spec/18)。
//
// 找不到檔案(還沒存過)不是錯誤 —— 沿用目前 <assets>/save/ 底下已經有的資料
// (出貨資料,或前一輪 cmd/convert 的輸出)。openPartySelect(shell_scene.go)
// 在列出隊伍槽之前呼叫這支函式,讓「隊伍選擇畫面讀 JSON 存檔」成立
// (docs/spec/15 §5、docs/spec/18 §6)。
func (g *Game) hydrateFromSave() {
	s, err := save.Read(save.Path(g.effectiveSaveDir(), g.effectiveSaveName()))
	if err != nil {
		if !os.IsNotExist(err) {
			g.warnings = append(g.warnings, "讀取存檔失敗:"+err.Error())
		}
		return
	}
	if err := g.applySave(s); err != nil {
		g.warnings = append(g.warnings, "套用存檔失敗:"+err.Error())
	}
}

// applySave 把一份 *save.Save 套進目前的執行狀態。
func (g *Game) applySave(s *save.Save) error {
	var charBytes []byte
	for _, c := range s.Chars {
		charBytes = append(charBytes, c.Bytes()...)
	}
	if err := os.WriteFile(filepath.Join(g.assets, "save", "CHARS.DAT"), charBytes, 0o644); err != nil {
		return err
	}
	parsed, err := original.ParseChars(charBytes)
	if err != nil {
		return err
	}

	var groupBytes []byte
	for _, grp := range s.Groups {
		groupBytes = append(groupBytes, grp.Bytes()...)
	}
	if err := os.WriteFile(g.savePath, groupBytes, 0o644); err != nil {
		return err
	}

	g.chars = parsed
	g.disabledEvents = s.Progress.DisabledEvents
	g.tombs = map[int]bool{}
	for _, n := range s.Progress.Tombs {
		g.tombs[n] = true
	}
	g.clanRewarded = s.Progress.ClanRewarded
	g.pendingActive = s.Active
	g.pendingMazeFile = s.Progress.MazeFile
	g.pendingMazeFacing = s.Progress.MazeFacing
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(cgaBlack)

	// 外殼接管畫面時,把整個畫布讓給它(docs/spec/15)。
	// drawOverlay 仍要呼叫 —— A6 按鍵表(P 鍵)借用的是敘述覆蓋層的機制,
	// 蓋在主選單上。
	if g.shell != nil && g.shell.mode != shellPlaying {
		g.drawShell(screen)
		g.drawOverlay(screen)
		return
	}

	// 圖層順序是一張表(scene.go 的 drawOrder)。**每一格全部都畫** ——
	// 各場景自己判斷該不該畫,這與重構前逐一呼叫 drawXxx() 的行為相同。
	// ⚠ 與 inputChain 的順序不同,而且不該一致:戰鬥吃鍵最優先,
	// 但畫在迷宮之上、覆蓋層之下。
	for _, sc := range g.drawOrder() {
		sc.Draw(screen)
	}

	// M2 還沒有字型(docs/spec/04 §4 的 TTF 是 M3 之後),
	// 所以狀態暫時走 Ebitengine 的除錯字。**這不是最終呈現。**
	ebiten.SetWindowTitle(fmt.Sprintf(
		"春之石 — (%d,%d) 朝向%d  月%d 日%d 時%d  遭遇倒數%d  未解地形%v",
		g.party.X, g.party.Y, g.party.Facing,
		g.party.Clock.Month, g.party.Clock.Day, g.party.Clock.Hour,
		g.party.Encounter, keys(g.noSrc)))
}

func keys(m map[int]bool) []int {
	out := []int{}
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (g *Game) Layout(int, int) (int, int) { return layout.ScreenW, layout.ScreenH }

// startupBanner 是原版開機那一行(START:0)。
const startupBanner = "正在載入《春之石》,(c) 1986-1987 by Strategic Simulations Inc."

// version 由 `tools/release.sh` 用 -ldflags -X 填。原始碼跑是 dev。
var version = "dev"

func main() {
	assets := flag.String("assets", "assets", "資產資料夾(由 cmd/convert 產生)")
	showVer := flag.Bool("version", false, "印版本後結束")
	// docs/spec/15 §1:預設走標題 → 主選單 → 隊伍選擇。-slot 只是除錯捷徑。
	slot := flag.Int("slot", 0,
		"除錯捷徑:跳過標題/主選單/隊伍選擇,直接進第幾隊(1–5)。0 = 預設,走標題流程")
	fontPath := flag.String("font", "", "字型檔;留空則依序試內建候選")
	seed := flag.Uint64("seed", 1, "戰鬥亂數種子(docs/spec/07 §2:同種子可重現)")
	enc := flag.Int("encounter", -1, "覆寫遭遇倒數(除錯用,只在搭配 -slot 時生效)")
	x := flag.Int("x", -1, "覆寫起始 x(除錯用,只在搭配 -slot 時生效)")
	y := flag.Int("y", -1, "覆寫起始 y")
	saveDir := flag.String("save", "",
		"存檔目錄(docs/spec/18 §2)。留空 = 資產目錄旁邊的 saves/")
	flag.Parse()

	if *showVer {
		fmt.Println("shard", version)
		return
	}

	// START:0 —— 原版開機時印的那一行。引擎**沒有讀取畫面**(標題只印
	// `(c)` 那半句),所以這一句印到 stderr,同 WRLDMOVE:19 的道別做法。
	fmt.Fprintln(os.Stderr, startupBanner)

	g, err := loadStatic(*assets, *fontPath, *seed)
	if err != nil {
		// 原版有**兩句**:資料檔壞掉(CMBT:172 / CAMP:87)與磁片有問題
		// (TOWN:75)。引擎的對應是「檔案讀得到但解不開」與「整個資產目錄
		// 根本不在」——後者對玩家而言就是「你的磁片有問題」那一類。
		// ⚠ 底下兩行是**本引擎加的**:原版叫玩家「重新還原」磁片,
		// 而這裡的對應動作是重跑轉換器,所以錯誤本身與怎麼修都要印出來。
		if _, statErr := os.Stat(*assets); statErr != nil {
			fmt.Fprintln(os.Stderr, "啊咧!你的磁片好像有問題!")
		} else {
			fmt.Fprintln(os.Stderr, "啊咧!資料檔好像出了問題,請重新還原!")
		}
		fmt.Fprintln(os.Stderr, "  ", err)
		fmt.Fprintln(os.Stderr, "  請先跑:go run ./cmd/convert -in <原版> -out assets")
		os.Exit(1)
	}
	if *saveDir != "" {
		g.saveDir = *saveDir
	}

	if *slot > 0 {
		// 除錯捷徑(docs/spec/15 §1):略過標題/主選單/隊伍選擇,直接進遊戲。
		if err := g.loadParty(*slot); err != nil {
			fmt.Fprintln(os.Stderr, "載入隊伍失敗:", err)
			os.Exit(1)
		}
		// 除錯用的座標/遭遇倒數覆寫。**預設不覆寫** —— 存檔裡的值才是真相。
		if *x >= 0 {
			g.party.X = *x
		}
		if *y >= 0 {
			g.party.Y = *y
		}
		if *enc >= 0 {
			g.party.Encounter = *enc
		}
		g.shell.mode = shellPlaying
	} else {
		g.openTitle() // A1(docs/spec/15 §3)
	}

	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	ebiten.SetWindowTitle("春之石 Shard of Spring")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// loadStatic 讀開局前就需要的靜態資產:世界地圖、圖塊、迷宮資料、規則表、
// 名冊、字型、音訊。**不載入任何一支隊伍** —— docs/spec/15 §1:
// 隊伍要選了才載入,所以拆成「載入靜態資產」與「載入某一支隊伍」兩段
// (後者是 loadParty)。
func loadStatic(dir, fontPath string, seed uint64) (*Game, error) {
	// ⚠ 每個欄位要有自己的 tag。寫成 `W, H int `json:"w"`` 會讓兩個欄位
	// 共用同一個 tag,H 永遠是 0 —— 而 JSON 解碼**不會報錯**。
	var wm struct {
		W     int   `json:"w"`
		H     int   `json:"h"`
		Cells []int `json:"cells"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "data", "worldmap.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &wm); err != nil {
		return nil, err
	}
	if wm.W != world.W || wm.H != world.H {
		return nil, fmt.Errorf("地圖尺寸 %d×%d,規格說 %d×%d", wm.W, wm.H, world.W, world.H)
	}

	tiles := loadTiles(filepath.Join(dir, "gfx", "world"), 38)

	g := &Game{
		world:  &world.Map{Cells: wm.Cells},
		tiles:  tiles,
		assets: dir,
		noSrc:  map[int]bool{},
		// docs/spec/18 §2:saves/ 預設在資產目錄旁邊,-save 可覆寫(main() 裡)。
		saveDir: filepath.Join(filepath.Dir(dir), "saves"),
	}
	if err := readJSON(filepath.Join(dir, "data", "mazedata.json"), &g.mazeData); err != nil {
		return nil, err
	}
	// 迷宮圖塊:MAZEITEM.PIC 第 k 行 = 格值 k(偏移 0)。
	g.mazeTiles = loadTiles(filepath.Join(dir, "gfx", "maze"), 32)

	// 名冊獨立於隊伍槽:C)har Utilities 從主選單就能進去,不必先 L)oad
	// 一支隊伍(docs/spec/15 §4)。GROUPS.DAT 的路徑也在這裡定案,
	// 隊伍選擇畫面(A3)與 loadParty 都靠它。
	g.savePath = filepath.Join(dir, "save", "GROUPS.DAT")
	cb, err := os.ReadFile(filepath.Join(dir, "save", "CHARS.DAT"))
	if err != nil {
		return nil, err
	}
	g.chars, err = original.ParseChars(cb)
	if err != nil {
		return nil, err
	}

	if err := g.loadCombat(dir, seed); err != nil {
		return nil, err
	}

	// 字型分兩個版本(font_ttf.go / font_eten.go,build tag `eten`)——
	// 發行版用開源向量字,本機版用倚天點陣字。字級與欄寬預算兩套都不同。
	panel, overlay, title, fontName, err := newPainters(fontPath)
	if err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr, "字型:", fontName)
	g.panel, g.overlayFont, g.titleFont = panel, overlay, title
	g.initSound() // docs/spec/13:失敗只記警告,不影響遊戲
	g.shell = &shellState{mode: shellTitle}
	return g, nil
}

// loadParty 選一支隊伍進遊戲。docs/spec/15 §5 = 開局的唯一入口。
//
// ⚠ 讀的是 <assets>/save/ 底下的**複本**,不是 game/sharspri/ ——
// 存檔要寫回去,而原版目錄是唯讀的(CLAUDE.md §8)。
// 呼叫前 g.chars 必須已經由 loadStatic 讀好 —— CHARS.DAT 與隊伍槽無關。
func (g *Game) loadParty(slot int) error {
	if slot < 1 || slot > original.GroupSlots {
		return fmt.Errorf("存檔槽 %d 超出 1–%d", slot, original.GroupSlots)
	}
	gb, err := os.ReadFile(g.savePath)
	if err != nil {
		return err
	}
	groups, err := original.ParseGroups(gb)
	if err != nil {
		return err
	}
	grp := groups[slot-1]
	if grp.Blank() {
		return fmt.Errorf("第 %d 隊還沒建立(記錄整份是空白)", slot)
	}

	g.slot = slot
	g.members = nil
	// 以 GROUPS.DAT 的成員槽為準(docs/spec/06 §1)
	for _, id := range grp.MemberIDs() {
		if id < 1 || id > len(g.chars) {
			g.warnings = append(g.warnings,
				fmt.Sprintf("成員編號 %d 超出名冊 1–%d", id, len(g.chars)))
			continue
		}
		c := g.chars[id-1]
		if !c.Occupied() {
			g.warnings = append(g.warnings, fmt.Sprintf("成員編號 %d 指向空槽", id))
			continue
		}
		// CHARS.DAT 位移 1 應該回指同一隊。不一致只記錄,**不自行修正**。
		if p, ok := c.InParty(); !ok || p != slot {
			g.warnings = append(g.warnings,
				fmt.Sprintf("%s 在第 %d 隊的成員表裡,自己的隊伍欄卻是 %q",
					c.Name, slot, string(c.Party)))
		}
		g.members = append(g.members, c)
	}

	g.group = grp
	g.party = world.State{
		X: grp.WorldX, Y: grp.WorldY, Facing: world.Facing(grp.Facing),
		Clock: world.Clock{
			Sub: grp.Sub, Hour: grp.Hour, Day: grp.Day, Month: grp.Month,
		},
		Encounter: grp.Encounter,
	}
	// 遭遇倒數的載入補值:原版在載入常式裡做 `≤ 2 → 25`
	// (`MENU` 0x119A2,docs/re/204 §3)。存進去的小值不會被拿來用。
	if g.party.Encounter <= world.EncounterFloor {
		g.party.Encounter = world.EncounterReload
	}
	g.loadLight()
	return nil
}

// loadCombat 讀 M4 需要的資料表。docs/spec/07。
func (g *Game) loadCombat(dir string, seed uint64) error {
	var monsters []original.Monster
	if err := readJSON(filepath.Join(dir, "data", "monsters.json"), &monsters); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(dir, "data", "spells.json"), &g.spells); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(dir, "data", "townsites.json"), &g.townSites); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(dir, "data", "shops.json"), &g.shops); err != nil {
		return err
	}
	// 傳聞。⚠ 找不到就是找不到(docs/re/138 §4:10 段對 11 個索引),
	// 畫面上會明講,不拿別段頂替。
	g.rumors = map[int]string{}
	_ = readJSON(filepath.Join(dir, "data", "rumors.json"), &g.rumors)
	if err := readJSON(filepath.Join(dir, "data", "items.json"), &g.itemList); err != nil {
		return err
	}
	g.items = map[int]combat.Item{}
	for _, it := range g.itemList {
		// ⚠ 欄4/欄5 是型別相依的(docs/re/74)。這裡原樣搬,
		// 由 combat 依「這是武器還是防具」去解讀 —— 不在載入時分類。
		// Name 只給戰鬥訊息用(F3 的「使用 <武器>」),不進公式。
		// Alias 是未鑑定時的說法(docs/re/192 §4),同樣只給訊息用。
		g.items[it.Index] = combat.Item{
			Main: it.Col4, Bonus: it.Col5, Name: it.Name, Alias: it.Alias,
		}
	}
	g.monsters = monsters
	g.rand = combat.NewRand(seed)
	return nil
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// loadTiles 讀 <dir>/t00.png … t<max>.png。讀不到的編號不進 map ——
// 執行期會畫佔位符,讓「沒有來源」在畫面上看得見(docs/spec/05 §4)。
func loadTiles(dir string, max int) map[int]*ebiten.Image {
	out := map[int]*ebiten.Image{}
	for v := 0; v <= max; v++ {
		f, err := os.Open(filepath.Join(dir, fmt.Sprintf("t%02d.png", v)))
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err == nil {
			out[v] = ebiten.NewImageFromImage(img)
		}
	}
	return out
}

// writeChars 把名冊寫回 <assets>/save/CHARS.DAT。
//
// ⚠ 與 GROUPS.DAT 一樣,**只覆寫已解的欄位** —— 未解的位置由 Raw 原樣保留
// (docs/spec/06 §3 的同一條原則)。
func (g *Game) writeChars() error {
	path := filepath.Join(g.assets, "save", "CHARS.DAT")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(b) != original.CharRecLen*original.CharSlots {
		return fmt.Errorf("CHARS.DAT 長度 %d 不對", len(b))
	}
	out := append([]byte(nil), b...)
	for i, c := range g.chars {
		copy(out[i*original.CharRecLen:], c.Bytes())
	}
	return os.WriteFile(path, out, 0o644)
}
