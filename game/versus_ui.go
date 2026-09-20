package game

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"strings"
)

func (g *Game) versusHost() bool { return g.Versus == nil || g.Versus.Host }
func (g *Game) drawVersus(s *ebiten.Image) {
	if g.State == StateVersusLobby && g.ChoosingTeam {
		g.drawVersusTeamSelect(s)
		return
	}
	switch g.State {
	case StateVersusSettings:
		g.drawVersusSettings(s)
	case StateVersusTeamSelect:
		g.drawVersusTeamSelect(s)
	case StateVersusSeries:
		g.drawVersusSeries(s)
	case StateVersusLobby:
		g.drawVersusLobby(s)
	case StateVersusMatchEnd:
		g.drawVersusFinal(s)
	default:
		if g.versusCanvas == nil {
			g.versusCanvas = ebiten.NewImage(416, 416)
		}
		field := g.versusCanvas
		field.Fill(uiBG)
		if g.Map != nil {
			g.Map.Draw(field)
		}
		for _, t := range g.Tanks {
			t.Draw(field)
		}
		for _, b := range g.Bullets {
			b.Draw(field)
		}
		for _, e := range g.Explosions {
			e.Draw(field)
		}
		if g.Map != nil {
			g.Map.DrawBushes(field)
		}
		op := &ebiten.DrawImageOptions{}
		if g.versusRotated() {
			op.GeoM.Scale(-1, -1)
			op.GeoM.Translate(416, 416)
		}
		s.DrawImage(field, op)
		for _, t := range g.Tanks {
			if t.Active {
				x, y := g.versusPosition(t.X, t.Y, 32)
				name := g.playerName(t.PlayerID)
				uiText(s, name, min(416-len(name)*6, max(0, int(x)+16-len(name)*3)), max(2, int(y)-10), 1, uiInk)
			}
		}
		g.drawVersusSidebar(s)
		if g.State == StateVersusCountdown {
			uiModal(s, fmt.Sprintf("ROUND %02d", g.Match.Round), "", fmt.Sprintf("START IN %d", (g.Match.Timer+59)/60), uiGold)
		}
		if g.State == StateVersusRoundEnd {
			title := "ROUND DRAW"
			if g.Match.Winner == 0 {
				title = "YELLOW TEAM WINS"
			}
			if g.Match.Winner == 1 {
				title = "GREEN TEAM WINS"
			}
			uiModal(s, title, "", fmt.Sprintf("NEXT ROUND IN %d", (g.Match.Timer+59)/60), uiGreen)
		}
		if g.ShowExitConfirm {
			uiExitDialog(s, "LEAVE MATCH?", "THIS MATCH WILL RESET FOR EVERYONE.", "LEAVE")
		}
	}
	if g.MessageTimer > 0 && (g.State == StateVersusSettings || g.State == StateVersusLobby) {
		uiText(s, g.LastMessage, 44, 352, 1, uiRed)
	}
}
func (g *Game) drawVersusSettings(s *ebiten.Image) {
	uiBackground(s, "", "MATCH")
	for i := 0; i < 2; i++ {
		y := 137 + i*92
		uiIconCard(s, y, g.Menu.ConfigCursor == i)
		uiVersusIcon(s, y, i+1)
		uiText(s, fmt.Sprintf("%dV%d", i+1, i+1), 464, y+31, 2, uiMuted)
	}
	uiFooter(s, "ENTER CONTINUE", "ESC BACK")
}
func (g *Game) drawVersusTeamSelect(s *ebiten.Image) {
	uiBackground(s, "2V2 / TEAM SELECTION", "TEAM")
	counts := [2]int{}
	if g.Versus != nil {
		for _, id := range g.LobbyMembers {
			team := g.playerTeam(id)
			if team >= 0 && team < 2 {
				counts[team]++
			}
		}
	}
	for cursor, name := range []string{"GREEN TEAM", "YELLOW TEAM"} {
		team := 1 - cursor
		detail := fmt.Sprintf("%d / 2", counts[team])
		if counts[team] >= 2 {
			detail += " / FULL"
		}
		y := 137 + cursor*92
		c := uiGold
		if team == 1 {
			c = uiGreen
		}
		uiFrame(s, 44, y, 488, 76, c)
		uiRect(s, 44, y, 5, 76, c)
		uiTank(s, 458, y+16, 3, c)
		if g.Menu.TeamCursor == cursor {
			uiText(s, ">", 62, y+23, 2, c)
		}
		uiText(s, name, 88, y+17, 2, c)
		uiText(s, detail, 88, y+47, 2, uiMuted)
	}

	if g.MessageTimer > 0 {
		uiText(s, g.LastMessage, 44, 352, 1, uiRed)
	}
	action := "ENTER CONTINUE"
	back := "ESC BACK"
	if g.Versus != nil {
		action = "ENTER JOIN TEAM"
		back = "ESC LEAVE ROOM"
	}
	uiFooter(s, action, back)
}
func (g *Game) drawVersusSeries(s *ebiten.Image) {
	uiBackground(s, "HOST / MATCH LENGTH", "ROUNDS")
	for i := 0; i < 3; i++ {
		y := 132 + i*72
		accent := uiLine
		if g.Menu.SeriesIndex == i {
			accent = uiGold
		}
		uiFrame(s, 44, y, 488, 62, accent)
		if g.Menu.SeriesIndex == i {
			uiRect(s, 44, y, 4, 62, uiGold)
			uiText(s, ">", 62, y+21, 2, uiGold)
		}
		uiText(s, fmt.Sprintf("%d ROUNDS", 3+i*2), 88, y+23, 2, uiInk)
		uiText(s, fmt.Sprintf("%d WINS", i+2), 414, y+24, 2, uiMuted)
	}
	if g.MessageTimer > 0 {
		uiText(s, g.LastMessage, 44, 352, 1, uiRed)
	}
	uiFooter(s, "CLICK OR ENTER CREATE LOBBY", "ESC BACK")
}
func (g *Game) drawVersusLobby(s *ebiten.Image) {
	uiBackground(s, "VERSUS / WAITING ROOM", "LOBBY")
	c := g.Match.Config
	uiText(s, fmt.Sprintf("%dV%d / BEST OF %d / FIRST TO %d", c.TeamSize, c.TeamSize, c.BestOf, c.WinsNeeded()), 44, 122, 1, uiGold)
	local := 0
	if g.Versus != nil {
		local = g.Versus.LocalID
	}
	for id := 0; id < c.Players(); id++ {
		present := false
		for _, member := range g.LobbyMembers {
			if member == id {
				present = true
			}
		}
		y := 146 + id*46
		accent := uiLine
		if present {
			accent = uiGreen
		}
		uiFrame(s, 44, y, 488, 38, accent)
		c := uiMuted
		if present {
			c = uiGold
			if g.playerTeam(id) == 1 {
				c = uiGreen
			}
		}
		label := "EMPTY"
		if present {
			label = g.playerName(id)
		}
		uiTank(s, 58, y+8, 1, c)
		uiText(s, label, 86, y+12, 2, c)
		tag := ""
		if present {
			if id == 0 {
				tag = "HOST"
			}
			if id == local {
				tag += " YOU"
			}
			if g.playerTeam(id) < 0 {
				tag = "PICK TEAM"
			}
		}
		uiText(s, tag, 422, y+16, 1, uiMuted)
	}
	ready := len(g.LobbyMembers) == c.Players()
	counts := [2]int{}
	for _, id := range g.LobbyMembers {
		team := g.playerTeam(id)
		if team >= 0 && team < 2 {
			counts[team]++
		}
	}
	ready = ready && counts == [2]int{c.TeamSize, c.TeamSize}
	footer := "WAITING FOR PLAYERS"
	if ready {
		footer = "WAITING FOR HOST TO START"
		if g.versusHost() {
			footer = "ENTER START MATCH"
		}
	}
	uiText(s, fmt.Sprintf("PLAYERS %d / %d", len(g.LobbyMembers), c.Players()), 44, 339, 1, uiMuted)
	if c.TeamSize == 2 {
		uiButton(s, "T CHANGE TEAM", 388, 329, 144, 27, ebiten.KeyT, uiGold)
	}
	uiFooter(s, footer, "ESC LEAVE ROOM")
}
func (g *Game) drawVersusSidebar(s *ebiten.Image) {
	uiRect(s, 416, 0, 160, 416, uiBG)
	uiRect(s, 416, 0, 2, 416, uiGold)
	uiText(s, "SCORE", 430, 22, 1, uiMuted)
	uiText(s, "GREEN", 430, 50, 1, uiGreen)
	uiText(s, fmt.Sprintf("%02d", g.Match.Scores[1]), 506, 44, 3, uiGreen)
	uiText(s, "YELLOW", 430, 92, 1, uiGold)
	uiText(s, fmt.Sprintf("%02d", g.Match.Scores[0]), 506, 86, 3, uiGold)
	uiRect(s, 430, 128, 130, 1, uiLine)
	uiText(s, "HITS", 430, 149, 1, uiMuted)
	for id := 0; id < g.Match.Config.Players(); id++ {
		hits := 0
		for _, t := range g.Tanks {
			if t.PlayerID == id {
				hits = t.Hits
				break
			}
		}
		y := 182 + id*54
		c := uiGold
		if g.playerTeam(id) == 1 {
			c = uiGreen
		}
		uiText(s, g.playerName(id), 430, y, 1, c)
		uiText(s, fmt.Sprintf("%d / 3", hits), 526, y, 1, uiInk)
		for i := 0; i < 3; i++ {
			col := uiLine
			if i < hits {
				col = uiRed
			}
			uiRect(s, 430+i*44, y+17, 34, 5, col)
		}
	}
}
func (g *Game) drawVersusFinal(s *ebiten.Image) {
	uiBackground(s, "VERSUS / MATCH COMPLETE", "RESULT")
	winner := "YELLOW TEAM WINS"
	if g.Match.Winner == 1 {
		winner = "GREEN TEAM WINS"
	}
	uiFrame(s, 44, 140, 488, 182, uiGold)
	winnerColor := uiGold
	if g.Match.Winner == 1 {
		winnerColor = uiGreen
	}
	uiText(s, winner, 70, 163, 3, winnerColor)
	uiText(s, "YELLOW", 97, 218, 1, uiGold)
	uiText(s, "GREEN", 374, 218, 1, uiGreen)
	uiText(s, fmt.Sprintf("%02d", g.Match.Scores[0]), 110, 246, 5, uiGold)
	uiText(s, "-", 277, 256, 3, uiMuted)
	uiText(s, fmt.Sprintf("%02d", g.Match.Scores[1]), 376, 246, 5, uiGreen)
	for team := 0; team < 2; team++ {
		names := []string{}
		for id := 0; id < g.Match.Config.Players(); id++ {
			if g.playerTeam(id) == team {
				names = append(names, g.playerName(id))
			}
		}
		c := uiGold
		if team == 1 {
			c = uiGreen
		}
		uiText(s, strings.Join(names, " / "), 60+team*246, 337, 1, c)
	}
	text := "WAITING FOR HOST"
	if g.versusHost() {
		text = "ENTER RETURN TO LOBBY"
	}
	uiFooter(s, text, "ESC LEAVE ROOM")
}
