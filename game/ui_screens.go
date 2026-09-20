package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func (m *MenuUI) DrawTitle(s *ebiten.Image) {
	s.Fill(uiBG)
	uiTank(s, 248, 56, 6, uiGold)
	uiCenter(s, "TANK BATTLE", 198, 5, uiInk)
	uiButton(s, "ENTER START", 188, 298, 200, 48, ebiten.KeyEnter, uiGold)
}
func (m *MenuUI) DrawModeSelect(s *ebiten.Image) {
	uiBackground(s, "", "PLAY")
	uiIconCard(s, 139, m.ModeCursor == 0)
	uiTank(s, 154, 156, 3, uiGold)
	uiTank(s, 214, 156, 3, uiGold)
	uiShield(s, 318, 155, uiGreen)
	uiText(s, "1-4", 452, 171, 2, uiMuted)
	uiIconCard(s, 231, m.ModeCursor == 1)
	uiVersusIcon(s, 231, 1)
	uiFooter(s, "ENTER CONTINUE", "ESC BACK")
}
func (m *MenuUI) DrawRoomSelect(s *ebiten.Image) {
	uiBackground(s, "02 / CONNECTION", "ROOM")
	uiOption(s, 139, "CREATE ROOM", "CREATE A LOBBY / INVITE YOUR TEAM", m.RoomCursor == 0)
	uiOption(s, 231, "FIND ROOM", "SEARCH FOR ROOMS ON YOUR NETWORK", m.RoomCursor == 1)
	uiFooter(s, "UP / DOWN SELECT  ENTER CONFIRM", "ESC BACK")
}
func (m *MenuUI) DrawRoomNameInput(s *ebiten.Image) {
	uiBackground(s, "03 / YOUR CALLSIGN", "ROOM NAME")
	uiFrame(s, 44, 140, 488, 185, uiGold)
	uiRect(s, 66, 186, 444, 46, uiBG)
	uiRect(s, 66, 230, 444, 2, uiGold)
	uiText(s, m.RoomName+"_", 80, 201, 2, uiInk)
	uiText(s, fmt.Sprintf("%02d / 15", len([]rune(m.RoomName))), 66, 256, 1, uiMuted)
	uiText(s, "A-Z / 0-9 / SPACE / - / _", 66, 286, 1, uiMuted)
	uiFooter(s, "ENTER CREATE ROOM", "BACKSPACE DELETE / ESC BACK")
}
func (m *MenuUI) DrawRoomList(s *ebiten.Image, rooms []RoomInfo) {
	uiBackground(s, "03 / LOCAL NETWORK", "ROOMS")
	if len(rooms) == 0 {
		uiFrame(s, 44, 146, 488, 178, uiGreen)
		uiTank(s, 72, 182, 3, uiGreen)
		uiText(s, "SEARCHING...", 148, 183, 2, uiInk)
		uiText(s, "NO ROOMS YET", 148, 221, 1, uiMuted)
	} else {
		selected := m.RoomListCursor
		if selected >= len(rooms) {
			selected = len(rooms) - 1
		}
		const perPage = 3
		start := selected / perPage * perPage
		for i := start; i < len(rooms) && i < start+perPage; i++ {
			r := rooms[i]
			y := 132 + (i-start)*71
			accent := uiLine
			if i == selected {
				accent = uiGold
			}
			uiFrame(s, 44, y, 488, 64, accent)
			if i == selected {
				uiText(s, ">", 56, y+12, 1, uiGold)
			}
			name := []rune(r.RoomName)
			if len(name) > 24 {
				name = name[:24]
			}
			uiText(s, string(name), 78, y+9, 2, uiInk)
			uiText(s, fmt.Sprintf("%d / %d", r.CurrentPlayers, r.MaxPlayers), 414, y+10, 2, uiGreen)
			rules := "COOP"
			if GameMode(r.Mode) == ModeVersus {
				rules = fmt.Sprintf("%dV%d / %d ROUNDS / %d TO WIN", r.TeamSize, r.TeamSize, r.BestOf, r.BestOf/2+1)
			}
			uiText(s, rules, 78, y+28, 1, uiGold)

			status := "OPEN"
			statusColor := uiGreen
			if r.MaxPlayers > 0 && r.CurrentPlayers >= r.MaxPlayers {
				status = "FULL"
				statusColor = uiRed
			}
			if r.InProgress {
				status = "PLAYING"
				statusColor = uiMuted
			}
			uiText(s, status, 474, y+46, 1, statusColor)
		}
		uiText(s, fmt.Sprintf("%d / %d", selected/perPage+1, (len(rooms)+perPage-1)/perPage), 44, 353, 1, uiMuted)
	}
	uiFooter(s, "UP / DOWN SELECT  ENTER JOIN", "ESC BACK")
}

