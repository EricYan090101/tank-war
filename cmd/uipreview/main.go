// uipreview exports the actual Ebitengine screens for visual review.
// Run from the project root: go run ./cmd/uipreview
package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"tank-war/game"
)

type scene struct {
	name  string
	state game.GameState
}

var scenes = []scene{
	{"01-title", game.StateTitle}, {"02-mode", game.StateModeSelect},
	{"03-room", game.StateRoomSelect}, {"04-name", game.StateRoomNameInput},
	{"05-search", game.StateRoomListSelect}, {"06-rooms", game.StateRoomListSelect},
	{"07-briefing", game.StateStageIntro}, {"08-battle", game.StatePlaying},
	{"09-pause", game.StatePlaying}, {"10-exit", game.StatePlaying},
	{"11-cleanup", game.StateStageCleanup}, {"12-clear", game.StateStageClear},
	{"13-final", game.StateGameOver},
	{"14-counting", game.StateStageClear}, {"15-count-bonus", game.StateStageClear}, {"16-count-animation", game.StateStageClear},
	{"17-versus-settings", game.StateVersusSettings},
	{"18-versus-lobby", game.StateVersusLobby},
	{"19-versus-countdown", game.StateVersusCountdown},
	{"20-versus-battle", game.StateVersusPlaying},
	{"21-versus-round", game.StateVersusRoundEnd},
	{"22-versus-final", game.StateVersusMatchEnd},
	{"25-player-name", game.StatePlayerNameInput}, {"26-green-view", game.StateVersusPlaying}, {"27-team-occupancy", game.StateVersusLobby},
	{"23-team-select", game.StateVersusTeamSelect}, {"24-series", game.StateVersusSeries},
	{"28-coop-lobby", game.StateCoopLobby},
	{"29-new-materials", game.StateVersusPlaying},
	{"30-gray-enemies", game.StateVersusPlaying},
	{"31-iron-shields", game.StateVersusPlaying},
	{"32-new-base", game.StatePlaying},
	{"33-game-over-fade", game.StateDeathResult},
	{"34-game-over-full", game.StateDeathResult},
}

type preview struct {
	g              *game.Game
	index          int
	err            error
	animation      gif.GIF
	animationFrame int
}

