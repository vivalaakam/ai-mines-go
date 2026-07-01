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

	phase, err := e.Read("get_game_phase", nil)
	if err != nil {
		t.Fatalf("Read(get_game_phase) error: %v", err)
	}
	if phase["phase"] != "shift_planning" {
		t.Fatalf("expected initial phase shift_planning, got %v", phase["phase"])
	}

	buyResult, err := e.Apply("buy_worker", map[string]any{"workerLevel": float64(1)})
	if err != nil {
		t.Fatalf("Apply(buy_worker) error: %v", err)
	}
	if !buyResult.OK {
		t.Fatalf("buy_worker failed: %+v", buyResult.Error)
	}

	startResult, err := e.Apply("start_next_shift", nil)
	if err != nil {
		t.Fatalf("Apply(start_next_shift) error: %v", err)
	}
	if !startResult.OK {
		t.Fatalf("start_next_shift failed: %+v", startResult.Error)
	}

	badFF, err := e.Apply("buy_worker", map[string]any{"workerLevel": float64(1)})
	if err != nil {
		t.Fatalf("Apply(buy_worker during shift) error: %v", err)
	}
	if badFF.OK || badFF.Error == nil || badFF.Error.Code != "not_in_planning" {
		t.Fatalf("expected not_in_planning error, got %+v", badFF)
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
	phaseAfterReload, err := e.Read("get_game_phase", nil)
	if err != nil {
		t.Fatalf("Read after reload error: %v", err)
	}
	if phaseAfterReload["phase"] != "shift_running" {
		t.Fatalf("expected shift_running phase preserved after reload, got %v", phaseAfterReload["phase"])
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
