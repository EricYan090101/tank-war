package game

import (
	"encoding/json"
	"testing"
)

func TestDefeatIntroThenScore(t *testing.T) {
	for _, reason := range []string{"ALL LIVES LOST", "BASE DESTROYED"} {
		g := &Game{Score: 1000, MapDifficulty: Difficulty{Score: 60}}
		g.finishRun(reason)
		if g.State != StateDeathResult {
			t.Fatal("skipped defeat intro")
		}
		for i := 0; i < defeatIntroFrames-1; i++ {
			g.tickDefeatIntro()
		}
		if g.State != StateDeathResult {
			t.Fatal("intro too short")
		}
		g.tickDefeatIntro()
		if g.State != StateGameOver || g.ResultCountFrame != 0 {
			t.Fatal("no score animation after intro")
		}
		if g.advanceResultCount(true) {
			t.Fatal("first enter should reveal score")
		}
		if !g.advanceResultCount(true) {
			t.Fatal("second enter should leave")
		}
		g.finishRun(reason)
		if g.Score != 1300 {
			t.Fatal("score settled more than once")
		}
	}
}
func TestBaseRuinsAndIntroSync(t *testing.T) {
	m := &Map{}
	m.fillBlock(12, 24, TileBase)
	hit, dead := m.HitBullet(193, 385, DirDown, 1, false, 0)
	if !hit || !dead {
		t.Fatal("base hit not recognized")
	}
	for r := 24; r < 26; r++ {
		for c := 12; c < 14; c++ {
			if m.Tiles[r][c] != TileBaseRuins {
				t.Fatal("partial ruined base")
			}
		}
	}
	_, dead = m.HitBullet(193, 385, DirDown, 1, false, 0)
	if dead {
		t.Fatal("ruins destroyed twice")
	}
	g := &Game{Map: m, State: StateDeathResult, DeathResultTimer: 45}
	data, err := json.Marshal(g.coopSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	var snap versusSnapshot
	if err = json.Unmarshal(data, &snap); err != nil {
		t.Fatal(err)
	}
	client := &Game{}
	client.applyCoopSnapshot(&snap)
	if client.State != StateDeathResult || client.DeathResultTimer != 45 || client.Map.Tiles[24][12] != TileBaseRuins {
		t.Fatal("defeat visual state not synchronized")
	}
}
