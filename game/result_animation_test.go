package game

import "testing"

func TestResultCountProgressAndAccuracy(t *testing.T) {
	g := &Game{Score: 8450, HighScore: 9000, DifficultyBonus: 850, StageStats: StageStats{Kills: [4]int{12, 4, 2, 0}, StageBonus: 1000, MapBonus: 850}}
	values, previous, _ := g.resultCountValues()
	if values != [6]int{} || previous != 4000 {
		t.Fatalf("wrong start: %v %d", values, previous)
	}
	var last [6]int
	for frame := 0; frame <= g.resultCountDuration()+10; frame++ {
		g.ResultCountFrame = frame
		values, total, active := g.resultCountValues()
		if total < previous || total > g.Score {
			t.Fatal("count fell or overshot")
		}
		for i, n := range values {
			if n < last[i] || n > g.resultTargets()[i] {
				t.Fatal("row count fell or overshot")
			}
			if active >= 0 && i > active && n != 0 {
				t.Fatal("later row counted before active row")
			}
		}
		last, previous = values, total
	}
	if last != g.resultTargets() || previous != g.Score {
		t.Fatal("final numbers are not exact")
	}
	if g.Score != 8450 || g.HighScore != 9000 || g.DifficultyBonus != 850 {
		t.Fatal("animation mutated actual score")
	}
}
func TestResultEnterNeedsSeparatePressToContinue(t *testing.T) {
	g := &Game{Score: 1200, StageStats: StageStats{StageBonus: 1000, MapBonus: 200}}
	if g.advanceResultCount(true) {
		t.Fatal("first Enter advanced without showing result")
	}
	_, total, _ := g.resultCountValues()
	if total != 1200 {
		t.Fatal("Enter did not finish count")
	}
	if g.advanceResultCount(false) {
		t.Fatal("animation completion advanced automatically")
	}
	if !g.advanceResultCount(true) {
		t.Fatal("second Enter did not continue")
	}
}
func TestResultAnimationFinishesAndResets(t *testing.T) {
	g := &Game{State: StateStageCleanup, StageClearTimer: 1, Score: 1000, MapDifficulty: Difficulty{Score: 60}, StageStats: StageStats{StageBonus: 1000}, ResultCountFrame: 999}
	g.tickStageCleanup(false)
	if g.ResultCountFrame != 0 {
		t.Fatal("new result did not reset animation")
	}
	for i := 0; i < g.resultCountDuration()+5; i++ {
		if g.advanceResultCount(false) {
			t.Fatal("auto advanced")
		}
	}
	_, total, _ := g.resultCountValues()
	if total != 1300 || g.ResultCountFrame != g.resultCountDuration() {
		t.Fatal("animation failed to finish")
	}
}
