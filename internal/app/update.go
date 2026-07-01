package app

import "log"

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

	phaseData, err := g.engine.Read("get_game_phase", nil)
	if err != nil {
		return err
	}

	if phaseData["phase"] != "shift_running" {
		g.accumulator.Reset()
		return nil
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
		case "shift_completed":
			log.Printf("shift completed: shiftIndex=%v", event["shiftIndex"])
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

// SaveNow performs a manual save, e.g. bound to a UI command or hotkey
// (REQUIREMENTS.md §30: "Manual save is also allowed from planning phase").
func (g *Game) SaveNow() error {
	if g.store == nil || g.saveID == "" {
		return nil
	}
	return g.store.SaveEngine(g.engine, g.saveID)
}
