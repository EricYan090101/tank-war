package game

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type StageStats struct {
	Kills          [4]int
	ItemsCollected [6]int
	BombKills      int
	StartScore     int
	StageBonus     int
	MapBonus       int
}

func (s *StageStats) TotalKills() int {
	return s.Kills[0] + s.Kills[1] + s.Kills[2] + s.Kills[3]
}

type Game struct {
	Coop            *VersusSession
	coopInputs      map[int]VersusInput
	versusCanvas    *ebiten.Image
	ChoosingTeam    bool
	RequestedTeam   int
	Versus          *VersusSession
	Match           VersusMatch
	LobbyMembers    []int
	RunSeed         int64
	MapSeed         int64
	MapDifficulty   Difficulty
	DifficultyBonus int
	TotalKills      int
	ClearedStages   int
	stageSettled    bool
	runFinished     bool
	EndReason       string
	State           GameState
	Menu            *MenuUI
	NetMgr          *NetworkManager
	Tanks           []*Tank
	Enemies         []*Enemy
	Bullets         []*Bullet
	Items           []*Item
	Map             *Map

	CurrentStage         int
	SpawnTimer           int
	NextEnemySpawn       int
	SpawnedEnemyCount    int
	TotalEnemiesPerStage int
	FreezeTimer          int
	StageIntroTimer      int
	StageClearTimer      int
	ResultCountFrame     int
	Score                int
	HighScore            int
	Frame                int
	StageStats           StageStats
	Explosions           []*Explosion
	StageItemCount       int
	GameOverTimer        int
	DeathResultTimer     int

	EnemySprites    map[EnemyType]*ebiten.Image
	ItemSprites     map[ItemType]*ebiten.Image
	ShowExitConfirm bool
	clientConn      net.Conn
	LastMessage     string
	MessageTimer    int
	IsPaused        bool
}

func NewGame() *Game {
	rand.Seed(time.Now().UnixNano())
	g := &Game{
		State:                StateTitle,
		Menu:                 NewMenuUI(),
		NetMgr:               NewNetworkManager(),
		TotalEnemiesPerStage: EnemiesPerStage,
	}
	g.EnemySprites = LoadEnemySprites()
	g.ItemSprites = LoadItemSprites()
	return g
}

func (g *Game) Update() error {
	g.Frame++
	if g.MessageTimer > 0 && g.State != StatePlaying && g.State != StateStageCleanup {
		g.MessageTimer--
	}
	if g.Coop != nil {
		g.updateCoop()
		return nil
	}
	if g.Versus != nil {
		g.updateVersus()
		return nil
	}
	switch g.State {
	case StateTitle:
		if actionPressed(ebiten.KeyEnter) {
			g.State = StateModeSelect
		}
	case StateModeSelect:
		if actionPressed(ebiten.KeyEscape) {
			g.State = StateTitle
			break
		}
		if g.Menu.UpdateModeSelect() {
			g.State = StateRoomSelect
			if g.Menu.SelectedMode == ModeVersus {
				g.State = StatePlayerNameInput
			}
		}
	case StatePlayerNameInput:
		if actionPressed(ebiten.KeyEscape) {
			g.State = StateModeSelect
			break
		}
		if g.Menu.UpdatePlayerName() {
			g.State = StateRoomSelect
		}
	case StateRoomSelect:
		if actionPressed(ebiten.KeyEscape) {
			g.State = StateModeSelect
			break
		}
		if g.Menu.UpdateRoomSelect() {
			if g.Menu.SelectedRole == RoleHost {
				g.State = StateRoomNameInput
			} else {
				g.NetMgr.ClearDiscoveredRooms()
				g.NetMgr.StartListeningRooms()
				g.State = StateRoomListSelect
			}
		}
	case StateRoomNameInput:
		if actionPressed(ebiten.KeyEscape) {
			g.State = StateRoomSelect
			break
		}
		if g.Menu.UpdateRoomNameInput() {
			if g.Menu.SelectedMode == ModeVersus {
				g.State = StateVersusSettings
			} else {
				g.hostCoopRoom()
			}
		}
	case StateRoomListSelect:
		if actionPressed(ebiten.KeyEscape) {
			g.State = StateRoomSelect
			break
		}
		rooms := g.availableRooms()
		idx := g.Menu.UpdateRoomListSelect(len(rooms))
		if idx >= 0 && idx < len(rooms) {
			if GameMode(rooms[idx].Mode) == ModeVersus {
				g.joinVersusRoom(rooms[idx].HostIP)
			} else {
				g.joinCoopRoom(rooms[idx].HostIP)
			}
		}
	case StateVersusSettings:
		if actionPressed(ebiten.KeyEscape) {
			g.State = StateRoomNameInput
			break
		}
		if g.Menu.UpdateVersusSettings() {
			if g.Menu.TeamSize == 2 {
				g.State = StateVersusTeamSelect
			} else {
				g.State = StateVersusSeries
			}
		}
	case StateVersusTeamSelect:
		if actionPressed(ebiten.KeyEscape) {
			g.State = StateVersusSettings
			break
		}
		if cardChoice(&g.Menu.TeamCursor, 2, 137, 92, 76) {
			g.State = StateVersusSeries
		}
	case StateVersusSeries:
		if actionPressed(ebiten.KeyEscape) {
			if g.Menu.TeamSize == 2 {
				g.State = StateVersusTeamSelect
			} else {
				g.State = StateVersusSettings
			}
			break
		}
		if cardChoice(&g.Menu.SeriesIndex, 3, 132, 72, 62) {
			g.hostVersusRoom()
		}

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
			break
		}
		if g.advanceResultCount(actionPressed(ebiten.KeyEnter)) {
			g.nextStage()
		}
	case StateDeathResult:
		g.tickDefeatIntro()
	case StateGameOver:
		if g.advanceResultCount(actionPressed(ebiten.KeyEnter)) {
			g.resetToTitle()
		}
	}
	return nil
}

