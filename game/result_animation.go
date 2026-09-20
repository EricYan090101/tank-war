package game

const resultLeadFrames = 18

func (g *Game) resultTargets() [6]int {
	return [6]int{g.StageStats.Kills[0], g.StageStats.Kills[1], g.StageStats.Kills[2], 0, g.StageStats.StageBonus, g.StageStats.MapBonus}
}
func resultSegmentFrames(index int) int {
	if index < 4 {
		return 36
	}
	return 48
}
func (g *Game) resultCountDuration() int {
	frames := resultLeadFrames
	for i, target := range g.resultTargets() {
		if target > 0 {
			frames += resultSegmentFrames(i)
		}
	}
	return frames
}

// Only presentation changes. Score and bonuses were already settled once.
func (g *Game) resultCountValues() (values [6]int, total, active int) {
	targets := g.resultTargets()
	remaining := g.ResultCountFrame - resultLeadFrames
	active = -1
	total = g.Score
	for i, target := range targets {
		factor := 1
		if i < 4 {
			factor = EnemyPoints(EnemyType(i))
		}
		total -= target * factor
		if target <= 0 {
			continue
		}
		duration := resultSegmentFrames(i)
		switch {
		case remaining >= duration:
			values[i] = target
		case remaining >= 0:
			values[i] = target * remaining / duration
			active = i
		}
		remaining -= duration
		total += values[i] * factor
	}
	return
}

// Enter while counting reveals the result. A separate press advances stages.
func (g *Game) advanceResultCount(enter bool) bool {
	duration := g.resultCountDuration()
	if enter {
		if g.ResultCountFrame >= duration {
			return true
		}
		g.ResultCountFrame = duration
		return false
	}
	if g.ResultCountFrame < duration {
		g.ResultCountFrame++
	}
	return false
}
