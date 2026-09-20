package game

import "testing"

func TestCannonStarsMatchClassicTiers(t *testing.T) {
	g := &Game{}
	tank := NewTank(64, 64, 0)
	normalSpeed := tank.Fire().Speed
	for power := 1; power <= 4; power++ {
		if power > 1 {
			item := &Item{Type: ItemStar, Active: true}
			item.Apply(g, tank)
		}
		if tank.Power != power {
			t.Fatalf("power=%d want %d", tank.Power, power)
		}
		wantShots := 1
		if power >= 3 {
			wantShots = 2
		}
		if tank.MaxBullets() != wantShots {
			t.Fatalf("tier %d has wrong quota", power)
		}
		bullet := tank.Fire()
		if power > 1 && bullet.Speed <= normalSpeed {
			t.Fatalf("tier %d did not gain bullet speed", power)
		}
		m := &Map{}
		m.Tiles[4][4] = TileSteel
		m.HitBullet(65, 65, DirDown, power, power >= 4, bullet.BlastRadius)
		if (m.Tiles[4][4] == TileEmpty) != (power == 4) {
			t.Fatalf("tier %d has incorrect steel penetration", power)
		}
		// Upgrades improve the cannon, not the damage against an iron shield.
		e := NewEnemy(128, 128, EnemyBasic, 0)
		e.SpawnTime = 0
		e.ArmorShield = 3
		g = &Game{Map: &Map{}, Menu: &MenuUI{}, Enemies: []*Enemy{e}, StageItemCount: 3}
		bullet.X, bullet.Y, bullet.Dir = 140, 130, DirDown
		g.Bullets = []*Bullet{bullet}
		g.updateBullets()
		if !e.Active || e.ArmorShield != 2 {
			t.Fatalf("tier %d should remove one armor point, HP=%d", power, e.HP)
		}
	}
	extra := &Item{Type: ItemStar}
	extra.Apply(g, tank)
	if tank.Power != 4 {
		t.Fatal("upgrade exceeds three stars")
	}
	tank.Respawn(64, 64)
	if tank.Power != 4 {
		t.Fatal("stage transition lost upgrades")
	}
	tank.TakeHit()
	if tank.Power != 1 {
		t.Fatal("death did not reset cannon")
	}
}

func TestClearWindowAndFinalSettlement(t *testing.T) {
	g := &Game{State: StatePlaying, Map: &Map{}, Menu: &MenuUI{}, Tanks: []*Tank{NewTank(64, 64, 0)}, TotalEnemiesPerStage: 20, SpawnedEnemyCount: 20, MapDifficulty: Difficulty{Score: 60}, Bullets: []*Bullet{NewBullet(0, 0, DirDown, OwnerEnemy, 1)}}
	g.checkStageClear()
	if g.State != StateStageCleanup || g.StageClearTimer != 180 || len(g.Bullets) != 0 {
		t.Fatal("clear did not start safe grace period")
	}
	g.checkStageClear()
	if g.ClearedStages != 1 || g.Score != 1000 {
		t.Fatal("clear bonus repeated")
	}
	g.Items = []*Item{{X: 64, Y: 64, Type: ItemStar, Active: true, Timer: 100}}
	g.updateItems()
	if g.Tanks[0].Power != 2 || g.StageStats.ItemsCollected[ItemStar] != 1 {
		t.Fatal("cannot collect final supply")
	}
	g.IsPaused = true
	g.tickStageCleanup(false)
	if g.StageClearTimer != 180 {
		t.Fatal("pause consumed grace time")
	}
	g.IsPaused = false
	for i := 0; i < 179; i++ {
		g.tickStageCleanup(false)
	}
	if g.State != StateStageCleanup || g.stageSettled {
		t.Fatal("result appeared early")
	}
	g.tickStageCleanup(false)
	if g.State != StateStageClear || g.Score != 1300 {
		t.Fatal("result was not settled after grace period")
	}
	g.tickStageCleanup(false)
	if g.Score != 1300 {
		t.Fatal("result settled twice")
	}
}

func TestClearWindowEnterSkips(t *testing.T) {
	g := &Game{State: StateStageCleanup, StageClearTimer: 180, Score: 1000, MapDifficulty: Difficulty{Score: 60}}
	g.tickStageCleanup(true)
	if g.State != StateStageClear || g.Score != 1300 {
		t.Fatal("Enter did not skip to result")
	}
}
