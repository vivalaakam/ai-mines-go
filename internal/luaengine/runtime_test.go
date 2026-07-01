package luaengine

import "testing"

func TestEngineLifecycle(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer e.Close()

	if err := e.NewGame("go-test-seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}

	buyResult, err := e.Apply("buy_worker", map[string]any{"workerLevel": float64(1)})
	if err != nil {
		t.Fatalf("Apply(buy_worker) error: %v", err)
	}
	if !buyResult.OK {
		t.Fatalf("buy_worker failed: %+v", buyResult.Error)
	}

	tickResult, err := e.Apply("tick", map[string]any{"ticksPassed": float64(1)})
	if err != nil {
		t.Fatalf("Apply(tick) error: %v", err)
	}
	if !tickResult.OK {
		t.Fatalf("tick failed: %+v", tickResult.Error)
	}

	// Purchases are no longer phase-gated; a second buy_worker must still succeed.
	buyResult2, err := e.Apply("buy_worker", map[string]any{"workerLevel": float64(1)})
	if err != nil {
		t.Fatalf("Apply(buy_worker) error: %v", err)
	}
	if !buyResult2.OK {
		t.Fatalf("second buy_worker failed: %+v", buyResult2.Error)
	}

	exported, err := e.ExportState()
	if err != nil {
		t.Fatalf("ExportState() error: %v", err)
	}
	if exported["seedPhrase"] != "go-test-seed" {
		t.Fatalf("exported state missing seedPhrase, got %v", exported["seedPhrase"])
	}

	if err := e.LoadState(exported); err != nil {
		t.Fatalf("LoadState() error: %v", err)
	}
	timeAfterReload, err := e.Read("get_game_time", nil)
	if err != nil {
		t.Fatalf("Read after reload error: %v", err)
	}
	if timeAfterReload["tick"] != float64(1) {
		t.Fatalf("expected tick=1 preserved after reload, got %v", timeAfterReload["tick"])
	}
}

func TestEngineUnknownCommand(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer e.Close()
	if err := e.NewGame("seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}

	result, err := e.Apply("does_not_exist", nil)
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if result.OK || result.Error.Code != "unknown_command" {
		t.Fatalf("expected unknown_command error, got %+v", result)
	}
}
