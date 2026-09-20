package game

import (
	"bufio"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	TileEmpty = iota
	TileBrick
	TileSteel
	TileBush
	TileBase
	TileWater
	TileIce
	TileBaseRuins
)

const (
	BrickTL uint8 = 1 << iota
	BrickTR
	BrickBL
	BrickBR
	BrickFull = BrickTL | BrickTR | BrickBL | BrickBR
)

type Map struct {
	Tiles            [MapRows][MapCols]int
	BrickMask        [MapRows][MapCols]uint8
	BaseFortified    bool
	baseFortifyTimer int

	BrickImg *ebiten.Image
	SteelImg *ebiten.Image
	BushImg  *ebiten.Image
	WaterImg *ebiten.Image
	IceImg   *ebiten.Image
}

func NewMap() *Map {
	m := &Map{}
	m.LoadTextures()
	return m
}

var loadedImageAssets = map[string]*ebiten.Image{}

func loadImageAsset(file string) *ebiten.Image {
	if img := loadedImageAssets[file]; img != nil {
		return img
	}
	for _, p := range []string{
		filepath.Join("Source", "Pictures", file),
		filepath.Join("..", "Source", "Pictures", file),
	} {
		if img, _, err := ebitenutil.NewImageFromFile(p); err == nil {
			loadedImageAssets[file] = img
			return img
		}
	}
	return nil
}

func (m *Map) LoadTextures() {
	m.BrickImg = loadImageAsset("brick.png")
	m.SteelImg = loadImageAsset("stone.png")
	m.BushImg = loadImageAsset("bush.png")
	m.WaterImg = loadImageAsset("water.png")
	m.IceImg = loadImageAsset("ice.png")
}

func (m *Map) LoadLevelFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	m.Tiles = [MapRows][MapCols]int{}
	m.BrickMask = [MapRows][MapCols]uint8{}

	scanner := bufio.NewScanner(file)
	row := 0
	for scanner.Scan() && row < MapRows {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		for col := 0; col < len(parts) && col < MapCols; col++ {
			v, e := strconv.Atoi(parts[col])
			if e != nil {
				continue
			}
			m.Tiles[row][col] = v
			if v == TileBrick {
				m.BrickMask[row][col] = BrickFull
			}
		}
		row++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	m.ensureBrickMasks()
	return nil
}

func (m *Map) ensureBrickMasks() {
	for r := 0; r < MapRows; r++ {
		for c := 0; c < MapCols; c++ {
			if m.Tiles[r][c] == TileBrick {
				if m.BrickMask[r][c] == 0 {
					m.BrickMask[r][c] = BrickFull
				}
			} else {
				m.BrickMask[r][c] = 0
			}
		}
	}
}

func (m *Map) Update() {
	if m.baseFortifyTimer <= 0 {
		return
	}
	m.baseFortifyTimer--
	if m.baseFortifyTimer == 0 {
		m.unfortifyBase()
	}
}

func (m *Map) IsSolidTile(tile int) bool {
	return tile == TileBrick || tile == TileSteel || tile == TileBase || tile == TileWater
}

func rectsOverlap(ax, ay, aw, ah, bx, by, bw, bh float32) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

func (m *Map) CheckTileCollision(x, y, w, h float32) bool {
	if x < 0 || y < 0 || x+w > PlayfieldWidth || y+h > ScreenHeight {
		return true
	}

	minC := int(x) / TileSize
	minR := int(y) / TileSize
	maxC := int((x + w - 0.001)) / TileSize
	maxR := int((y + h - 0.001)) / TileSize
	if minC < 0 || minR < 0 || maxC >= MapCols || maxR >= MapRows {
		return true
	}

	for r := minR; r <= maxR; r++ {
		for c := minC; c <= maxC; c++ {
			tx, ty := float32(c*TileSize), float32(r*TileSize)
			if m.Tiles[r][c] == TileBrick {
				if m.rectHitsBrickMask(x, y, w, h, tx, ty, m.BrickMask[r][c]) {
					return true
				}
				continue
			}
			if m.IsSolidTile(m.Tiles[r][c]) && rectsOverlap(x, y, w, h, tx, ty, TileSize, TileSize) {
				return true
			}
		}
	}
	return false
}

func (m *Map) rectHitsBrickMask(x, y, w, h, tileX, tileY float32, mask uint8) bool {
	pieces := []struct {
		bit  uint8
		x, y float32
	}{{BrickTL, tileX, tileY}, {BrickTR, tileX + 8, tileY}, {BrickBL, tileX, tileY + 8}, {BrickBR, tileX + 8, tileY + 8}}
	for _, p := range pieces {
		if mask&p.bit != 0 && rectsOverlap(x, y, w, h, p.x, p.y, 8, 8) {
			return true
		}
	}
	return false
}

