package game

import (
	"image"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

type ItemType int

const (
	ItemStar ItemType = iota
	ItemHelmet
	ItemShovel
	ItemBomb
	ItemClock
	ItemTank
)

func ItemName(t ItemType) string {
	switch t {
	case ItemHelmet:
		return "HELMET"
	case ItemShovel:
		return "SHOVEL"
	case ItemBomb:
		return "BOMB"
	case ItemClock:
		return "CLOCK"
	case ItemTank:
		return "TANK"
	default:
		return "STAR"
	}
}

type Item struct {
	X, Y   float32
	Type   ItemType
	Active bool
	Timer  int
}

var itemFiles = map[ItemType]string{
	ItemStar: "bonus_star.png", ItemHelmet: "bonus_helmet.png", ItemShovel: "bonus_shovel.png",
	ItemBomb: "bonus_grenade.png", ItemClock: "bonus_clock.png", ItemTank: "bonus_tank.png",
}

func NewItem(x, y float32) *Item {
	// Classic-feeling weighted distribution: Star and Tank show slightly less often.
	typ := ItemType(rand.Intn(100))
	var item ItemType
	switch {
	case typ < 20:
		item = ItemStar
	case typ < 38:
		item = ItemHelmet
	case typ < 53:
		item = ItemShovel
	case typ < 70:
		item = ItemBomb
	case typ < 86:
		item = ItemClock
	default:
		item = ItemTank
	}
	return &Item{X: x, Y: y, Type: item, Active: true, Timer: 60 * 10}
}

func (i *Item) Update() {
	if !i.Active {
		return
	}
	i.Timer--
	if i.Timer <= 0 {
		i.Active = false
	}
}

func LoadItemSprites() map[ItemType]*ebiten.Image {
	out := make(map[ItemType]*ebiten.Image)
	for typ, file := range itemFiles {
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

func drawScaled16(screen *ebiten.Image, img *ebiten.Image, x, y float32) {
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	bounds := img.Bounds()
	scale := 16 / float64(max(bounds.Dx(), bounds.Dy()))
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x)+(16-float64(bounds.Dx())*scale)/2, float64(y)+(16-float64(bounds.Dy())*scale)/2)
	screen.DrawImage(img, op)
}

func (i *Item) Draw(screen *ebiten.Image, sprites map[ItemType]*ebiten.Image, frame int) {
	if !i.Active {
		return
	}
	if i.Timer < 120 && (frame/5)%2 == 0 {
		return
	}
	drawScaled16(screen, sprites[i.Type], i.X, i.Y)
}

func (i *Item) Apply(g *Game, player *Tank) {
	switch i.Type {
	case ItemStar:
		if player.Power < 4 {
			player.Power++
		}
		g.setMessage([]string{"", "", "STAR 1 / HIGH SPEED", "STAR 2 / DOUBLE SHOT", "STAR 3 / STEEL BREAKER"}[player.Power])
	case ItemHelmet:
		player.ArmorShield = IronShieldHits
		g.setMessage("SHIELD!")
	case ItemShovel:
		g.Map.FortifyBase()
		g.setMessage("BASE FORTIFIED!")
	case ItemBomb:
		for _, e := range g.Enemies {
			if e.Active && e.SpawnTime == 0 {
				e.Active = false
				g.Score += e.Score
				g.StageStats.Kills[e.Type]++
				g.StageStats.BombKills++
				g.Explosions = append(g.Explosions, NewExplosion(e.X+8, e.Y+8, true))
			}
		}
		g.setMessage("TOTAL ANNIHILATION!")
	case ItemClock:
		g.FreezeTimer = 60 * 10
		g.setMessage("ENEMIES FROZEN!")
	case ItemTank:
		player.Lives++
		g.setMessage("1-UP!")
	}
	g.StageStats.ItemsCollected[i.Type]++
	i.Active = false
}