func (g *Game) drawSidebar(s *ebiten.Image) {
	uiRect(s, 416, 0, 160, 416, uiBG)
	uiRect(s, 416, 0, 2, 416, uiGold)
	uiText(s, fmt.Sprintf("STAGE %02d", g.CurrentStage+1), 430, 36, 2, uiInk)
	uiRect(s, 430, 61, 132, 1, uiLine)
	uiText(s, "TOTAL SCORE", 430, 76, 1, uiMuted)
	uiText(s, fmt.Sprintf("%06d", g.Score), 430, 92, 2, uiGold)

	uiRect(s, 430, 136, 132, 1, uiLine)
	localID := 0
	if g.Coop != nil {
		localID = g.Coop.LocalID
	}
	uiText(s, fmt.Sprintf("PLAYER %02d / YOU", localID+1), 430, 150, 1, uiMuted)
	if len(g.Tanks) > 0 && g.Tanks[0] != nil {
		t := g.Tanks[0]
		for _, candidate := range g.Tanks {
			if candidate.PlayerID == localID {
				t = candidate
			}
		}
		uiTank(s, 432, 166, 1, uiGold)
		uiText(s, fmt.Sprintf("%02d LIVES", max(0, t.Lives)), 456, 169, 2, uiInk)
		uiText(s, "CANNON", 430, 195, 1, uiMuted)
		for i := 0; i < 3; i++ {
			c := uiLine
			if t.Power > i+1 {
				c = uiGold
			}
			uiText(s, "*", 488+i*22, 192, 2, c)
		}

		if t.ArmorShield > 0 {
			uiText(s, fmt.Sprintf("SHIELD %d/3", t.ArmorShield), 430, 231, 1, uiGreen)
		}
	}
	uiRect(s, 430, 249, 132, 1, uiLine)
	left := max(0, g.TotalEnemiesPerStage-g.SpawnedEnemyCount+g.activeEnemyCount())
	uiText(s, fmt.Sprintf("HOSTILES LEFT %02d", left), 430, 262, 1, uiMuted)
	for i := 0; i < 20; i++ {
		c := uiLine
		if i < left {
			c = uiRed
		}
		uiRect(s, 430+i%10*13, 280+i/10*11, 8, 6, c)
	}
	uiText(s, fmt.Sprintf("MAP RISK %03d", g.MapDifficulty.Score), 430, 309, 1, uiMuted)
	uiProgress(s, 430, 324, 130, g.MapDifficulty.Score, 100, uiGold)
	if g.State == StateStageCleanup {
		if g.Coop == nil || g.Coop.Host {
			uiButton(s, "ENTER RESULTS", 426, 357, 144, 18, ebiten.KeyEnter, uiGold)
		}
	}
	if g.State == StateDeathResult {
		return
	}
	if g.Coop == nil || g.Coop.Host {
		uiButton(s, "P PAUSE", 426, 378, 66, 29, ebiten.KeyP, uiMuted)
	}
	uiButton(s, "ESC EXIT", 498, 378, 74, 29, ebiten.KeyEscape, uiRed)
}
func (g *Game) drawStageIntro(s *ebiten.Image) {
	g.Map.Draw(s)
	uiRect(s, 0, 0, 576, 416, color.RGBA{5, 10, 13, 205})
	uiFrame(s, 52, 44, 472, 320, uiGold)
	uiText(s, fmt.Sprintf("STAGE %02d", g.CurrentStage+1), 76, 99, 4, uiInk)
	uiFrame(s, 76, 181, 200, 101, uiLine)
	uiText(s, "ENEMY WAVE", 92, 197, 1, uiMuted)
	uiText(s, fmt.Sprintf("%02d", g.TotalEnemiesPerStage), 92, 220, 4, uiRed)
	uiFrame(s, 292, 181, 208, 101, uiLine)
	uiText(s, "TERRAIN RISK", 308, 197, 1, uiMuted)
	uiText(s, fmt.Sprintf("%02d", g.MapDifficulty.Score), 308, 220, 4, uiGold)
	uiText(s, "/ 100", 374, 239, 1, uiMuted)
	uiProgress(s, 308, 262, 174, g.MapDifficulty.Score, 100, uiGold)
	if g.Coop == nil || g.Coop.Host {
		uiButton(s, "ENTER BEGIN MISSION", 76, 308, 242, 34, ebiten.KeyEnter, uiGold)
	} else {
		uiText(s, "WAITING FOR HOST", 76, 323, 1, uiMuted)
	}
}
func (g *Game) drawStageClear(s *ebiten.Image) {
	uiBackground(s, "MISSION DEBRIEF", "STAGE CLEAR")
	values, total, active := g.resultCountValues()
	done := g.ResultCountFrame >= g.resultCountDuration()
	uiText(s, fmt.Sprintf("%02d", g.CurrentStage+1), 487, 75, 3, uiGreen)
	uiFrame(s, 28, 128, 312, 222, uiGreen)
	uiText(s, "TARGET", 44, 145, 1, uiMuted)
	uiText(s, "KILLS", 177, 145, 1, uiMuted)
	uiText(s, "POINTS", 267, 145, 1, uiMuted)
	for i, name := range []string{"BASIC", "FAST", "POWER"} {
		y := 174 + i*35
		uiRect(s, 44, y-9, 280, 1, uiLine)
		if active == i {
			uiRect(s, 38, y-4, 292, 18, uiRaised)
			uiRect(s, 28, y-4, 3, 18, uiGold)
		}
		uiText(s, name, 44, y, 2, uiInk)
		uiText(s, fmt.Sprintf("%02d", values[i]), 182, y, 2, uiInk)
		uiText(s, fmt.Sprintf("%04d", values[i]*EnemyPoints(EnemyType(i))), 273, y, 2, uiGold)
	}
	uiText(s, fmt.Sprintf("ITEMS %02d", sumItems(g.StageStats.ItemsCollected)), 44, 329, 1, uiMuted)
	uiFrame(s, 356, 128, 192, 222, uiGold)
	if active == 4 {
		uiRect(s, 364, 137, 176, 51, uiRaised)
	}
	if active == 5 {
		uiRect(s, 364, 193, 176, 51, uiRaised)
	}
	uiText(s, "CLEAR BONUS", 372, 145, 1, uiMuted)
	uiText(s, fmt.Sprintf("+%04d", values[4]), 372, 165, 2, uiGreen)
	uiText(s, "MAP BONUS", 372, 201, 1, uiMuted)
	uiText(s, fmt.Sprintf("+%04d", values[5]), 372, 221, 2, uiGreen)
	uiRect(s, 372, 252, 160, 1, uiLine)
	uiText(s, "TOTAL SCORE", 372, 273, 1, uiGold)
	uiText(s, fmt.Sprintf("%06d", total), 372, 299, 2, uiInk)
	uiProgress(s, 372, 324, 160, g.ResultCountFrame, g.resultCountDuration(), uiGold)
	if g.Coop != nil && !g.Coop.Host {
		uiFooter(s, "WAITING FOR HOST", "ESC LEAVE ROOM")
	} else if done {
		uiFooter(s, "ENTER NEXT STAGE", "ESC FINISH RUN")
	} else {
		uiFooter(s, "ENTER SKIP COUNT", "ESC FINISH RUN")
	}
}

