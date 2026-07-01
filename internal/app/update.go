package app

import (
	"log"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// Update runs once per Ebitengine frame. It never computes gameplay outcomes
// itself: input only adjusts local camera state, and the only game-affecting
// action is periodically calling engine.Apply("tick", ...) once per accumulated
// real second (REQUIREMENTS.md §34).
func (g *Game) Update() error {
	input := PollInput()
	g.camera.Move(input.CameraDX, input.CameraDY)
	if input.ZoomDelta != 0 {
		g.camera.SetZoom(g.camera.Zoom + input.ZoomDelta)
	}
	if g.mapBounds != nil {
		g.camera.Clamp(
			g.mapBounds.MinX*render.TileSize,
			g.mapBounds.MinY*render.TileSize,
			(g.mapBounds.MaxX+1)*render.TileSize,
			(g.mapBounds.MaxY+1)*render.TileSize,
			render.ScreenWidth/g.camera.Zoom,
			render.ScreenHeight/g.camera.Zoom,
		)
	}
	if input.HireWorkerClicked {
		if err := g.hireWorker(); err != nil {
			return err
		}
	}

	if !g.accumulator.Advance() {
		return nil
	}

	result, err := g.engine.Apply("tick", map[string]any{"ticksPassed": float64(1)})
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

// hireWorker buys the cheapest currently-purchasable worker level (the same
// level/cost the HUD button shows - see render.HireWorkerButton), letting Lua's
// buy_worker validate funds/level rather than duplicating that logic here.
func (g *Game) hireWorker() error {
	workers, err := g.engine.Read("get_workers", nil)
	if err != nil {
		return err
	}
	level, _ := workers["nextPurchasableWorkerLevel"].(float64)

	result, err := g.engine.Apply("buy_worker", map[string]any{"workerLevel": level, "levelId": g.levelID})
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
			log.Printf("order completed: orderId=%v", event["orderId"])
		}
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
