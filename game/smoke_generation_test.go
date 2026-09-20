package game

import (
	"testing"
)

func TestGeneratedSmokeHasAtLeastFourBlocks(t *testing.T) {
	for _, versus := range []bool{false, true} {
		smokeMaps := 0
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
					if visited[r][c] || m.Tiles[r][c] != TileBush {
						continue
					}
					found = true
					queue := []mapPoint{{c, r}}
					visited[r][c] = true
					for head := 0; head < len(queue); head++ {
						p := queue[head]
						for _, d := range []mapPoint{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
							q := mapPoint{p.c + d.c, p.r + d.r}
							if q.c >= 0 && q.c < MapCols && q.r >= 0 && q.r < MapRows && !visited[q.r][q.c] && m.Tiles[q.r][q.c] == TileBush {
								visited[q.r][q.c] = true
								queue = append(queue, q)
							}
						}
					}
					if len(queue) < 16 {
						t.Fatalf("versus=%v seed=%d: isolated smoke region of %d cells", versus, seed, len(queue))
					}
				}
			}
			if found {
				smokeMaps++
			}
		}
		if smokeMaps < 100 {
			t.Fatalf("versus=%v: smoke disappeared from most maps (%d/500)", versus, smokeMaps)
		}
		t.Logf("versus=%v: %d/500 maps contain connected smoke", versus, smokeMaps)
	}
}
