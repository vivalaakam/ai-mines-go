package render

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// workerLevelPalette gives each worker level a distinct, stable color so
// same-level workers (the only ones that can merge) are visually grouped at
// a glance. Levels beyond the palette wrap around rather than erroring.
var workerLevelPalette = []color.RGBA{
	{90, 170, 230, 255},  // 1: blue
	{90, 220, 120, 255},  // 2: green
	{230, 210, 90, 255},  // 3: yellow
	{230, 150, 90, 255},  // 4: orange
	{200, 100, 220, 255}, // 5: purple
	{230, 90, 90, 255},   // 6: red
}

func workerLevelColor(level int) color.RGBA {
	if level < 1 {
		level = 1
	}
	return workerLevelPalette[(level-1)%len(workerLevelPalette)]
}

// ParseCellID turns a "x,y" cell id (as used for positionCellId/targetCellId)
// back into world coordinates for placing a worker sprite.
func ParseCellID(id string) (float64, float64, bool) {
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
		cx, cy, ok := ParseCellID(positionCellID)
		if !ok {
			continue
		}
		level, _ := worker["level"].(float64)
		clr := workerLevelColor(int(level))

		x, y := worldToScreen(cx+0.5, cy+0.5, vm.Camera)
		vector.FillCircle(screen, float32(x), float32(y), radius, clr, true)

		id, _ := worker["id"].(string)
		if id != "" && id == vm.DraggingWorkerID {
			vector.StrokeCircle(screen, float32(x), float32(y), radius+2, 2, color.RGBA{255, 255, 255, 255}, true)
		}
		if state, _ := worker["state"].(string); state == "blocked_by_storage" {
			vector.StrokeCircle(screen, float32(x), float32(y), radius, 2, color.RGBA{220, 40, 40, 255}, true)
		}

		levelLabel := strconv.Itoa(int(level))
		ebitenutil.DebugPrintAt(screen, levelLabel, int(x)-3*len(levelLabel), int(y)-6)
	}
}
