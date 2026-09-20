package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type BulletOwner int

const (
	OwnerPlayer BulletOwner = iota
	OwnerEnemy
)

type Bullet struct {
	PlayerID    int
	Emitter     *Enemy
	X, Y        float32
	Speed       float32
	Dir         Direction
	Active      bool
	Owner       BulletOwner
	Power       int
	Width       float32
	BlastRadius float32
	TrailAge    int
}

func NewBullet(x, y float32, dir Direction, owner BulletOwner, power int) *Bullet {
	speed := float32(4.0)
	if power >= 2 {
		speed = 6
	}
	blast := float32(10)
	if power >= 4 {
		blast = 16
	}
	return &Bullet{X: x, Y: y, Speed: speed, Dir: dir, Active: true, Owner: owner, Power: power, Width: 4, BlastRadius: blast}
}

func (b *Bullet) Update(m *Map) (hitBase bool) {
	if !b.Active {
		return false
	}
	dx, dy := b.Dir.Delta()
	remaining := b.Speed
	for remaining > 0 && b.Active {
		step := float32(1)
		if remaining < step {
			step = remaining
		}
		b.X += dx * step
		b.Y += dy * step
		b.TrailAge++
		remaining -= step

		if b.X < -b.Width || b.Y < 0 || b.X > PlayfieldWidth || b.Y > ScreenHeight {
			b.Active = false
			return false
		}

		hit, base := m.HitBullet(b.X+b.Width/2, b.Y+b.Width/2, b.Dir, b.Power, b.Power >= 4, b.BlastRadius)
		if hit {
			b.Active = false
			return base
		}
	}
	return false
}

func (b *Bullet) Draw(screen *ebiten.Image) {
	if !b.Active {
		return
	}
	col := color.RGBA{244, 244, 229, 255}
	if b.Owner == OwnerEnemy {
		col = color.RGBA{255, 214, 116, 255}
	}
	if b.Power >= 4 {
		col = color.RGBA{255, 236, 160, 255}
	}
	if dx, dy := b.Dir.Delta(); dx != 0 {
		vector.DrawFilledRect(screen, b.X-1, b.Y+1, 6, 2, col, true)
	} else if dy != 0 {
		vector.DrawFilledRect(screen, b.X+1, b.Y-1, 2, 6, col, true)
	}
}