// HitBullet returns true when the bullet has hit a tile. Brick tiles lose one 8x8 quadrant.
// Bullets pass through water and ice; steel requires power 4 (three stars) to break.
func (m *Map) HitBullet(x, y float32, dir Direction, power int, super bool, blastRadius float32) (hit bool, baseDestroyed bool) {
	if x < 0 || y < 0 || x >= PlayfieldWidth || y >= ScreenHeight {
		return true, false
	}
	c, r := int(x)/TileSize, int(y)/TileSize
	if r < 0 || r >= MapRows || c < 0 || c >= MapCols {
		return true, false
	}

	switch m.Tiles[r][c] {
	case TileBrick:
		// A bullet has an actual impact/explosion radius. The blast is applied
		// to every 8x8 brick quadrant touched by the circular radius. This
		// prevents the old "only one side of the 16x16 wall disappears" feeling.
		changed := m.damageBrickBlast(x, y, blastRadius)
		if !changed {
			// The projectile may have entered an already-open part of a brick.
			return false, false
		}
		return true, false
	case TileSteel:
		if super || power >= 4 {
			m.Tiles[r][c] = TileEmpty
		}
		return true, false
	case TileBase:
		for y := 0; y < MapRows; y++ {
			for x := 0; x < MapCols; x++ {
				if m.Tiles[y][x] == TileBase {
					m.Tiles[y][x] = TileBaseRuins
				}
			}
		}
		return true, true
	case TileWater, TileIce:
		return false, false
	default:
		return false, false
	}
}

func (m *Map) damageBrickBlast(cx, cy, radius float32) bool {
	if radius <= 0 {
		radius = 10
	}
	minX := int((cx-radius)/TileSize) - 1
	maxX := int((cx+radius)/TileSize) + 1
	minY := int((cy-radius)/TileSize) - 1
	maxY := int((cy+radius)/TileSize) + 1
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= MapCols {
		maxX = MapCols - 1
	}
	if maxY >= MapRows {
		maxY = MapRows - 1
	}

	changed := false
	for r := minY; r <= maxY; r++ {
		for c := minX; c <= maxX; c++ {
			if m.Tiles[r][c] != TileBrick || m.BrickMask[r][c] == 0 {
				continue
			}
			tx, ty := float32(c*TileSize), float32(r*TileSize)
			pieces := []struct {
				bit  uint8
				x, y float32
			}{
				{BrickTL, tx, ty},
				{BrickTR, tx + 8, ty},
				{BrickBL, tx, ty + 8},
				{BrickBR, tx + 8, ty + 8},
			}
			for _, q := range pieces {
				if m.BrickMask[r][c]&q.bit == 0 {
					continue
				}
				if circleIntersectsRect(cx, cy, radius, q.x, q.y, 8, 8) {
					m.BrickMask[r][c] &^= q.bit
					changed = true
				}
			}
			if m.BrickMask[r][c] == 0 {
				m.Tiles[r][c] = TileEmpty
			}
		}
	}
	return changed
}

func circleIntersectsRect(cx, cy, radius, rx, ry, rw, rh float32) bool {
	closestX := cx
	if closestX < rx {
		closestX = rx
	}
	if closestX > rx+rw {
		closestX = rx + rw
	}
	closestY := cy
	if closestY < ry {
		closestY = ry
	}
	if closestY > ry+rh {
		closestY = ry + rh
	}
	dx, dy := cx-closestX, cy-closestY
	return dx*dx+dy*dy <= radius*radius
}

func (m *Map) brickMaskContainsPoint(mask uint8, localX, localY float32) bool {
	if localX < 0 || localX >= TileSize || localY < 0 || localY >= TileSize {
		return false
	}
	var bit uint8
	if localY < 8 {
		if localX < 8 {
			bit = BrickTL
		} else {
			bit = BrickTR
		}
	} else {
		if localX < 8 {
			bit = BrickBL
		} else {
			bit = BrickBR
		}
	}
	return mask&bit != 0
}

