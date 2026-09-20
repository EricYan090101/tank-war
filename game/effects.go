package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Explosion struct {
	X, Y   float32
	Large  bool
	Radius float32
	Frame  int
	Active bool
}

func NewExplosion(x, y float32, large bool) *Explosion {
	return NewExplosionWithRadius(x, y, large, 0)
}

func NewExplosionWithRadius(x, y float32, large bool, radius float32) *Explosion {
	return &Explosion{X: x, Y: y, Large: large, Radius: radius, Active: true}
}

func (e *Explosion) Update() {
	if !e.Active {
		return
	}
	e.Frame++
	limit := 12
	if e.Large {
		limit = 18
	}
	if e.Frame >= limit {
		e.Active = false
	}
}

func (e *Explosion) Draw(screen *ebiten.Image) {
	if !e.Active {
		return
	}
	cx, cy := e.X, e.Y
	progress := float32(e.Frame) / 12.0
	if e.Large {
		progress = float32(e.Frame) / 18.0
	}
	if progress > 1 {
		progress = 1
	}
	alpha := uint8(255)
	if progress > 0.7 {
		alpha = uint8(math.Max(0, float64((1-progress)*850)))
	}

	outer := float32(3)
	if e.Large {
		outer = 8
	} else if e.Radius > 0 {
		outer = 4 + e.Radius*0.18
	}
	r := outer + progress*outer*1.7
	vector.DrawFilledRect(screen, cx-r, cy-2, r*2, 4, color.RGBA{255, 222, 84, alpha}, true)
	vector.DrawFilledRect(screen, cx-2, cy-r, 4, r*2, color.RGBA{255, 178, 48, alpha}, true)
	if e.Frame%3 != 0 {
		vector.DrawFilledRect(screen, cx-r*0.65, cy-r*0.65, 4, 4, color.RGBA{255, 94, 38, alpha}, true)
		vector.DrawFilledRect(screen, cx+r*0.4, cy-r*0.55, 3, 3, color.RGBA{255, 94, 38, alpha}, true)
		vector.DrawFilledRect(screen, cx-r*0.45, cy+r*0.5, 3, 3, color.RGBA{255, 94, 38, alpha}, true)
	}
	vector.DrawFilledRect(screen, cx-2, cy-2, 4, 4, color.RGBA{255, 255, 235, alpha}, true)
}
