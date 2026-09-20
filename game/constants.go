package game

const (
	PlayfieldWidth = 416
	ScreenWidth    = 576 // 416px battlefield + 160px information panel
	ScreenHeight   = 416
	MapCols        = 26
	MapRows        = 26
	TileSize       = 16

	MaxLives          = 3
	EnemiesPerStage   = 20
	PlayerSpawnX      = 240 // immediately to the right of the base
	PlayerSpawnY      = 384
	SecondSpawnX      = 144 // second spawn mirrors the base area on the left
	SecondSpawnY      = 384
	StageIntroFrames  = 90
	StageClearFrames  = 180 // Three seconds to collect remaining items after victory.
	GameOverFrames    = 180
	DeathResultFrames = 45
)

type GameState int

const (
	StateTitle GameState = iota
	StateModeSelect
	StateRoomSelect
	StateRoomNameInput
	StateRoomListSelect
	StateStageIntro
	StatePlaying
	StateStageClear
	StateGameOver
	StateDeathResult
	StateStageCleanup
	StateVersusSettings
	StateVersusLobby
	StateVersusCountdown
	StateVersusPlaying
	StateVersusRoundEnd
	StateVersusMatchEnd
	StateVersusTeamSelect
	StateVersusSeries
	StatePlayerNameInput
	StateCoopLobby
)

type GameMode int

const (
	ModeCoop GameMode = iota
	ModeVersus
)

type NetworkRole int

const (
	RoleHost NetworkRole = iota
	RoleJoin
)

type Direction int

const (
	DirUp Direction = iota
	DirDown
	DirLeft
	DirRight
)

func (d Direction) Delta() (float32, float32) {
	switch d {
	case DirUp:
		return 0, -1
	case DirDown:
		return 0, 1
	case DirLeft:
		return -1, 0
	default:
		return 1, 0
	}
}
