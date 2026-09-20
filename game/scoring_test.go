package game

import "testing"

func TestFinalScoreIncludesPartialStageOnce(t *testing.T) {
	g := &Game{Score: 1000, MapDifficulty: Difficulty{Score: 60}, StageStats: StageStats{Kills: [4]int{2, 1, 0, 0}}}
	g.finishRun("BASE DESTROYED")
	if g.Score != 1300 || g.DifficultyBonus != 300 || g.TotalKills != 3 || g.HighScore != 1300 || g.State != StateDeathResult {
		t.Fatalf("wrong final result: %+v", g)
	}
	g.finishRun("RUN ENDED")
	if g.Score != 1300 || g.TotalKills != 3 || g.EndReason != "BASE DESTROYED" {
		t.Fatal("final result changed after finalization")
	}
}
func TestMultiStageScoreDoesNotCompound(t *testing.T) {
	g := &Game{Score: 1000, MapDifficulty: Difficulty{Score: 60}}
	g.settleStage()
	g.settleStage()
	if g.Score != 1300 {
		t.Fatal("stage awarded twice")
	}
	g.StageStats = StageStats{StartScore: g.Score}
	g.stageSettled = false
	g.Score += 2000
	g.MapDifficulty.Score = 40
	g.finishRun("ALL LIVES LOST")
	if g.Score != 3700 || g.DifficultyBonus != 700 {
		t.Fatalf("wrong multi-stage score: %d bonus %d", g.Score, g.DifficultyBonus)
	}
}
func TestDeathAndStageClearPriority(t *testing.T) {
	g := &Game{State: StatePlaying, Tanks: []*Tank{{Lives: 0, Active: false}}, TotalEnemiesPerStage: 20, SpawnedEnemyCount: 20}
	g.checkStageClear()
	if g.State != StateDeathResult || g.ClearedStages != 0 || g.Score != 0 {
		t.Fatal("defeat must take priority over clear")
	}
	g = &Game{State: StatePlaying, Tanks: []*Tank{{Lives: 1, Active: false}}, TotalEnemiesPerStage: 20, SpawnedEnemyCount: 20, MapDifficulty: Difficulty{Score: 60}}
	g.checkStageClear()
	if g.State != StateStageCleanup || g.Score != 1000 || g.ClearedStages != 1 {
		t.Fatal("remaining life should permit stage clear")
	}
	g.finishRun("RUN ENDED")
	if g.Score != 1300 {
		t.Fatal("ending after clear awarded points twice")
	}
}
