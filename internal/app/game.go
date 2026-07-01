package app

import (
	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/persistence"
	"github.com/vivalaakam/ai-mines-go/internal/render"
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
	mapBounds   *MapBounds
}

// MapBounds is the known-generated extent of the current level, in world
// cell coordinates (inclusive). Populated from get_level_view's "bounds"
// field each Draw and used by Update to keep the camera from panning past
// the generated map into empty space.
type MapBounds struct {
	MinX, MinY, MaxX, MaxY float64
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

// Layout returns a fixed logical resolution rather than echoing back
// outsideWidth/outsideHeight. Ebitengine then scales this logical canvas up
// to fill the actual window/fullscreen output (letterboxed, aspect-preserved),
// which is what makes the map/HUD/buttons scale with the screen instead of
// staying pinned to a small corner of a large fullscreen framebuffer.
// CursorPosition() already reports clicks in these same logical coordinates,
// so hit-testing (e.g. render.HireWorkerButton) needs no extra conversion.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return render.ScreenWidth, render.ScreenHeight
}
