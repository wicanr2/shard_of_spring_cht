package combat

import "testing"

// 戰場地形層的驗收(docs/re/227 §2)。
//
// ⚠ 這一組驗的是**攤開的幾何**:一格地圖 → 5×5 格、3×3 → 15×15、
// 以隊伍基準 (13,13) 為中心。原版的怪物出場錨點是 6…20
// (docs/re/186 §1),兩者必須疊得起來 —— 對不上就是攤開的原點錯了。

func TestTerrainSpreadsThreeByThreeIntoFifteen(t *testing.T) {
	var f Field
	// 九格各給一個好認的值(用格值本身當標記,不必是合法地形)
	cells := []int{11, 12, 13, 21, 22, 23, 31, 32, 33}
	f.SetTerrain(cells)

	// 每一格地圖都該攤成 5×5,而且四個角落都是同一個值
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			want := cells[i*3+j]
			y0, x0 := TerrainOrigin+i*TerrainBlock, TerrainOrigin+j*TerrainBlock
			for _, c := range [][2]int{{y0, x0}, {y0 + 4, x0}, {y0, x0 + 4}, {y0 + 4, x0 + 4}} {
				if got := f.Terrain[c[0]][c[1]]; got != want {
					t.Errorf("(%d,%d) 是 %d,應為 %d(來源格 %d,%d)", c[0], c[1], got, want, i, j)
				}
			}
		}
	}
	// 15×15 那一塊之外不該被碰到
	for _, c := range [][2]int{{TerrainOrigin - 1, 13}, {TerrainOrigin + 15, 13}, {13, TerrainOrigin - 1}} {
		if got := f.Terrain[c[0]][c[1]]; got != 0 {
			t.Errorf("(%d,%d) 在 15×15 之外卻是 %d", c[0], c[1], got)
		}
	}
	// 隊伍的基準格落在正中間那一格地圖上
	if f.Terrain[PartyBaseY][PartyBaseX] != cells[4] {
		t.Errorf("隊伍基準格 (%d,%d) 的地形是 %d,應該是中央那一格 %d",
			PartyBaseX, PartyBaseY, f.Terrain[PartyBaseY][PartyBaseX], cells[4])
	}
}

// 長度不對就整片不填 —— 地城戰鬥的呼叫端會傳 nil。
func TestTerrainIgnoresWrongLength(t *testing.T) {
	var f Field
	f.SetTerrain(nil)
	f.SetTerrain([]int{1, 2, 3})
	for y := range f.Terrain {
		for x := range f.Terrain[y] {
			if f.Terrain[y][x] != 0 {
				t.Fatalf("(%d,%d) 被填了 %d", y, x, f.Terrain[y][x])
			}
		}
	}
}
