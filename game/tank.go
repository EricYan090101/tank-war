package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Tank struct {
	Team         int
	Hits         int
	MatchTank    bool
	X, Y         float32
	Speed        float32
	Size         float32
	Dir          Direction
	PlayerID     int
	Active       bool
	Lives        int
	Power        int
	ArmorShield  int // Persistent iron shield hits remaining, 0..3.
	FireCooldown int
	RespawnTimer int
	HitFlash     int
	SlideTimer   int
}

func NewTank(x, y float32, playerID int) *Tank {
	return &Tank{X: x, Y: y, Speed: 2, Size: 32, Dir: DirUp, PlayerID: playerID, Active: true, Lives: MaxLives, Power: 1}
}

func (t *Tank) Bounds() (float32, float32, float32, float32) { return t.X, t.Y, t.Size, t.Size }

func (t *Tank) IsOnIce(m *Map) bool {
	cx := int((t.X + t.Size/2) / TileSize)
	cy := int((t.Y + t.Size/2) / TileSize)
	return cx >= 0 && cx < MapCols && cy >= 0 && cy < MapRows && m.Tiles[cy][cx] == TileIce
}

func (t *Tank) UpdatePlayer(m *Map, enemies []*Enemy, otherPlayers []*Tank) {
	keys := [...]ebiten.Key{ebiten.KeyW, ebiten.KeyS, ebiten.KeyA, ebiten.KeyD}
	if t.PlayerID == 1 {
		keys = [...]ebiten.Key{ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight}
	}
	durations := [4]int{}
	for i, key := range keys {
		durations[i] = inpututil.KeyPressDuration(key)
	}
	dir, moving := preferredDirection(durations, t.Dir)
	t.updateWithInput(m, enemies, otherPlayers, VersusInput{Dir: dir, Moving: moving})
}

// The most recently pressed direction wins when keys overlap during a turn.
func preferredDirection(held [4]int, current Direction) (Direction, bool) {
	best := 0
	dir := current
	if held[current] > 0 {
		best = held[current]
	}
	for i, frames := range held {
		if frames > 0 && (best == 0 || frames < best) {
			best = frames
			dir = Direction(i)
		}
	}
	return dir, best > 0
}

func (t *Tank) applyMovement(m *Map, enemies []*Enemy, players []*Tank, dir Direction, moving bool) {
	onIce := t.IsOnIce(m)
	if moving {
		if (dir == DirUp || dir == DirDown) != (t.Dir == DirUp || t.Dir == DirDown) {
			// Align to the doubled NES 8px lane grid. Sweep the adjustment so turns
			// cannot teleport through brick fragments or adjacent tanks.
			if dir == DirUp || dir == DirDown {
				target := snapTo(t.X, 16, 8)
				moveWithCollisionAndTanks(t, m, sign(target-t.X), 0, abs(target-t.X), enemies, players)
			} else {
				target := snapTo(t.Y, 16, 8)
				moveWithCollisionAndTanks(t, m, 0, sign(target-t.Y), abs(target-t.Y), enemies, players)
			}
		}
		t.Dir = dir
		if onIce {
			t.SlideTimer = 24
		} else {
			t.SlideTimer = 0
		}
	} else {
		if !onIce || t.SlideTimer <= 0 {
			t.SlideTimer = 0
			return
		}
		t.SlideTimer--
	}
	dx, dy := t.Dir.Delta()
	x, y := t.X, t.Y
	moveWithCollisionAndTanks(t, m, dx, dy, t.Speed, enemies, players)
	if t.X == x && t.Y == y {
		t.SlideTimer = 0
	}
	t.ConstrainBounds(PlayfieldWidth, ScreenHeight)
}

func moveWithCollisionAndTanks(t *Tank, m *Map, dx, dy, distance float32, enemies []*Enemy, otherPlayers []*Tank) {
	remaining := distance
	for remaining > 0 {
		step := float32(1)
		if remaining < step {
			step = remaining
		}
		nx, ny := t.X+dx*step, t.Y+dy*step
		if m.CheckTileCollision(nx, ny, t.Size, t.Size) {
			return
		}
		for _, e := range enemies {
			if e == nil || !e.Active {
				continue
			}
			if rectsOverlap(nx, ny, t.Size, t.Size, e.X, e.Y, e.Size, e.Size) {
				return
			}
		}
		for _, p := range otherPlayers {
			if p == nil || p == t || !p.Active {
				continue
			}
			if rectsOverlap(nx, ny, t.Size, t.Size, p.X, p.Y, p.Size, p.Size) {
				return
			}
		}
		t.X, t.Y = nx, ny
		remaining -= step
	}
}

