package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"log"
	"os"
	"path/filepath"

	"tank-war/game" // 导入自定义的 game 包

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/tank-icon.png
var windowIcon []byte

func main() {
	// Resolve the bundled assets relative to the EXE, including shortcut launches.
	if exe, err := os.Executable(); err == nil {
		if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "Source")); err == nil {
			_ = os.Chdir(filepath.Dir(exe))
		}
	}
	if icon, _, err := image.Decode(bytes.NewReader(windowIcon)); err == nil {
		ebiten.SetWindowIcon([]image.Image{icon})
	}

	gameInstance := game.NewGame()

	ebiten.SetWindowSize(game.ScreenWidth*2, game.ScreenHeight*2)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(game.ScreenWidth, game.ScreenHeight, -1, -1)
	ebiten.SetRunnableOnUnfocused(true) // Keep the host and lobby connected while switching windows.
	ebiten.SetWindowTitle("Tank Battle | Field Command")

	if err := ebiten.RunGame(gameInstance); err != nil {
		log.Fatal(err)
	}
}
