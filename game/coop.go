package game

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
)

type coopSnapshot struct {
	Enemies              []Enemy
	Items                []Item
	CurrentStage         int
	Score                int
	HighScore            int
	SpawnedEnemyCount    int
	TotalEnemiesPerStage int
	FreezeTimer          int
	StageClearTimer      int
	ResultCountFrame     int
	DeathResultTimer     int
	StageStats           StageStats
	DifficultyBonus      int
	TotalKills           int
	ClearedStages        int
	EndReason            string
	MapSeed              int64
	MapDifficulty        Difficulty
	IsPaused             bool
	LastMessage          string
	MessageTimer         int
}

func coopSpawn(id int) (float32, float32) {
	switch id {
	case 1:
		return 144, 384
	case 2:
		return 96, 384
	case 3:
		return 288, 384
	default:
		return 240, 384
	}
}
func (g *Game) hostCoopRoom() {
	s, err := HostCoop(TCPHostPort, RoomInfo{RoomName: g.Menu.RoomName})
	if err != nil {
		g.setMessage("CANNOT HOST / PORT IN USE")
		return
	}
	g.Coop = s
	g.Menu.SelectedMode = ModeCoop
	g.LobbyMembers = []int{0}
	g.Tanks = nil
	g.Enemies = nil
	g.Bullets = nil
	g.Items = nil
	g.Explosions = nil
	g.State = StateCoopLobby
	g.ShowExitConfirm = false
	g.NetMgr.SetRoomInfo(s.Info())
	g.NetMgr.StartBroadcasting()
}
func (g *Game) joinCoopRoom(ip string) {
	s, err := JoinVersus(ip + TCPHostPort)
	if err != nil {
		g.setMessage("ROOM FULL / STARTED / UNAVAILABLE")
		return
	}
	if s.Info().Mode != int(ModeCoop) {
		s.Close()
		g.setMessage("ROOM MODE CHANGED")
		return
	}
	g.Coop = s
	g.Menu.SelectedMode = ModeCoop
	g.State = StateCoopLobby
	g.LobbyMembers = nil
	g.ShowExitConfirm = false
}
func (g *Game) closeCoopRoom(message string) {
	if g.Coop != nil {
		if g.Coop.Host {
			g.NetMgr.StopBroadcasting()
		}
		g.Coop.Close()
		g.Coop = nil
	}
	g.State = StateRoomSelect
	g.ShowExitConfirm = false
	g.IsPaused = false
	g.Tanks = nil
	g.coopInputs = nil
	if message != "" {
		g.setMessage(message)
	}
}
func (g *Game) beginCoopBattle() {
	s := g.Coop
	g.startBattle(false)
	g.Tanks = nil
	for _, id := range s.Connected() {
		x, y := coopSpawn(id)
		g.Tanks = append(g.Tanks, NewTank(x, y, id))
	}
	g.loadStage(0)
	g.State = StatePlaying
}
func (g *Game) updateCoop() {
	s := g.Coop
	if g.State == StateCoopLobby {
		if actionPressed(ebiten.KeyEscape) {
			g.closeCoopRoom("")
			return
		}
		if s.Host && actionPressed(ebiten.KeyEnter) && s.Begin() {
			g.beginCoopBattle()
		}
	}
	if !s.Host {
		input := readVersusInput()
		if g.State != StateCoopLobby && actionPressed(ebiten.KeyEscape) {
			g.ShowExitConfirm = !g.ShowExitConfirm
		}
		if g.ShowExitConfirm {
			input = VersusInput{}
			if actionPressed(ebiten.KeyY) {
				g.closeCoopRoom("")
				return
			}
			if actionPressed(ebiten.KeyN) {
				g.ShowExitConfirm = false
			}
		}
		s.SendInput(input)
		snap, failure := s.Receive()
		if failure != "" {
			g.closeCoopRoom("HOST DISCONNECTED")
			return
		}
		if snap != nil {
			g.applyCoopSnapshot(snap)
		}
		if g.State == StateGameOver && g.ResultCountFrame >= g.resultCountDuration() && actionPressed(ebiten.KeyEnter) {
			g.closeCoopRoom("")
		}
		return
	}
	g.LobbyMembers = s.Connected()
	g.coopInputs = s.Inputs()
	g.coopInputs[0] = readVersusInput()
	// Disconnected slots are removed from this run; the locked room never admits replacements.
	for _, t := range g.Tanks {
		connected := false
		for _, id := range g.LobbyMembers {
			if t.PlayerID == id {
				connected = true
			}
		}
		if !connected {
			t.Active = false
			t.Lives = 0
		}
	}
	switch g.State {
	case StateStageIntro:
		if actionPressed(ebiten.KeyEnter) {
			g.State = StatePlaying
		}
	case StatePlaying:
		g.updatePlaying()
	case StateStageCleanup:
		g.updateStageCleanup()
	case StateStageClear:
		if actionPressed(ebiten.KeyEscape) {
			g.finishRun("RUN ENDED")
		} else if g.advanceResultCount(actionPressed(ebiten.KeyEnter)) {
			g.nextStage()
		}
	case StateDeathResult:
		g.tickDefeatIntro()
	case StateGameOver:
		if g.advanceResultCount(actionPressed(ebiten.KeyEnter)) {
			g.closeCoopRoom("")
			return
		}
	}
	g.NetMgr.SetRoomInfo(s.Info())
	if g.Frame%2 == 0 {
		s.Publish(g.coopSnapshot())
	}
}
func (g *Game) coopSnapshot() *versusSnapshot {
	snap := g.versusSnapshot()
	c := &coopSnapshot{}
	c.CurrentStage = g.CurrentStage
	c.Score = g.Score
	c.HighScore = g.HighScore
	c.SpawnedEnemyCount = g.SpawnedEnemyCount
	c.TotalEnemiesPerStage = g.TotalEnemiesPerStage
	c.FreezeTimer = g.FreezeTimer
	c.StageClearTimer = g.StageClearTimer
	c.ResultCountFrame = g.ResultCountFrame
	c.DeathResultTimer = g.DeathResultTimer
	c.StageStats = g.StageStats
	c.DifficultyBonus = g.DifficultyBonus
	c.TotalKills = g.TotalKills
	c.ClearedStages = g.ClearedStages
	c.EndReason = g.EndReason
	c.MapSeed = g.MapSeed
	c.MapDifficulty = g.MapDifficulty
	c.IsPaused = g.IsPaused
	c.LastMessage = g.LastMessage
	c.MessageTimer = g.MessageTimer

	for _, e := range g.Enemies {
		c.Enemies = append(c.Enemies, *e)
	}
	for _, i := range g.Items {
		c.Items = append(c.Items, *i)
	}
	snap.Coop = c
	return snap
}
func (g *Game) applyCoopSnapshot(snap *versusSnapshot) {
	if snap.Coop == nil {
		return
	}
	g.applyVersusSnapshot(snap)
	c := snap.Coop
	g.CurrentStage = c.CurrentStage
	g.Score = c.Score
	g.HighScore = c.HighScore
	g.SpawnedEnemyCount = c.SpawnedEnemyCount
	g.TotalEnemiesPerStage = c.TotalEnemiesPerStage
	g.FreezeTimer = c.FreezeTimer
	g.StageClearTimer = c.StageClearTimer
	g.ResultCountFrame = c.ResultCountFrame
	g.DeathResultTimer = c.DeathResultTimer
	g.StageStats = c.StageStats
	g.DifficultyBonus = c.DifficultyBonus
	g.TotalKills = c.TotalKills
	g.ClearedStages = c.ClearedStages
	g.EndReason = c.EndReason
	g.MapSeed = c.MapSeed
	g.MapDifficulty = c.MapDifficulty
	g.IsPaused = c.IsPaused
	g.LastMessage = c.LastMessage
	g.MessageTimer = c.MessageTimer

	g.Enemies = nil
	for _, e := range c.Enemies {
		e := e
		g.Enemies = append(g.Enemies, &e)
	}
	g.Items = nil
	for _, i := range c.Items {
		i := i
		g.Items = append(g.Items, &i)
	}
}
func (g *Game) drawCoopLobby(s *ebiten.Image) {
	uiBackground(s, "COOP / WAITING ROOM", "COOP LOBBY")
	uiText(s, fmt.Sprintf("PLAYERS %d / 4", len(g.LobbyMembers)), 44, 122, 1, uiGreen)
	for id := 0; id < 4; id++ {
		present := false
		for _, member := range g.LobbyMembers {
			if member == id {
				present = true
			}
		}
		y := 145 + id*45
		c := uiMuted
		label := "EMPTY"
		if present {
			c = uiGreen
			label = fmt.Sprintf("PLAYER %d", id+1)
			if id == 0 {
				label += " / HOST"
			}
			if g.Coop != nil && id == g.Coop.LocalID {
				label += " / YOU"
			}
		}
		uiFrame(s, 44, y, 488, 37, c)
		uiTank(s, 58, y+7, 1, c)
		uiText(s, label, 88, y+12, 2, c)
	}

	action := "WAITING FOR HOST"
	if g.Coop == nil || g.Coop.Host {
		action = "ENTER START GAME"
	}
	uiFooter(s, action, "ESC LEAVE ROOM")
}
