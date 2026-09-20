package game

import "github.com/hajimehoshi/ebiten/v2"

const IronShieldHits = 3
const EnemyShieldChance = 25

func absorbIronHit(hits *int) bool {
	if *hits <= 0 {
		return false
	}
	*hits--
	return true
}

// All damage frames share the original canvas, preserving placement as edges break.
func drawIronShield(screen *ebiten.Image, x, y float32, dir Direction, hits int) {
	if hits <= 0 {
		return
	}
	files := [...]string{"shield_hit2.png", "shield_hit1.png", "shield_full.png"}
	img := loadImageAsset(files[min(hits, IronShieldHits)-1])
	if img == nil {
		return
	}
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	// Plate overlays most of the hull; top notch and forward barrel remain visible.
	op.GeoM.Scale(34/float64(b.Dx()), 30/float64(b.Dy()))
	op.GeoM.Translate(-17, -12)
	op.GeoM.Rotate(directionRotation(dir))
	op.GeoM.Translate(float64(x)+16, float64(y)+16)
	screen.DrawImage(img, op)
}
