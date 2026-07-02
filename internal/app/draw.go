package app

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// Draw reads view-models from the Lua engine and hands them to the render
// package. It never touches authoritative state (REQUIREMENTS.md: "Go must not
// mutate authoritative state in Draw").
func (g *Game) Draw(screen *ebiten.Image) {
	viewportX := math.Floor(g.camera.X / render.TileSize)
	viewportY := math.Floor(g.camera.Y / render.TileSize)
	cellsWide, cellsTall := render.ViewportCellCounts(g.camera.Zoom)

	levelView, err := g.engine.Read("get_level_view", map[string]any{
		"levelId": g.levelID,
		"viewport": map[string]any{
			"x":      viewportX,
			"y":      viewportY,
			"width":  float64(cellsWide),
			"height": float64(cellsTall),
		},
	})
	if err != nil {
		log.Printf("get_level_view failed: %v", err)
		return
	}
	g.lastLevelView = levelView
	if bounds, ok := levelView["bounds"].(map[string]any); ok {
		minX, _ := bounds["minX"].(float64)
		minY, _ := bounds["minY"].(float64)
		maxX, _ := bounds["maxX"].(float64)
		maxY, _ := bounds["maxY"].(float64)
		g.mapBounds = &MapBounds{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}
	}

	playerSummary, err := g.engine.Read("get_player_summary", nil)
	if err != nil {
		log.Printf("get_player_summary failed: %v", err)
		return
	}

	workers, err := g.engine.Read("get_workers", nil)
	if err != nil {
		log.Printf("get_workers failed: %v", err)
		return
	}

	resources, err := g.engine.Read("get_resources", nil)
	if err != nil {
		log.Printf("get_resources failed: %v", err)
		return
	}

	availableOrders, err := g.engine.Read("get_available_orders", nil)
	if err != nil {
		log.Printf("get_available_orders failed: %v", err)
		return
	}
	g.lastAvailableOrderIDs = availableOrderIDs(availableOrders)

	activeOrders, err := g.engine.Read("get_active_orders", nil)
	if err != nil {
		log.Printf("get_active_orders failed: %v", err)
		return
	}

	var mergeConfirm *render.MergeConfirm
	if g.pendingMerge != nil {
		mergeConfirm = &render.MergeConfirm{Level: g.pendingMerge.Level}
	}

	render.Draw(screen, render.ViewModel{
		Camera:           render.Camera{X: g.camera.X, Y: g.camera.Y, Zoom: g.camera.Zoom},
		LevelView:        levelView,
		PlayerSummary:    playerSummary,
		Workers:          workers,
		Resources:        resources,
		AvailableOrders:  availableOrders,
		ActiveOrders:     activeOrders,
		DraggingWorkerID: g.draggingWorkerID,
		SelectedWorkerID: g.selectedWorkerID,
		MergeConfirm:     mergeConfirm,
	})
}

// availableOrderIDs extracts order ids in the same (already Lua-sorted) order
// the panel will draw them, capped to the rows that actually get buttons.
func availableOrderIDs(availableOrders map[string]any) []string {
	list, _ := availableOrders["orders"].([]any)
	var ids []string
	for _, raw := range list {
		if len(ids) >= render.MaxAvailableOrderRows {
			break
		}
		order, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := order["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
