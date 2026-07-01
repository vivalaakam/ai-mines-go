package render

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var workerStateColors = map[string]color.RGBA{
	"idle":               {90, 170, 230, 255},
	"working":            {90, 220, 120, 255},
	"blocked_by_storage": {220, 90, 90, 255},
}

// parseCellID turns a "x,y" cell id (as used for positionCellId/targetCellId)
// back into world coordinates for placing a worker sprite.
func parseCellID(id string) (float64, float64, bool) {
	parts := strings.SplitN(id, ",", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	x, err1 := strconv.ParseFloat(parts[0], 64)
	y, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return x, y, true
}

func drawWorkers(screen *ebiten.Image, vm ViewModel) {
	workers, _ := vm.LevelView["workers"].([]any)
	radius := float32(TileSize*vm.Camera.Zoom) / 3

	for _, raw := range workers {
		worker, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		positionCellID, _ := worker["positionCellId"].(string)
		if positionCellID == "" {
			continue
		}
		cx, cy, ok := parseCellID(positionCellID)
		if !ok {
			continue
		}
		state, _ := worker["state"].(string)
		clr, ok := workerStateColors[state]
		if !ok {
			clr = workerStateColors["idle"]
		}

		x, y := worldToScreen(cx+0.5, cy+0.5, vm.Camera)
		vector.FillCircle(screen, float32(x), float32(y), radius, clr, true)
	}
}