func (g *Game) resetToTitle() {
	if g.Coop != nil {
		g.Coop.Close()
		g.Coop = nil
		g.NetMgr.StopBroadcasting()
	}
	if g.Versus != nil {
		g.Versus.Close()
		g.Versus = nil
		g.NetMgr.StopBroadcasting()
	}
	if g.NetMgr.IsHost {
		g.NetMgr.DestroyRoom()
	}
	if g.clientConn != nil {
		_ = g.clientConn.Close()
		g.clientConn = nil
	}
	g.State = StateTitle
	g.Score = 0
	g.IsPaused = false
	g.ShowExitConfirm = false
}

func (g *Game) startBattle(host bool) {
	g.NetMgr.IsHost = host
	g.CurrentStage = 0
	g.RunSeed = rand.Int63()
	g.DifficultyBonus, g.TotalKills, g.ClearedStages = 0, 0, 0
	g.runFinished = false
	g.EndReason = ""
	g.Score = 0
	g.ShowExitConfirm = false
	g.IsPaused = false
	g.SpawnedEnemyCount = 0
	g.Tanks = []*Tank{NewTank(PlayerSpawnX, PlayerSpawnY, 0)}
	if g.Menu.SelectedMode == ModeVersus {
		g.Tanks = append(g.Tanks, NewTank(SecondSpawnX, SecondSpawnY, 1))
	}

	g.loadStage(g.CurrentStage)
}

func (g *Game) connectToHost(hostIP string) bool {
	conn, err := net.DialTimeout("tcp", hostIP+TCPHostPort, 2*time.Second)
	if err != nil {
		g.setMessage("CONNECT FAILED")
		return false
	}
	g.clientConn = conn
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			if n > 0 && string(buf[:n]) == "ROOM_DESTROYED\n" {
				g.State = StateModeSelect
				return
			}
		}
	}()
	return true
}

func (g *Game) loadStage(stage int) {
	g.Map = NewMap()
	g.MapSeed = g.RunSeed + int64(stage)*7919
	g.MapDifficulty = g.Map.GenerateRandom(g.MapSeed)
	g.stageSettled = false
	if g.Coop != nil {
		for _, t := range g.Tanks {
			if t.PlayerID < 2 {
				continue
			}
			x, y := coopSpawn(t.PlayerID)
			for r := int(y)/16 - 2; r < MapRows; r++ {
				for c := int(x) / 16; c < int(x)/16+2; c++ {
					g.Map.Tiles[r][c] = TileEmpty
					g.Map.BrickMask[r][c] = 0
				}
			}
		}
	}

	g.Enemies = nil
	g.Bullets = nil
	g.Items = nil
	g.Explosions = nil
	g.SpawnTimer = 0
	g.NextEnemySpawn = 0
	g.SpawnedEnemyCount = 0
	g.TotalEnemiesPerStage = EnemiesPerStage
	g.FreezeTimer = 0
	g.Map.baseFortifyTimer = 0
	g.Map.BaseFortified = false
	g.StageItemCount = 0
	g.StageStats = StageStats{StartScore: g.Score}
	g.StageIntroTimer = StageIntroFrames
	g.StageClearTimer = 0
	g.ResultCountFrame = 0
	g.GameOverTimer = 0
	g.DeathResultTimer = 0
	g.LastMessage = ""
	g.MessageTimer = 0

	for i, t := range g.Tanks {
		id := i
		if t != nil {
			id = t.PlayerID
		}
		x, y := coopSpawn(id)
		if t == nil {
			g.Tanks[i] = NewTank(x, y, i)
			t = g.Tanks[i]
		}
		if t.Lives <= 0 {
			t.Lives = MaxLives
		}
		t.Respawn(x, y)
		if stage == 0 {
			t.ArmorShield = IronShieldHits
		}
	}
	g.spawnEnemy()
	g.State = StateStageIntro
}