func (g *Game) drawGameOver(s *ebiten.Image) {
	_, displayScore, _ := g.resultCountValues()
	uiBackground(s, "RUN COMPLETE", "RESULT")
	uiText(s, g.EndReason, 28, 118, 1, uiRed)
	uiFrame(s, 28, 142, 520, 84, uiGold)
	uiText(s, "TOTAL SCORE", 48, 159, 1, uiGold)
	uiText(s, fmt.Sprintf("%06d", displayScore), 48, 181, 4, uiInk)
	uiText(s, "SESSION BEST", 374, 161, 1, uiMuted)
	uiText(s, fmt.Sprintf("%06d", g.HighScore), 374, 187, 2, uiGold)
	uiFrame(s, 28, 242, 252, 83, uiLine)
	uiText(s, fmt.Sprintf("BASE      %06d", g.Score-g.DifficultyBonus), 44, 258, 2, uiInk)
	uiText(s, fmt.Sprintf("BONUS    +%06d", g.DifficultyBonus), 44, 282, 2, uiGreen)
	uiFrame(s, 296, 242, 252, 83, uiLine)
	uiText(s, fmt.Sprintf("STAGES  %02d", g.ClearedStages), 312, 258, 2, uiInk)
	uiText(s, fmt.Sprintf("KILLS   %02d", g.TotalKills), 312, 282, 2, uiInk)

	if g.ResultCountFrame < g.resultCountDuration() {
		if g.Coop != nil && !g.Coop.Host {
			uiFooter(s, "COUNTING SCORE", "WAITING FOR HOST")
		} else {
			uiFooter(s, "ENTER SKIP COUNT", "")
		}
	} else if g.Coop != nil {
		uiFooter(s, "ENTER LEAVE ROOM", "THANK YOU FOR PLAYING")
	} else {
		uiFooter(s, "ENTER RETURN TO TITLE", "THANK YOU FOR PLAYING")
	}
}

