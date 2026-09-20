package game

// Each stage's earned points get a 0-50% map bonus, rounded down once.
// Settling on either clear or final defeat avoids losing partial-stage credit.
func (g *Game) settleStage() {
	if g.stageSettled {
		return
	}
	g.stageSettled = true
	earned := max(0, g.Score-g.StageStats.StartScore)
	bonus := earned * g.MapDifficulty.Score / 200
	g.StageStats.MapBonus = bonus
	g.DifficultyBonus += bonus
	g.Score += bonus
	g.TotalKills += g.StageStats.TotalKills()
	g.HighScore = max(g.HighScore, g.Score)
}
func (g *Game) finishRun(reason string) {
	if g.runFinished {
		return
	}
	g.settleStage()
	g.runFinished = true
	g.EndReason = reason
	g.ShowExitConfirm, g.IsPaused = false, false
	g.State = StateGameOver
	g.ResultCountFrame = 0
	if reason == "BASE DESTROYED" || reason == "ALL LIVES LOST" {
		g.State = StateDeathResult
		g.DeathResultTimer = 0
	}
}

// Victory is already secured: only movement, items and effects continue here.
func (g *Game) tickStageCleanup(skip bool) {
	if g.State != StateStageCleanup || g.IsPaused || g.ShowExitConfirm {
		return
	}
	if g.StageClearTimer > 0 {
		g.StageClearTimer--
	}
	if skip || g.StageClearTimer == 0 {
		g.settleStage()
		g.ResultCountFrame = 0
		g.State = StateStageClear
	}
}
