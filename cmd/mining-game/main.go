package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/vivalaakam/ai-mines-go/internal/app"
	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
)

const devSeedPhrase = "dev-seed"

func main() {
	engine, err := luaengine.New()
	if err != nil {
		log.Fatalf("failed to start lua engine: %v", err)
	}
	defer engine.Close()

	if err := engine.NewGame(devSeedPhrase); err != nil {
		log.Fatalf("failed to start new game: %v", err)
	}

	game := app.NewGame(engine, "level_1")

	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Idle Mining Game")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