func (g *Game) activeEnemyCount() int {
	n := 0
	for _, e := range g.Enemies {
		if e.Active {
			n++
		}
	}
	return n
}

func enemyTypeFor(stage, index int) EnemyType {
	// The first stages lean on BASIC/FAST; stronger forms are introduced gradually.
	if stage == 0 {
		if index == 4 || index == 11 {
			return EnemyFast
		}
		return EnemyBasic
	}
	if stage == 1 {
		switch index {
		case 3, 7, 12, 16:
			return EnemyFast
		case 9, 18:
			return EnemyPower
		default:
			return EnemyBasic
		}
	}
	if index%5 == 2 || index%7 == 5 {
		return EnemyFast
	}
	if index%6 == 3 {
		return EnemyPower
	}
	return EnemyBasic
}

// Spawn points cycle left, center, right; an occupied point waits its turn.
func (g *Game) spawnEnemy() bool {
	if g.SpawnedEnemyCount >= g.TotalEnemiesPerStage || g.Map == nil || (g.Menu != nil && g.Menu.SelectedMode == ModeVersus) {
		return false
	}
	points := [...]float32{0, 192, 384}
	x := points[g.NextEnemySpawn%len(points)]
	if g.Map.CheckTileCollision(x, 0, 32, 32) || overlapsAnyTank(x, 0, 32, g.Tanks) {
		return false
	}
	for _, e := range g.Enemies {
		if e.Active && rectsOverlap(x, 0, 32, 32, e.X, e.Y, e.Size, e.Size) {
			return false
		}
	}
	typ := enemyTypeFor(g.CurrentStage, g.SpawnedEnemyCount)
	enemy := NewEnemy(x, 0, typ, g.CurrentStage)
	if rand.Intn(100) < EnemyShieldChance {
		enemy.ArmorShield = IronShieldHits
	}
	g.Enemies = append(g.Enemies, enemy)
	g.SpawnedEnemyCount++
	g.NextEnemySpawn++
	return true
}

func (g *Game) battleControlsBlocked() bool {
	if actionPressed(ebiten.KeyEscape) {
		g.ShowExitConfirm = !g.ShowExitConfirm
	}
	if g.ShowExitConfirm {
		if actionPressed(ebiten.KeyY) {
			g.finishRun("RUN ENDED")
			return true
		}
		if actionPressed(ebiten.KeyN) {
			g.ShowExitConfirm = false
		}
		return true
	}
	if actionPressed(ebiten.KeyP) {
		g.IsPaused = !g.IsPaused
		return true
	}
	if g.IsPaused {
		return true
	}

	return false
}

func (g *Game) updatePlaying() {
	if g.battleControlsBlocked() {
		return
	}

	g.Map.Update()
	g.updatePlayers()
	g.updateEnemies()
	g.updateBullets()
	if g.State != StatePlaying {
		g.updateExplosions()
		return
	}
	g.updateItems()
	g.resolveTankCollisions()
	g.maybeSpawnItem()
	g.maybeSpawnEnemy()
	g.updateExplosions()
	g.checkStageClear()

	if g.MessageTimer > 0 {
		g.MessageTimer--
	}
	if g.FreezeTimer > 0 {
		g.FreezeTimer--
	}
}

