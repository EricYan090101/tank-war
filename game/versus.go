package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"math/rand"
)

func (g *Game) hostVersusRoom() {
	config := VersusConfig{TeamSize: g.Menu.TeamSize, BestOf: 3 + g.Menu.SeriesIndex*2}
	room := RoomInfo{RoomName: g.Menu.RoomName, Mode: int(ModeVersus), TeamSize: config.TeamSize, BestOf: config.BestOf, MaxPlayers: config.Players(), CurrentPlayers: 1}
	session, err := HostVersus(TCPHostPort, room, g.Menu.PlayerName)
	if err != nil {
		g.setMessage("CANNOT HOST / PORT IN USE")
		return
	}
	if config.TeamSize == 2 {
		session.SelectTeam(0, 1-g.Menu.TeamCursor)
	}
	g.ChoosingTeam = false
	g.RequestedTeam = -1
	g.Versus = session
	g.Match = VersusMatch{Config: config, Winner: -1}
	g.Match.Names = session.Names()
	g.Match.Teams = session.Teams()
	g.Match.TeamsChosen = true
	g.LobbyMembers = []int{0}
	g.RunSeed = rand.Int63()
	g.State = StateVersusLobby
	g.NetMgr.SetRoomInfo(room)
	g.NetMgr.StartBroadcasting()
	g.LastMessage = ""
	g.ShowExitConfirm = false
	g.IsPaused = false
}
func (g *Game) joinVersusRoom(ip string) {
	session, err := JoinVersus(ip+TCPHostPort, g.Menu.PlayerName)
	if err != nil {
		g.setMessage("ROOM UNAVAILABLE / TRY AGAIN")
		return
	}
	g.Versus = session
	info := session.Info()
	g.Menu.SelectedMode = ModeVersus
	g.Match = VersusMatch{Config: VersusConfig{info.TeamSize, info.BestOf}, Winner: -1}
	g.State = StateVersusLobby
	g.Match.Teams = [4]int{-1, -1, -1, -1}
	g.Match.TeamsChosen = true
	g.ChoosingTeam = info.TeamSize == 2
	g.RequestedTeam = -1
	g.LobbyMembers = nil
	g.Explosions = nil
	g.Tanks = nil
	g.Bullets = nil
	g.LastMessage = ""
	g.ShowExitConfirm = false
}
func (g *Game) closeVersusRoom(message string) {
	if g.Versus != nil {
		if g.Versus.Host {
			g.NetMgr.StopBroadcasting()
		}
		g.Versus.Close()
		g.Versus = nil
		g.ChoosingTeam = false
		g.RequestedTeam = -1
	}
	g.Tanks = nil
	g.Bullets = nil
	g.State = StateRoomSelect
	g.ShowExitConfirm = false
	if message != "" {
		g.setMessage(message)
	}
}
func readVersusInput() VersusInput {
	held := [4]int{}
	for i, k := range []ebiten.Key{ebiten.KeyW, ebiten.KeyS, ebiten.KeyA, ebiten.KeyD} {
		held[i] = inpututil.KeyPressDuration(k)
	}
	dir, moving := preferredDirection(held, DirUp)
	return VersusInput{Dir: dir, Moving: moving, Fire: ebiten.IsKeyPressed(ebiten.KeySpace)}
}
func (g *Game) updateVersus() {
	s := g.Versus
	wasChoosingTeam := g.ChoosingTeam
	if actionPressed(ebiten.KeyEscape) {
		if g.State == StateVersusLobby || g.State == StateVersusMatchEnd {
			g.closeVersusRoom("")
			return
		}
		g.ShowExitConfirm = !g.ShowExitConfirm
	}
	if g.ShowExitConfirm {
		if actionPressed(ebiten.KeyY) {
			g.closeVersusRoom("")
			return
		}
		if actionPressed(ebiten.KeyN) {
			g.ShowExitConfirm = false
		}
	}
	if g.State == StateVersusLobby && g.Match.Config.TeamSize == 2 {
		if actionPressed(ebiten.KeyT) {
			g.ChoosingTeam = true
			g.RequestedTeam = -1
		}
		if g.ChoosingTeam && cardChoice(&g.Menu.TeamCursor, 2, 137, 92, 76) {
			g.RequestedTeam = 1 - g.Menu.TeamCursor
			if s.Host {
				if s.SelectTeam(0, g.RequestedTeam) {
					g.ChoosingTeam = false
					g.RequestedTeam = -1
				} else {
					g.setMessage("TEAM FULL / CHOOSE THE OTHER SIDE")
				}
			}
		}
	}
	input := readVersusInput()
	if g.versusRotated() {
		input.Dir = oppositeDirection(input.Dir)
	}
	if g.ChoosingTeam {
		input = VersusInput{}
		if g.RequestedTeam >= 0 {
			input.TeamChoice = g.RequestedTeam + 1
		}
	}
	if g.ShowExitConfirm {
		input = VersusInput{}
	}
	if !s.Host {
		s.SendInput(input)
		snap, failure := s.Receive()
		if failure != "" {
			g.closeVersusRoom(failure)
			return
		}
		if snap != nil {
			g.applyVersusSnapshot(snap)
		}
		return
	}
	g.LobbyMembers = s.Connected()
	g.Match.Names = s.Names()
	g.Match.Teams = s.Teams()
	g.Match.TeamsChosen = true
	if g.State != StateVersusLobby && len(g.LobbyMembers) != g.Match.Config.Players() {
		g.Match.Scores = [2]int{}
		g.Match.Round = 0
		g.Tanks = nil
		g.Bullets = nil
		g.State = StateVersusLobby
		g.ShowExitConfirm = false
		s.UnlockLobby()
		g.setMessage("PLAYER LEFT / MATCH RESET")
	}
	switch g.State {
	case StateVersusLobby:
		if !wasChoosingTeam && !g.ChoosingTeam && actionPressed(ebiten.KeyEnter) && s.Begin() {
			g.startVersusRound()
		}
	case StateVersusCountdown:
		g.Match.Timer--
		if g.Match.Timer <= 0 {
			g.State = StateVersusPlaying
		}
	case StateVersusPlaying:
		inputs := s.Inputs()
		inputs[0] = input
		g.simulateVersus(inputs)
	case StateVersusRoundEnd:
		g.updateExplosions()
		g.Match.Timer--
		if g.Match.Timer <= 0 {
			g.startVersusRound()
		}
	case StateVersusMatchEnd:
		if actionPressed(ebiten.KeyEnter) {
			g.Match.Scores = [2]int{}
			g.Match.Round = 0
			g.State = StateVersusLobby
			s.UnlockLobby()
		}
	}
	g.NetMgr.SetRoomInfo(s.Info())
	if g.Frame%2 == 0 {
		s.Publish(g.versusSnapshot())
	}
}
func (g *Game) versusSnapshot() *versusSnapshot {
	snap := &versusSnapshot{State: g.State, Match: g.Match, Connected: append([]int(nil), g.LobbyMembers...), Teams: g.Match.Teams, Frame: g.Frame}
	if g.Map != nil {
		snap.Tiles = g.Map.Tiles
		snap.Bricks = g.Map.BrickMask
	}
	for _, t := range g.Tanks {
		snap.Tanks = append(snap.Tanks, *t)
	}
	for _, b := range g.Bullets {
		copyBullet := *b
		copyBullet.Emitter = nil // Host-only identity; never share mutable enemy pointers with socket writers.
		snap.Bullets = append(snap.Bullets, copyBullet)
	}
	for _, e := range g.Explosions {
		snap.Explosions = append(snap.Explosions, *e)
	}
	return snap
}
func (g *Game) applyVersusSnapshot(snap *versusSnapshot) {
	if g.State != snap.State && snap.State == StateVersusLobby {
		g.ShowExitConfirm = false
	}
	g.State = snap.State
	if snap.State != StateVersusLobby {
		g.ChoosingTeam = false
		g.RequestedTeam = -1
	}
	g.Explosions = nil
	for i := range snap.Explosions {
		e := snap.Explosions[i]
		g.Explosions = append(g.Explosions, &e)
	}
	g.Match = snap.Match
	if g.State == StateVersusLobby && g.ChoosingTeam && g.RequestedTeam >= 0 && g.Versus != nil && snap.Teams[g.Versus.LocalID] == g.RequestedTeam {
		g.ChoosingTeam = false
		g.RequestedTeam = -1
	}
	g.LobbyMembers = snap.Connected
	if g.Map == nil {
		g.Map = NewMap()
	}
	g.Map.Tiles = snap.Tiles
	g.Map.BrickMask = snap.Bricks
	g.Tanks = nil
	for i := range snap.Tanks {
		t := snap.Tanks[i]
		g.Tanks = append(g.Tanks, &t)
	}
	g.Bullets = nil
	for i := range snap.Bullets {
		b := snap.Bullets[i]
		g.Bullets = append(g.Bullets, &b)
	}
}

// The server keeps world coordinates; only local input and rendering rotate.
func oppositeDirection(d Direction) Direction {
	switch d {
	case DirUp:
		return DirDown
	case DirDown:
		return DirUp
	case DirLeft:
		return DirRight
	default:
		return DirLeft
	}
}
func (g *Game) versusRotated() bool {
	id := 0
	if g.Versus != nil {
		id = g.Versus.LocalID
	}
	return g.playerTeam(id) == 1
}
func (g *Game) versusPosition(x, y, size float32) (float32, float32) {
	if g.versusRotated() {
		return 416 - size - x, 416 - size - y
	}
	return x, y
}
func (g *Game) playerName(id int) string {
	if id >= 0 && id < 4 && g.Match.Names[id] != "" {
		return g.Match.Names[id]
	}
	return "PLAYER"
}
