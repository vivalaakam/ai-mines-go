package app

import (
	"testing"

	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// TestOrderButtonsAcceptAndDecline exercises the orders-panel click handler
// without a real Ebitengine input/graphics context: it hit-tests the same
// exported button rects render draws and sends accept_order/decline_order to
// Lua, same as handleWorkerDrag does on a mouse press.
func TestOrderButtonsAcceptAndDecline(t *testing.T) {
	engine, err := luaengine.New()
	if err != nil {
		t.Fatalf("luaengine.New() error: %v", err)
	}
	defer engine.Close()
	if err := engine.NewGame("order-buttons-seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}

	game := NewGame(engine, nil, "", "level_1")

	available, err := engine.Read("get_available_orders", nil)
	if err != nil {
		t.Fatalf("Read(get_available_orders) error: %v", err)
	}
	game.lastAvailableOrderIDs = availableOrderIDs(available)
	if len(game.lastAvailableOrderIDs) < 2 {
		t.Fatalf("expected at least 2 available orders at game start, got %d", len(game.lastAvailableOrderIDs))
	}
	declined := game.lastAvailableOrderIDs[0]
	accepted := game.lastAvailableOrderIDs[1]

	pt := render.AvailableOrderDeclineButton(0).Min.Add(render.AvailableOrderDeclineButton(0).Size().Div(2))
	consumed, err := game.handleOrderButtonClick(pt.X, pt.Y)
	if err != nil || !consumed {
		t.Fatalf("decline click not consumed: consumed=%v err=%v", consumed, err)
	}

	// The pool does not refill instantly, so index 1 still points at the same
	// order (the cached ids are only refreshed by Draw).
	pt = render.AvailableOrderAcceptButton(1).Min.Add(render.AvailableOrderAcceptButton(1).Size().Div(2))
	consumed, err = game.handleOrderButtonClick(pt.X, pt.Y)
	if err != nil || !consumed {
		t.Fatalf("accept click not consumed: consumed=%v err=%v", consumed, err)
	}

	// A click far outside the panel must not be consumed.
	consumed, err = game.handleOrderButtonClick(1, 1)
	if err != nil || consumed {
		t.Fatalf("unrelated click must not be consumed: consumed=%v err=%v", consumed, err)
	}

	states := map[string]string{}
	for _, query := range []string{"get_available_orders", "get_active_orders"} {
		result, err := engine.Read(query, nil)
		if err != nil {
			t.Fatalf("Read(%s) error: %v", query, err)
		}
		list, _ := result["orders"].([]any)
		for _, raw := range list {
			order, _ := raw.(map[string]any)
			id, _ := order["id"].(string)
			state, _ := order["state"].(string)
			states[id] = state
		}
	}
	if states[declined] != "" {
		t.Fatalf("declined order %s should no longer be listed, got state %q", declined, states[declined])
	}
	// With empty storages the accepted order can't complete instantly, so it
	// must show up as accepted (an active order).
	if states[accepted] != "accepted" {
		t.Fatalf("expected order %s to be accepted, got state %q (all states: %v)", accepted, states[accepted], states)
	}
}