func (m *Map) removeBrickPiece(r, c int, localX, localY float32, dir Direction) {
	mask := m.BrickMask[r][c]
	if mask == 0 {
		m.Tiles[r][c] = TileEmpty
		return
	}

	// Each 16x16 brick is made of two 8x16 halves. A shot removes the
	// half that the projectile actually enters. This matches the classic
	// feel much better than removing only one 8x8 quadrant per shot.
	var target uint8
	if dir == DirLeft || dir == DirRight {
		if localX < 8 {
			target = BrickTL | BrickBL
		} else {
			target = BrickTR | BrickBR
		}
	} else {
		if localY < 8 {
			target = BrickTL | BrickTR
		} else {
			target = BrickBL | BrickBR
		}
	}

	mask &^= target
	m.BrickMask[r][c] = mask
	if mask == 0 {
		m.Tiles[r][c] = TileEmpty
	}
}

func (m *Map) FortifyBase() {
	m.BaseFortified = true
	m.baseFortifyTimer = 60 * 20
	for r := 22; r <= 25 && r < MapRows; r++ {
		for c := 9; c <= 16 && c < MapCols; c++ {
			if m.Tiles[r][c] == TileBrick {
				m.Tiles[r][c] = TileSteel
				m.BrickMask[r][c] = 0
			}
		}
	}
}

func (m *Map) unfortifyBase() {
	m.BaseFortified = false
	for r := 22; r <= 25 && r < MapRows; r++ {
		for c := 9; c <= 16 && c < MapCols; c++ {
			if m.Tiles[r][c] == TileSteel {
				m.Tiles[r][c] = TileBrick
				m.BrickMask[r][c] = BrickFull
			}
		}
	}
}

func (m *Map) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, ScreenWidth, ScreenHeight, color.RGBA{0, 0, 0, 255}, true)
	for r := 0; r < MapRows; r++ {
		for c := 0; c < MapCols; c++ {
			x, y := float32(c*TileSize), float32(r*TileSize)
			switch m.Tiles[r][c] {
			case TileBrick:
				m.drawBrick(screen, x, y, m.BrickMask[r][c])
			case TileSteel:
				m.drawAsset16(screen, m.SteelImg, x, y, color.RGBA{130, 136, 140, 255})
			case TileBush:
				// Smoke is deliberately rendered after units; TileBush retains map compatibility.
			case TileWater:
				if m.WaterImg != nil {
					m.drawTerrainPatch(screen, m.WaterImg, x, y, 1)
					m.drawWaterShore(screen, r, c)
				} else {
					m.drawAsset16(screen, nil, x, y, color.RGBA{79, 157, 184, 255})
				}
			case TileIce:
				if m.IceImg != nil {
					m.drawTerrainPatch(screen, m.IceImg, x, y, 1)
				} else {
					m.drawAsset16(screen, nil, x, y, color.RGBA{180, 220, 230, 255})
				}
			case TileBase, TileBaseRuins:
				m.drawBaseSprite(screen, r, c)
			}
		}
	}
}

func (m *Map) drawAsset16(screen *ebiten.Image, img *ebiten.Image, x, y float32, fallback color.RGBA) {
	if img != nil {
		b := img.Bounds()
		w, h := b.Dx(), b.Dy()
		if w >= 16 && h >= 16 {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			op.GeoM.Scale(16/float64(w), 16/float64(h))
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(img, op)
			return
		}
	}
	vector.DrawFilledRect(screen, x, y, 16, 16, fallback, true)
}

func (m *Map) drawBrick(screen *ebiten.Image, x, y float32, mask uint8) {
	if mask == 0 {
		return
	}
	if m.BrickImg != nil && m.BrickImg.Bounds().Dx() >= 16 && m.BrickImg.Bounds().Dy() >= 16 {
		pieces := []struct {
			bit    uint8
			sx, sy int
			dx, dy float64
		}{{BrickTL, 0, 0, 0, 0}, {BrickTR, 8, 0, 8, 0}, {BrickBL, 0, 8, 0, 8}, {BrickBR, 8, 8, 8, 8}}
		for _, p := range pieces {
			if mask&p.bit == 0 {
				continue
			}
			bounds := m.BrickImg.Bounds()
			w, h := bounds.Dx(), bounds.Dy()
			src := m.BrickImg.SubImage(image.Rect(bounds.Min.X+p.sx*w/16, bounds.Min.Y+p.sy*h/16, bounds.Min.X+(p.sx+8)*w/16, bounds.Min.Y+(p.sy+8)*h/16)).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			op.GeoM.Scale(8/float64(src.Bounds().Dx()), 8/float64(src.Bounds().Dy()))
			op.GeoM.Translate(float64(x)+p.dx, float64(y)+p.dy)
			screen.DrawImage(src, op)
		}
		return
	}

	brick := color.RGBA{184, 76, 48, 255}
	for _, p := range []struct {
		bit    uint8
		dx, dy float32
	}{{BrickTL, 0, 0}, {BrickTR, 8, 0}, {BrickBL, 0, 8}, {BrickBR, 8, 8}} {
		if mask&p.bit != 0 {
			vector.DrawFilledRect(screen, x+p.dx, y+p.dy, 8, 8, brick, true)
		}
	}
}

