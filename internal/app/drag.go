package app

import (
	"image"
	"log"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// handleWorkerDrag lets the player drag a worker sprite: dropping it onto
// another worker of the same level asks Lua to merge them (merge_workers
// itself enforces the same-level/idle rules); dropping it onto a deposit
// cell asks Lua to reassign it to the nearest free adjacent cell. Go only
// picks the drag source/target from the cached view - Lua remains the only
// place that validates and mutates game state.
func (g *Game) handleWorkerDrag() error {
	if g.lastLevelView == nil {
		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if image.Pt(mx, my).In(render.HireWorkerButton) {
			return nil
		}
		cx, cy := render.ScreenToCell(mx, my, g.renderCamera())
		g.draggingWorkerID = workerAtCell(g.lastLevelView, cx, cy)
		return nil
	}

	if g.draggingWorkerID == "" || !inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return nil
	}

	workerID := g.draggingWorkerID
	g.draggingWorkerID = ""

	mx, my := ebiten.CursorPosition()
	cx, cy := render.ScreenToCell(mx, my, g.renderCamera())

	if targetWorkerID := workerAtCell(g.lastLevelView, cx, cy); targetWorkerID != "" && targetWorkerID != workerID {
		result, err := g.engine.Apply("merge_workers", map[string]any{"workerIds": []any{workerID, targetWorkerID}})
		if err != nil {
			return err
		}
		if !result.OK {
			log.Printf("merge_workers rejected: %+v", result.Error)
		}
		return nil
	}

	if targetCellID, ok := depositCellAt(g.lastLevelView, cx, cy); ok {
		result, err := g.engine.Apply("assign_worker_to_deposit", map[string]any{
			"workerId":     workerID,
			"levelId":      g.levelID,
			"targetCellId": targetCellID,
		})
		if err != nil {
			return err
		}
		if !result.OK {
			log.Printf("assign_worker_to_deposit rejected: %+v", result.Error)
		}
	}

	return nil
}

func (g *Game) renderCamera() render.Camera {
	return render.Camera{X: g.camera.X, Y: g.camera.Y, Zoom: g.camera.Zoom}
}

func workerAtCell(levelView map[string]any, cx, cy float64) string {
	workers, _ := levelView["workers"].([]any)
	for _, raw := range workers {
		worker, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		positionCellID, _ := worker["positionCellId"].(string)
		wx, wy, ok := render.ParseCellID(positionCellID)
		if ok && wx == cx && wy == cy {
			id, _ := worker["id"].(string)
			return id
		}
	}
	return ""
}

func depositCellAt(levelView map[string]any, cx, cy float64) (string, bool) {
	cells, _ := levelView["cells"].([]any)
	for _, raw := range cells {
		cell, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		x, _ := cell["x"].(float64)
		y, _ := cell["y"].(float64)
		if x == cx && y == cy && cell["kind"] == "deposit" {
			return cellID(x, y), true
		}
	}
	return "", false
}

// cellID formats world coordinates into the "x,y" id Lua uses as a cell key
// (state.levels[*].cells is keyed by fmt.Sprintf-free Lua string concat of
// integer x/y, e.g. "5,3").
func cellID(x, y float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64) + "," + strconv.FormatFloat(y, 'f', -1, 64)
}
