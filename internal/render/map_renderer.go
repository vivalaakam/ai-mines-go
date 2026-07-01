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
	cellX := math.Floor((float64(mx)/vm.Camera.Zoom + vm.Camera.X) / TileSize)
	cellY := math.Floor((float64(my)/vm.Camera.Zoom + vm.Camera.Y) / TileSize)

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
		name := resourceName(vm, component["resourceId"])
		remaining, _ := component["remainingAmount"].(float64)
		lines = append(lines, fmt.Sprintf("%s: %.0f", name, remaining))
	}

	tx, ty := mx+16, my+8
	for _, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, tx, ty)
		ty += 14
	}
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
