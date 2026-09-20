package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
)

func uiIconCard(s *ebiten.Image, y int, selected bool) {
	c := uiLine
	mx, my := ebiten.CursorPosition()
	if selected || mx >= 44 && mx < 532 && my >= y && my < y+76 {
		c = uiGold
	}
	uiFrame(s, 44, y, 488, 76, c)
	if selected {
		uiRect(s, 45, y+1, 486, 74, uiRaised)
		uiRect(s, 44, y, 4, 76, uiGold)
		uiText(s, ">", 62, y+30, 2, uiGold)
	}
}
func uiShield(s *ebiten.Image, x, y int, c color.Color) {
	uiRect(s, x, y, 32, 5, c)
	uiRect(s, x, y, 5, 25, c)
	uiRect(s, x+27, y, 5, 25, c)
	for i := 0; i < 4; i++ {
		uiRect(s, x+i*4, y+25+i*4, 5, 5, c)
		uiRect(s, x+27-i*4, y+25+i*4, 5, 5, c)
	}
}
func uiVersusIcon(s *ebiten.Image, y, count int) {
	for i := 0; i < count; i++ {
		uiTank(s, 180-i*42, y+16, 3, uiGold)
		uiTank(s, 332+i*42, y+16, 3, uiGreen)
	}
	uiText(s, "VS", 264, y+30, 2, uiInk)
}