func (g *Game) updatePlayers() {
	for _, t := range g.Tanks {
		if t == nil {
			continue
		}
		xBefore, yBefore, activeBefore := t.X, t.Y, t.Active
		if g.Coop != nil {
			t.updateWithInput(g.Map, g.Enemies, g.Tanks, g.coopInputs[t.PlayerID])
		} else {
			t.UpdatePlayer(g.Map, g.Enemies, g.Tanks)
		}
		if activeBefore && !t.Active {
			g.Explosions = append(g.Explosions, NewExplosion(xBefore+t.Size/2, yBefore+t.Size/2, true))

		}
		if !t.Active {
			continue
		}
		fireKey := ebiten.KeySpace
		if t.PlayerID == 1 {
			fireKey = ebiten.KeyEnter
		}
		maxB := t.MaxBullets()
		count := 0
		for _, b := range g.Bullets {
			if b.Active && b.Owner == OwnerPlayer && b.PlayerID == t.PlayerID {
				count++
			}
		}
		fire := ebiten.IsKeyPressed(fireKey)
		if g.Coop != nil {
			fire = g.coopInputs[t.PlayerID].Fire
		}
		if fire && t.CanFire(maxB, count) {
			g.Bullets = append(g.Bullets, t.Fire())
		}
	}
}

func (g *Game) updateEnemies() {
	if g.Menu.SelectedMode == ModeVersus {
		return
	}
	for _, e := range g.Enemies {
		if !e.Active {
			continue
		}
		x, y := e.X, e.Y
		bullet := e.Update(g.Map, g.Tanks, g.FreezeTimer > 0)
		for _, other := range g.Enemies {
			if other != e && other.Active && rectsOverlap(e.X, e.Y, e.Size, e.Size, other.X, other.Y, other.Size, other.Size) {
				e.X, e.Y = x, y
				e.MoveTimer = 0
				bullet = nil
				break
			}
		}
		if bullet != nil {
			busy := false
			for _, b := range g.Bullets {
				if b.Active && b.Emitter == e {
					busy = true
					break
				}
			}
			if !busy {
				g.Bullets = append(g.Bullets, bullet)
			}
		}
	}
}

func (g *Game) updateBullets() {
	active := g.Bullets[:0]
	for _, b := range g.Bullets {
		if !b.Active {
			continue
		}
		x0, y0 := b.X, b.Y
		baseDestroyed := false
		if b.Update(g.Map) {
			baseDestroyed = true
		}
		if !b.Active {
			if baseDestroyed {
				g.Explosions = append(g.Explosions, NewExplosion(b.X+2, b.Y+2, true))
				g.setMessage("BASE DESTROYED!")
				g.finishRun("BASE DESTROYED")
				return
			}
			// A map impact always gets a tiny flash. The bullet changed position in its final substep.
			if abs(b.X-x0)+abs(b.Y-y0) > 0 {
				g.Explosions = append(g.Explosions, NewExplosionWithRadius(b.X+2, b.Y+2, false, b.BlastRadius))
			}
			continue
		}

		for _, o := range g.Bullets {
			if o == b || !o.Active || o.Owner == b.Owner {
				continue
			}
			if rectsOverlap(b.X, b.Y, 4, 4, o.X, o.Y, 4, 4) {
				b.Active = false
				o.Active = false
				g.Explosions = append(g.Explosions, NewExplosion(b.X+2, b.Y+2, false))
				break
			}
		}
		if !b.Active {
			continue
		}

		if b.Owner == OwnerPlayer {
			for _, e := range g.Enemies {
				if !e.Active || !rectsOverlap(b.X, b.Y, 4, 4, e.X, e.Y, e.Size, e.Size) {
					continue
				}
				if e.Hit(1) {
					g.Score += e.Score
					g.HighScore = max(g.HighScore, g.Score)
					g.StageStats.Kills[e.Type]++
					g.Explosions = append(g.Explosions, NewExplosion(e.X+8, e.Y+8, true))
					g.tryDropItem(e.X, e.Y)
				} else {
					g.Explosions = append(g.Explosions, NewExplosion(e.X+8, e.Y+8, false))
				}
				b.Active = false
				break
			}
			if b.Active && g.Menu.SelectedMode == ModeVersus {
				for _, t := range g.Tanks {
					if t == nil || t.PlayerID == 0 || !t.Active {
						continue
					}
					if rectsOverlap(b.X, b.Y, 4, 4, t.X, t.Y, t.Size, t.Size) {
						if t.TakeHit() {
							g.Explosions = append(g.Explosions, NewExplosion(t.X+8, t.Y+8, true))
						}
						b.Active = false
						break
					}
				}
			}
		} else {
			for _, t := range g.Tanks {
				if t == nil || !t.Active {
					continue
				}
				if rectsOverlap(b.X, b.Y, 4, 4, t.X, t.Y, t.Size, t.Size) {
					if t.TakeHit() {
						g.Explosions = append(g.Explosions, NewExplosion(t.X+8, t.Y+8, true))
					}
					b.Active = false
					break
				}
			}
		}

		if b.Active {
			active = append(active, b)
		}
	}
	g.Bullets = active
}

