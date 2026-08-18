package main

// T1:無頭跑一輪的整合測試(docs/spec/14-remake-worklist.md §8)。
//
//	讀檔 → 走幾步 → 觸發戰鬥 → 打完 → 進城 → 買東西 → 存檔 → 讀回
//
// 這是 C(場景重構)的前置條件:重構之後行為要一模一樣,而既有的 68 個測試
// 全是單點的,擋不住「場景之間接錯線」那一類回歸。
//
// ⚠ **用真正的資產**(版控裡的 assets/,cmd/convert 的產物),不是合成 fixture ——
// 資料格式漂移(欄位改名、圖塊編號跳號、shops.json 的範圍變了)只有讀真檔會紅。
// 資產先整份複製到 t.TempDir():存檔會寫回 <assets>/save/*.DAT,
// 版控裡那份不能被測試改到(tools/go.sh 也是 :ro 掛進來的)。
//
// ⚠ **一律走 Update() 的真按鍵路徑。** 輸入在 Update() 開頭收一次
// (scene.go 的 Input),測試用 g.testKeys 餵進去 —— 所以派工表
// (scene.go 的 inputChain)本身也在涵蓋範圍內:某個場景漏接一個鍵,
// 或優先序被改動,這裡會紅。⛔ 不要為了方便改成直呼 g.townKey() 之類的
// handler:那樣測到的是規則,測不到接線,而接線斷掉的代價見 docs/spec/14 §8。

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/combat"
	"shardofspring/internal/original"
	"shardofspring/internal/save"
	"shardofspring/internal/town"
	"shardofspring/internal/world"
)

// t1Seed 固定戰鬥亂數種子。docs/spec/07 §2:同種子同輸入 → 同結果。
// 換種子等於換一場戰鬥,底下「打贏」的斷言要重新確認。
const t1Seed = 20260816

// assetsSource 找出版控裡的資產目錄。
//
// ⚠ **找不到就讓測試紅,不 t.Skip()** —— skip 的綠燈與「跑過且通過」
// 在輸出裡長得一樣,而這份測試的價值正在於它真的讀了那些檔
// (~/diagnosis-notes 03:沉默不是成功)。
func assetsSource(t *testing.T) string {
	t.Helper()
	// SHARD_ASSETS 由 tools/go.sh 設成 /assets(唯讀掛載);
	// ../assets 是直接在主機上跑 go test 時的位置。
	for _, p := range []string{os.Getenv("SHARD_ASSETS"), "/assets", "../assets"} {
		if p == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(p, "data", "worldmap.json")); err == nil {
			return p
		}
	}
	t.Fatal("找不到資產目錄:試過 $SHARD_ASSETS、/assets、../assets " +
		"(容器裡由 tools/go.sh 掛載;主機上請確認 repo 的 assets/ 還在)")
	return ""
}

// assetsCopy 把資產整份複製到 t.TempDir(),回傳複本路徑。
// 遊戲會寫 <assets>/save/{CHARS,GROUPS}.DAT,所以不能就地用版控那份。
func assetsCopy(t *testing.T) string {
	t.Helper()
	src, dst := assetsSource(t), filepath.Join(t.TempDir(), "assets")
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
	if err != nil {
		t.Fatal("複製資產失敗:", err)
	}
	return dst
}

// newIntegrationGame 載入真資產,回傳還停在標題畫面的 *Game。
func newIntegrationGame(t *testing.T, dir string) *Game {
	t.Helper()
	g, err := loadStatic(dir, "", t1Seed)
	if err != nil {
		t.Fatal("載入資產失敗:", err)
	}
	// ⚠ 容器裡沒有音訊裝置。initSound 的 recover 擋得住初始化,
	// **播放時的錯誤是另一條路**(shot_test.go 的教訓)。
	g.sound = nil
	return g
}

