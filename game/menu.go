package game

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type MenuUI struct {
	PlayerName     string
	TeamCursor     int
	TeamSize       int
	SeriesIndex    int
	ConfigCursor   int
	ModeCursor     int
	RoomCursor     int
	RoomListCursor int
	SelectedMode   GameMode
	SelectedRole   NetworkRole
	RoomName       string
}

func NewMenuUI() *MenuUI { return &MenuUI{RoomName: "Player's Room", TeamSize: 1} }

func menuNav(cursor *int, max int) {
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		*cursor = (*cursor + 1) % max
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		*cursor = (*cursor - 1 + max) % max
	}
}

func (m *MenuUI) UpdateModeSelect() bool {
	if cardChoice(&m.ModeCursor, 2, 139, 92, 76) {
		m.SelectedMode = GameMode(m.ModeCursor)
		return true
	}
	return false
}

func (m *MenuUI) UpdateRoomSelect() bool {
	if cardChoice(&m.RoomCursor, 2, 139, 92, 76) {
		m.SelectedRole = NetworkRole(m.RoomCursor)
		return true
	}
	return false
}

func (m *MenuUI) UpdateRoomNameInput() bool {
	chars := ebiten.AppendInputChars(nil)
	for _, r := range chars {
		if len([]rune(m.RoomName)) >= 15 {
			break
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' {
			m.RoomName += string(r)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		rs := []rune(m.RoomName)
		if len(rs) > 0 {
			m.RoomName = string(rs[:len(rs)-1])
		}
	}
	if actionPressed(ebiten.KeyEnter) {
		m.RoomName = strings.TrimSpace(m.RoomName)
		if m.RoomName == "" {
			m.RoomName = "My Tank Room"
		}
		return true
	}
	return false
}

func (m *MenuUI) UpdateRoomListSelect(roomsCount int) int {
	if roomsCount == 0 {
		return -1
	}
	if m.RoomListCursor >= roomsCount {
		m.RoomListCursor = roomsCount - 1
	}
	menuNav(&m.RoomListCursor, roomsCount)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		start := m.RoomListCursor / 3 * 3
		for i := 0; i < 3 && start+i < roomsCount; i++ {
			if x >= 44 && x < 532 && y >= 132+i*71 && y < 196+i*71 {
				m.RoomListCursor = start + i
				return start + i
			}
		}
	}
	if actionPressed(ebiten.KeyEnter) {
		return m.RoomListCursor
	}
	return -1
}

// Card choices support both keyboard and left mouse click.
func cardChoice(cursor *int, count, top, step, height int) bool {
	menuNav(cursor, count)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x >= 44 && x < 532 {
			for i := 0; i < count; i++ {
				if y >= top+i*step && y < top+i*step+height {
					*cursor = i
					return true
				}
			}
		}
	}
	return actionPressed(ebiten.KeyEnter)
}
func (m *MenuUI) UpdateVersusSettings() bool {
	if cardChoice(&m.ConfigCursor, 2, 137, 92, 76) {
		m.TeamSize = m.ConfigCursor + 1
		return true
	}
	return false
}

func (m *MenuUI) UpdatePlayerName() bool {
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(m.PlayerName) < 12 && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			m.PlayerName += string(r)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(m.PlayerName) > 0 {
		m.PlayerName = m.PlayerName[:len(m.PlayerName)-1]
	}
	return actionPressed(ebiten.KeyEnter) && m.PlayerName != ""
}
func (m *MenuUI) DrawPlayerName(s *ebiten.Image) {
	uiBackground(s, "VERSUS / PLAYER PROFILE", "YOUR NAME")
	uiFrame(s, 44, 145, 488, 95, uiGold)
	uiText(s, m.PlayerName+"_", 64, 177, 3, uiInk)
	uiText(s, "1-12: A-Z / 0-9 / - / _", 44, 265, 1, uiMuted)
	uiFooter(s, "ENTER CONTINUE", "ESC BACK")
}
