package game

import (
	"math/rand"
	"testing"
)

func TestGeneratedWaterHasAtLeastThreeBlocks(t *testing.T) {
	for _, versus := range []bool{false, true} {
		waterMaps := 0
		for seed := int64(0); seed < 500; seed++ {
			m := &Map{}
			if versus {
				m.GenerateVersus(seed)
			} else {
				m.GenerateRandom(seed)
			}
			visited := [MapRows][MapCols]bool{}
			found := false
			// Independent flood fill over 16px cells verifies the actual final map.
			for r := 0; r < MapRows; r++ {
				for c := 0; c < MapCols; c++ {
					if visited[r][c] || m.Tiles[r][c] != TileWater {
						continue
					}
					found = true
					queue := []mapPoint{{c, r}}
					visited[r][c] = true
					for head := 0; head < len(queue); head++ {
						p := queue[head]
						for _, d := range []mapPoint{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
							q := mapPoint{p.c + d.c, p.r + d.r}
							if q.c >= 0 && q.c < MapCols && q.r >= 0 && q.r < MapRows && !visited[q.r][q.c] && m.Tiles[q.r][q.c] == TileWater {
								visited[q.r][q.c] = true
								queue = append(queue, q)
							}
						}
					}
					if len(queue) < 12 {
						t.Fatalf("versus=%v seed=%d: isolated water region of %d cells", versus, seed, len(queue))
					}
				}
			}
			if found {
				waterMaps++
			}
		}
		if waterMaps < 100 {
			t.Fatalf("versus=%v: water disappeared from most maps (%d/500)", versus, waterMaps)
		}
		t.Logf("versus=%v: %d/500 maps contain connected water", versus, waterMaps)
	}
}
func TestWaterGrowthPreservesRoutesAndRejectsDiagonalNeighbors(t *testing.T) {
	m := &Map{}
	m.fillBlock(2, 2, TileWater)
	m.fillBlock(4, 4, TileWater)
	m.connectWaterRegions(rand.New(rand.NewSource(1)), 12)
	if m.Tiles[2][2] != TileEmpty || m.Tiles[4][4] != TileEmpty {
		t.Fatal("diagonal isolated pools retained")
	}
	m.fillBlock(2, 2, TileWater)
	m.fillBlock(4, 2, TileBrick)
	m.fillBlock(6, 2, TileSteel)
	m.fillBlock(2, 4, TileBush)
	m.fillBlock(4, 4, TileIce)
	before := m.Tiles
	m.connectWaterRegions(rand.New(rand.NewSource(3)), 12)
	m.ensureBrickMasks()
	for r := 0; r < MapRows; r++ {
		for c := 0; c < MapCols; c++ {
			if before[r][c] == TileEmpty || before[r][c] == TileBush || before[r][c] == TileIce {
				if m.Tiles[r][c] != before[r][c] {
					t.Fatal("water blocked traversable terrain")
				}
			}
		}
	}
	if m.Tiles[2][2] != TileWater || m.Tiles[2][4] != TileWater || m.Tiles[2][6] != TileWater {
		t.Fatal("river did not grow to three blocks")
	}
	if m.BrickMask[2][4] != 0 {
		t.Fatal("water retained brick damage mask")
	}
}
