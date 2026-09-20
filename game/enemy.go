package game

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type EnemyType int

const (
	EnemyBasic EnemyType = iota
	EnemyFast
	EnemyPower
)

func (t EnemyType) String() string {
	switch t {
	case EnemyFast:
		return "FAST"
	case EnemyPower:
		return "POWER"
	default:
		return "BASIC"
	}
}

func EnemyPoints(t EnemyType) int {
	switch t {
	case EnemyFast:
		return 200
	case EnemyPower:
		return 300
	default:
		return 100
	}
}

type Enemy struct {
	X, Y        float32
	Speed       float32
	Size        float32
	Dir         Direction
	Active      bool
	Type        EnemyType
	HP          int
	ArmorShield int
	MoveTimer   int
	FireTimer   int
	SpawnTime   int
	Flash       int
	Score       int
	SpawnX      float32
	BumpTimer   int
	AggroTimer  int
}

func NewEnemy(x, y float32, typ EnemyType, stage int) *Enemy {
	e := &Enemy{
		X: x, Y: y, Size: 32, Dir: DirDown, Active: true,
		Type: typ, MoveTimer: 150 + rand.Intn(100), FireTimer: 55 + rand.Intn(75),
		SpawnTime: 45, SpawnX: x, AggroTimer: 30 + rand.Intn(90), Score: EnemyPoints(typ),
	}
	switch typ {
	case EnemyFast:
		e.Speed, e.HP = 2, 1
	case EnemyPower:
		e.Speed, e.HP = 1.0, 1
	default:
		e.Speed, e.HP = 1, 1
	}
	// Later stages become slightly more aggressive without changing the visual pace too much.
	// Keep movement speed stable between stages; only wave pacing changes.
	return e
}

func (e *Enemy) Update(m *Map, targets []*Tank, frozen bool) *Bullet {
	if !e.Active {
		return nil
	}
	if e.SpawnTime > 0 {
		e.SpawnTime--
		return nil
	}
	if frozen {
		return nil
	}
	if e.Flash > 0 {
		e.Flash--
	}
	if e.BumpTimer > 0 {
		e.BumpTimer--
	}
	if e.AggroTimer > 0 {
		e.AggroTimer--
	}

	e.MoveTimer--
	if e.MoveTimer <= 0 {
		e.turnOnGrid(m, targets)
		e.MoveTimer = e.nextTurnDelay()
	}

	dx, dy := e.Dir.Delta()
	nx, ny := e.X+dx*e.Speed, e.Y+dy*e.Speed
	blocked := m.CheckTileCollision(nx, ny, e.Size, e.Size) || nx < 0 || ny < 0 || nx > PlayfieldWidth-e.Size || ny > ScreenHeight-e.Size
	if !blocked && overlapsAnyTank(nx, ny, e.Size, targets) {
		// Never push the player. Treat tanks as solid blockers for enemy movement.
		blocked = true
	}
	if blocked {
		// Turn only once when a route is blocked. A cooldown prevents rapid left/right oscillation.
		if e.BumpTimer == 0 {
			e.turnOnGrid(m, targets)
			e.MoveTimer = e.nextTurnDelay()
			e.BumpTimer = 18
		}
	} else {
		e.X, e.Y = nx, ny
	}

	e.FireTimer--
	if e.FireTimer <= 0 {
		switch e.Type {
		case EnemyPower:
			e.FireTimer = 55 + rand.Intn(80)
		case EnemyFast:
			e.FireTimer = 60 + rand.Intn(75)
		default:
			e.FireTimer = 80 + rand.Intn(90)
		}
		power := 1
		if e.Type == EnemyPower {
			power = 2
		}
		return e.Fire(power)
	}
	return nil
}

func (e *Enemy) nextTurnDelay() int {
	switch e.Type {
	case EnemyFast:
		return 120 + rand.Intn(90)
	case EnemyPower:
		return 180 + rand.Intn(100)
	default:
		return 180 + rand.Intn(120)
	}
}

func (e *Enemy) turnOnGrid(m *Map, targets []*Tank) {
	old := e.Dir
	e.chooseDirection(targets)
	if (old == DirUp || old == DirDown) == (e.Dir == DirUp || e.Dir == DirDown) {
		return
	}
	proxy := &Tank{X: e.X, Y: e.Y, Size: e.Size}
	if e.Dir == DirUp || e.Dir == DirDown {
		target := snapTo(e.X, 16, 8)
		moveWithCollisionAndTanks(proxy, m, sign(target-e.X), 0, abs(target-e.X), nil, targets)
	} else {
		target := snapTo(e.Y, 16, 8)
		moveWithCollisionAndTanks(proxy, m, 0, sign(target-e.Y), abs(target-e.Y), nil, targets)
	}
	e.X, e.Y = proxy.X, proxy.Y
}

