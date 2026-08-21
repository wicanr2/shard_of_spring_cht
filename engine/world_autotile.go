package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"shardofspring/internal/world"
)

func modernAtlasTile(atlas *ebiten.Image, x, y int) *ebiten.Image {
	if atlas == nil {
		return nil
	}
	const tile, cols = 68, 4
	sx, sy := ((x%cols)+cols)%cols, ((y%cols)+cols)%cols
	return atlas.SubImage(image.Rect(sx*tile, sy*tile, (sx+1)*tile, (sy+1)*tile)).(*ebiten.Image)
}

type modernTerrain uint8

const (
	modernTerrainOther modernTerrain = iota
	modernTerrainGrass
	modernTerrainForest
	modernTerrainOcean
	modernTerrainCoast
)

func classifyModernTerrain(v int) modernTerrain {
	switch v {
	case 1, 2:
		return modernTerrainGrass
	case 3, 4:
		return modernTerrainForest
	case 5, 6, 11:
		return modernTerrainOcean
	case 15, 16, 17, 18, 35, 36, 37, 38:
		return modernTerrainCoast
	default:
		return modernTerrainOther
	}
}

// modernNeighborMask 的 bit 順序固定為北、東、南、西 = 1、2、4、8。
// map.At 對界外回 0，因此地圖外不會被誤當成自然地形延伸。
func modernNeighborMask(m *world.Map, x, y int, want modernTerrain) int {
	mask := 0
	for _, d := range []struct{ dx, dy, bit int }{
		{0, -1, 1}, {1, 0, 2}, {0, 1, 4}, {-1, 0, 8},
	} {
		if classifyModernTerrain(m.At(x+d.dx, y+d.dy)) == want {
			mask |= d.bit
		}
	}
	return mask
}

func (g *Game) modernWorldTile(v, x, y int) *ebiten.Image {
	switch classifyModernTerrain(v) {
	case modernTerrainGrass:
		return modernAtlasTile(g.modernWorldAuto.grass[v&1], x, y)
	case modernTerrainOcean:
		return modernAtlasTile(g.modernWorldAuto.ocean[(x+y+v)&1], x, y)
	case modernTerrainForest:
		return modernAtlasTile(g.modernWorldAuto.forest[modernNeighborMask(g.world, x, y, modernTerrainForest)], x, y)
	case modernTerrainCoast:
		mask := modernNeighborMask(g.world, x, y, modernTerrainOcean)
		// 對角海岸可能沒有四向海洋鄰格(mask 0)，仍須使用現代草地端材質；
		// 退回舊 tNN 會在連續世界中露出一格完全不同的拼貼圖。
		return modernAtlasTile(g.modernWorldAuto.coast[mask], x, y)
	}
	return g.modernTiles[v]
}