// press 走 Update() 的真按鍵路徑(只在外殼/覆蓋層那幾段有效,見檔頭)。
func press(t *testing.T, g *Game, keys ...ebiten.Key) {
	t.Helper()
	g.testKeys = keys
	if err := g.Update(); err != nil {
		t.Fatal("Update:", err)
	}
}

// dirKey 是朝向 → 方向鍵。世界地圖與迷宮共用同一組(scene.go 的 dirs)。
func dirKey(dir world.Facing) ebiten.Key {
	switch dir {
	case world.North:
		return ebiten.KeyUp
	case world.East:
		return ebiten.KeyRight
	case world.South:
		return ebiten.KeyDown
	}
	return ebiten.KeyLeft
}

// worldStep 按一次方向鍵,回傳有沒有**實際位移**。
//
// ⚠ 朝向不同時只轉身不位移(docs/spec/05 §6),所以第一下常常是 false。
// 走的是 Update() → inputChain → worldScene 的完整路徑,不是呼叫內部函式。
func worldStep(t *testing.T, g *Game, dir world.Facing) bool {
	t.Helper()
	x, y := g.party.X, g.party.Y
	press(t, g, dirKey(dir))
	return g.party.X != x || g.party.Y != y
}

// fightToEnd 用戰場按鍵把一場戰鬥打到分出結果。
//
// 策略與怪物那半邊的佔位 AI 同形(combat.MonsterTurn):相鄰就轉向 + A 攻擊,
// 否則往差距大的那一軸靠一格。⚠ 這是**測試的驅動策略**,不是遊戲規則。
func fightToEnd(t *testing.T, g *Game) combat.Outcome {
	t.Helper()
	const maxKeys = 40000 // 上限只為了「卡住時要紅,不要掛住」
	detours := []ebiten.Key{ebiten.KeyUp, ebiten.KeyRight, ebiten.KeyDown, ebiten.KeyLeft}
	last, detour := "", 0
	for i := 0; i < maxKeys; i++ {
		f := g.field
		if f == nil {
			t.Fatal("戰鬥憑空消失了")
		}
		if o := f.Outcome(); o != combat.Ongoing {
			return o
		}
		if g.actor < 0 {
			// endTurn 判定「全隊都動不了」時會留 −1,得由外面推下一輪。
			g.endTurn()
			continue
		}
		// ⚠ 失敗的動作**不扣行動點數**(board.go 的 spend 只在成功時扣)——
		// 被隊友擋住的隊員按同一個鍵會永遠沒有效果。沒進展就**換一個方向**
		// (轉身會扣點,所以點數終究會用完、換下一個人),四個方向都試過
		// 才結束這一輪。
		//
		// ⚠ 這一段是接上原版目標鎖定(docs/re/186)之後補的:怪物改從八個
		// 錨點出場,隊形與距離都變了,而原本「只往一個方向推」的驅動剛好在
		// 舊佈陣下走得通 —— **測試通過過,不代表驅動是對的**。
		if now := fieldFingerprint(g); now == last {
			detour++
			if detour > len(detours) {
				press(t, g, ebiten.KeyEnter)
				detour, last = 0, ""
				continue
			}
			press(t, g, detours[detour-1])
			continue
		} else {
			last, detour = now, 0
		}
		u := f.Units[g.actor]
		j := -1
		best := 1 << 30
		for k := combat.MonsterBase; k < combat.MonsterBase+combat.MonsterMax; k++ {
			v := f.Units[k]
			if !v.Alive() || !v.OnField() {
				continue
			}
			if d := iabs(v.X-u.X) + iabs(v.Y-u.Y); d < best {
				j, best = k, d
			}
		}
		if j < 0 {
			t.Fatal("場上沒有活著的怪物,但 Outcome 說戰鬥還在進行")
		}
		tgt := f.Units[j]
		dir := towardKey(u.X, u.Y, tgt.X, tgt.Y)
		if best == 1 && u.Facing == facingOf(dir) {
			press(t, g, ebiten.KeyA) // 面對面了 → 攻擊
			continue
		}
		press(t, g, dir) // 朝向不同 → 轉身;相同 → 前進一格
	}
	var st strings.Builder
	for i := range g.field.Units {
		if u := g.field.Units[i]; u.Name != "" {
			fmt.Fprintf(&st, "\n  #%d %s (%d,%d) 朝%d HP%d 點%d",
				i, u.Name, u.X, u.Y, int(u.Facing), u.HP, g.points[i])
		}
	}
	tail := g.field.Log
	if len(tail) > 6 {
		tail = tail[len(tail)-6:]
	}
	t.Fatalf("按了 %d 次還沒打完 —— 戰場驅動卡住了。actor=%d 回合=%d%s\n  log:%v",
		maxKeys, g.actor, g.field.Round, st.String(), tail)
	return combat.Ongoing
}

