package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ponytail: "unknown" is deliberately dark (fog of war) but kept well above
// pure black so the scaled-up fullscreen canvas reads as "there's a map here,
// mostly unexplored" rather than looking like empty screen real estate.
var kindColors = map[string]color.RGBA{
	"unknown":     {38, 38, 44, 255},
	"empty":       {70, 70, 78, 255},
	"deposit":     {110, 78, 55, 255},
	"obstacle":    {45, 45, 50, 255},
	"stairs_area": {200, 170, 60, 255},
}

func worldToScreen(cellX, cellY float64, cam Camera) (float64, float64) {
	x := (cellX*TileSize - cam.X) * cam.Zoom
	y := (cellY*TileSize - cam.Y) * cam.Zoom
	return x, y
}

// ScreenToCell converts a screen-space point (e.g. cursor position) into the
// world cell it falls in, inverting worldToScreen. Exported so the input
// layer can hit-test drag/drop against the same cell grid this package draws.
func ScreenToCell(screenX, screenY int, cam Camera) (float64, float64) {
	cellX := math.Floor((float64(screenX)/cam.Zoom + cam.X) / TileSize)
	cellY := math.Floor((float64(screenY)/cam.Zoom + cam.Y) / TileSize)
	return cellX, cellY
}

func drawMap(screen *ebiten.Image, vm ViewModel) {
	cells, _ := vm.LevelView["cells"].([]any)
	size := float32(TileSize * vm.Camera.Zoom)

	for _, raw := range cells {
		cell, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cx, _ := cell["x"].(float64)
		cy, _ := cell["y"].(float64)
		kind, _ := cell["kind"].(string)

		clr, ok := kindColors[kind]
		if !ok {
			clr = kindColors["unknown"]
		}
		if accessibility, _ := cell["accessibility"].(string); accessibility == "reachable" {
			clr = brighten(clr, 25)
		}

		x, y := worldToScreen(cx, cy, vm.Camera)
		vector.FillRect(screen, float32(x), float32(y), size-1, size-1, clr, false)
	}

	drawHoveredCell(screen, vm, cells, size)
}

// drawHoveredCell highlights the map cell under the mouse cursor and lists its
// resource components, so the player can inspect a deposit before assigning
// workers to it. Reads the cursor directly (Draw-time input read, no mutation).
func drawHoveredCell(screen *ebiten.Image, vm ViewModel, cells []any, size float32) {
	mx, my := ebiten.CursorPosition()
	cellX, cellY := ScreenToCell(mx, my, vm.Camera)

	var hovered map[string]any
	for _, raw := range cells {
		cell, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cx, _ := cell["x"].(float64)
		cy, _ := cell["y"].(float64)
		if cx == cellX && cy == cellY {
			hovered = cell
			break
		}
	}
	if hovered == nil {
		return
	}

	x, y := worldToScreen(cellX, cellY, vm.Camera)
	vector.StrokeRect(screen, float32(x), float32(y), size-1, size-1, 2, color.RGBA{255, 230, 90, 255}, false)

	lines := []string{fmt.Sprintf("(%.0f,%.0f) %v", cellX, cellY, hovered["kind"])}
	components, _ := hovered["components"].([]any)
	for _, raw := range components {
		component, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		remaining, _ := component["remainingAmount"].(float64)
		lines = append(lines, fmt.Sprintf("%s: %.0f", componentName(vm, component), remaining))
	}

	// A worker stands on the empty cell next to its deposit, not on the
	// deposit itself, so this cell's own components are usually empty. Add a
	// clearly separate block (its own coordinates, not merged into the lines
	// above) showing what the occupying worker actually mines, so hovering
	// the worker never looks like this empty cell holds resources itself.
	if targetComponents, targetX, targetY, ok := workerTargetComponents(vm, cells, hovered); ok {
		lines = append(lines, fmt.Sprintf("mining (%.0f,%.0f):", targetX, targetY))
		for _, raw := range targetComponents {
			component, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			remaining, _ := component["remainingAmount"].(float64)
			lines = append(lines, fmt.Sprintf("  %s: %.0f", componentName(vm, component), remaining))
		}
	}

	tx, ty := mx+16, my+8
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}
	bgW := float32(maxLen*6 + 8)
	bgH := float32(len(lines)*14 + 8)
	vector.FillRect(screen, float32(tx-4), float32(ty-4), bgW, bgH, color.RGBA{20, 20, 24, 220}, false)

	for _, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, tx, ty)
		ty += 14
	}
}

// workerTargetComponents finds the components (and coordinates) of the
// deposit cell mined by the worker occupying `hovered` (if any and if it's
// still actively mining - a worker just gone idle has no targetCellId), by
// following that worker's targetCellId into the same viewport cell list.
func workerTargetComponents(vm ViewModel, cells []any, hovered map[string]any) (components []any, targetX, targetY float64, ok bool) {
	occupiedBy, _ := hovered["occupiedBy"].(string)
	if occupiedBy == "" {
		return nil, 0, 0, false
	}

	workers, _ := vm.LevelView["workers"].([]any)
	var targetCellID string
	for _, raw := range workers {
		worker, wok := raw.(map[string]any)
		if !wok {
			continue
		}
		if id, _ := worker["id"].(string); id == occupiedBy {
			targetCellID, _ = worker["targetCellId"].(string)
			break
		}
	}
	if targetCellID == "" {
		return nil, 0, 0, false
	}
	tx, ty, parsed := ParseCellID(targetCellID)
	if !parsed {
		return nil, 0, 0, false
	}

	for _, raw := range cells {
		cell, cok := raw.(map[string]any)
		if !cok {
			continue
		}
		cx, _ := cell["x"].(float64)
		cy, _ := cell["y"].(float64)
		if cx == tx && cy == ty {
			cellComponents, _ := cell["components"].([]any)
			if len(cellComponents) == 0 {
				return nil, 0, 0, false
			}
			return cellComponents, tx, ty, true
		}
	}
	return nil, 0, 0, false
}

// componentName labels a deposit component for the tooltip: "rock" components
// carry no resourceId (REQUIREMENTS.md deposit mix is rock + one resource), so
// they get a fixed "Rock" label instead of an empty lookup miss.
func componentName(vm ViewModel, component map[string]any) string {
	if componentType, _ := component["type"].(string); componentType == "rock" {
		return "Rock"
	}
	return resourceName(vm, component["resourceId"])
}

// resourceName looks up a resource's display name from vm.Resources (populated
// by get_resources), falling back to the raw id if the list isn't loaded yet.
func resourceName(vm ViewModel, resourceID any) string {
	id, _ := resourceID.(string)
	list, _ := vm.Resources["resources"].([]any)
	for _, raw := range list {
		resource, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if resource["id"] == id {
			name, _ := resource["name"].(string)
			return name
		}
	}
	return id
}

func brighten(c color.RGBA, amount uint8) color.RGBA {
	add := func(v uint8) uint8 {
		if int(v)+int(amount) > 255 {
			return 255
		}
		return v + amount
	}
	return color.RGBA{add(c.R), add(c.G), add(c.B), c.A}
}
