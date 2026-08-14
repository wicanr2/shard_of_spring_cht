// Shard of Spring remake。
//
// M2:世界地圖場景(docs/spec/05-world-scene.md)。
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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"shardofspring/internal/layout"
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
	world *world.Map
	party world.State
	tiles map[int]*ebiten.Image // 地形值 → 圖;沒有來源的值不在裡面
	// noSrc 記下畫面上出現過的未解地形值,顯示在提示列。
	// 讓未解項目在**執行時**也看得見,不是只在文件裡。
	noSrc map[int]bool
}

func (g *Game) Update() error {
	for key, dir := range map[ebiten.Key]world.Facing{
		ebiten.KeyUp: world.North, ebiten.KeyRight: world.East,
		ebiten.KeyDown: world.South, ebiten.KeyLeft: world.West,
		ebiten.KeyDigit1: world.North, ebiten.KeyDigit2: world.East,
		ebiten.KeyDigit3: world.South, ebiten.KeyDigit4: world.West,
	} {
		if inpututil.IsKeyJustPressed(key) {
			g.party.Step(dir, g.world)
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(cgaBlack)

	// 9×9 視野,隊伍固定在正中央(docs/spec/05 §3、§4)。
	const half = layout.ViewTiles / 2
	for vy := 0; vy < layout.ViewTiles; vy++ {
		for vx := 0; vx < layout.ViewTiles; vx++ {
			mx, my := g.party.X-half+vx, g.party.Y-half+vy
			v := g.world.At(mx, my)
			px := float32(layout.View.X + vx*layout.TileDst)
			py := float32(layout.View.Y + vy*layout.TileDst)

			if img, ok := g.tiles[v]; ok {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(layout.ArtScale, layout.ArtScale)
				op.GeoM.Translate(float64(px), float64(py))
				// 最近鄰 —— 整數倍放大不該有插值(docs/spec/04 §1)。
				op.Filter = ebiten.FilterNearest
				screen.DrawImage(img, op)
			} else {
				g.noSrc[v] = true
				vector.DrawFilledRect(screen, px, py,
					layout.TileDst, layout.TileDst, missing, false)
			}
		}
	}

	// 隊伍所在格的框
	c := float32(layout.View.X + half*layout.TileDst)
	r := float32(layout.View.Y + half*layout.TileDst)
	vector.StrokeRect(screen, c, r, layout.TileDst, layout.TileDst, 3, cgaWhite, false)

	frame := func(rc layout.Rect) {
		vector.StrokeRect(screen, float32(rc.X), float32(rc.Y),
			float32(rc.W), float32(rc.H), 2, cgaWhite, false)
	}
	frame(layout.View)
	frame(layout.Party)
	frame(layout.Message)
	frame(layout.Prompt)

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

func main() {
	assets := flag.String("assets", "assets", "資產資料夾(由 cmd/convert 產生)")
	x := flag.Int("x", 50, "起始 x")
	y := flag.Int("y", 60, "起始 y")
	flag.Parse()

	g, err := load(*assets, *x, *y)
	if err != nil {
		fmt.Fprintln(os.Stderr, "載入失敗:", err)
		fmt.Fprintln(os.Stderr, "請先跑:go run ./cmd/convert -in <原版> -out assets")
		os.Exit(1)
	}

	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	ebiten.SetWindowTitle("春之石 Shard of Spring")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func load(dir string, x, y int) (*Game, error) {
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

	tiles := map[int]*ebiten.Image{}
	for v := 0; v <= 38; v++ {
		p := filepath.Join(dir, "gfx", "world", fmt.Sprintf("t%02d.png", v))
		f, err := os.Open(p)
		if err != nil {
			continue // 沒有來源的地形值,執行期畫佔位符
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s:%w", p, err)
		}
		tiles[v] = ebiten.NewImageFromImage(img)
	}

	return &Game{
		world: &world.Map{Cells: wm.Cells},
		party: world.State{
			X: x, Y: y, Facing: world.North,
			Clock:     world.Clock{Sub: 1, Hour: 4, Day: 1, Month: 1},
			Encounter: 12,
		},
		tiles: tiles,
		noSrc: map[int]bool{},
	}, nil
}
