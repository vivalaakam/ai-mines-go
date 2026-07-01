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

// handleLuaEvents reacts to events returned by apply. Persistence (Phase 15) is
// not implemented yet, so autosave_requested is only logged for now.
func (g *Game) handleLuaEvents(events []any) {
	for _, raw := range events {
		event, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		switch event["type"] {
		case "autosave_requested":
			log.Printf("autosave requested: reason=%v", event["reason"])
		case "shift_completed":
			log.Printf("shift completed: shiftIndex=%v", event["shiftIndex"])
		case "order_completed":
			log.Printf("order completed: orderId=%v", event["orderId"])
		}
	}
}
