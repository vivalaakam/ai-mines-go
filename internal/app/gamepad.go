package app

import (
	"log"
	"math"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// Cooldowns: how many frames a held direction must wait before repeating.
// ponytail: tuned by feel — lower = snappier lists, higher = less skip.
const (
	listMoveInterval = 8
)

// handleGamepad dispatches gamepad actions by the current focus. The mouse and
// the pad share one pointer entity (g.cursor, see input.go); A is already fed
// through g.pointer as a click, so this only handles focus-specific pad input
// (cursor movement on the map, list navigation in orders/hire).
func (g *Game) handleGamepad(gp gamepadInput) {
	if !gp.present {
		return
	}
	switch g.focus {
	case focusMap:
		g.handleGamepadMap(gp)
	case focusOrders:
		g.handleGamepadOrders(gp)
	case focusHire:
		g.handleGamepadHire(gp)
	}
}

// handleGamepadMap drives the unified cursor over the map. The left stick
// moves g.cursor in pixels (the cell under it is highlighted + tooltip'd via
// HoverPos in draw.go, so it reads as a cell cursor); A is a click through
// g.pointer (drag.go: select/place/merge, same as the mouse); B cancels a
// pending merge or selection; Select opens the hire panel; R2 switches focus
// to the orders panel.
func (g *Game) handleGamepadMap(gp gamepadInput) {
	if gp.selectBtn {
		g.openHire()
		return
	}
	if gp.r2 {
		g.focus = focusOrders
		g.clampOrderSelection()
		return
	}
	if gp.b {
		if g.pendingMerge != nil {
			g.pendingMerge = nil
		} else if g.selectedWorkerID != "" {
			g.selectedWorkerID = ""
		}
		return
	}
	// A is handled by g.pointer/drag.go — nothing to do here for A.

	if math.Abs(gp.leftX) > stickActThreshold {
		g.cursor.X += int(gp.leftX * cursorStickSpeed)
	}
	if math.Abs(gp.leftY) > stickActThreshold {
		g.cursor.Y += int(gp.leftY * cursorStickSpeed)
	}
	g.clampCursorPoint()
}

// clampCursorPoint keeps the unified cursor inside the logical screen so the
// stick can never shove it off the clickable area.
func (g *Game) clampCursorPoint() {
	if g.cursor.X < 0 {
		g.cursor.X = 0
	} else if g.cursor.X > render.ScreenWidth {
		g.cursor.X = render.ScreenWidth
	}
	if g.cursor.Y < 0 {
		g.cursor.Y = 0
	} else if g.cursor.Y > render.ScreenHeight {
		g.cursor.Y = render.ScreenHeight
	}
}

// handleGamepadOrders navigates the available-orders list. Up/down (D-pad or
// left stick) moves the selection; A accepts the highlighted order; B
// declines it; R2 returns focus to the map.
func (g *Game) handleGamepadOrders(gp gamepadInput) {
	if gp.r2 {
		g.focus = focusMap
		return
	}
	n := len(g.lastAvailableOrderIDs)
	if g.listCD > 0 {
		g.listCD--
	}
	if move := g.listMove(gp); move != 0 && g.listCD == 0 && n > 0 {
		g.orderSel += move
		g.clampOrderSelection()
		g.listCD = listMoveInterval
	}
	if gp.a {
		g.acceptOrderAtIndex(g.orderSel)
	}
	if gp.b {
		g.declineOrderAtIndex(g.orderSel)
	}
}

// handleGamepadHire navigates the purchasable-worker list. Up/down moves the
// selection; A buys the highlighted level; B or Select closes the panel.
func (g *Game) handleGamepadHire(gp gamepadInput) {
	if gp.b || gp.selectBtn {
		g.closeHire()
		return
	}
	n := len(g.hireLevels)
	if g.listCD > 0 {
		g.listCD--
	}
	if move := g.listMove(gp); move != 0 && g.listCD == 0 && n > 0 {
		g.hireSel += move
		if g.hireSel < 0 {
			g.hireSel = 0
		}
		if g.hireSel >= n {
			g.hireSel = n - 1
		}
		g.listCD = listMoveInterval
	}
	if gp.a && n > 0 {
		g.buyWorkerLevel(g.hireLevels[g.hireSel].Level)
		g.refreshHireLevels()
		if g.hireSel >= len(g.hireLevels) {
			g.hireSel = len(g.hireLevels) - 1
		}
	}
}

// listMove returns -1 (up), +1 (down), or 0 from D-pad or left stick Y.
func (g *Game) listMove(gp gamepadInput) int {
	if gp.dpadUp || gp.leftY < -stickActThreshold {
		return -1
	}
	if gp.dpadDown || gp.leftY > stickActThreshold {
		return 1
	}
	return 0
}

func (g *Game) clampOrderSelection() {
	if n := len(g.lastAvailableOrderIDs); n > 0 {
		if g.orderSel < 0 {
			g.orderSel = 0
		}
		if g.orderSel >= n {
			g.orderSel = n - 1
		}
	}
}

// openHire switches to the hire panel and caches the purchasable levels,
// defaulting the selection to the recommended (next purchasable) level.
func (g *Game) openHire() {
	g.focus = focusHire
	g.refreshHireLevels()
	if workers, err := g.engine.Read("get_workers", nil); err == nil {
		if next, _ := workers["nextPurchasableWorkerLevel"].(float64); next > 0 {
			g.hireSel = int(next) - 1 // levels are 1-indexed in the list
		}
	}
	if g.hireSel < 0 {
		g.hireSel = 0
	}
	if n := len(g.hireLevels); g.hireSel >= n && n > 0 {
		g.hireSel = n - 1
	}
}

func (g *Game) closeHire() {
	g.focus = focusMap
	g.hireLevels = nil
}

// refreshHireLevels re-reads the purchasable-worker list (costs/availability
// change after a purchase or when highestUnlockedWorkerLevel rises).
func (g *Game) refreshHireLevels() {
	res, err := g.engine.Read("get_purchasable_workers", nil)
	if err != nil {
		log.Printf("get_purchasable_workers failed: %v", err)
		g.hireLevels = nil
		return
	}
	levels, _ := res["levels"].([]any)
	g.hireLevels = g.hireLevels[:0]
	for _, raw := range levels {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		lvl, _ := m["level"].(float64)
		cost, _ := m["cost"].(float64)
		g.hireLevels = append(g.hireLevels, hireLevel{Level: lvl, Cost: cost})
	}
}

// buyWorkerLevel buys a specific worker level via the engine (Lua validates
// funds/level), used by the hire panel. hireWorker() (mouse) buys the
// cheapest by contrast.
func (g *Game) buyWorkerLevel(level float64) {
	result, err := g.apply("buy_worker", map[string]any{"workerLevel": level, "levelId": g.levelID})
	if err != nil {
		log.Printf("buy_worker error: %v", err)
		return
	}
	if !result.OK {
		log.Printf("buy_worker rejected: %+v", result.Error)
	}
}

// acceptOrderAtIndex / declineOrderAtIndex send the indexed order's command
// to Lua. Shared by the gamepad orders focus and mouse click hit-testing.
func (g *Game) acceptOrderAtIndex(i int) {
	if i < 0 || i >= len(g.lastAvailableOrderIDs) {
		return
	}
	result, err := g.apply("accept_order", map[string]any{"orderId": g.lastAvailableOrderIDs[i]})
	if err != nil {
		log.Printf("accept_order error: %v", err)
		return
	}
	if !result.OK {
		log.Printf("accept_order rejected: %+v", result.Error)
	}
}

func (g *Game) declineOrderAtIndex(i int) {
	if i < 0 || i >= len(g.lastAvailableOrderIDs) {
		return
	}
	result, err := g.apply("decline_order", map[string]any{"orderId": g.lastAvailableOrderIDs[i]})
	if err != nil {
		log.Printf("decline_order error: %v", err)
		return
	}
	if !result.OK {
		log.Printf("decline_order rejected: %+v", result.Error)
	}
}