func (g *Game) updateExplosions() {
	active := g.Explosions[:0]
	for _, e := range g.Explosions {
		e.Update()
		if e.Active {
			active = append(active, e)
		}
	}
	g.Explosions = active
}

func (g *Game) updateItems() {
	for _, i := range g.Items {
		i.Update()
	}
	for _, t := range g.Tanks {
		if t == nil || !t.Active {
			continue
		}
		for _, i := range g.Items {
			if i.Active && rectsOverlap(t.X, t.Y, t.Size, t.Size, i.X, i.Y, 16, 16) {
				i.Apply(g, t)
			}
		}
	}
	active := g.Items[:0]
	for _, i := range g.Items {
		if i.Active {
			active = append(active, i)
		}
	}
	g.Items = active
}

func (g *Game) tryDropItem(x, y float32) {
	if g.StageItemCount >= 3 || len(g.Items) > 0 {
		return
	}
	// Bonus appears often enough to be noticed, but not on every kill.
	chance := 12
	if g.CurrentStage == 0 {
		chance = 15
	}
	if rand.Intn(100) < chance {
		g.Items = append(g.Items, NewItem(x, y))
		g.StageItemCount++
	}
}

func (g *Game) maybeSpawnEnemy() {
	if g.Menu.SelectedMode == ModeVersus || g.SpawnedEnemyCount >= g.TotalEnemiesPerStage {
		return
	}
	if g.SpawnTimer < g.spawnInterval() {
		g.SpawnTimer++
	}
	if g.SpawnTimer >= g.spawnInterval() && g.spawnEnemy() {
		g.SpawnTimer = 0
	}
}

// Custom uncapped wave: gradual reinforcements, faster in later stages.
func (g *Game) spawnInterval() int { return max(90, 180-g.CurrentStage*6) }

func (g *Game) maybeSpawnItem() {}

func (g *Game) resolveTankCollisions() {
	for i, a := range g.Tanks {
		if a == nil || !a.Active {
			continue
		}
		for j := i + 1; j < len(g.Tanks); j++ {
			b := g.Tanks[j]
			if b == nil || !b.Active {
				continue
			}
			separateTanks(a, b)
		}
	}
	// Enemy/player overlap is resolved by blocking the enemy, never by pushing the
	// player. This prevents an enemy from shoving the player through a wall.
}

func separateTanks(a, b *Tank) {
	if !rectsOverlap(a.X, a.Y, a.Size, a.Size, b.X, b.Y, b.Size, b.Size) {
		return
	}
	if abs(a.X-b.X) > abs(a.Y-b.Y) {
		if a.X < b.X {
			a.X -= 1
			b.X += 1
		} else {
			a.X += 1
			b.X -= 1
		}
	} else if a.Y < b.Y {
		a.Y -= 1
		b.Y += 1
	} else {
		a.Y += 1
		b.Y -= 1
	}
	a.ConstrainBounds(PlayfieldWidth, ScreenHeight)
	b.ConstrainBounds(PlayfieldWidth, ScreenHeight)
}

func (g *Game) checkStageClear() {
	if g.State != StatePlaying {
		return
	}
	aliveOrRespawning := false
	for _, t := range g.Tanks {
		if t != nil && (t.Active || t.Lives > 0) {
			aliveOrRespawning = true
			break
		}
	}
	if !aliveOrRespawning {
		g.finishRun("ALL LIVES LOST")
		return
	}
	if g.SpawnedEnemyCount < g.TotalEnemiesPerStage || g.activeEnemyCount() != 0 {
		return
	}
	if g.State != StatePlaying {
		return
	}
	g.StageStats.StageBonus = 1000
	g.Score += g.StageStats.StageBonus
	g.HighScore = max(g.HighScore, g.Score)
	g.ClearedStages++
	g.Bullets = nil
	g.StageClearTimer = StageClearFrames
	g.State = StateStageCleanup
}