// fieldFingerprint 是「這一格按鍵有沒有改變任何東西」的判準:
// 誰在動、剩幾點、所有單位的位置/朝向/生命。
func fieldFingerprint(g *Game) string {
	var b strings.Builder
	fmt.Fprintf(&b, "a%d r%d", g.actor, g.field.Round)
	for i := range g.field.Units {
		u := g.field.Units[i]
		fmt.Fprintf(&b, "|%d,%d,%d,%d,%d", u.X, u.Y, int(u.Facing), u.HP, g.points[i])
	}
	return b.String()
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// towardKey 回傳朝目標的方向鍵:相鄰時指向目標,否則挑差距大的那一軸。
func towardKey(x, y, tx, ty int) ebiten.Key {
	if iabs(tx-x)+iabs(ty-y) == 1 {
		switch {
		case ty < y:
			return ebiten.KeyUp
		case ty > y:
			return ebiten.KeyDown
		case tx > x:
			return ebiten.KeyRight
		}
		return ebiten.KeyLeft
	}
	if iabs(tx-x) >= iabs(ty-y) {
		if tx > x {
			return ebiten.KeyRight
		}
		return ebiten.KeyLeft
	}
	if ty > y {
		return ebiten.KeyDown
	}
	return ebiten.KeyUp
}

func facingOf(k ebiten.Key) combat.Facing {
	switch k {
	case ebiten.KeyUp:
		return combat.North
	case ebiten.KeyRight:
		return combat.East
	case ebiten.KeyDown:
		return combat.South
	}
	return combat.West
}

func TestT1FullLoop(t *testing.T) {
	dir := assetsCopy(t)
	g := newIntegrationGame(t, dir)

	// ── 1. 讀檔:標題 → 主選單 → L)oad → 匯入原版存檔 → 選第 5 隊 ────────
	// 這一段走 Update() 的真按鍵路徑(外殼有 g.testKeys 接縫)。
	press(t, g, ebiten.KeyEnter)
	if g.shell.mode != shellMainMenu {
		t.Fatalf("標題按鍵後應進主選單,得到 %v", g.shell.mode)
	}
	press(t, g, ebiten.KeyL)
	if g.shell.mode != shellImportPrompt {
		t.Fatalf("還沒有任何具名存檔,L 應進匯入入口,得到 %v", g.shell.mode)
	}
	press(t, g, ebiten.KeyY)
	if g.shell.mode != shellPartySelect {
		t.Fatalf("匯入後應進隊伍選擇,得到 %v(msg=%q)", g.shell.mode, g.shell.msg)
	}
	press(t, g, ebiten.KeyDigit5)
	if g.shell.mode != shellPlaying {
		t.Fatalf("選第 5 隊後應進遊戲,得到 %v(msg=%q)", g.shell.mode, g.shell.msg)
	}
	if g.slot != 5 || len(g.members) == 0 {
		t.Fatalf("載入的隊伍不對:slot=%d 成員 %d 人", g.slot, len(g.members))
	}
	startX, startY := g.party.X, g.party.Y
	t.Logf("PARTY #5:%d 人,座標 (%d,%d),金幣 %.0f,食糧 %d",
		len(g.members), startX, startY, g.group.Gold, g.group.Provisions)

	// ── 2. 走幾步:轉身也算一步(docs/re/149)────────────────────────────
	clock0 := g.party.Clock
	moved := 0
	for _, d := range []world.Facing{world.North, world.North, world.North} {
		if worldStep(t, g, d) {
			moved++
		}
		if g.field != nil || g.town != nil || g.level != nil {
			break // 半路遇到事情就停,底下各步驟自己會處理
		}
	}
	if moved == 0 {
		t.Fatalf("往北走三次一步都沒移動(起點 (%d,%d))", startX, startY)
	}
	if g.party.Clock == clock0 {
		t.Error("走了路時鐘卻沒動 —— docs/re/149:每個動作一格")
	}

	// ── 3. 觸發戰鬥:倒數歸零的下一步就開打 ─────────────────────────────
	if g.field == nil {
		g.town, g.level = nil, nil
		g.party.Encounter = 0
		for i := 0; i < 8 && g.field == nil; i++ {
			// 朝向已經是北,所以每一下都是實際位移
			if !worldStep(t, g, world.North) {
				break
			}
		}
	}
	if g.field == nil {
		t.Fatalf("遭遇倒數歸零後走了幾步仍沒開打(座標 (%d,%d),warnings=%v)",
			g.party.X, g.party.Y, g.warnings)
	}
	hp0 := make([]int, len(g.members))
	for i, c := range g.members {
		hp0[i] = c.HP
	}
	// ⚠ 經驗發給 **g.members**(exp.go 的 awardExp),不是戰場單位 ——
	// 戰場單位的 Exp 欄是「打倒這隻怪物給多少」(屬性 19),兩者同名不同義。
	exp0 := g.members[0].Exp

	// ── 4. 打完:一路打到分出結果,再按 ESC 收尾 ────────────────────────
	outcome := fightToEnd(t, g)
	if outcome != combat.MonstersDead {
		t.Fatalf("這一場沒打贏(%v)—— 種子 %d 的戰鬥變了就要重新確認這條斷言;log:\n%v",
			outcome, t1Seed, g.field.Log)
	}
	if got := g.members[0].Exp; got <= exp0 {
		t.Errorf("打贏了第一位隊員的經驗卻沒增加:%v → %v(log:%v)",
			exp0, got, g.field.Log)
	}
	// 收尾走 Update():戰鬥分支的 ESC 是少數有接縫的一段。
	press(t, g, ebiten.KeyEscape)
	if g.field != nil {
		t.Fatal("戰鬥結束後按 ESC,g.field 應該清成 nil")
	}
	if g.shell.mode != shellPlaying {
		t.Fatalf("打贏一場普通遭遇不該離開遊戲,得到 %v", g.shell.mode)
	}
	if g.party.Encounter == 0 {
		t.Error("戰鬥結束後遭遇倒數沒有重置 —— 下一步會立刻再開打")
	}

	// ⚠ 戰鬥的傷害只改到 g.members(g.chars 的複本),要靠 save() 的
	// syncMembers 才寫得回名冊。這裡先記下來,存讀檔那一段要對得上。
	hurt := -1
	for i, c := range g.members {
		if c.HP != hp0[i] {
			hurt = i
			break
		}
	}

	// ── 5. 進城:用座標表走進去(TOWNDATA.BIN,docs/re/53 §2)───────────
	if len(g.townSites) == 0 {
		t.Fatal("townsites.json 沒有任何城鎮 —— 資產不完整")
	}
	site := g.townSites[0]
	// 站到城鎮西邊一格,朝東,再走一步進去 —— 走的是 worldStep 的城鎮分支,
	// 不是直接呼叫 enterTown。
	g.party.X, g.party.Y, g.party.Facing = site.X-1, site.Y, world.East
	if !worldStep(t, g, world.East) {
		t.Fatalf("走不進城鎮 (%d,%d)", site.X, site.Y)
	}
	if g.town == nil {
		t.Fatalf("踩到城鎮格 (%d,%d) 卻沒進城", site.X, site.Y)
	}
	if g.town.mode != townBuildings {
		t.Fatalf("進城應停在建築清單,得到 %v", g.town.mode)
	}
	t.Logf("進城:%s,%d 間建築", g.town.name, len(g.town.shops))

	// ── 6. 買東西:挑一間賣道具的店,買一件買得起的 ─────────────────────
	shopIdx := -1
	for i, s := range g.town.shops {
		if s.Kind == original.ShopGoods {
			shopIdx = i
			break
		}
	}
	if shopIdx < 0 {
		t.Fatalf("%s 沒有賣道具的店 —— 換一個城鎮或檢查 shops.json", g.town.name)
	}
	press(t, g, ebiten.KeyA+ebiten.Key(shopIdx))
	if g.town.mode != townShop {
		t.Fatalf("按 %c 應進商店,得到 %v", 'A'+shopIdx, g.town.mode)
	}
	stock := g.town.shopStock(g.itemList)
	if len(stock) == 0 {
		t.Fatal("這間店沒有任何商品")
	}
	buyIdx, gold0 := -1, g.group.Gold
	for i, it := range stock {
		if i >= shopPageSize {
			break
		}
		if float64(town.Price(it.BasePrice, g.town.shop.PriceMult)) <= gold0 {
			buyIdx = i
			break
		}
	}
	if buyIdx < 0 {
		t.Fatalf("%.0f 金幣買不起這間店的任何東西(最便宜 %d)",
			gold0, town.Price(stock[0].BasePrice, g.town.shop.PriceMult))
	}
	// ⚠ **買給第 2 位**,不是第一位 —— 原版會問「交給角色 #」(docs/re/187),
	// 而先前的實作寫死給第一位。買給別人才驗得出那一步真的接上了。
	const buyer = 1
	packBefore := g.members[buyer].Pack
	press(t, g, ebiten.KeyA+ebiten.Key(buyIdx))
	if g.town.pendingBuy == nil {
		t.Fatalf("選了道具之後應該問「交給角色 #」(msg=%q)", g.town.msg)
	}
	press(t, g, ebiten.Key1+ebiten.Key(buyer))
	if g.group.Gold >= gold0 {
		t.Fatalf("買了東西金幣卻沒減少:%.0f → %.0f(msg=%q)",
			gold0, g.group.Gold, g.town.msg)
	}
	if g.members[buyer].Pack == packBefore {
		t.Fatalf("買了東西背包卻沒變(msg=%q)", g.town.msg)
	}
	if g.members[0].Pack != g.chars[g.members[0].ID-1].Pack {
		t.Error("東西不該進第一位的背包")
	}
	bought := g.members[buyer].Pack
	t.Logf("買下 %s:金幣 %.0f → %.0f", stock[buyIdx].Name, gold0, g.group.Gold)

	// 離開商店與城鎮 —— ESC 回建築清單,城鎮本身由世界地圖分支關掉。
	press(t, g, ebiten.KeyEscape)
	if g.town.mode != townBuildings {
		t.Fatalf("商店按 ESC 應回建築清單,得到 %v", g.town.mode)
	}
	g.town = nil

	// ── 7. 存檔 ────────────────────────────────────────────────────────
	// S)ave 走 worldScene → saveHere:**它自己會推進時鐘一格**
	// (docs/re/149:只按 S 不移動,位移 33 仍 +1)。
	press(t, g, ebiten.KeyS)
	if g.saveMsg == "" || strings.Contains(g.saveMsg, "失敗") {
		t.Fatal("存檔沒成功:", g.saveMsg)
	}
	savedPath := save.Path(g.effectiveSaveDir(), g.effectiveSaveName())
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatal("存檔檔案不存在:", err)
	}
	want := struct {
		x, y, encounter int
		gold            float64
		provisions      int
		clock           world.Clock
	}{g.party.X, g.party.Y, g.party.Encounter, g.group.Gold, g.group.Provisions, g.party.Clock}
	wantExp := g.members[0].Exp
	wantHP := -1
	if hurt >= 0 {
		wantHP = g.members[hurt].HP
	}

	// ── 8. 讀回:全新的 *Game,走同一條 L)oad 路徑 ──────────────────────
	g2 := newIntegrationGame(t, dir)
	press(t, g2, ebiten.KeyEnter)
	press(t, g2, ebiten.KeyL)
	if g2.shell.mode != shellSaveList && g2.shell.mode != shellPartySelect {
		t.Fatalf("已經有存檔了,L 應進存檔清單或直接進隊伍選擇,得到 %v(msg=%q)",
			g2.shell.mode, g2.shell.msg)
	}
	if g2.shell.mode == shellSaveList {
		press(t, g2, ebiten.KeyDigit1)
	}
	if g2.shell.mode != shellPartySelect {
		t.Fatalf("選存檔後應進隊伍選擇,得到 %v(msg=%q)", g2.shell.mode, g2.shell.msg)
	}
	press(t, g2, ebiten.KeyDigit5)
	if g2.shell.mode != shellPlaying {
		t.Fatalf("讀回後選第 5 隊應進遊戲,得到 %v(msg=%q)", g2.shell.mode, g2.shell.msg)
	}

	if g2.party.X != want.x || g2.party.Y != want.y {
		t.Errorf("座標沒讀回來:存 (%d,%d),讀 (%d,%d)", want.x, want.y, g2.party.X, g2.party.Y)
	}
	if g2.party.Clock != want.clock {
		t.Errorf("時鐘沒讀回來:存 %+v,讀 %+v", want.clock, g2.party.Clock)
	}
	if g2.party.Encounter != want.encounter {
		t.Errorf("遭遇倒數沒讀回來:存 %d,讀 %d", want.encounter, g2.party.Encounter)
	}
	if g2.group.Gold != want.gold {
		t.Errorf("金幣沒讀回來:存 %.0f,讀 %.0f", want.gold, g2.group.Gold)
	}
	if g2.group.Provisions != want.provisions {
		t.Errorf("食糧沒讀回來:存 %d,讀 %d", want.provisions, g2.group.Provisions)
	}
	if len(g2.members) != len(g.members) {
		t.Fatalf("成員人數變了:存 %d,讀 %d", len(g.members), len(g2.members))
	}
	if g2.members[buyer].Pack != bought {
		t.Errorf("買到的東西沒讀回來:存 %v,讀 %v", bought, g2.members[buyer].Pack)
	}
	if g2.members[0].Exp != wantExp {
		t.Errorf("經驗值沒讀回來:存 %v,讀 %v", wantExp, g2.members[0].Exp)
	}
	if wantHP >= 0 && g2.members[hurt].HP != wantHP {
		t.Errorf("%s 的生命沒讀回來:存 %d,讀 %d",
			g.members[hurt].Name, wantHP, g2.members[hurt].HP)
	}
}

