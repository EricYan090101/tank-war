package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
)

var playerArt [2]*ebiten.Image
var playerArtLoaded bool

func playerSprite(team int) *ebiten.Image {
	if !playerArtLoaded {
		playerArtLoaded = true
		for i, name := range []string{"player_yellow.png", "player_green.png"} {
			for _, path := range []string{filepath.Join("Source", "Pictures", name), filepath.Join("..", "Source", "Pictures", name)} {
				f, err := os.Open(path)
				if err != nil {
					continue
				}
				src, _, err := image.Decode(f)
				f.Close()
				if err != nil {
					continue
				}
				// Ignore transparent export padding when fitting both teams to the same 32px hitbox.
				b := src.Bounds()
				r := image.Rectangle{Min: b.Max, Max: b.Min}
				for y := b.Min.Y; y < b.Max.Y; y++ {
					for x := b.Min.X; x < b.Max.X; x++ {
						_, _, _, a := src.At(x, y).RGBA()
						if a > 32768 {
							r.Min.X = min(r.Min.X, x)
							r.Min.Y = min(r.Min.Y, y)
							r.Max.X = max(r.Max.X, x+1)
							r.Max.Y = max(r.Max.Y, y+1)
						}
					}
				}
				if !r.Empty() {
					if crop, ok := src.(interface {
						SubImage(image.Rectangle) image.Image
					}); ok {
						src = crop.SubImage(r)
					}
				}
				playerArt[i] = ebiten.NewImageFromImage(src)
				break
			}
		}
	}
	if team != 1 {
		team = 0
	}
	return playerArt[team]
}
func (t *Tank) drawPlayerSprite(s *ebiten.Image) bool {
	team := 0
	if t.MatchTank {
		team = t.Team
	} else if t.PlayerID%2 == 1 {
		team = 1
	}
	img := playerSprite(team)
	if img == nil {
		return false
	}
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(32/float64(b.Dx()), 32/float64(b.Dy()))
	op.GeoM.Translate(-16, -16)
	angle := 0.0
	switch t.Dir {
	case DirDown:
		angle = math.Pi
	case DirLeft:
		angle = -math.Pi / 2
	case DirRight:
		angle = math.Pi / 2
	}
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(float64(t.X)+16, float64(t.Y)+16)
	if t.HitFlash > 0 {
		op.ColorScale.Scale(2, 2, 2, 1)
	}
	s.DrawImage(img, op)
	return true
}
