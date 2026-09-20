package game

import "testing"

func TestIronShieldThreeHitsAndRefill(t *testing.T) {
	g := &Game{}
	p := NewTank(64, 64, 0)
	(&Item{Type: ItemHelmet}).Apply(g, p)
	for frame := 0; frame < 1200; frame++ {
		p.updateWithInput(&Map{}, nil, nil, VersusInput{})
	}
	if p.ArmorShield != 3 {
		t.Fatal("iron shield expired with time")
	}
	lives := p.Lives
	for want := 2; want >= 0; want-- {
		if p.TakeHit() || !p.Active || p.ArmorShield != want || p.Lives != lives {
			t.Fatal("shield did not absorb exactly one shot")
		}
	}
	if !p.TakeHit() || p.Lives != lives-1 {
		t.Fatal("fourth shot should damage unshielded tank")
	}
	p.Respawn(64, 64)
	p.ArmorShield = 1
	(&Item{Type: ItemHelmet}).Apply(g, p)
	if p.ArmorShield != 3 {
		t.Fatal("pickup did not refill shield")
	}
	p.TakeHit()
	if p.ArmorShield != 2 {
		t.Fatal("respawn must not grant invulnerability")
	}
}

func TestEnemyIronShieldIgnoresBulletPower(t *testing.T) {
	for _, typ := range []EnemyType{EnemyBasic, EnemyFast, EnemyPower} {
		e := NewEnemy(64, 64, typ, 1)
		e.ArmorShield = 3
		if e.Hit(4) || e.ArmorShield != 3 {
			t.Fatal("spawn protection failed")
		}
		e.SpawnTime = 0
		hp := e.HP
		for want := 2; want >= 0; want-- {
			if e.Hit(4) || e.ArmorShield != want || e.HP != hp {
				t.Fatal("power bypassed shield or third shot damaged hull")
			}
		}
		killed := e.Hit(1)
		if !killed {
			t.Fatal("unshielded enemy survived")
		}
	}
}

func TestSpawnedEnemiesSometimesCarryShield(t *testing.T) {
	g := &Game{Map: &Map{}, TotalEnemiesPerStage: 400}
	shielded := 0
	for i := 0; i < 400; i++ {
		g.Enemies = nil
		if !g.spawnEnemy() {
			t.Fatal("spawn failed")
		}
		if g.Enemies[0].ArmorShield == 3 {
			shielded++
		} else if g.Enemies[0].ArmorShield != 0 {
			t.Fatal("invalid initial durability")
		}
	}
	if shielded < 30 || shielded > 180 {
		t.Fatalf("unexpected shield spawn count %d", shielded)
	}
}

func TestOpeningArmorAndNoRespawnInvulnerability(t *testing.T) {
	g := &Game{Tanks: []*Tank{NewTank(64, 64, 0), NewTank(96, 64, 1)}}
	g.loadStage(0)
	for _, p := range g.Tanks {
		if p.ArmorShield != 3 {
			t.Fatal("opening armor missing")
		}
		p.ArmorShield = 1
	}
	g.loadStage(1)
	for _, p := range g.Tanks {
		if p.ArmorShield != 1 {
			t.Fatal("later stage refilled armor")
		}
		p.ArmorShield = 0
		p.Respawn(64, 64)
		if !p.TakeHit() {
			t.Fatal("respawn is still invulnerable")
		}
	}
	v := &Game{Match: VersusMatch{Config: VersusConfig{1, 3}}}
	v.startVersusRound()
	for _, p := range v.Tanks {
		if p.ArmorShield != 3 {
			t.Fatal("first round armor missing")
		}
	}
	v.startVersusRound()
	for _, p := range v.Tanks {
		if p.ArmorShield != 0 {
			t.Fatal("later round got free armor")
		}
	}
	for stage := 0; stage < 100; stage++ {
		for i := 0; i < 20; i++ {
			if enemyTypeFor(stage, i) > EnemyPower {
				t.Fatal("heavy enemy still spawns")
			}
		}
	}
}