func (g *Game) drawBattleOverlays(s *ebiten.Image) {
	if g.State == StateStageCleanup {
		uiNoticeFrame(s, 60, 14, 296, 58, uiGreen)
		uiText(s, "AREA SECURED", 76, 25, 2, uiGreen)
		uiText(s, fmt.Sprintf("RESULTS IN %d", (g.StageClearTimer+59)/60), 76, 48, 1, uiInk)
		uiProgress(s, 76, 62, 264, g.StageClearTimer, StageClearFrames, uiGreen)
	} else if g.MessageTimer > 0 {
		uiNoticeFrame(s, 36, 14, 344, 30, uiGold)
		uiText(s, g.LastMessage, 50, 26, 1, uiInk)
	}
	if g.FreezeTimer > 0 && g.State == StatePlaying {
		uiNoticeFrame(s, 130, 378, 156, 25, uiGreen)
		uiText(s, fmt.Sprintf("FREEZE %02d SEC", (g.FreezeTimer+59)/60), 146, 388, 1, uiGreen)
	}
	if g.IsPaused {
		uiModal(s, "PAUSED", "", "", uiGold)
		if g.Coop == nil || g.Coop.Host {
			uiButton(s, "P RESUME", 56, 233, 143, 32, ebiten.KeyP, uiGreen)
		} else {
			uiText(s, "HOST PAUSED", 56, 246, 1, uiMuted)
		}
		uiButton(s, "ESC END RUN", 211, 233, 149, 32, ebiten.KeyEscape, uiRed)
	}
	if g.ShowExitConfirm {
		if g.Coop != nil && !g.Coop.Host {
			uiExitDialog(s, "LEAVE ROOM?", "OTHER PLAYERS CAN CONTINUE.", "LEAVE")
		} else {
			uiExitDialog(s, "END THIS RUN?", "YOUR SCORE WILL BE COUNTED.", "FINISH")
		}
	}
}
