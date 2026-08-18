package main

// 產生 README 用的畫面截圖。**這不是驗收測試** —— 只在 SHOT_DIR 指定時跑。
//
//	SHOT_DIR=/out go test -run TestShots -count=1
//
// ⚠ **一定要跑 ebiten.RunGame**。先前試過「離屏 NewImage + g.Draw + png.Encode」,
// 結果是 `panic: ui: ReadPixels cannot be called before the game starts` ——
// 讀像素需要遊戲迴圈已經啟動,離屏影像也不例外。所以這裡包一層 shotRunner:
// Update() 擺好要拍的狀態、Draw() 拍完立刻寫檔,拍完最後一張回 Termination。
import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/layout"
)

type shot struct {
	name  string
	setup func(*Game)
}

type shotRunner struct {
	g     *Game
	shots []shot
	dir   string
	i     int
	t     *testing.T
}

func (r *shotRunner) Update() error {
	if r.i >= len(r.shots) {
		return ebiten.Termination
	}
	r.shots[r.i].setup(r.g)
	return nil
}

func (r *shotRunner) Draw(screen *ebiten.Image) {
	if r.i >= len(r.shots) {
		return
	}
	r.g.Draw(screen)

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	buf := make([]byte, 4*w*h)
	screen.ReadPixels(buf)
	img := &image.RGBA{Pix: buf, Stride: 4 * w, Rect: image.Rect(0, 0, w, h)}

	f, err := os.Create(filepath.Join(r.dir, r.shots[r.i].name+".png"))
	if err != nil {
		r.t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		r.t.Fatal(err)
	}
	r.t.Log("拍到", r.shots[r.i].name)
	r.i++
}

func (r *shotRunner) Layout(int, int) (int, int) { return layout.ScreenW, layout.ScreenH }

func TestShots(t *testing.T) {
	outDir, assets := os.Getenv("SHOT_DIR"), os.Getenv("SHOT_ASSETS")
	if outDir == "" {
		t.Skip("沒有 SHOT_DIR —— 這不是驗收測試,平常不跑")
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
	g, err := loadStatic(assets, "", 20260815)
	if err != nil {
		t.Fatal("載入資產失敗:", err)
	}
	// ⚠ **一定要關掉聲音**。容器裡沒有音訊裝置,而 ebiten 的 audio context
	// 會把 oto 的 ALSA 錯誤丟回遊戲迴圈 —— RunGame 直接以錯誤結束,
	// 只拍得到第一張。docs/spec/13 說「沒有音訊裝置時只記警告」,
	// 那是指 initSound 的 recover;**播放時的錯誤是另一條路,它會中斷迴圈**。
	g.sound = nil

	shots := []shot{
		{"01-title", func(g *Game) {}},
		{"02-menu", func(g *Game) { g.openMainMenu() }},
		{"03-world", func(g *Game) {
			if err := g.loadParty(5); err != nil {
				t.Fatal("載入 PARTY #5 失敗:", err)
			}
			g.shell.mode = shellPlaying
		}},
		{"04-town", func(g *Game) {
			if len(g.townSites) == 0 {
				return
			}
			s := g.townSites[0]
			g.party.X, g.party.Y = s.X, s.Y
			g.enterTown(s.X, s.Y)
		}},
		{"05-maze", func(g *Game) {
			g.town = nil
			if len(g.mazeData) == 0 {
				return
			}
			e := g.mazeData[0]
			g.party.X, g.party.Y = e.WorldX, e.WorldY
			g.enterMaze(e.WorldX, e.WorldY)
			g.overlay = ""
		}},
		{"07-camp", func(g *Game) {
			// 野外紮營:地圖留著、隊伍那一格是帳篷、選單在右下角
			// (原版 workplace/qa/k0-camp.png 的樣子)。
			g.level, g.town = nil, nil
			g.party.X, g.party.Y = 52, 60
			g.makeCamp(true)
		}},
		{"06-combat", func(g *Game) {
			// ⚠ **setup 必須冪等。** ebiten 可能在一次 Draw 之前跑好幾次
			// Update(掉格時就會),而 startScriptedCombat 每次都重建 *Field
			// 並重擲怪物生命 —— 拍到的數字因此會隨機器忙碌程度變動,
			// 兩次跑出來的 PNG 不一樣。這不是遊戲的不確定性,是拍照工具的。
			g.town = nil // ⚠ 前一張是營地 —— 狀態要收乾淨再開戰
			if g.field == nil && !g.startScriptedCombat(533) {
				t.Log("⚠ 腳本戰鬥開不起來")
			}
			g.overlay = ""
		}},
	}

	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	if err := ebiten.RunGame(&shotRunner{g: g, shots: shots, dir: outDir, t: t}); err != nil {
		t.Fatal(err)
	}
}
