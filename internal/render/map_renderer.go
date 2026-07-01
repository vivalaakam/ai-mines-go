package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var kindColors = map[string]color.RGBA{
	"unknown":     {20, 20, 24, 255},
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
