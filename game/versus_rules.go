package game

import "math/rand"

// VersusConfig describes a complete match; TeamSize is players per side.
type VersusConfig struct {
	TeamSize int
	BestOf   int
}

func (c VersusConfig) Valid() bool {
	return (c.TeamSize == 1 || c.TeamSize == 2) && (c.BestOf == 3 || c.BestOf == 5 || c.BestOf == 7)
}
func (c VersusConfig) Players() int    { return c.TeamSize * 2 }
func (c VersusConfig) WinsNeeded() int { return c.BestOf/2 + 1 }

type VersusMatch struct {
	Names       [4]string
	Teams       [4]int
	TeamsChosen bool
	Config      VersusConfig
	Scores      [2]int
	Round       int
	Winner      int // -1 is a drawn round.
	Timer       int
}

func (m *Map) GenerateVersus(seed int64) {
	rng := rand.New(rand.NewSource(seed))
	m.Tiles = [MapRows][MapCols]int{}
	m.BrickMask = [MapRows][MapCols]uint8{}
	m.BaseFortified = false
	m.baseFortifyTimer = 0
	for r := 2; r < 12; r += 2 {
		for c := 0; c < MapCols; c += 2 {
			tile := TileEmpty
			switch n := rng.Intn(100); {
			case n < 38:
				tile = TileBrick
			case n < 48:
				tile = TileSteel
			case n < 55:
				tile = TileWater
			case n < 65:
				tile = TileBush
			case n < 70:
				tile = TileIce
			}
			m.fillBlock(c, r, tile)
		}
	}
	// Full-width spawn strip, a central crossing, and three tank-wide routes.
	for _, c := range []int{0, 12, 24} {
		for r := 0; r < 12; r += 2 {
			m.fillBlock(c, r, TileEmpty)
		}
	}
	m.connectWaterRegions(rng, 12)
	m.connectSmokeRegions(rng, 2, 12)
	for r := 0; r < MapRows/2; r++ {
		m.Tiles[MapRows-1-r] = m.Tiles[r]
	}
	m.ensureBrickMasks()
}
func versusSpawn(id, teamSize int) (float32, float32, Direction) {
	x := float32(192)
	if teamSize == 2 {
		x = 128 + float32(id/2)*128
	}
	if id%2 == 0 {
		return x, 384, DirUp
	}
	return x, 0, DirDown
}
func (g *Game) startVersusRound() {
	g.Match.Round++
	g.MapSeed = g.RunSeed + int64(g.Match.Round)*7919
	if g.Map == nil {
		g.Map = NewMap()
	}
	g.Map.GenerateVersus(g.MapSeed)
	g.Tanks = nil
	g.Bullets = nil
	g.Enemies = nil
	g.Items = nil
	g.Explosions = nil
	seats := [2]int{}
	for id := 0; id < g.Match.Config.Players(); id++ {
		team := g.playerTeam(id)
		x, y, dir := versusSpawn(team+seats[team]*2, g.Match.Config.TeamSize)
		seats[team]++
		t := NewTank(x, y, id)
		t.Team = team
		t.MatchTank = true
		t.Dir = dir
		t.Lives = 1
		if g.Match.Round == 1 {
			t.ArmorShield = IronShieldHits
		}
		g.Tanks = append(g.Tanks, t)
	}
	g.Match.Winner = -1
	g.Match.Timer = 120
	g.State = StateVersusCountdown
}
func (g *Game) simulateVersus(inputs map[int]VersusInput) {
	for _, t := range g.Tanks {
		if !t.Active {
			continue
		}
		if t.FireCooldown > 0 {
			t.FireCooldown--
		}
		if t.HitFlash > 0 {
			t.HitFlash--
		}
		input := inputs[t.PlayerID]
		t.applyMovement(g.Map, nil, g.Tanks, input.Dir, input.Moving)
		count := 0
		for _, b := range g.Bullets {
			if b.Active && b.PlayerID == t.PlayerID {
				count++
			}
		}
		if input.Fire && t.CanFire(1, count) {
			g.Bullets = append(g.Bullets, t.Fire())
		}
	}
	for _, b := range g.Bullets {
		if !b.Active {
			continue
		}
		b.Update(g.Map)
		if !b.Active {
			continue
		}
		for _, other := range g.Bullets {
			if other != b && other.Active && g.playerTeam(other.PlayerID) != g.playerTeam(b.PlayerID) && rectsOverlap(b.X, b.Y, 4, 4, other.X, other.Y, 4, 4) {
				b.Active = false
				other.Active = false
				break
			}
		}
		if !b.Active {
			continue
		}
		for _, t := range g.Tanks {
			if !t.Active || t.Team == g.playerTeam(b.PlayerID) {
				continue
			}
			if rectsOverlap(b.X, b.Y, 4, 4, t.X, t.Y, t.Size, t.Size) {
				b.Active = false
				if absorbIronHit(&t.ArmorShield) {
					t.HitFlash = 5
					break
				}
				t.Hits++
				t.HitFlash = 12
				if t.Hits >= 3 {
					t.Active = false
					t.Lives = 0
					g.Explosions = append(g.Explosions, NewExplosion(t.X+16, t.Y+16, true))
				}
				break
			}
		}
	}
	alive := g.Bullets[:0]
	for _, b := range g.Bullets {
		if b.Active {
			alive = append(alive, b)
		}
	}
	g.Bullets = alive
	g.updateExplosions()
	g.checkVersusRound()
}
func (g *Game) checkVersusRound() {
	if g.State != StateVersusPlaying {
		return
	}
	alive := [2]int{}
	for _, t := range g.Tanks {
		if t.Active {
			alive[t.Team]++
		}
	}
	if alive[0] > 0 && alive[1] > 0 {
		return
	}
	g.Match.Winner = -1
	if alive[0] > 0 {
		g.Match.Winner = 0
	} else if alive[1] > 0 {
		g.Match.Winner = 1
	}
	if g.Match.Winner >= 0 {
		g.Match.Scores[g.Match.Winner]++
	}
	g.Bullets = nil
	g.Match.Timer = 180
	g.State = StateVersusRoundEnd
	if g.Match.Winner >= 0 && g.Match.Scores[g.Match.Winner] >= g.Match.Config.WinsNeeded() {
		g.State = StateVersusMatchEnd
	}
}

func (g *Game) playerTeam(id int) int {
	if g.Match.TeamsChosen && id >= 0 && id < len(g.Match.Teams) {
		return g.Match.Teams[id]
	}
	return id % 2
}
