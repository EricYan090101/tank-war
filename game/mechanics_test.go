package game

import (
	"math"
	"testing"
)

func TestUncappedSpawnsCycleAndWait(t *testing.T) {
	g := &Game{Map: &Map{}, Menu: &MenuUI{}, TotalEnemiesPerStage: 20}
	for i := 0; i < 20; i++ {
		if !g.spawnEnemy() {
			t.Fatalf("enemy %d was capped", i+1)
		}
		e := g.Enemies[i]
		want := [3]float32{0, 192, 384}
		if e.X != want[i%3] || e.Dir != DirDown {
			t.Fatal("wrong spawn order/direction")
		}
		e.Y = 64 // Vacate the spawn while keeping every enemy alive.
	}
	if g.activeEnemyCount() != 20 || g.spawnEnemy() {
		t.Fatal("wave total incorrect")
	}
	g = &Game{Map: &Map{}, Menu: &MenuUI{}, TotalEnemiesPerStage: 20, Tanks: []*Tank{NewTank(0, 0, 0)}}
	if g.spawnEnemy() || g.SpawnedEnemyCount != 0 || g.NextEnemySpawn != 0 {
		t.Fatal("occupied spawn must wait")
	}
	g.Tanks = nil
	for i := 0; i < g.spawnInterval()-1; i++ {
		g.maybeSpawnEnemy()
	}
	if len(g.Enemies) != 0 {
		t.Fatal("spawned too early")
	}
	g.maybeSpawnEnemy()
	if len(g.Enemies) != 1 {
		t.Fatal("did not spawn on schedule")
	}
}

func TestSpawnOccupiedByEnemy(t *testing.T) {
	g := &Game{Map: &Map{}, Menu: &MenuUI{}, TotalEnemiesPerStage: 20}
	for i := 0; i < 3; i++ {
		if !g.spawnEnemy() {
			t.Fatal("initial spawn failed")
		}
	}
	if g.spawnEnemy() || g.NextEnemySpawn != 3 {
		t.Fatal("spawn animation must reserve its space")
	}
	g.Enemies[0].Y = 32
	if !g.spawnEnemy() {
		t.Fatal("vacated spawn did not resume")
	}
}

func TestEnemySpriteAndProjectileAgree(t *testing.T) {
	for _, dir := range []Direction{DirUp, DirDown, DirLeft, DirRight} {
		a := directionRotation(dir)
		dx, dy := dir.Delta()
		// The source image's barrel vector is (0,-1).
		if math.Abs(math.Sin(a)-float64(dx)) > 1e-6 || math.Abs(-math.Cos(a)-float64(dy)) > 1e-6 {
			t.Fatalf("sprite faces away from %v", dir)
		}
		e := NewEnemy(64, 64, EnemyBasic, 0)
		e.Dir = dir
		b := e.Fire(1)
		if b.Dir != dir || b.Emitter != e || (b.X-78)*dx+(b.Y-78)*dy <= 0 {
			t.Fatal("projectile does not leave barrel")
		}
	}
	e := NewEnemy(0, 0, EnemyBasic, 0)
	if e.Hit(3) || !e.Active {
		t.Fatal("spawning tank was destroyed")
	}
	for e.SpawnTime > 0 {
		e.Update(&Map{}, nil, false)
	}
	if e.Y != 0 || e.Dir != DirDown {
		t.Fatal("spawn animation moved/turned tank")
	}
	e.Update(&Map{}, nil, false)
	if e.Y <= 0 {
		t.Fatal("enemy did not enter battlefield")
	}
}

func TestLatestDirectionAndGroundStop(t *testing.T) {
	dir, moving := preferredDirection([4]int{40, 0, 0, 1}, DirUp)
	if !moving || dir != DirRight {
		t.Fatal("old held direction swallowed turn")
	}
	tank := NewTank(64, 64, 0)
	tank.applyMovement(&Map{}, nil, nil, DirRight, true)
	x, y := tank.X, tank.Y
	tank.applyMovement(&Map{}, nil, nil, DirRight, false)
	if tank.X != x || tank.Y != y {
		t.Fatal("tank drifted on normal ground")
	}
}

func TestTurnCannotSnapThroughWallOrTank(t *testing.T) {
	for _, withWall := range []bool{true, false} {
		m := &Map{}
		tank := NewTank(40, 64, 0)
		tank.Dir = DirRight
		var enemies []*Enemy
		if withWall {
			// The intact upper-right brick fragment starts at X=72, exactly
			// at the tank's right edge. Snapping X=40 to X=48 must be blocked.
			m.Tiles[4][4] = TileBrick
			m.BrickMask[4][4] = BrickTR
		} else {
			enemies = []*Enemy{{X: 72, Y: 64, Size: 32, Active: true}}
		}
		tank.applyMovement(m, enemies, nil, DirDown, true)
		if tank.X != 40 || m.CheckTileCollision(tank.X, tank.Y, 32, 32) {
			t.Fatal("turn snapped through blocker")
		}
		for _, e := range enemies {
			if rectsOverlap(tank.X, tank.Y, 32, 32, e.X, e.Y, 32, 32) {
				t.Fatal("turn snapped into tank")
			}
		}
	}
}

func TestIceCoastsThenStops(t *testing.T) {
	m := &Map{}
	for r := range m.Tiles {
		for c := range m.Tiles[r] {
			m.Tiles[r][c] = TileIce
		}
	}
	tank := NewTank(64, 64, 0)
	tank.applyMovement(m, nil, nil, DirRight, true)
	x := tank.X
	tank.applyMovement(m, nil, nil, DirRight, false)
	if tank.X <= x {
		t.Fatal("ice had no inertia")
	}
	for i := 0; i < 30; i++ {
		tank.applyMovement(m, nil, nil, DirRight, false)
	}
	x = tank.X
	tank.applyMovement(m, nil, nil, DirRight, false)
	if tank.X != x {
		t.Fatal("ice slide never stops")
	}
}

func TestEnemyCollisionAndIndividualBullets(t *testing.T) {
	a, b := NewEnemy(0, 0, EnemyBasic, 0), NewEnemy(0, 32, EnemyBasic, 0)
	a.SpawnTime, b.SpawnTime = 0, 0
	b.Dir = DirUp
	g := &Game{Map: &Map{}, Menu: &MenuUI{}, Enemies: []*Enemy{a, b}}
	g.updateEnemies()
	if rectsOverlap(a.X, a.Y, 32, 32, b.X, b.Y, 32, 32) {
		t.Fatal("enemies overlapped")
	}
	g.Enemies = nil
	g.Bullets = nil
	for i := 0; i < 6; i++ {
		e := NewEnemy(float32(i*64), 64, EnemyBasic, 0)
		e.SpawnTime, e.FireTimer = 0, 0
		g.Enemies = append(g.Enemies, e)
	}
	g.updateEnemies()
	if len(g.Bullets) != 6 {
		t.Fatal("global bullet cap silenced enemies")
	}
	for _, e := range g.Enemies {
		e.FireTimer = 0
	}
	g.updateEnemies()
	if len(g.Bullets) != 6 {
		t.Fatal("enemy fired with its previous bullet still active")
	}
}
