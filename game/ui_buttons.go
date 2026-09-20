package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"image/color"
	"strings"
)

type buttonArea struct {
	x, y, w, h int
	key        ebiten.Key
}

var drawnButtons []buttonArea

func actionPressed(key ebiten.Key) bool {
	if inpututil.IsKeyJustPressed(key) {
		return true
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	x, y := ebiten.CursorPosition()
	for _, b := range drawnButtons {
		if b.key == key && x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h {
			return true
		}
	}
	return false
}
func uiButton(s *ebiten.Image, label string, x, y, w, h int, key ebiten.Key, c color.Color) {
	label = compactButtonLabel(label)
	bg := uiPanel
	mx, my := ebiten.CursorPosition()
	if mx >= x && mx < x+w && my >= y && my < y+h {
		bg = uiRaised
	}
	uiRect(s, x, y, w, h, c)
	uiRect(s, x+1, y+1, w-2, h-2, bg)
	scale := 1
	if len(label)*12 <= w-18 && h >= 29 {
		scale = 2
	}
	uiText(s, label, x+(w-len(label)*6*scale)/2, y+(h-7*scale)/2, scale, c)
	drawnButtons = append(drawnButtons, buttonArea{x, y, w, h, key})
}
func compactButtonLabel(label string) string {
	switch label {
	case "UP / DOWN SELECT  ENTER CONFIRM":
		return "SELECT"
	case "UP / DOWN SELECT  ENTER JOIN":
		return "JOIN"
	case "CLICK OR ENTER CREATE LOBBY", "ENTER CREATE ROOM":
		return "CREATE"
	case "BACKSPACE DELETE / ESC BACK":
		return "BACK"
	case "ENTER RETURN TO TITLE":
		return "HOME"
	case "ENTER RETURN TO LOBBY":
		return "LOBBY"
	case "ENTER BEGIN MISSION", "ENTER START GAME", "ENTER START MATCH":
		return "START"
	case "ENTER SKIP COUNT":
		return "SKIP"
	case "ENTER NEXT STAGE":
		return "NEXT"
	case "ESC LEAVE ROOM", "ENTER LEAVE ROOM":
		return "LEAVE"
	case "ESC FINISH RUN", "ESC END RUN":
		return "FINISH"
	case "ENTER JOIN TEAM":
		return "JOIN"
	case "T CHANGE TEAM":
		return "TEAM"
	}
	for _, prefix := range []string{"ENTER ", "ESC ", "P ", "N  ", "Y  "} {
		label = strings.TrimPrefix(label, prefix)
	}
	return label
}
func footerAction(s *ebiten.Image, label string, x int, c color.Color) {
	key := ebiten.Key(-1)
	if strings.Contains(label, "ENTER") {
		key = ebiten.KeyEnter
	}
	if strings.Contains(label, "ESC") {
		key = ebiten.KeyEscape
	}
	if key < 0 {
		if strings.Contains(label, "WAITING") {
			label = "WAITING..."
			uiText(s, label, x, 385, 2, uiMuted)
		}
		return
	}
	w := 200
	if x >= 380 {
		w = 168
	}
	uiButton(s, label, x, 375, w, 33, key, c)
}
func uiExitDialog(s *ebiten.Image, title, detail, confirm string) {
	drawnButtons = nil
	uiModal(s, title, detail, "", uiRed)
	uiButton(s, "N  CONTINUE", 56, 233, 143, 32, ebiten.KeyN, uiGreen)
	uiButton(s, "Y  "+confirm, 211, 233, 149, 32, ebiten.KeyY, uiRed)
}