func (e *Enemy) chooseDirection(targets []*Tank) {
	// Mostly directional wandering, with short bursts of target-seeking.
	if len(targets) > 0 && rand.Intn(100) < 55 {
		var target *Tank
		best := float32(1e30)
		for _, t := range targets {
			if t == nil || !t.Active {
				continue
			}
			dx, dy := t.X-e.X, t.Y-e.Y
			d := dx*dx + dy*dy
			if d < best {
				best, target = d, t
			}
		}
		if target != nil {
			dx, dy := target.X-e.X, target.Y-e.Y
			if rand.Intn(100) < 70 {
				if abs(dx) > abs(dy) {
					if dx < 0 {
						e.Dir = DirLeft
					} else {
						e.Dir = DirRight
					}
				} else if dy < 0 {
					e.Dir = DirUp
				} else {
					e.Dir = DirDown
				}
				return
			}
		}
	}

	choices := []Direction{DirUp, DirDown, DirLeft, DirRight}
	// Bias against instant 180-degree turns, which looks too twitchy.
	for tries := 0; tries < 4; tries++ {
		d := choices[rand.Intn(len(choices))]
		if !isOpposite(d, e.Dir) || tries >= 2 {
			e.Dir = d
			return
		}
	}
}

func isOpposite(a, b Direction) bool {
	return (a == DirUp && b == DirDown) || (a == DirDown && b == DirUp) || (a == DirLeft && b == DirRight) || (a == DirRight && b == DirLeft)
}

func (e *Enemy) Fire(power int) *Bullet {
	dx, dy := e.Dir.Delta()
	b := NewBullet(e.X+14+dx*15, e.Y+14+dy*15, e.Dir, OwnerEnemy, power)
	b.Emitter = e
	b.PlayerID = -1
	return b
}

func (e *Enemy) Hit(power int) bool {
	if !e.Active || e.SpawnTime > 0 {
		return false
	}
	if absorbIronHit(&e.ArmorShield) {
		e.Flash = 5
		return false
	}
	e.HP = 0
	e.Flash = 5
	if e.HP <= 0 {
		e.Active = false
		return true
	}
	return false
}

func overlapsAnyTank(x, y, size float32, tanks []*Tank) bool {
	for _, t := range tanks {
		if t == nil || !t.Active {
			continue
		}
		if rectsOverlap(x, y, size, size, t.X, t.Y, t.Size, t.Size) {
			return true
		}
	}
	return false
}

// Source PNG barrels face UP. Screen coordinates use positive Y downward.
func directionRotation(d Direction) float64 {
	switch d {
	case DirDown:
		return math.Pi
	case DirLeft:
		return -math.Pi / 2
	case DirRight:
		return math.Pi / 2
	default:
		return 0
	}
}

func LoadEnemySprites() map[EnemyType]*ebiten.Image {
	out := make(map[EnemyType]*ebiten.Image)
	files := map[EnemyType]string{
		EnemyBasic: "enemy_a.png",
		EnemyFast:  "enemy_b.png",
		EnemyPower: "enemy_c.png",
	}
	for typ, file := range files {
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
			out[typ] = ebiten.NewImageFromImage(src)
			break
		}
	}
	return out
}

func (e *Enemy) Draw(screen *ebiten.Image, sprites map[EnemyType]*ebiten.Image) {
	if !e.Active {
		return
	}
	if e.SpawnTime > 0 {
		radius := float32(5 + (e.SpawnTime/5)%3*4)
		cx, cy := e.X+16, e.Y+16
		col := color.RGBA{245, 235, 185, 255}
		vector.StrokeLine(screen, cx-radius, cy, cx+radius, cy, 2, col, false)
		vector.StrokeLine(screen, cx, cy-radius, cx, cy+radius, 2, col, false)
		vector.StrokeLine(screen, cx-radius*.6, cy-radius*.6, cx+radius*.6, cy+radius*.6, 2, col, false)
		vector.StrokeLine(screen, cx-radius*.6, cy+radius*.6, cx+radius*.6, cy-radius*.6, 2, col, false)
		return
	}
	if img := sprites[e.Type]; img != nil {
		b := img.Bounds()
		w, h := float64(b.Dx()), float64(b.Dy())
		if w > 0 && h > 0 {
			scale := 32.0 / math.Max(w, h)
			angle := directionRotation(e.Dir)
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			op.GeoM.Translate(-w/2, -h/2)
			op.GeoM.Scale(scale, scale)
			op.GeoM.Rotate(angle)
			op.GeoM.Translate(float64(e.X)+16, float64(e.Y)+16)
			if e.Flash > 0 {
				op.ColorScale.Scale(1.8, 1.8, 1.8, 1)
			}
			screen.DrawImage(img, op)
		}
	} else {
		body := color.RGBA{200, 65, 52, 255}
		if e.Type == EnemyFast {
			body = color.RGBA{236, 178, 46, 255}
		}
		if e.Type == EnemyPower {
			body = color.RGBA{157, 80, 179, 255}
		}
		if e.Flash > 0 {
			body = color.RGBA{255, 255, 255, 255}
		}
		vector.DrawFilledRect(screen, e.X, e.Y, 32, 32, body, true)
		vector.DrawFilledRect(screen, e.X+4, e.Y+4, 6, 24, color.RGBA{30, 30, 30, 255}, true)
		vector.DrawFilledRect(screen, e.X+22, e.Y+4, 6, 24, color.RGBA{30, 30, 30, 255}, true)
		vector.DrawFilledRect(screen, e.X+12, e.Y+10, 8, 12, body, true)
	}
	drawIronShield(screen, e.X, e.Y, e.Dir, e.ArmorShield)
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