func snapTo(v, grid, threshold float32) float32 {
	nearest := float32(int(v/grid+0.5)) * grid
	if abs(v-nearest) <= threshold {
		return nearest
	}
	return v
}

func (t *Tank) ConstrainBounds(screenWidth, screenHeight float32) {
	if t.X < 0 {
		t.X = 0
	}
	if t.Y < 0 {
		t.Y = 0
	}
	if t.X > screenWidth-t.Size {
		t.X = screenWidth - t.Size
	}
	if t.Y > screenHeight-t.Size {
		t.Y = screenHeight - t.Size
	}
}

func (t *Tank) MaxBullets() int {
	if t.Power >= 3 {
		return 2
	}
	return 1
}

func (t *Tank) CanFire(maxBullets, current int) bool {
	return t.Active && t.FireCooldown == 0 && current < maxBullets
}

func (t *Tank) Fire() *Bullet {
	if t.Power < 1 {
		t.Power = 1
	}
	// Bullet occupancy controls repeat fire; keep a short input-independent delay.
	t.FireCooldown = 8
	dx, dy := t.Dir.Delta()
	b := NewBullet(t.X+14+dx*15, t.Y+14+dy*15, t.Dir, OwnerPlayer, t.Power)
	b.PlayerID = t.PlayerID
	return b
}

func (t *Tank) TakeHit() bool {
	if !t.Active {
		return false
	}
	if absorbIronHit(&t.ArmorShield) {
		t.HitFlash = 5
		return false
	}
	t.Active = false
	t.Lives--
	t.RespawnTimer = 90
	t.Power = 1
	t.FireCooldown = 0
	t.HitFlash = 20
	return true
}

func (t *Tank) Respawn(x, y float32) {
	t.X, t.Y = x, y
	t.Dir = DirUp
	t.Active = true
	t.FireCooldown = 0
	t.HitFlash = 0
	t.SlideTimer = 0
}

func (t *Tank) Draw(screen *ebiten.Image) {
	if !t.Active {
		return
	}

	if !t.drawPlayerSprite(screen) {
		body := color.RGBA{226, 192, 40, 255}
		if t.PlayerID == 1 {
			body = color.RGBA{72, 174, 92, 255}
		}
		if t.MatchTank {
			body = uiGold
			if t.Team == 1 {
				body = uiGreen
			}
		}
		if t.HitFlash > 0 {
			body = color.RGBA{255, 255, 255, 255}
		}

		// 32x32 tank: twice the old size so it fills the 32px-wide corridors in the map.
		vector.DrawFilledRect(screen, t.X, t.Y, 32, 32, color.RGBA{34, 34, 30, 255}, true)
		vector.DrawFilledRect(screen, t.X+3, t.Y+2, 8, 28, body, true)
		vector.DrawFilledRect(screen, t.X+21, t.Y+2, 8, 28, body, true)
		vector.DrawFilledRect(screen, t.X+10, t.Y+6, 12, 20, body, true)
		vector.DrawFilledRect(screen, t.X+15, t.Y+7, 2, 16, color.RGBA{247, 240, 186, 255}, true)

		barrel := color.RGBA{247, 240, 186, 255}
		switch t.Dir {
		case DirUp:
			vector.DrawFilledRect(screen, t.X+15, t.Y-7, 3, 10, barrel, true)
		case DirDown:
			vector.DrawFilledRect(screen, t.X+15, t.Y+29, 3, 10, barrel, true)
		case DirLeft:
			vector.DrawFilledRect(screen, t.X-7, t.Y+15, 10, 3, barrel, true)
		case DirRight:
			vector.DrawFilledRect(screen, t.X+29, t.Y+15, 10, 3, barrel, true)
		}

	}

	drawIronShield(screen, t.X, t.Y, t.Dir, t.ArmorShield)
}

func (t *Tank) updateWithInput(m *Map, enemies []*Enemy, otherPlayers []*Tank, input VersusInput) {
	if !t.Active {
		if t.RespawnTimer > 0 {
			t.RespawnTimer--
		}
		if t.RespawnTimer == 0 && t.Lives > 0 {
			x, y := coopSpawn(t.PlayerID)
			blocked := false
			for _, other := range otherPlayers {
				if other != t && other.Active && rectsOverlap(x, y, 32, 32, other.X, other.Y, 32, 32) {
					blocked = true
				}
			}
			if !blocked {
				t.Respawn(x, y)
			}
		}
		return
	}
	if t.FireCooldown > 0 {
		t.FireCooldown--
	}
	if t.HitFlash > 0 {
		t.HitFlash--
	}

	t.applyMovement(m, enemies, otherPlayers, input.Dir, input.Moving)
}
