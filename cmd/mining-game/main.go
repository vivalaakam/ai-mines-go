package main

import (
	"database/sql"
	"errors"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/vivalaakam/ai-mines-go/internal/app"
	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/persistence"
)

const (
	saveDBPath    = "./save.db"
	saveID        = "default"
	devSeedPhrase = "dev-seed"
)

func main() {
	store, err := persistence.Open(saveDBPath)
	if err != nil {
		log.Fatalf("failed to open save database: %v", err)
	}
	defer store.Close()

	var engine *luaengine.Engine
	loaded, err := store.LoadEngine(saveID)
	switch {
	case err == nil:
		engine = loaded
		log.Printf("loaded existing save %q", saveID)
	case errors.Is(err, sql.ErrNoRows):
		engine, err = store.CreateNewEngine(saveID, devSeedPhrase)
		if err != nil {
			log.Fatalf("failed to start new game: %v", err)
		}
		log.Printf("created new save %q", saveID)
	default:
		log.Fatalf("failed to load save %q: %v", saveID, err)
	}
	defer engine.Close()

	game := app.NewGame(engine, store, saveID, "level_1")

	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Idle Mining Game")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
