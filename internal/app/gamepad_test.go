package app

import (
	"fmt"
	"image"
	"testing"

	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// TestGamepadOrderIndexActions exercises the gamepad orders-focus handlers
// (acceptOrderAtIndex/declineOrderAtIndex) without an Ebitengine input
// context: they only talk to the Lua engine via apply, same as the mouse
// click path. Verifies the indexed dispatch maps to the right order id.
func TestGamepadOrderIndexActions(t *testing.T) {
	engine, err := luaengine.New()
	if err != nil {
		t.Fatalf("luaengine.New() error: %v", err)
	}
	defer engine.Close()
	if err := engine.NewGame("gamepad-orders-seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}
	game := NewGame(engine, nil, "", "level_1")

	available, err := engine.Read("get_available_orders", nil)
	if err != nil {
		t.Fatalf("Read(get_available_orders) error: %v", err)
	}
	game.lastAvailableOrderIDs = availableOrderIDs(available)
	if len(game.lastAvailableOrderIDs) < 1 {
		t.Fatalf("expected at least 1 available order at game start, got %d", len(game.lastAvailableOrderIDs))
	}
	accepted := game.lastAvailableOrderIDs[0]

	game.orderSel = 0
	game.acceptOrderAtIndex(game.orderSel)

	active, err := engine.Read("get_active_orders", nil)
	if err != nil {
		t.Fatalf("Read(get_active_orders) error: %v", err)
	}
	list, _ := active["orders"].([]any)
	found := false
	for _, raw := range list {
		o, _ := raw.(map[string]any)
		if id, _ := o["id"].(string); id == accepted {
			if state, _ := o["state"].(string); state == "accepted" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("acceptOrderAtIndex(0) did not accept order %s", accepted)
	}
}

// TestGamepadHirePanel exercises the hire-select modal: openHire caches the
// purchasable levels from Lua (cost formula stays in Lua), buyWorkerLevel buys
// the selected level, and the cache refreshes afterwards.
func TestGamepadHirePanel(t *testing.T) {
	engine, err := luaengine.New()
	if err != nil {
		t.Fatalf("luaengine.New() error: %v", err)
	}
	defer engine.Close()
	if err := engine.NewGame("gamepad-hire-seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}
	game := NewGame(engine, nil, "", "level_1")

	game.openHire()
	if game.focus != focusHire {
		t.Fatalf("openHire() focus = %v, want focusHire", game.focus)
	}
	if len(game.hireLevels) == 0 {
		t.Fatalf("openHire() did not populate hireLevels")
	}
	wantLevel := game.hireLevels[game.hireSel].Level
	wantCost := game.hireLevels[game.hireSel].Cost

	summaryBefore, _ := engine.Read("get_player_summary", nil)
	moneyBefore, _ := summaryBefore["money"].(float64)

	game.buyWorkerLevel(wantLevel)

	summaryAfter, _ := engine.Read("get_player_summary", nil)
	moneyAfter, _ := summaryAfter["money"].(float64)
	if moneyAfter != moneyBefore-wantCost {
		t.Fatalf("money after buy = %v, want %v (before %v - cost %v)", moneyAfter, moneyBefore-wantCost, moneyBefore, wantCost)
	}

	workers, _ := engine.Read("get_workers", nil)
	list, _ := workers["workers"].([]any)
	var bought bool
	for _, raw := range list {
		w, _ := raw.(map[string]any)
		if lvl, _ := w["level"].(float64); lvl == wantLevel {
			bought = true
		}
	}
	if !bought {
		t.Fatalf("buyWorkerLevel(%v) did not add a level-%v worker", wantLevel, wantLevel)
	}

	game.closeHire()
	if game.focus != focusMap {
		t.Fatalf("closeHire() focus = %v, want focusMap", game.focus)
	}
	if game.hireLevels != nil {
		t.Fatalf("closeHire() should clear hireLevels, got %v", game.hireLevels)
	}
}

// TestGamepadMapCursorSelectsWorker exercises the unified cursor's A action on
// the map: with the cursor on a worker's cell, an A press/release (fed through
// g.pointer exactly like a mouse click) selects the worker via the same
// handleWorkerDrag/handleWorkerClick flow the mouse uses.
func TestGamepadMapCursorSelectsWorker(t *testing.T) {
	engine, err := luaengine.New()
	if err != nil {
		t.Fatalf("luaengine.New() error: %v", err)
	}
	defer engine.Close()
	if err := engine.NewGame("gamepad-cursor-seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}
	game := NewGame(engine, nil, "", "level_1")

	// Hire a worker so the roster is non-empty, then read the level view to
	// find a cell the worker actually occupies.
	if err := game.hireWorker(); err != nil {
		t.Fatalf("hireWorker() error: %v", err)
	}
	view, err := engine.Read("get_level_view", map[string]any{
		"levelId":  "level_1",
		"viewport": map[string]any{"x": -50.0, "y": -50.0, "width": 100.0, "height": 100.0},
	})
	if err != nil {
		t.Fatalf("Read(get_level_view) error: %v", err)
	}
	game.lastLevelView = view

	workers, _ := view["workers"].([]any)
	if len(workers) == 0 {
		t.Skip("no placed workers in viewport; cursor test needs one")
	}
	w, _ := workers[0].(map[string]any)
	posCellID, _ := w["positionCellId"].(string)
	if posCellID == "" {
		t.Skip("worker has no positionCellId; cursor test needs a placed worker")
	}
	// positionCellId is "x,y"; the camera starts at (0,0) zoom=1, so the
	// worker's screen position is its cell * TileSize. Point the unified
	// cursor there.
	var cx, cy float64
	if n, _ := fmt.Sscanf(posCellID, "%f,%f", &cx, &cy); n != 2 {
		t.Fatalf("could not parse positionCellId %q", posCellID)
	}
	game.focus = focusMap
	game.gamepadPresent = true
	game.cursorInit = true
	game.cursor = image.Pt(int(cx*render.TileSize), int(cy*render.TileSize))

	// A press, then A release — the same edges syncPointer would build from
	// gp.a / gp.aReleased in focusMap.
	game.pointer = pointerState{pos: game.cursor, justPressed: true}
	if err := game.handleWorkerDrag(); err != nil {
		t.Fatalf("handleWorkerDrag (press) error: %v", err)
	}
	game.pointer = pointerState{pos: game.cursor, justReleased: true}
	if err := game.handleWorkerDrag(); err != nil {
		t.Fatalf("handleWorkerDrag (release) error: %v", err)
	}

	if game.selectedWorkerID != w["id"] {
		t.Fatalf("A on worker cell selected %q, want %q", game.selectedWorkerID, w["id"])
	}
}
