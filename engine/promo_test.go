package main

// 錄推廣影片的影格。**這不是驗收測試** —— 只在 PROMO_DIR 指定時跑。
//
//	PROMO_DIR=/out go test -run TestPromo -count=1
//
// 與 shot_test.go 的分工:那邊拍**六張定格**給 README,這邊錄**連續影格**
// 給影片。共用同一個道理 —— 一定要跑 `ebiten.RunGame`,離屏 `NewImage`
// 讀不到像素(shot_test.go 的檔頭)。
//
// ⚠ **輸入走真的按鍵**(`g.testKeys`),不是直接改狀態。
// 直接擺狀態錄出來的畫面看起來一樣,但那不是「玩得動」的證據 ——
// 而推廣片要證明的正是這件事。少數幾個進場點(載入隊伍、進城、進地城)
// 例外,因為走完整流程要按十幾次鍵,對影片節奏沒有幫助。

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
)

// beat 是影片的一段。`setup` 進場時做一次,`step` 每一格供一次按鍵。
type beat struct {
	name   string
	frames int
	setup  func(*Game)
	step   func(g *Game, i int) []ebiten.Key
}

// hold 是「什麼都不按」的 step。
func hold(*Game, int) []ebiten.Key { return nil }

// every 每 n 格按一次 keys 裡的下一個鍵,其餘格空手。
//
// ⚠ 中間要留空格,不能每一格都按 —— 一秒 30 步在畫面上是瞬移,
// 而且遊戲的每一步都會推進時鐘與遭遇倒數。
func every(n int, keys ...ebiten.Key) func(*Game, int) []ebiten.Key {
	return func(_ *Game, i int) []ebiten.Key {
		if i%n != 0 || i == 0 {
			return nil
		}
		return []ebiten.Key{keys[(i/n-1)%len(keys)]}
	}
}

type promoRunner struct {
	g      *Game
	beats  []beat
	dir    string
	b, f   int // 目前第幾段、段內第幾格
	total  int // 已經寫出幾張
	t      *testing.T
	labels []string // 每一段從第幾格開始,寫進 index 檔給剪接用
}

func (r *promoRunner) Update() error {
	if r.b >= len(r.beats) {
		return ebiten.Termination
	}
	bt := r.beats[r.b]
	if r.f == 0 {
		// ⚠ 一段只記一行。`r.f == 0` 會**成立好幾次** —— ebiten 掉格時
		// 一次 Draw 之前跑好幾次 Update,而 r.f 是在 Draw 裡才 +1。
		// 這也是 setup 必須冪等的原因(見下面各段的註解)。
		if len(r.labels) == r.b {
			r.labels = append(r.labels, fmt.Sprintf("%d\t%s", r.total, bt.name))
		}
		if bt.setup != nil {
			bt.setup(r.g)
		}
	}
	// ⚠ testKeys 要**非 nil**才算測試模式(main.go 的 pressedKeys),
	// 空切片與 nil 在這裡不是同一件事。
	keys := bt.step(r.g, r.f)
	if keys == nil {
		keys = []ebiten.Key{}
	}
	r.g.testKeys = keys
	return r.g.Update()
}

func (r *promoRunner) Draw(screen *ebiten.Image) {
	if r.b >= len(r.beats) {
		return
	}
	r.g.Draw(screen)

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	buf := make([]byte, 4*w*h)
	screen.ReadPixels(buf)
	img := &image.RGBA{Pix: buf, Stride: 4 * w, Rect: image.Rect(0, 0, w, h)}

	f, err := os.Create(filepath.Join(r.dir, fmt.Sprintf("frame-%06d.png", r.total)))
	if err != nil {
		r.t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		r.t.Fatal(err)
	}
	f.Close()

	r.total++
	r.f++
	if r.f >= r.beats[r.b].frames {
		r.t.Logf("錄完 %s(%d 格)", r.beats[r.b].name, r.f)
		r.b++
		r.f = 0
	}
}

