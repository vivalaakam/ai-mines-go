package app

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

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
		HoverPos:         g.gamepadHoverPos(),
	})

	g.drawGamepadOverlays(screen)
}

// gamepadHoverPos returns the unified cursor position so render draws the cell
// highlight + tooltip over the cell the pad/mouse is pointing at, or nil to
// fall back to the raw OS mouse cursor (no gamepad connected).
func (g *Game) gamepadHoverPos() *image.Point {
	if !g.gamepadPresent || !g.cursorInit {
		return nil
	}
	return &g.cursor
}

// drawGamepadOverlays draws the gamepad-only UI: the unified cursor reticle
// (the one visible pointer entity while a pad is connected — the OS cursor is
// hidden, see update.go), the orders-panel selection highlight, and the
// hire-select modal. The cell highlight + tooltip under the cursor come from
// render via HoverPos, not a second rect here.
func (g *Game) drawGamepadOverlays(screen *ebiten.Image) {
	if g.gamepadPresent && g.cursorInit {
		// Small reticle marks the exact pixel of the single cursor entity;
		// the cell it falls in is highlighted separately by render.
		const s = 5
		x := float32(g.cursor.X) - s
		y := float32(g.cursor.Y) - s
		vector.StrokeRect(screen, x, y, s*2, s*2, 2, color.RGBA{255, 255, 255, 255}, false)
	}

	hl := color.RGBA{255, 230, 0, 255}
	if g.focus == focusOrders && g.orderSel >= 0 && g.orderSel < len(g.lastAvailableOrderIDs) {
		r := render.AvailableOrderRow(g.orderSel)
		vector.StrokeRect(screen, float32(r.Min.X)-2, float32(r.Min.Y)-2, float32(r.Dx())+4, float32(r.Dy())+4, 2, hl, false)
	}

	if g.focus == focusHire {
		g.drawHirePanel(screen)
	}
}

// drawHirePanel renders the hire-worker selection modal: a list of purchasable
// levels with costs and a highlight on the selected row.
func (g *Game) drawHirePanel(screen *ebiten.Image) {
	const w = 280
	rowH := 18
	h := 36 + rowH*max(1, len(g.hireLevels)) + 16
	x := (render.ScreenWidth - w) / 2
	y := (render.ScreenHeight - h) / 2

	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), color.RGBA{30, 30, 30, 230}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, color.RGBA{255, 255, 255, 255}, false)
	ebitenutil.DebugPrintAt(screen, "Hire worker  (A=buy, B=close)", x+8, y+8)

	if len(g.hireLevels) == 0 {
		ebitenutil.DebugPrintAt(screen, "(none purchasable)", x+10, y+30)
		return
	}
	for i, lv := range g.hireLevels {
		ry := y + 30 + i*rowH
		if i == g.hireSel {
			vector.FillRect(screen, float32(x+4), float32(ry-1), float32(w-8), float32(rowH), color.RGBA{80, 80, 40, 255}, false)
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Lv%.0f   $%.0f", lv.Level, lv.Cost), x+12, ry)
	}
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
