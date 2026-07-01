package app

import (
	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/persistence"
)

// TicksPerSecond is Ebitengine's default TPS; one game tick = one real second
// (REQUIREMENTS.md §6), so this is also updatesPerGameTick for the accumulator.
const TicksPerSecond = 60

// Game is the Ebitengine entry point. It holds no gameplay state of its own -
// everything authoritative is fetched from the Lua engine via apply/read on
// demand (REQUIREMENTS.md: "Go must not mutate authoritative state directly").
type Game struct {
	engine      *luaengine.Engine
	store       *persistence.Adapter
	saveID      string
	camera      *Camera
	accumulator *TickAccumulator
	levelID     string
}

// NewGame wires an already-loaded/created engine to the Ebitengine loop. store
// and saveID may be left as nil/"" to run without autosave (e.g. in tests).
func NewGame(engine *luaengine.Engine, store *persistence.Adapter, saveID string, levelID string) *Game {
	return &Game{
		engine:      engine,
		store:       store,
		saveID:      saveID,
		camera:      NewCamera(),
		accumulator: NewTickAccumulator(TicksPerSecond),
		levelID:     levelID,
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
