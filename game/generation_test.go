package game

import "testing"

func TestRandomMapsPlayableAndReproducible(t *testing.T) {
	unique := map[[MapRows][MapCols]int]bool{}
	for seed := int64(0); seed < 500; seed++ {
		m, n := &Map{}, &Map{}
		d := m.GenerateRandom(seed)
		if d != n.GenerateRandom(seed) || m.Tiles != n.Tiles || m.BrickMask != n.BrickMask {
			t.Fatalf("seed %d is not reproducible", seed)
		}
		if d.Score < 0 || d.Score > 100 {
			t.Fatalf("invalid difficulty: %+v", d)
		}
		unique[m.Tiles] = true
		reachable := m.tankDistances(mapPoint{PlayerSpawnX / 16, PlayerSpawnY / 16})
		for _, p := range []mapPoint{{0, 0}, {12, 0}, {24, 0}, {SecondSpawnX / 16, SecondSpawnY / 16}, {12, 21}} {
			if _, ok := reachable[p]; !ok {
				t.Fatalf("seed %d: unreachable spawn/defense point %+v", seed, p)
			}
		}
		bases := 0
		for r := 0; r < MapRows; r++ {
			for c := 0; c < MapCols; c++ {
				if m.Tiles[r][c] == TileBase {
					bases++
				}
				if (m.Tiles[r][c] == TileBrick) != (m.BrickMask[r][c] == BrickFull) {
					t.Fatalf("seed %d: invalid brick mask", seed)
				}
			}
		}
		if bases != 4 {
			t.Fatalf("seed %d: base has %d tiles", seed, bases)
		}
	}
	if len(unique) < 490 {
		t.Fatalf("too few unique maps: %d", len(unique))
	}
}

func TestDifficultyRespondsToAttackRoutes(t *testing.T) {
	m := &Map{}
	open := m.EvaluateDifficulty()
	for c := 0; c < MapCols; c++ {
		m.Tiles[10][c] = TileSteel
	}
	blocked := m.EvaluateDifficulty()
	if blocked.Pressure >= open.Pressure || blocked.Score >= open.Score {
		t.Fatalf("blocked approach must reduce pressure: open=%+v blocked=%+v", open, blocked)
	}
}
