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
//
// ponytail: the 6 engine reads are cached and only re-fetched when the game
// state changed (viewsDirty, set after every Apply) or the camera viewport
// cell range changed. The engine state is otherwise immutable between frames,
// so re-issuing 6 Lua round-trips + recursive luaToGo conversions every frame
// (360/sec) was pure allocation churn and the main driver of baseline memory.
func (g *Game) Draw(screen *ebiten.Image) {
	viewportX := math.Floor(g.camera.X / render.TileSize)
	viewportY := math.Floor(g.camera.Y / render.TileSize)
	cellsWide, cellsTall := render.ViewportCellCounts(g.camera.Zoom)
	viewportW := float64(cellsWide)
	viewportH := float64(cellsTall)

	viewportChanged := !g.hasCachedViewport ||
		viewportX != g.lastViewportX || viewportY != g.lastViewportY ||
		viewportW != g.lastViewportW || viewportH != g.lastViewportH

	if g.viewsDirty || g.cachedLevelView == nil || viewportChanged {
		levelView, err := g.engine.Read("get_level_view", map[string]any{
			"levelId": g.levelID,
			"viewport": map[string]any{
				"x":      viewportX,
				"y":      viewportY,
				"width":  viewportW,
				"height": viewportH,
			},
		})
		if err != nil {
			log.Printf("get_level_view failed: %v", err)
			return
		}
		g.cachedLevelView = levelView
		g.lastViewportX = viewportX
		g.lastViewportY = viewportY
		g.lastViewportW = viewportW
		g.lastViewportH = viewportH
		g.hasCachedViewport = true
		if bounds, ok := levelView["bounds"].(map[string]any); ok {
			minX, _ := bounds["minX"].(float64)
			minY, _ := bounds["minY"].(float64)
			maxX, _ := bounds["maxX"].(float64)
			maxY, _ := bounds["maxY"].(float64)
			g.mapBounds = &MapBounds{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}
		}
	}
	g.lastLevelView = g.cachedLevelView

	if g.viewsDirty || g.cachedPlayerSummary == nil {
		if err := g.refreshStateViews(); err != nil {
			log.Printf("refresh state views failed: %v", err)
			return
		}
		g.viewsDirty = false
	}
	g.lastAvailableOrderIDs = availableOrderIDs(g.cachedAvailableOrders)

	var mergeConfirm *render.MergeConfirm
	if g.pendingMerge != nil {
		mergeConfirm = &render.MergeConfirm{Level: g.pendingMerge.Level}
	}

	render.Draw(screen, render.ViewModel{
		Camera:           render.Camera{X: g.camera.X, Y: g.camera.Y, Zoom: g.camera.Zoom},
		LevelView:        g.cachedLevelView,
		PlayerSummary:    g.cachedPlayerSummary,
		Workers:          g.cachedWorkers,
		Resources:        g.cachedResources,
		AvailableOrders:  g.cachedAvailableOrders,
		ActiveOrders:     g.cachedActiveOrders,
		OrderEventLog:    g.orderEventLog,
		DraggingWorkerID: g.draggingWorkerID,
		SelectedWorkerID: g.selectedWorkerID,
		MergeConfirm:     mergeConfirm,
	})
}

// refreshStateViews re-fetches the 5 camera-independent view-models in one
// batch. Called only when the engine state changed (after Apply), not every
// frame, so the hot Draw path allocates nothing while the player is idle.
func (g *Game) refreshStateViews() error {
	playerSummary, err := g.engine.Read("get_player_summary", nil)
	if err != nil {
		return err
	}
	workers, err := g.engine.Read("get_workers", nil)
	if err != nil {
		return err
	}
	resources, err := g.engine.Read("get_resources", nil)
	if err != nil {
		return err
	}
	availableOrders, err := g.engine.Read("get_available_orders", nil)
	if err != nil {
		return err
	}
	activeOrders, err := g.engine.Read("get_active_orders", nil)
	if err != nil {
		return err
	}
	g.cachedPlayerSummary = playerSummary
	g.cachedWorkers = workers
	g.cachedResources = resources
	g.cachedAvailableOrders = availableOrders
	g.cachedActiveOrders = activeOrders
	return nil
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