func (p *preview) Update() error {
	if p.err != nil {
		return p.err
	}
	if p.index == len(scenes) {
		return ebiten.Termination
	}
	return nil
}
func (p *preview) Draw(s *ebiten.Image) {
	if p.index >= len(scenes) || p.err != nil {
		return
	}
	scene := scenes[p.index]
	g := p.g
	g.State = scene.state
	g.Coop = nil
	g.Versus = nil
	g.ChoosingTeam = false
	g.Menu.PlayerName = "TankAce"
	g.IsPaused = scene.name == "09-pause"
	g.ShowExitConfirm = scene.name == "10-exit"
	if scene.name == "06-rooms" {
		g.Menu.SelectedMode = game.ModeVersus
		for i := 0; i < 7; i++ {
			ip := fmt.Sprintf("192.168.1.%d", 20+i)
			g.NetMgr.DiscoveredRooms[ip] = game.RoomInfo{RoomName: fmt.Sprintf("FIELD TEAM %02d", i+1), HostIP: ip, Mode: int(game.ModeVersus), TeamSize: 1 + i%2, BestOf: 3 + 2*(i%3), CurrentPlayers: 1 + i%2, MaxPlayers: 2 * (1 + i%2), InProgress: i == 3}
		}
		g.Menu.RoomListCursor = 5
	}
	if scene.state == game.StateStageCleanup {
		g.Enemies = nil
		g.SpawnedEnemyCount = 20
	}
	g.ResultCountFrame = 0
	switch scene.name {
	case "12-clear":
		g.ResultCountFrame = 999
	case "14-counting":
		g.ResultCountFrame = 72
	case "15-count-bonus":
		g.ResultCountFrame = 220
	case "16-count-animation":
		g.ResultCountFrame = p.animationFrame
	}
	g.MessageTimer = 0
	g.FreezeTimer = 0
	if scene.name == "08-battle" {
		g.MessageTimer = 90
		g.LastMessage = "STAR 2 / DOUBLE SHOT"
		g.FreezeTimer = 180
	}
	if scene.state >= game.StateVersusSettings {
		g.Menu.SelectedMode = game.ModeVersus
		g.Menu.TeamSize = 2
		g.Menu.ConfigCursor = 1
		g.Menu.TeamCursor = 0
		g.Menu.SeriesIndex = 1
		g.Match = game.VersusMatch{Config: game.VersusConfig{TeamSize: 2, BestOf: 5}, Scores: [2]int{2, 1}, Round: 4, Winner: 0, Timer: 120}
		g.Match.Names = [4]string{"YellowAce", "GreenAce", "Sunny", "Forest"}
		g.LobbyMembers = []int{0, 1, 2, 3}
		g.Map.GenerateVersus(781)
		g.Bullets = nil
		g.Explosions = nil
		g.Tanks = nil
		for id := 0; id < 4; id++ {
			x := float32(128 + (id/2)*128)
			y := float32(384)
			dir := game.DirUp
			if id%2 == 1 {
				y = 0
				dir = game.DirDown
			}
			t := game.NewTank(x, y, id)
			t.Team = id % 2
			t.MatchTank = true
			t.Dir = dir
			t.Hits = id % 3
			g.Tanks = append(g.Tanks, t)
		}
		if scene.state == game.StateVersusMatchEnd {
			g.Match.Scores = [2]int{3, 1}
		}
	}
	if scene.name == "26-green-view" {
		g.Versus = &game.VersusSession{LocalID: 1}
	}
	if scene.name == "27-team-occupancy" {
		g.Versus = &game.VersusSession{LocalID: 3}
		g.ChoosingTeam = true
		g.LobbyMembers = []int{0, 1, 2}
		g.Match.TeamsChosen = true
		g.Match.Teams = [4]int{0, 1, 0, -1}
	}
	if scene.name == "28-coop-lobby" {
		g.LobbyMembers = []int{0, 1, 2}
	}
	if scene.name == "29-new-materials" {
		g.Map.Tiles = [game.MapRows][game.MapCols]int{}
		g.Map.BrickMask = [game.MapRows][game.MapCols]uint8{}
		for r := 2; r < 10; r++ {
			for c := 2; c < 12; c++ {
				g.Map.Tiles[r][c] = game.TileWater
			}
			for c := 14; c < 24; c++ {
				g.Map.Tiles[r][c] = game.TileBrick
				g.Map.BrickMask[r][c] = game.BrickFull
			}
		}
		for r := 14; r < 24; r++ {
			for c := 2; c < 10; c++ {
				g.Map.Tiles[r][c] = game.TileSteel
			}
			for c := 16; c < 24; c++ {
				g.Map.Tiles[r][c] = game.TileBush
			}
		}
		for r := 10; r < 13; r++ {
			for c := 2; c < 12; c++ {
				g.Map.Tiles[r][c] = game.TileIce
			}
		}
		g.Tanks[0].X = 176
		g.Tanks[0].Y = 224
		g.Tanks[0].Dir = game.DirUp
		g.Tanks[1].X = 176
		g.Tanks[1].Y = 288
		g.Tanks[1].Dir = game.DirLeft
		g.Tanks[2].X = 288
		g.Tanks[2].Y = 288
		g.Tanks[2].Dir = game.DirRight
		g.Tanks[3].X = 176
		g.Tanks[3].Y = 352
		g.Tanks[3].Dir = game.DirDown
		g.Map.BrickMask[9][14] = game.BrickTL | game.BrickBR
	}
	if scene.name == "32-new-base" || scene.name == "33-game-over-fade" || scene.name == "34-game-over-full" {
		g.Map = game.NewMap()
		g.Map.GenerateRandom(42)
		if scene.name != "32-new-base" {
			g.Map.HitBullet(193, 385, game.DirDown, 1, false, 0)
			g.DeathResultTimer = 30
			if scene.name == "34-game-over-full" {
				g.DeathResultTimer = 100
			}
		}
	}
	g.Draw(s)
	if scene.name == "31-iron-shields" {
		s.Fill(color.Black)
		for row, dir := range []game.Direction{game.DirUp, game.DirDown, game.DirLeft, game.DirRight} {
			for col, hits := range []int{3, 2, 1, 0} {
				x, y := float32(40+col*128), float32(28+row*96)
				tank := game.NewTank(x, y, 0)
				tank.Dir = dir
				tank.ArmorShield = hits
				tank.Draw(s)
				enemy := game.NewEnemy(x+48, y, game.EnemyBasic, 1)
				enemy.SpawnTime = 0
				enemy.Dir = dir
				enemy.ArmorShield = hits
				enemy.Draw(s, g.EnemySprites)
			}
		}
	}
	if scene.name == "30-gray-enemies" {
		s.Fill(color.Black)
		for row, typ := range []game.EnemyType{game.EnemyBasic, game.EnemyPower, game.EnemyFast} {
			for col, dir := range []game.Direction{game.DirUp, game.DirDown, game.DirLeft, game.DirRight, game.DirUp} {
				e := game.NewEnemy(float32(48+col*96), float32(40+row*88), typ, 1)
				e.SpawnTime = 0
				e.Dir = dir
				if col == 4 {
					e.Flash = 5
				}
				e.Draw(s, g.EnemySprites)
			}
		}
	}
	if scene.name == "29-new-materials" {
		for typ := game.ItemStar; typ <= game.ItemTank; typ++ {
			item := &game.Item{X: float32(32 + int(typ)*56), Y: 208, Type: typ, Active: true, Timer: 600}
			item.Draw(s, g.ItemSprites, 0)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, game.ScreenWidth, game.ScreenHeight))
	s.ReadPixels(img.Pix)
	if scene.name == "16-count-animation" {
		frame := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(frame, frame.Bounds(), img, image.Point{})
		p.animation.Image = append(p.animation.Image, frame)
		p.animation.Delay = append(p.animation.Delay, 10)
		if p.animationFrame < 300 {
			p.animationFrame += 6
			return
		}
		f, err := os.Create(filepath.Join("previews", "16-count-animation.gif"))
		if err != nil {
			p.err = err
			return
		}
		p.err = gif.EncodeAll(f, &p.animation)
		if err := f.Close(); p.err == nil {
			p.err = err
		}
		p.index++
		return
	}
	f, err := os.Create(filepath.Join("previews", scene.name+".png"))
	if err != nil {
		p.err = err
		return
	}
	p.err = png.Encode(f, img)
	if err := f.Close(); p.err == nil {
		p.err = err
	}
	p.index++
}
func (p *preview) Layout(int, int) (int, int) { return game.ScreenWidth, game.ScreenHeight }
func main() {
	if err := os.MkdirAll("previews", 0755); err != nil {
		panic(err)
	}
	g := game.NewGame()
	g.Map = game.NewMap()
	g.MapSeed = 98172181
	g.MapDifficulty = g.Map.GenerateRandom(g.MapSeed)
	g.CurrentStage = 2
	g.Score = 8450
	g.HighScore = 12700
	g.TotalEnemiesPerStage = 20
	g.SpawnedEnemyCount = 6
	g.StageStats = game.StageStats{Kills: [4]int{12, 4, 2, 2}, StageBonus: 1000, MapBonus: 850, ItemsCollected: [6]int{2, 1, 0, 1, 0, 0}}
	g.Tanks = []*game.Tank{game.NewTank(game.PlayerSpawnX, game.PlayerSpawnY, 0)}
	g.Tanks[0].Power = 3
	for i, x := range []float32{0, 192, 384} {
		e := game.NewEnemy(x, 0, game.EnemyType(i), 0)
		e.SpawnTime = 0
		g.Enemies = append(g.Enemies, e)
	}
	g.StageClearTimer = 120
	g.ClearedStages = 3
	g.TotalKills = 60
	g.DifficultyBonus = 1450
	g.EndReason = "ALL LIVES LOST"
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowPosition(-10000, -10000)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGameWithOptions(&preview{g: g}, &ebiten.RunGameOptions{SkipTaskbar: true, InitUnfocused: true}); err != nil {
		panic(err)
	}
	fmt.Printf("Exported %d screens to previews/\n", len(scenes))
}