func (g *Game) updateStageCleanup() {
	if g.battleControlsBlocked() {
		return
	}
	g.Map.Update()
	for _, t := range g.Tanks {
		if t != nil {
			t.UpdatePlayer(g.Map, nil, g.Tanks)
		}
	}
	g.updateItems()
	g.resolveTankCollisions()
	g.updateExplosions()
	if g.MessageTimer > 0 {
		g.MessageTimer--
	}
	g.tickStageCleanup(actionPressed(ebiten.KeyEnter))
}

func (g *Game) nextStage() {
	g.CurrentStage++
	g.loadStage(g.CurrentStage)
}

func (g *Game) leaveBattle() {
	if g.NetMgr.IsHost {
		g.NetMgr.DestroyRoom()
	} else if g.clientConn != nil {
		_, _ = g.clientConn.Write([]byte("LEAVE_ROOM\n"))
		_ = g.clientConn.Close()
		g.clientConn = nil
	}
	g.ShowExitConfirm = false
	g.IsPaused = false
	g.State = StateTitle
}

func (g *Game) setMessage(s string) {
	g.LastMessage = s
	g.MessageTimer = 90
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sign(v float32) float32 {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

func (g *Game) Draw(screen *ebiten.Image) {
	drawnButtons = nil
	if g.State >= StateVersusSettings && g.State <= StateVersusSeries {
		g.drawVersus(screen)
		return
	}
	screen.Fill(uiBG)
	switch g.State {
	case StateTitle:
		g.Menu.DrawTitle(screen)
	case StateModeSelect:
		g.Menu.DrawModeSelect(screen)
	case StateCoopLobby:
		g.drawCoopLobby(screen)
	case StatePlayerNameInput:
		g.Menu.DrawPlayerName(screen)
	case StateRoomSelect:
		g.Menu.DrawRoomSelect(screen)
	case StateRoomNameInput:
		g.Menu.DrawRoomNameInput(screen)
	case StateRoomListSelect:
		g.Menu.DrawRoomList(screen, g.availableRooms())
	case StateStageIntro:
		g.drawStageIntro(screen)
	case StatePlaying, StateStageCleanup:
		g.drawBattle(screen)
	case StateStageClear:
		g.drawStageClear(screen)
	case StateDeathResult:
		g.drawDeathResult(screen)
	case StateGameOver:
		g.drawGameOver(screen)
	}
	if g.Coop != nil && g.ShowExitConfirm && g.State != StatePlaying && g.State != StateStageCleanup {
		uiExitDialog(screen, "LEAVE ROOM?", "OTHER PLAYERS CAN CONTINUE.", "LEAVE")
	}
	if g.MessageTimer > 0 && g.State <= StateRoomListSelect {
		uiNoticeFrame(screen, 36, 335, 504, 26, uiRed)
		uiText(screen, g.LastMessage, 48, 345, 1, uiRed)
	}

}

func (g *Game) drawBattle(screen *ebiten.Image) {
	g.Map.Draw(screen)
	for _, e := range g.Enemies {
		e.Draw(screen, g.EnemySprites)
	}
	for _, t := range g.Tanks {
		if t != nil {
			t.Draw(screen)
		}
	}
	for _, b := range g.Bullets {
		b.Draw(screen)
	}
	for _, i := range g.Items {
		i.Draw(screen, g.ItemSprites, g.Frame)
	}
	for _, x := range g.Explosions {
		x.Draw(screen)
	}
	g.Map.DrawBushes(screen)
	g.drawSidebar(screen)

	if g.Coop != nil {
		for _, t := range g.Tanks {
			if t.Active {
				label := fmt.Sprintf("P%d", t.PlayerID+1)
				if t.PlayerID == g.Coop.LocalID {
					label += " YOU"
				}
				uiText(screen, label, int(t.X), max(2, int(t.Y)-10), 1, uiInk)
			}
		}
	}
	if g.State != StateDeathResult {
		g.drawBattleOverlays(screen)
	}
}

func sumItems(v [6]int) int {
	total := 0
	for _, n := range v {
		total += n
	}
	return total
}
func (g *Game) Layout(_, _ int) (int, int) { return ScreenWidth, ScreenHeight }

func (g *Game) availableRooms() []RoomInfo {
	rooms := g.NetMgr.GetDiscoveredRooms()
	filtered := rooms[:0]
	for _, room := range rooms {
		if GameMode(room.Mode) == g.Menu.SelectedMode {
			filtered = append(filtered, room)
		}
	}
	return filtered
}
