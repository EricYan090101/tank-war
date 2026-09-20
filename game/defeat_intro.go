package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
)

const defeatIntroFrames = 150

func (g *Game) tickDefeatIntro() {
	if g.State != StateDeathResult {
		return
	}
	g.DeathResultTimer++
	g.updateExplosions()
	if g.DeathResultTimer >= defeatIntroFrames {
		g.State = StateGameOver
		g.ResultCountFrame = 0
	}
}
func (g *Game) drawDeathResult(s *ebiten.Image) {
	if g.Map != nil {
		g.drawBattle(s)
	} else {
		s.Fill(color.Black)
	}
	drawnButtons = nil
	fade := float32(min(g.DeathResultTimer, 60)) / 60
	fade = fade * fade * (3 - 2*fade)
	uiRect(s, 0, 0, ScreenWidth, ScreenHeight, color.NRGBA{A: uint8(195 * fade)})
	uiCenter(s, "GAME OVER", ScreenHeight/2-20, 5, color.NRGBA{R: 210, G: 61, B: 64, A: uint8(255 * fade)})
}
