package game

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestReplacementAssetsHaveUsableTransparency(t *testing.T) {
	for _, name := range []string{"base", "base_destroyed", "shield_full", "shield_hit1", "shield_hit2", "enemy_a", "enemy_b", "enemy_c", "enemy_armored", "bonus_star", "bonus_helmet", "bonus_shovel", "bonus_grenade", "bonus_clock", "bonus_tank", "water", "brick", "stone", "ice", "bush", "player_yellow", "player_green"} {
		f, err := os.Open(filepath.Join("..", "Source", "Pictures", name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		b := img.Bounds()
		if b.Dx() < 16 || b.Dy() < 16 {
			t.Fatal("asset too small")
		}
		if name == "player_yellow" || name == "player_green" {
			_, _, _, alpha := img.At(0, 0).RGBA()
			if alpha > 256 {
				t.Fatal("tank has opaque outside background")
			}
		}
		if name == "bush" {
			semi := 0
			for y := 0; y < b.Dy(); y += 32 {
				for x := 0; x < b.Dx(); x += 32 {
					_, _, _, a := img.At(x, y).RGBA()
					if a > 0 && a < 65535 {
						semi++
					}
				}
			}
			if semi == 0 {
				t.Fatal("smoke lost alpha")
			}
		}
	}
}
