package game

import "math/rand"

// Grow smoke over existing walls only, preserving routes, ice and water.
// Components may have any orthogonally connected shape.
func (m *Map) connectSmokeRegions(rng *rand.Rand, firstRow, rows int) {
	const minimumBlocks = 4
	seen := map[mapPoint]bool{}
	for r := firstRow; r < rows; r += 2 {
		for c := 0; c < MapCols; c += 2 {
			start := mapPoint{c, r}
			if seen[start] || m.Tiles[r][c] != TileBush {
				continue
			}
			region := m.smokeBlockRegion(start, firstRow, rows)
			for len(region) < minimumBlocks {
				candidates := []mapPoint{}
				queued := map[mapPoint]bool{}
				for _, p := range region {
					for _, d := range []mapPoint{{2, 0}, {-2, 0}, {0, 2}, {0, -2}} {
						q := mapPoint{p.c + d.c, p.r + d.r}
						if q.c < 0 || q.c+1 >= MapCols || q.r < firstRow || q.r+1 >= rows || queued[q] {
							continue
						}
						solid := true
						for y := q.r; y < q.r+2; y++ {
							for x := q.c; x < q.c+2; x++ {
								tile := m.Tiles[y][x]
								if tile != TileBrick && tile != TileSteel {
									solid = false
								}
							}
						}
						if solid {
							candidates = append(candidates, q)
							queued[q] = true
						}
					}
				}
				if len(candidates) == 0 {
					break
				}
				q := candidates[rng.Intn(len(candidates))]
				m.fillBlock(q.c, q.r, TileBush)
				// Re-scan because the new block can merge two existing smoke regions.
				region = m.smokeBlockRegion(start, firstRow, rows)
			}
			for _, p := range region {
				seen[p] = true
				if len(region) < minimumBlocks {
					m.fillBlock(p.c, p.r, TileEmpty)
				}
			}
		}
	}
}
func (m *Map) smokeBlockRegion(start mapPoint, firstRow, rows int) []mapPoint {
	region := []mapPoint{start}
	seen := map[mapPoint]bool{start: true}
	for head := 0; head < len(region); head++ {
		p := region[head]
		for _, d := range []mapPoint{{2, 0}, {-2, 0}, {0, 2}, {0, -2}} {
			q := mapPoint{p.c + d.c, p.r + d.r}
			if q.c < 0 || q.c+1 >= MapCols || q.r < firstRow || q.r+1 >= rows || seen[q] || m.Tiles[q.r][q.c] != TileBush {
				continue
			}
			seen[q] = true
			region = append(region, q)
		}
	}
	return region
}
