package app

import (
	"image"
	"log"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// clickMoveThreshold is how far (in screen pixels) the cursor may move
// between mouse-down and mouse-up for the gesture to still count as a click
// (click-to-select) rather than a drag (drag-and-drop).
const clickMoveThreshold = 6

// handleWorkerDrag handles both worker gestures. Dragging a worker sprite
// onto another worker of the same level asks Lua to merge them immediately
// (merge_workers itself enforces the same-level/idle rules); dragging it
// onto a deposit cell asks Lua to reassign it there. A plain click instead
// feeds handleWorkerClick's click-to-select "cut/paste" flow. Go only picks
// the source/target from the cached view - Lua remains the only place that
// validates and mutates game state.
func (g *Game) handleWorkerDrag() error {
	if g.lastLevelView == nil {
		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if g.pendingMerge != nil {
			g.suppressNextClick = true
			return g.resolvePendingMergeClick(mx, my)
		}
		if image.Pt(mx, my).In(render.HireWorkerButton) {
			g.suppressNextClick = true
			return nil
		}
		if consumed, err := g.handleOrderButtonClick(mx, my); consumed || err != nil {
			g.suppressNextClick = consumed
			return err
		}
		if mx >= render.MapWidth {
			// Sidebar area but not on a button - never a map interaction.
			g.suppressNextClick = true
			return nil
		}
		g.suppressNextClick = false
		g.pressPos = image.Pt(mx, my)
		cx, cy := render.ScreenToCell(mx, my, g.renderCamera())
		g.draggingWorkerID = workerAtCell(g.lastLevelView, cx, cy)
		return nil
	}

	if !inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return nil
	}
	if g.suppressNextClick {
		g.suppressNextClick = false
		return nil
	}

	workerID := g.draggingWorkerID
	g.draggingWorkerID = ""

	mx, my := ebiten.CursorPosition()
	cx, cy := render.ScreenToCell(mx, my, g.renderCamera())

	if isNearby(g.pressPos, image.Pt(mx, my)) {
		return g.handleWorkerClick(workerID, cx, cy)
	}
	if workerID == "" {
		return nil
	}

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

// handleWorkerClick implements the click-to-select "cut/paste" gesture:
// click a worker to select it, click a deposit cell to move the selected
// worker there (equivalent to dragging it), click another worker of the
// same level to open the merge-confirmation modal, or click the
// already-selected worker again to deselect it.
func (g *Game) handleWorkerClick(workerID string, cx, cy float64) error {
	if workerID != "" {
		switch g.selectedWorkerID {
		case "":
			g.selectedWorkerID = workerID
		case workerID:
			g.selectedWorkerID = ""
		default:
			selectedLevel, _ := workerLevelAt(g.lastLevelView, g.selectedWorkerID)
			targetLevel, _ := workerLevelAt(g.lastLevelView, workerID)
			if selectedLevel == targetLevel {
				g.pendingMerge = &PendingMerge{WorkerA: g.selectedWorkerID, WorkerB: workerID, Level: int(selectedLevel)}
			}
			g.selectedWorkerID = ""
		}
		return nil
	}

	if g.selectedWorkerID == "" {
		return nil
	}

	targetCellID, ok := depositCellAt(g.lastLevelView, cx, cy)
	if !ok {
		return nil
	}

	workerToMove := g.selectedWorkerID
	g.selectedWorkerID = ""
	result, err := g.engine.Apply("assign_worker_to_deposit", map[string]any{
		"workerId":     workerToMove,
		"levelId":      g.levelID,
		"targetCellId": targetCellID,
	})
	if err != nil {
		return err
	}
	if !result.OK {
		log.Printf("assign_worker_to_deposit rejected: %+v", result.Error)
	}
	return nil
}

// resolvePendingMergeClick handles a click while the merge-confirmation
// modal is open: clicking Yes merges the pending pair; clicking anywhere
// else (No, or outside the modal) cancels without mutating state.
func (g *Game) resolvePendingMergeClick(mx, my int) error {
	merge := g.pendingMerge
	g.pendingMerge = nil
	if !(image.Pt(mx, my).In(render.MergeModalYesButton)) {
		return nil
	}
	result, err := g.engine.Apply("merge_workers", map[string]any{"workerIds": []any{merge.WorkerA, merge.WorkerB}})
	if err != nil {
		return err
	}
	if !result.OK {
		log.Printf("merge_workers rejected: %+v", result.Error)
	}
	return nil
}

func isNearby(a, b image.Point) bool {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx+dy*dy <= clickMoveThreshold*clickMoveThreshold
}

func workerLevelAt(levelView map[string]any, workerID string) (float64, bool) {
	workers, _ := levelView["workers"].([]any)
	for _, raw := range workers {
		worker, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := worker["id"].(string); id == workerID {
			level, _ := worker["level"].(float64)
			return level, true
		}
	}
	return 0, false
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
