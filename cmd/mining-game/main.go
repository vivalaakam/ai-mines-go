package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

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

// startPprof, if MINES_PPROF is set (e.g. "localhost:6060"), serves Go's pprof
// endpoints on a background goroutine so a heap/goroutine profile can be
// captured from a live windowed run with `go tool pprof
// http://localhost:6060/debug/pprof/heap`. The authoritative Lua path and the
// render layer were measured headlessly and hold steady at ~84MB; the
// remaining memory growth appears on focus loss and lives outside runtime
// MemStats (Ebitengine's GPU atlas / Metal textures), so a real profile from
// the leaking state is the fastest way to pin the retaining path.
func startPprof() {
	addr := os.Getenv("MINES_PPROF")
	if addr == "" {
		return
	}
	go func() {
		log.Printf("pprof serving on http://%s/debug/pprof/", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("pprof server stopped: %v", err)
		}
	}()
}

func main() {
	startPprof()

	store, err := persistence.Open(saveDBPath)
	if err != nil {
		log.Fatalf("failed to open save database: %v", err)
	}
	defer func() { _ = store.Close() }()

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
	ebiten.SetFullscreen(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
