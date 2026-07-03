package app

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// orderEventLogCap bounds how many recent order events Game retains; the
// sidebar only displays the first few (render.orderEventLogMaxLines), this
// just keeps the backing slice from growing unbounded over a long session.
const orderEventLogCap = 20

// Update runs once per Ebitengine frame. It never computes gameplay outcomes
// itself: input only adjusts local camera state, and the only game-affecting
// action is periodically calling engine.Apply("tick", ...) once per accumulated
// real second (REQUIREMENTS.md §34).
func (g *Game) Update() error {
	g.syncGamepads()
	input := g.pollInput()
	// Hide the OS cursor while a gamepad is connected so only the single
	// unified cursor entity (g.cursor, drawn in draw.go) is visible — the
	// two-overlapping-cursors bug on mouse+pad (e.g. Steam Deck). Tracked so
	// we only touch the mode on a change, not every frame.
	wantHidden := g.gamepadPresent
	if wantHidden != g.cursorHidden {
		if wantHidden {
			ebiten.SetCursorMode(ebiten.CursorModeHidden)
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		}
		g.cursorHidden = wantHidden
	}
	g.camera.Move(input.CameraDX, input.CameraDY)
	zoomDelta := input.ZoomDelta
	if input.Gamepad.present && g.focus == focusMap {
		if input.Gamepad.dpadUp {
			zoomDelta += dpadZoomSpeed
		}
		if input.Gamepad.dpadDown {
			zoomDelta -= dpadZoomSpeed
		}
	}
	if zoomDelta != 0 {
		g.camera.SetZoom(g.camera.Zoom + zoomDelta)
	}
	if g.mapBounds != nil {
		g.camera.Clamp(
			g.mapBounds.MinX*render.TileSize,
			g.mapBounds.MinY*render.TileSize,
			(g.mapBounds.MaxX+1)*render.TileSize,
			(g.mapBounds.MaxY+1)*render.TileSize,
			render.MapWidth/g.camera.Zoom,
			render.ScreenHeight/g.camera.Zoom,
		)
	}
	if g.pointer.justPressed && g.pointer.pos.In(render.HireWorkerButton) {
		if err := g.hireWorker(); err != nil {
			return err
		}
	}

	if err := g.handleWorkerDrag(); err != nil {
		return err
	}

	g.handleGamepad(input.Gamepad)

	if !g.accumulator.Advance() {
		return nil
	}

	result, err := g.apply("tick", map[string]any{"ticksPassed": float64(1)})
	if err != nil {
		return err
	}
	if !result.OK {
		log.Printf("tick command rejected: %+v", result.Error)
		return nil
	}
	g.handleLuaEvents(result.Events)

	g.ticksSinceSave++
	if g.ticksSinceSave >= AutosaveIntervalTicks {
		g.ticksSinceSave = 0
		g.autosave("periodic")
	}
	return nil
}

// apply sends a mutating command to the engine and, on success, marks the
// cached view-models stale so the next Draw re-fetches them. Centralizing this
// keeps Draw's read-cache correct without every call site remembering to flag
// it. A rejected command (ok=false) did not mutate state, so the cache stays.
func (g *Game) apply(cmd string, payload map[string]any) (luaengine.ApplyResult, error) {
	result, err := g.engine.Apply(cmd, payload)
	if err == nil && result.OK {
		g.viewsDirty = true
	}
	return result, err
}

// hireWorker buys the cheapest currently-purchasable worker level (the same
// level/cost the HUD button shows - see render.HireWorkerButton), letting Lua's
// buy_worker validate funds/level rather than duplicating that logic here.
func (g *Game) hireWorker() error {
	workers, err := g.engine.Read("get_workers", nil)
	if err != nil {
		return err
	}
	level, _ := workers["nextPurchasableWorkerLevel"].(float64)

	result, err := g.apply("buy_worker", map[string]any{"workerLevel": level, "levelId": g.levelID})
	if err != nil {
		return err
	}
	if !result.OK {
		log.Printf("buy_worker rejected: %+v", result.Error)
	}
	return nil
}

// handleLuaEvents reacts to events returned by apply. Lua never writes to
// SQLite itself (REQUIREMENTS.md §30) - the app layer is responsible for
// calling the persistence adapter when an autosave_requested event arrives.
func (g *Game) handleLuaEvents(events []any) {
	for _, raw := range events {
		event, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		switch event["type"] {
		case "autosave_requested":
			g.autosave(event["reason"])
		case "order_completed":
			g.logOrderEvent(fmt.Sprintf("order %v completed", event["orderId"]))
		case "order_shipped":
			g.logOrderEvent(fmt.Sprintf("order %v: shipped %v %v (+$%v)",
				event["orderId"], event["amount"], event["resourceId"], event["payment"]))
		case "order_arrived":
			g.logOrderEvent(fmt.Sprintf("order %v arrived", event["orderId"]))
		case "order_expired":
			g.logOrderEvent(fmt.Sprintf("order %v expired", event["orderId"]))
		}
	}
}

// logOrderEvent records a human-readable order event, newest first, both in
// the application log and in Game.orderEventLog (rendered in the sidebar so
// order activity is visible in the UI itself, not just the console).
func (g *Game) logOrderEvent(line string) {
	log.Print(line)
	g.orderEventLog = append([]string{line}, g.orderEventLog...)
	if len(g.orderEventLog) > orderEventLogCap {
		g.orderEventLog = g.orderEventLog[:orderEventLogCap]
	}
}

func (g *Game) autosave(reason any) {
	if g.store == nil || g.saveID == "" {
		log.Printf("autosave requested (no store configured): reason=%v", reason)
		return
	}
	if err := g.store.SaveEngine(g.engine, g.saveID); err != nil {
		log.Printf("autosave failed: reason=%v err=%v", reason, err)
		return
	}
	log.Printf("autosave completed: reason=%v", reason)
}

// SaveNow performs a manual save, e.g. bound to a UI command or hotkey.
func (g *Game) SaveNow() error {
	if g.store == nil || g.saveID == "" {
		return nil
	}
	return g.store.SaveEngine(g.engine, g.saveID)
}
