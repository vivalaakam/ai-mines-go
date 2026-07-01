package app

import (
	"testing"

	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
)

// TestHireWorkerBuysNextPurchasableLevel exercises the click handler behind
// the HUD's "Hire Worker" button (render.HireWorkerButton) without needing a
// real Ebitengine input/graphics context: hireWorker() only talks to the Lua
// engine via apply/read, same as Update() does after PollInput() detects a click.
func TestHireWorkerBuysNextPurchasableLevel(t *testing.T) {
	engine, err := luaengine.New()
	if err != nil {
		t.Fatalf("luaengine.New() error: %v", err)
	}
	defer engine.Close()
	if err := engine.NewGame("hire-worker-seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}

	game := NewGame(engine, nil, "", "level_1")
	if err := game.hireWorker(); err != nil {
		t.Fatalf("hireWorker() error: %v", err)
	}

	workers, err := engine.Read("get_workers", nil)
	if err != nil {
		t.Fatalf("Read(get_workers) error: %v", err)
	}
	list, _ := workers["workers"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected hireWorker() to have purchased exactly one worker, got %d", len(list))
	}
	worker, _ := list[0].(map[string]any)
	if worker["level"] != 1.0 {
		t.Fatalf("expected the first hired worker to be level 1, got %v", worker["level"])
	}

	summary, err := engine.Read("get_player_summary", nil)
	if err != nil {
		t.Fatalf("Read(get_player_summary) error: %v", err)
	}
	if summary["money"] != 50.0 {
		t.Fatalf("expected money=50 after hiring a level-1 worker (100 - 50), got %v", summary["money"])
	}
}
