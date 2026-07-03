// Package render turns Lua view-models (from engine.read) into pixels. It never
// reads authoritative state directly and never mutates it - only local
// hover/selection/camera state may live here (REQUIREMENTS.md §Render/UI Layer).
package render

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// TileSize is the on-screen pixel size of one game cell at zoom=1.
const TileSize = 24

// ScreenWidth/Height is the fixed logical resolution Game.Layout reports.
// Ebitengine scales this canvas up to fill the real window/fullscreen output,
// so UI elements sized against these constants scale with the screen for free.
//
// The screen splits into two columns: the map/game field on the left
// (MapWidth wide) and a full-height info sidebar (SidebarWidth wide, opaque
// background) on the right for resources/orders - anything that isn't a
// direct map interaction.
const (
	ScreenWidth  = 1280
	ScreenHeight = 720
	SidebarWidth = 300
	MapWidth     = ScreenWidth - SidebarWidth
)

// ViewportCellCounts returns how many cells to query/draw so the map field
// (not the full screen - the sidebar reserves its own SidebarWidth column)
// always fills its area regardless of zoom - zooming in shows fewer, bigger
// cells; zooming out shows more, smaller cells, but the covered area never
// grows or shrinks.
func ViewportCellCounts(zoom float64) (int, int) {
	if zoom <= 0 {
		zoom = 1
	}
	w := int(math.Ceil(MapWidth/(TileSize*zoom))) + 1
	h := int(math.Ceil(ScreenHeight/(TileSize*zoom))) + 1
	return w, h
}

type Camera struct {
	X, Y float64
	Zoom float64
}

// MergeConfirm describes an in-progress "merge these two workers?" modal,
// shown after the player click-selects a worker and then clicks another
// worker of the same level.
type MergeConfirm struct {
	Level int
}

// ViewModel bundles everything one frame's Draw needs, all sourced from
// engine.read query results (map[string]any as decoded by luaengine).
type ViewModel struct {
	Camera           Camera
	LevelView        map[string]any
	PlayerSummary    map[string]any
	Workers          map[string]any
	Resources        map[string]any
	AvailableOrders  map[string]any
	ActiveOrders     map[string]any
	OrderEventLog    []string
	DraggingWorkerID string
	SelectedWorkerID string
	MergeConfirm     *MergeConfirm
	// HoverPos overrides the mouse cursor as the cell-inspection hover point,
	// used by the gamepad cell cursor. nil means fall back to the mouse.
	HoverPos *image.Point
}

func Draw(screen *ebiten.Image, vm ViewModel) {
	drawMap(screen, vm)
	drawWorkers(screen, vm)
	drawUI(screen, vm)
	drawSidebar(screen, vm)
	drawMergeModal(screen, vm)
}