// Each 32px smoke block draws a complete feathered cloud with a small overlap.
// No rectangular crop is applied to its soft perimeter.
func (m *Map) DrawBushes(screen *ebiten.Image) {
	field := screen.SubImage(image.Rect(0, 0, PlayfieldWidth, ScreenHeight)).(*ebiten.Image)
	for r := 0; r < MapRows; r += 2 {
		for c := 0; c < MapCols; c += 2 {
			smoke := false
			for dy := 0; dy < 2 && r+dy < MapRows; dy++ {
				for dx := 0; dx < 2 && c+dx < MapCols; dx++ {
					smoke = smoke || m.Tiles[r+dy][c+dx] == TileBush
				}
			}
			if !smoke {
				continue
			}
			if m.BushImg == nil {
				vector.DrawFilledRect(field, float32(c*TileSize), float32(r*TileSize), 32, 32, color.RGBA{102, 96, 118, 224}, true)
				continue
			}
			b := m.BushImg.Bounds()
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(48/float64(b.Dx()), 48/float64(b.Dy()))
			op.GeoM.Translate(float64(c*TileSize-8), float64(r*TileSize-8))
			op.ColorScale.ScaleAlpha(0.88)
			field.DrawImage(m.BushImg, op)
		}
	}
}

func (m *Map) drawBase(screen *ebiten.Image, x, y float32) {
	vector.DrawFilledRect(screen, x, y, 16, 16, color.RGBA{198, 164, 48, 255}, true)
	vector.DrawFilledRect(screen, x+2, y+3, 12, 10, color.RGBA{40, 38, 24, 255}, true)
	vector.DrawFilledRect(screen, x+5, y+4, 2, 8, color.RGBA{236, 203, 92, 255}, true)
	vector.DrawFilledRect(screen, x+9, y+4, 2, 8, color.RGBA{236, 203, 92, 255}, true)
}

func loadFallbackStage(m *Map, stage int) {
	m.Tiles = [MapRows][MapCols]int{}
	m.BrickMask = [MapRows][MapCols]uint8{}

	for r := 0; r < MapRows; r++ {
		m.Tiles[r][0] = TileSteel
		m.Tiles[r][MapCols-1] = TileSteel
	}
	for c := 0; c < MapCols; c++ {
		m.Tiles[0][c] = TileSteel
		m.Tiles[MapRows-1][c] = TileSteel
	}
	seed := stage * 7919
	for r := 2; r < 22; r += 3 {
		for c := 2; c < 24; c += 4 {
			if (r+c+seed)%7 < 5 {
				for rr := r; rr < r+2 && rr < 23; rr++ {
					m.Tiles[rr][c] = TileBrick
					m.BrickMask[rr][c] = BrickFull
					if c+1 < 25 {
						m.Tiles[rr][c+1] = TileBrick
						m.BrickMask[rr][c+1] = BrickFull
					}
				}
			}
		}
	}

	// Base: four tiles in the lower center.
	for r := 24; r < 26; r++ {
		for c := 12; c < 14; c++ {
			m.Tiles[r][c] = TileBase
		}
	}
	m.ensureBrickMasks()
}

// A 64px material patch spans four map cells, rather than shrinking every
// ripple or smoke cloud into a separate 16px square. Collision stays per cell.
func (m *Map) drawTerrainPatch(s *ebiten.Image, img *ebiten.Image, x, y float32, alpha float32) {
	b := img.Bounds()
	// Use the cloud's filled interior for repeating cells, avoiding export-edge
	// transparency forming black seams across a continuous smoke field.
	if alpha < 1 {
		b = b.Inset(b.Dx() / 10)
	}
	w, h := b.Dx(), b.Dy()
	cx, cy := int(x)%64/16, int(y)%64/16
	left, top := b.Min.X+cx*w/4, b.Min.Y+cy*h/4
	right, bottom := b.Min.X+(cx+1)*w/4, b.Min.Y+(cy+1)*h/4
	part := img.SubImage(image.Rect(left, top, right, bottom)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(16/float64(right-left), 16/float64(bottom-top))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleAlpha(alpha)
	s.DrawImage(part, op)
}
