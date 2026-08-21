package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/world"
)

func TestClassifyModernTerrain(t *testing.T) {
	cases := map[modernTerrain][]int{
		modernTerrainGrass:  {1, 2},
		modernTerrainForest: {3, 4},
		modernTerrainOcean:  {5, 6, 11},
		modernTerrainCoast:  {15, 16, 17, 18, 35, 36, 37, 38},
	}
	for want, values := range cases {
		for _, v := range values {
			if got := classifyModernTerrain(v); got != want {
				t.Errorf("地形 %d 分類 = %d，want %d", v, got, want)
			}
		}
	}
	if got := classifyModernTerrain(30); got != modernTerrainOther {
		t.Fatalf("城鎮不得自動拼接，分類 = %d", got)
	}
}

func TestModernNeighborMaskNESW(t *testing.T) {
	m := &world.Map{Cells: make([]int, world.W*world.H)}
	x, y := 50, 50
	set := func(x, y, v int) { m.Cells[x*world.H+y] = v }
	set(x, y-1, 11)
	set(x+1, y, 5)
	set(x, y+1, 3)
	set(x-1, y, 6)
	if got := modernNeighborMask(m, x, y, modernTerrainOcean); got != 1|2|8 {
		t.Fatalf("海洋鄰接遮罩 = %04b，want 1011", got)
	}
}

func TestCoastWithoutCardinalOceanStillUsesAutoTile(t *testing.T) {
	g := &Game{
		world: &world.Map{Cells: make([]int, world.W*world.H)},
		modernWorldAuto: modernWorldAuto{coast: map[int]*ebiten.Image{
			0: ebiten.NewImage(272, 272),
		}},
	}
	if got := g.modernWorldTile(15, 50, 50); got == nil {
		t.Fatal("四向遮罩為 0 的海岸不得退回舊單格資產")
	}
}
