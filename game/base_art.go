package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"os"
	"path/filepath"
)

var baseSprites [2]*ebiten.Image

func loadBaseSprite(file string) *ebiten.Image {
	for _, p := range []string{filepath.Join("Source", "Pictures", file), filepath.Join("..", "Source", "Pictures", file)} {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		src, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			continue
		}
		// Trim only transparent export margins; keep the selected artwork unchanged.
		bounds := src.Bounds()
		visible := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := src.At(x, y).RGBA()
				if a > 32768 {
					visible.Min.X = min(visible.Min.X, x)
					visible.Min.Y = min(visible.Min.Y, y)
					visible.Max.X = max(visible.Max.X, x+1)
					visible.Max.Y = max(visible.Max.Y, y+1)
				}
			}
		}
		if !visible.Empty() {
			if cropped, ok := src.(interface {
				SubImage(image.Rectangle) image.Image
			}); ok {
				src = cropped.SubImage(visible)
			}
		}
		return ebiten.NewImageFromImage(src)
	}
	return nil
}
func (m *Map) drawBaseSprite(s *ebiten.Image, r, c int) {
	state := 0
	if m.Tiles[r][c] == TileBaseRuins {
		state = 1
	}
	if baseSprites[state] == nil {
		baseSprites[state] = loadBaseSprite([]string{"base.png", "base_destroyed.png"}[state])
	}
	img := baseSprites[state]
	if img == nil {
		m.drawBase(s, float32(c*16), float32(r*16))
		return
	}
	b := img.Bounds()
	// Four cells sample quarters of one 32px headquarters, never four buildings.
	cx, cy := c%2, r%2
	part := img.SubImage(image.Rect(b.Min.X+cx*b.Dx()/2, b.Min.Y+cy*b.Dy()/2, b.Min.X+(cx+1)*b.Dx()/2, b.Min.Y+(cy+1)*b.Dy()/2)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(16/float64(part.Bounds().Dx()), 16/float64(part.Bounds().Dy()))
	op.GeoM.Translate(float64(c*16), float64(r*16))
	s.DrawImage(part, op)
}
