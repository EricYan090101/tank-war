package game

import "math/rand"

// Difficulty is a terrain heuristic frozen before combat changes the map.
// Components are in [0,1], Score in [0,100].
type Difficulty struct {
	Score                                 int
	Pressure, Branching, Concealment, Ice float64
}
type mapPoint struct{ c, r int }

// GenerateRandom uses a private RNG: the same seed reproduces the terrain.
func (m *Map) GenerateRandom(seed int64) Difficulty {
	rng := rand.New(rand.NewSource(seed))
	m.Tiles = [MapRows][MapCols]int{}
	m.BrickMask = [MapRows][MapCols]uint8{}
	m.BaseFortified, m.baseFortifyTimer = false, 0
	for r := 0; r < 22; r += 2 {
		for c := 0; c < MapCols; c += 2 {
			tile := TileEmpty
			switch n := rng.Intn(100); {
			case n < 30:
				tile = TileBrick
			case n < 42:
				tile = TileSteel
			case n < 50:
				tile = TileWater
			case n < 65:
				tile = TileBush
			case n < 75:
				tile = TileIce
			}
			m.fillBlock(c, r, tile)
		}
	}
	// Carve full-width routes from all enemy spawns to the defense area.
	for _, start := range []int{0, 12, 24} {
		c := start
		m.fillBlock(c, 0, TileEmpty)
		for r := 0; r < 22; r += 2 {
			next := c + (rng.Intn(5)-2)*2
			if next < 0 {
				next = 0
			}
			if next > 24 {
				next = 24
			}
			for c != next {
				if c < next {
					c += 2
				} else {
					c -= 2
				}
				m.fillBlock(c, r, TileEmpty)
			}
			m.fillBlock(c, r+2, TileEmpty)
		}
	}
	for c := 0; c < MapCols; c += 2 {
		m.fillBlock(c, 20, TileEmpty)
	}
	m.connectWaterRegions(rng, 22)
	m.connectSmokeRegions(rng, 0, 20)
	// Both player spawns remain clear. The base has a destructible shell.
	for r := 23; r < 26; r++ {
		for c := 11; c <= 14; c++ {
			m.Tiles[r][c] = TileBrick
		}
	}
	m.fillBlock(12, 24, TileBase)
	m.ensureBrickMasks()
	return m.EvaluateDifficulty()
}
func (m *Map) fillBlock(c, r, tile int) {
	for y := r; y < r+2; y++ {
		for x := c; x < c+2; x++ {
			m.Tiles[y][x] = tile
		}
	}
}

// BFS uses the full 32px tank footprint at 16px grid positions.
func (m *Map) tankDistances(start mapPoint) map[mapPoint]int {
	dist := map[mapPoint]int{}
	if m.CheckTileCollision(float32(start.c*16), float32(start.r*16), 32, 32) {
		return dist
	}
	dist[start] = 0
	queue := []mapPoint{start}
	for head := 0; head < len(queue); head++ {
		p := queue[head]
		for _, d := range []mapPoint{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := mapPoint{p.c + d.c, p.r + d.r}
			if _, seen := dist[n]; seen || m.CheckTileCollision(float32(n.c*16), float32(n.r*16), 32, 32) {
				continue
			}
			dist[n] = dist[p] + 1
			queue = append(queue, n)
		}
	}
	return dist
}
func (m *Map) EvaluateDifficulty() Difficulty {
	d := Difficulty{}
	dist := m.tankDistances(mapPoint{12, 21})
	for _, c := range []int{0, 12, 24} {
		if steps, ok := dist[mapPoint{c, 0}]; ok {
			d.Pressure += 21.0 / float64(max(21, steps)) / 3
		}
	}
	branches, bushes, ice := 0, 0, 0
	for p := range dist {
		degree := 0
		for _, n := range []mapPoint{{p.c + 1, p.r}, {p.c - 1, p.r}, {p.c, p.r + 1}, {p.c, p.r - 1}} {
			if _, ok := dist[n]; ok {
				degree++
			}
		}
		if degree >= 3 {
			branches++
		}
		if m.Tiles[p.r][p.c] == TileBush {
			bushes++
		}
		if m.Tiles[p.r][p.c] == TileIce {
			ice++
		}
	}
	if len(dist) > 0 {
		d.Branching = float64(branches) / float64(len(dist))
		d.Concealment = float64(bushes) / float64(len(dist))
		d.Ice = float64(ice) / float64(len(dist))
	}
	d.Score = int(100*(.45*d.Pressure+.25*d.Branching+.20*d.Concealment+.10*d.Ice) + .5)
	return d
}

// Water regions are counted in 32px generation blocks (2x2 tile cells).
// Only existing solid brick/steel can be converted: carved routes, snow,
// smoke, spawn strips and other walkable cells are never blocked by growth.
func (m *Map) connectWaterRegions(rng *rand.Rand, rows int) {
	const minimumBlocks = 3
	seen := map[mapPoint]bool{}
	for r := 0; r < rows; r += 2 {
		for c := 0; c < MapCols; c += 2 {
			start := mapPoint{c, r}
			if seen[start] || m.Tiles[r][c] != TileWater {
				continue
			}
			region := m.waterBlockRegion(start, rows)
			for len(region) < minimumBlocks {
				candidates := []mapPoint{}
				queued := map[mapPoint]bool{}
				for _, p := range region {
					for _, d := range []mapPoint{{2, 0}, {-2, 0}, {0, 2}, {0, -2}} {
						q := mapPoint{p.c + d.c, p.r + d.r}
						if q.c < 0 || q.c+1 >= MapCols || q.r < 0 || q.r+1 >= rows || queued[q] {
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
				m.fillBlock(q.c, q.r, TileWater)
				// Re-scan because the new block can merge two existing water regions.
				region = m.waterBlockRegion(start, rows)
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
func (m *Map) waterBlockRegion(start mapPoint, rows int) []mapPoint {
	region := []mapPoint{start}
	seen := map[mapPoint]bool{start: true}
	for head := 0; head < len(region); head++ {
		p := region[head]
		for _, d := range []mapPoint{{2, 0}, {-2, 0}, {0, 2}, {0, -2}} {
			q := mapPoint{p.c + d.c, p.r + d.r}
			if q.c < 0 || q.c+1 >= MapCols || q.r < 0 || q.r+1 >= rows || seen[q] || m.Tiles[q.r][q.c] != TileWater {
				continue
			}
			seen[q] = true
			region = append(region, q)
		}
	}
	return region
}
