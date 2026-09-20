package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"strings"
)

var (
	uiBG     = color.RGBA{12, 18, 22, 255}
	uiPanel  = color.RGBA{20, 30, 35, 255}
	uiRaised = color.RGBA{29, 42, 46, 255}
	uiLine   = color.RGBA{47, 65, 67, 255}
	uiInk    = color.RGBA{233, 233, 211, 255}
	uiMuted  = color.RGBA{144, 165, 161, 255}
	uiGold   = color.RGBA{241, 189, 83, 255}
	uiGreen  = color.RGBA{115, 208, 163, 255}
	uiRed    = color.RGBA{237, 128, 104, 255}
)

// Small bitmap alphabet: crisp at integer scales, no font download required.
var uiGlyphs = map[rune][7]byte{
	'A': {14, 17, 17, 31, 17, 17, 17}, 'B': {30, 17, 17, 30, 17, 17, 30},
	'C': {14, 17, 16, 16, 16, 17, 14}, 'D': {30, 17, 17, 17, 17, 17, 30},
	'E': {31, 16, 16, 30, 16, 16, 31}, 'F': {31, 16, 16, 30, 16, 16, 16},
	'G': {14, 17, 16, 23, 17, 17, 15}, 'H': {17, 17, 17, 31, 17, 17, 17},
	'I': {14, 4, 4, 4, 4, 4, 14}, 'J': {7, 2, 2, 2, 18, 18, 12},
	'K': {17, 18, 20, 24, 20, 18, 17}, 'L': {16, 16, 16, 16, 16, 16, 31},
	'M': {17, 27, 21, 21, 17, 17, 17}, 'N': {17, 25, 21, 19, 17, 17, 17},
	'O': {14, 17, 17, 17, 17, 17, 14}, 'P': {30, 17, 17, 30, 16, 16, 16},
	'Q': {14, 17, 17, 17, 21, 18, 13}, 'R': {30, 17, 17, 30, 20, 18, 17},
	'S': {15, 16, 16, 14, 1, 1, 30}, 'T': {31, 4, 4, 4, 4, 4, 4},
	'U': {17, 17, 17, 17, 17, 17, 14}, 'V': {17, 17, 17, 17, 17, 10, 4},
	'W': {17, 17, 17, 21, 21, 21, 10}, 'X': {17, 17, 10, 4, 10, 17, 17},
	'Y': {17, 17, 10, 4, 4, 4, 4}, 'Z': {31, 1, 2, 4, 8, 16, 31},
	'0': {14, 17, 19, 21, 25, 17, 14}, '1': {4, 12, 4, 4, 4, 4, 14},
	'2': {14, 17, 1, 2, 4, 8, 31}, '3': {30, 1, 1, 14, 1, 1, 30},
	'4': {2, 6, 10, 18, 31, 2, 2}, '5': {31, 16, 16, 30, 1, 1, 30},
	'6': {14, 16, 16, 30, 17, 17, 14}, '7': {31, 1, 2, 4, 8, 8, 8},
	'8': {14, 17, 17, 14, 17, 17, 14}, '9': {14, 17, 17, 15, 1, 1, 14},
	'-': {0, 0, 0, 31, 0, 0, 0}, '+': {0, 4, 4, 31, 4, 4, 0},
	'/': {1, 2, 2, 4, 8, 8, 16}, ':': {0, 4, 4, 0, 4, 4, 0},
	'.': {0, 0, 0, 0, 0, 6, 6}, '>': {16, 8, 4, 2, 4, 8, 16},
	'<': {1, 2, 4, 8, 4, 2, 1}, '_': {0, 0, 0, 0, 0, 0, 31},
	'?': {14, 17, 1, 2, 4, 0, 4}, '!': {4, 4, 4, 4, 4, 0, 4},
	'*': {0, 21, 14, 31, 14, 21, 0}, '\'': {4, 4, 0, 0, 0, 0, 0},
}

func uiRect(s *ebiten.Image, x, y, w, h int, c color.Color) {
	vector.DrawFilledRect(s, float32(x), float32(y), float32(w), float32(h), c, false)
}
func uiText(s *ebiten.Image, text string, x, y, scale int, c color.Color) {
	for _, r := range strings.ToUpper(text) {
		glyph := uiGlyphs[r]
		for row, bits := range glyph {
			for col := 0; col < 5; col++ {
				if bits&(1<<uint(4-col)) != 0 {
					uiRect(s, x+col*scale, y+row*scale, scale, scale, c)
				}
			}
		}
		x += 6 * scale
	}
}
func uiCenter(s *ebiten.Image, text string, y, scale int, c color.Color) {
	uiText(s, text, (ScreenWidth-(len([]rune(text))*6-1)*scale)/2, y, scale, c)
}
func uiFrame(s *ebiten.Image, x, y, w, h int, accent color.Color) {
	uiRect(s, x+4, y+4, w, h, color.RGBA{0, 0, 0, 80})
	uiRect(s, x, y, w, h, uiLine)
	uiRect(s, x+1, y+1, w-2, h-2, uiPanel)
	uiRect(s, x, y, 20, 2, accent)
	uiRect(s, x, y, 2, 12, accent)
	uiRect(s, x+w-20, y+h-2, 20, 2, accent)
}