// ── T2:固定種子的行為快照(docs/spec/14 §8)────────────────────────────────
//
// 同樣的種子 + 同樣的輸入 → 同樣的狀態。這一條擋的是「重構之後行為悄悄變了」:
// 場景重構(C)不該改動任何規則,而規則層的單元測試看不到「走這條路徑會發生
// 什麼」。快照本身不寫進版控 —— 兩次跑在同一個測試裡互比,免得快照檔變成
// 另一份要維護的真相(docs/spec/07 §2 的可重現性驗收就是這個意思)。

// t2Run 跑一段固定腳本,回傳最終狀態的指紋。
func t2Run(t *testing.T) string {
	t.Helper()
	g := newIntegrationGame(t, assetsCopy(t))
	press(t, g, ebiten.KeyEnter)
	press(t, g, ebiten.KeyL)
	press(t, g, ebiten.KeyY)
	press(t, g, ebiten.KeyDigit5)
	if g.shell.mode != shellPlaying {
		t.Fatalf("腳本沒進到遊戲:%v(msg=%q)", g.shell.mode, g.shell.msg)
	}

	// 固定的一段路:三步向北,然後把遭遇倒數清零再走到開打為止。
	for i := 0; i < 3; i++ {
		worldStep(t, g, world.North)
	}
	g.party.Encounter = 0
	for i := 0; i < 8 && g.field == nil; i++ {
		worldStep(t, g, world.North)
	}
	if g.field == nil {
		t.Fatal("腳本沒有觸發戰鬥")
	}
	outcome := fightToEnd(t, g)

	var b strings.Builder
	fmt.Fprintf(&b, "pos=%d,%d facing=%d clock=%+v enc=%d gold=%.2f prov=%d outcome=%v\n",
		g.party.X, g.party.Y, int(g.party.Facing), g.party.Clock,
		g.party.Encounter, g.group.Gold, g.group.Provisions, outcome)
	for _, c := range g.members {
		fmt.Fprintf(&b, "%s hp=%d sp=%d exp=%v\n", c.Name, c.HP, c.SP, c.Exp)
	}
	// 戰鬥紀錄整份進指紋 —— 擲骰序列變了這裡就會不一樣。
	for _, line := range g.field.Log {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestT2SameSeedSameResult(t *testing.T) {
	a, b := t2Run(t), t2Run(t)
	if a != b {
		t.Errorf("同種子同輸入卻得到不同結果\n--- 第一次 ---\n%s\n--- 第二次 ---\n%s", a, b)
	}
	if !strings.Contains(a, "outcome=") {
		t.Fatalf("指紋不完整:%s", a)
	}
}

// TestCommittedAssetsAreComplete —— 版控裡的 `assets/` 要含 `cmd/convert` 的**全部**產物。
//
// ⚠ 這條擋的是一個**全綠的失敗**:轉換器新增了輸出(人形、開機畫面),
// 我把它轉進 `workplace/` 給截圖用,卻沒有同步版控那份 ——
// 於是 clone 這個 repo 跑起來看到的是白框,而**沒有任何測試會紅**,
// 因為當時截圖測試讀的是 workplace 那份。
//
// 兩道防線,缺一不可:
//  1. 這張表(數量對不上就紅)
//  2. `shot_test.go` / `promo_test.go` 預設讀版控這份 —— 讓截圖與遊戲看到同一批檔
//
// **新增 `cmd/convert` 的輸出時要加一行**,而加了之後這裡會紅到你重跑轉換器
// 並且把產物 commit 進來為止。那正是它的用途。
func TestCommittedAssetsAreComplete(t *testing.T) {
	src := assetsSource(t)
	for _, w := range []struct {
		glob string
		n    int
	}{
		{"gfx/world/*.png", 32},      // 世界地圖圖塊(FASTWRLD + WRLDITEM)
		{"gfx/maze/*.png", 29},       // 地城圖塊(MAZEITEM + 牆/門/水…)
		{"gfx/tiles/*.png", 9},       // 98-byte 小圖塊
		{"gfx/monst/*.png", 22},      // MONST1–22
		{"gfx/pict/*.png", 5},        // PICT1/2/6/7/8
		{"gfx/walk/w*.png", 10},      // WALKDRAW 十段(世界地圖色盤,docs/re/219)
		{"gfx/walk-maze/w*.png", 10}, // 同一批,地城色盤(0x3D8 = 0x0E)
		{"gfx/startup.png", 1},       // STARTUP.BIN 的整頁 CGA
		{"data/*.json", 35}, // 含 encounters.json(docs/re/225)
		{"save/*.DAT", 2},
	} {
		got, err := filepath.Glob(filepath.Join(src, w.glob))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != w.n {
			t.Errorf("%s:版控裡有 %d 個,應該是 %d 個 —— "+
				"重跑 `tools/go.sh run ./cmd/convert -in /game/sharspri -out …` "+
				"並把產物 commit 進 assets/", w.glob, len(got), w.n)
		}
	}
}