func (r *promoRunner) Layout(int, int) (int, int) { return layout.ScreenW, layout.ScreenH }

func TestPromoFrames(t *testing.T) {
	outDir, assets := os.Getenv("PROMO_DIR"), os.Getenv("SHOT_ASSETS")
	if outDir == "" {
		t.Skip("沒有 PROMO_DIR —— 這不是驗收測試,平常不跑")
	}
	// ⚠ **預設讀版控裡的 `assets/`**,不是 workplace 的複本 ——
	// 兩份分家的時候,截圖看起來一切正常,而玩家 clone 下來跑的是另一批檔
	// (integration_test.go 的 TestCommittedAssetsAreComplete 是另一道防線)。
	if assets == "" {
		assets = assetsSource(t)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	g, err := loadStatic(assets, "", 20260817)
	if err != nil {
		t.Fatal("載入資產失敗:", err)
	}
	// ⚠ **`g.sound = nil` 不夠**,要連 audio context 都不要建(`SHARD_NOSOUND`)。
	// `loadStatic` 已經呼叫過 `initSound`,把 `g.sound` 設成 nil 只是不再播,
	// **context 還在**,而 oto 的 ALSA 錯誤是由**遊戲迴圈**丟回來的 ——
	// `RunGame` 直接以錯誤結束。
	//
	// ⚠ shot_test.go 用同一招卻沒事,因為它只拍六張、在錯誤浮上來之前就結束了。
	// **「短的跑得過」不代表「長的跑得過」**;這一支要跑近千格。
	if os.Getenv(NoSoundEnv) == "" {
		t.Skip("要設 " + NoSoundEnv + " —— 容器裡沒有音訊裝置,ALSA 的錯誤會中斷遊戲迴圈")
	}
	g.sound = nil

	beats := []beat{
		{"01-title", 75, nil, hold},
		{"02-menu", 60, func(g *Game) { g.openMainMenu() }, hold},
		{"03-world", 200, func(g *Game) {
			if err := g.loadParty(5); err != nil {
				t.Fatal("載入 PARTY #5 失敗:", err)
			}
			g.shell.mode = shellPlaying
		}, every(12, ebiten.KeyDown, ebiten.KeyDown, ebiten.KeyRight,
			ebiten.KeyRight, ebiten.KeyDown, ebiten.KeyLeft)},
		{"04-town", 140, func(g *Game) {
			if len(g.townSites) == 0 {
				return
			}
			s := g.townSites[0]
			g.party.X, g.party.Y = s.X, s.Y
			g.enterTown(s.X, s.Y)
		}, hold},
		{"05-maze", 180, func(g *Game) {
			g.town = nil
			if len(g.mazeData) == 0 {
				return
			}
			e := g.mazeData[0]
			g.party.X, g.party.Y = e.WorldX, e.WorldY
			g.enterMaze(e.WorldX, e.WorldY)
			g.overlay = ""
		}, every(15, ebiten.KeyUp, ebiten.KeyUp, ebiten.KeyRight, ebiten.KeyUp)},
		{"06-combat", 220, func(g *Game) {
			// ⚠ setup 冪等:ebiten 掉格時一次 Draw 之前可能跑好幾次 Update,
			// 而 startScriptedCombat 每次都重建 *Field 並重擲生命(shot_test.go)。
			if g.field == nil && !g.startScriptedCombat(533) {
				t.Log("⚠ 腳本戰鬥開不起來")
			}
			g.overlay = ""
		}, hold},
	}

	r := &promoRunner{g: g, beats: beats, dir: outDir, t: t}
	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	if err := ebiten.RunGame(r); err != nil {
		t.Fatal(err)
	}
	// 段落索引給剪接用:哪一格開始是哪一段,才知道字卡與配樂放哪裡。
	idx := ""
	for _, l := range r.labels {
		idx += l + "\n"
	}
	if err := os.WriteFile(filepath.Join(outDir, "beats.tsv"), []byte(idx), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("共 %d 格", r.total)
}