// In-battle notices use one translucent fill; do not stack opaque panel layers.
func uiNoticeFrame(s *ebiten.Image, x, y, w, h int, accent color.Color) {
	uiRect(s, x, y, w, h, color.NRGBA{R: 20, G: 30, B: 35, A: 155})
	border := color.NRGBA{R: 105, G: 136, B: 135, A: 150}
	uiRect(s, x, y, w, 1, border)
	uiRect(s, x, y+h-1, w, 1, border)
	uiRect(s, x, y, 1, h, border)
	uiRect(s, x+w-1, y, 1, h, border)
	uiRect(s, x, y, 20, 2, accent)
	uiRect(s, x, y, 2, 12, accent)
	uiRect(s, x+w-20, y+h-2, 20, 2, accent)
}

func uiBackground(s *ebiten.Image, kicker, title string) {
	s.Fill(uiBG)
	uiRect(s, 28, 42, 4, 47, uiGold)
	uiText(s, title, 48, 58, 3, uiInk)
	uiRect(s, 28, 110, 520, 1, uiLine)
}
func uiFooter(s *ebiten.Image, primary, secondary string) {
	footerAction(s, primary, 28, uiGold)
	footerAction(s, secondary, 380, uiMuted)
}
func uiOption(s *ebiten.Image, y int, title, detail string, selected bool) {
	accent := uiLine
	if selected {
		accent = uiGold
	}
	uiFrame(s, 44, y, 488, 76, accent)
	if selected {
		uiRect(s, 45, y+1, 486, 74, uiRaised)
		uiRect(s, 44, y, 4, 76, uiGold)
		uiText(s, ">", 62, y+23, 2, uiGold)
	}
	uiText(s, title, 88, y+30, 2, uiInk)
}
func uiTank(s *ebiten.Image, x, y, scale int, c color.Color) {
	team := 0
	if c == uiGreen {
		team = 1
	}
	if img := playerSprite(team); img != nil {
		b := img.Bounds()
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(float64(14*scale)/float64(b.Dx()), float64(15*scale)/float64(b.Dy()))
		op.GeoM.Translate(float64(x), float64(y))
		if c == uiMuted {
			op.ColorScale.Scale(.45, .45, .45, 1)
		}
		s.DrawImage(img, op)
		return
	}

	// Stylized tank icon facing up, also used in title and result cards.
	uiRect(s, x, y+3*scale, 3*scale, 12*scale, uiMuted)
	uiRect(s, x+11*scale, y+3*scale, 3*scale, 12*scale, uiMuted)
	uiRect(s, x+3*scale, y+5*scale, 8*scale, 9*scale, c)
	uiRect(s, x+5*scale, y+3*scale, 4*scale, 7*scale, c)
	uiRect(s, x+6*scale, y, 2*scale, 7*scale, uiInk)
	for i := 0; i < 4; i++ {
		uiRect(s, x, y+(4+i*3)*scale, 3*scale, scale, uiBG)
		uiRect(s, x+11*scale, y+(4+i*3)*scale, 3*scale, scale, uiBG)
	}
}
func uiProgress(s *ebiten.Image, x, y, w, value, total int, c color.Color) {
	uiRect(s, x, y, w, 4, uiLine)
	if total <= 0 {
		return
	}
	if value < 0 {
		value = 0
	}
	if value > total {
		value = total
	}
	uiRect(s, x, y, w*value/total, 4, c)
}
func uiModal(s *ebiten.Image, title, detail, action string, accent color.Color) {
	uiRect(s, 0, 0, PlayfieldWidth, ScreenHeight, color.NRGBA{R: 5, G: 10, B: 13, A: 55})
	uiNoticeFrame(s, 36, 136, 344, 144, accent)
	uiText(s, title, 56, 158, 2, accent)
	uiText(s, detail, 56, 194, 1, uiInk)
	uiRect(s, 56, 222, 304, 1, uiLine)
	uiText(s, action, 56, 245, 1, uiGold)
}
