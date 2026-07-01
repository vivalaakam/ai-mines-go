package app

import (
	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
)

// TicksPerSecond is Ebitengine's default TPS; one game tick = one real second
// (REQUIREMENTS.md §6), so this is also updatesPerGameTick for the accumulator.
const TicksPerSecond = 60

// Game is the Ebitengine entry point. It holds no gameplay state of its own -
// everything authoritative is fetched from the Lua engine via apply/read on
// demand (REQUIREMENTS.md: "Go must not mutate authoritative state directly").
type Game struct {
	engine      *luaengine.Engine
	camera      *Camera
	accumulator *TickAccumulator
	levelID     string
}

func NewGame(engine *luaengine.Engine, levelID string) *Game {
	return &Game{
		engine:      engine,
		camera:      NewCamera(),
		accumulator: NewTickAccumulator(TicksPerSecond),
		levelID:     levelID,
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
