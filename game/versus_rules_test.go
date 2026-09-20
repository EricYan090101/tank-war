package game

import "testing"

func TestVersusMapsMirrorAndConnect(t *testing.T) {
	for seed := int64(0); seed < 100; seed++ {
		m, n := &Map{}, &Map{}
		m.GenerateVersus(seed)
		n.GenerateVersus(seed)
		if m.Tiles != n.Tiles {
			t.Fatal("same seed changed terrain")
		}
		for r := 0; r < MapRows; r++ {
			for c := 0; c < MapCols; c++ {
				if m.Tiles[r][c] != m.Tiles[MapRows-1-r][c] || m.BrickMask[r][c] != m.BrickMask[MapRows-1-r][c] {
					t.Fatalf("map %d not mirrored", seed)
				}
				if m.Tiles[r][c] == TileBase {
					t.Fatal("versus map contains arcade base")
				}
			}
		}
		reach := m.tankDistances(mapPoint{12, 0})
		for _, size := range []int{1, 2} {
			for id := 0; id < size*2; id++ {
				x, y, _ := versusSpawn(id, size)
				if _, ok := reach[mapPoint{int(x) / 16, int(y) / 16}]; !ok {
					t.Fatalf("spawn disconnected: seed=%d id=%d", seed, id)
				}
			}
		}
	}
}
func versusTestGame(size, best int) *Game {
	g := &Game{Map: &Map{}, Match: VersusMatch{Config: VersusConfig{size, best}}, RunSeed: 42}
	g.startVersusRound()
	g.State = StateVersusPlaying
	// These tests isolate hull hits; starting armor is covered separately.
	for _, tank := range g.Tanks {
		tank.ArmorShield = 0
	}
	// Open the terrain for predictable hit tests.
	g.Map = &Map{}
	return g
}
func hitVersusTank(g *Game, target, shooter int) {
	tank := g.Tanks[target]
	b := NewBullet(tank.X+14, tank.Y+4, DirDown, OwnerPlayer, 1)
	b.PlayerID = shooter
	g.Bullets = append(g.Bullets, b)
	g.simulateVersus(nil)
}
func TestThreeHitsAndNoFriendlyFire(t *testing.T) {
	g := versusTestGame(2, 3)
	hitVersusTank(g, 0, 2)
	if g.Tanks[0].Hits != 0 {
		t.Fatal("teammate inflicted damage")
	}
	g.Bullets = nil
	for i := 1; i <= 3; i++ {
		hitVersusTank(g, 0, 1)
		if g.Tanks[0].Hits != i || g.Tanks[0].Active != (i < 3) {
			t.Fatalf("incorrect elimination at hit %d", i)
		}
	}
	if g.State != StateVersusPlaying {
		t.Fatal("2v2 ended while teammate alive")
	}
	for i := 0; i < 3; i++ {
		hitVersusTank(g, 2, 1)
	}
	if g.State != StateVersusRoundEnd || g.Match.Scores != [2]int{0, 1} {
		t.Fatal("team elimination did not award one round")
	}
	g.checkVersusRound()
	if g.Match.Scores[1] != 1 {
		t.Fatal("round counted twice")
	}
	g.startVersusRound()
	for _, tank := range g.Tanks {
		if tank.Hits != 0 || !tank.Active {
			t.Fatal("new round did not revive all players")
		}
	}
	if g.Match.Scores[1] != 1 {
		t.Fatal("new round lost match score")
	}
}
func TestVersusBestOfAndDraw(t *testing.T) {
	for _, best := range []int{3, 5, 7} {
		g := versusTestGame(1, best)
		for win := 1; win <= best/2+1; win++ {
			g.Tanks[1].Active = false
			g.checkVersusRound()
			if g.Match.Scores[0] != win {
				t.Fatal("score incorrect")
			}
			if win < best/2+1 {
				if g.State != StateVersusRoundEnd {
					t.Fatal("series ended early")
				}
				g.startVersusRound()
				g.State = StateVersusPlaying
			} else if g.State != StateVersusMatchEnd {
				t.Fatal("series failed to finish")
			}
		}
	}
	g := versusTestGame(1, 3)
	for _, tank := range g.Tanks {
		tank.Active = false
	}
	g.checkVersusRound()
	if g.Match.Scores != [2]int{} || g.Match.Winner != -1 || g.State != StateVersusRoundEnd {
		t.Fatal("draw awarded victory")
	}
}
func TestVersusSnapshotIsIndependent(t *testing.T) {
	host := versusTestGame(2, 5)
	host.LobbyMembers = []int{0, 1, 2, 3}
	snap := host.versusSnapshot()
	original := snap.Tanks[0].X
	host.Tanks[0].X = 77
	host.Map.Tiles[0][0] = TileSteel
	if snap.Tanks[0].X != original || snap.Tiles[0][0] == TileSteel {
		t.Fatal("snapshot shared mutable game memory")
	}
	client := &Game{Map: &Map{}}
	client.applyVersusSnapshot(snap)
	if client.Match != snap.Match || client.Tanks[0].X != original || client.Map.Tiles != snap.Tiles {
		t.Fatal("client did not apply host state")
	}
}

func TestChosenTeamsControlSpawnAndFriendlyFire(t *testing.T) {
	g := &Game{Map: &Map{}, Match: VersusMatch{Config: VersusConfig{2, 3}, TeamsChosen: true, Teams: [4]int{1, 1, 0, 0}}}
	g.startVersusRound()
	g.State = StateVersusPlaying
	g.Map = &Map{}
	for id, tank := range g.Tanks {
		tank.ArmorShield = 0
		wantY := float32(0)
		if id >= 2 {
			wantY = 384
		}
		if tank.Y != wantY || tank.Team != g.Match.Teams[id] {
			t.Fatal("spawn ignored selected team")
		}
	}
	if g.Tanks[0].X == g.Tanks[1].X || g.Tanks[2].X == g.Tanks[3].X {
		t.Fatal("teammates share spawn")
	}
	hitVersusTank(g, 0, 1)
	if g.Tanks[0].Hits != 0 {
		t.Fatal("chosen teammate inflicted damage")
	}
	g.Bullets = nil
	hitVersusTank(g, 0, 2)
	if g.Tanks[0].Hits != 1 {
		t.Fatal("opponent with same ID parity was treated as teammate")
	}
}
func TestCountdownClosesLocalTeamChooser(t *testing.T) {
	g := &Game{Map: &Map{}, ChoosingTeam: true, RequestedTeam: 1}
	g.applyVersusSnapshot(&versusSnapshot{State: StateVersusCountdown})
	if g.ChoosingTeam || g.RequestedTeam != -1 {
		t.Fatal("team chooser suppressed controls after host started")
	}
}
