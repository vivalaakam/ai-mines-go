// Package render turns Lua view-models (from engine.read) into pixels. It never
// reads authoritative state directly and never mutates it - only local
// hover/selection/camera state may live here (REQUIREMENTS.md §Render/UI Layer).
package render

import "github.com/hajimehoshi/ebiten/v2"

// TileSize is the on-screen pixel size of one game cell at zoom=1.
const TileSize = 24

// ViewportCellsWide/Tall bound how many cells get queried and drawn per frame.
const (
	ViewportCellsWide = 40
	ViewportCellsTall = 30
)

type Camera struct {
	X, Y float64
	Zoom float64
}

// ViewModel bundles everything one frame's Draw needs, all sourced from
// engine.read query results (map[string]any as decoded by luaengine).
type ViewModel struct {
	Camera        Camera
	LevelView     map[string]any
	PlayerSummary map[string]any
	ShiftSummary  map[string]any
}

func Draw(screen *ebiten.Image, vm ViewModel) {
	drawMap(screen, vm)
	drawWorkers(screen, vm)
	drawUI(screen, vm)
}
