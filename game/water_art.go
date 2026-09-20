package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
)

var waterShoreMasks [16]*ebiten.Image

// Feather only exposed banks; joined water cells keep their full texture.
func (m *Map) drawWaterShore(screen *ebiten.Image, row, col int) {
	water := func(r, c int) bool {
		return r >= 0 && r < MapRows && c >= 0 && c < MapCols && m.Tiles[r][c] == TileWater
	}
	mask := 0
	if !water(row-1, col) {
		mask |= 1
	}
	if !water(row, col+1) {
		mask |= 2
	}
	if !water(row+1, col) {
		mask |= 4
	}
	if !water(row, col-1) {
		mask |= 8
	}
	if mask == 0 {
		return
	}
	if waterShoreMasks[mask] == nil {
		pixels := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				distance := TileSize
				if mask&1 != 0 && y < distance {
					distance = y
				}
				if mask&2 != 0 && TileSize-1-x < distance {
					distance = TileSize - 1 - x
				}
				if mask&4 != 0 && TileSize-1-y < distance {
					distance = TileSize - 1 - y
				}
				if mask&8 != 0 && x < distance {
					distance = x
				}
				// Three pixels of graduated shading retain a readable bank.
				if distance < 3 {
					alpha := [...]uint8{210, 125, 45}
					pixels.SetNRGBA(x, y, color.NRGBA{A: alpha[distance]})
				}
			}
		}
		waterShoreMasks[mask] = ebiten.NewImageFromImage(pixels)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(col*TileSize), float64(row*TileSize))
	screen.DrawImage(waterShoreMasks[mask], op)
}
