package app

import (
	"path/filepath"
	"testing"

	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/persistence"
)

// TestAutosaveEventTriggersPersistenceAdapter is the Go-side counterpart of
// REQUIREMENTS.md §40 ("autosave event triggers persistence adapter"): Lua
// never writes to SQLite itself, so the app layer must react to
// autosave_requested by calling the store.
func TestAutosaveEventTriggersPersistenceAdapter(t *testing.T) {
	store, err := persistence.Open(filepath.Join(t.TempDir(), "autosave.db"))
	if err != nil {
		t.Fatalf("persistence.Open() error: %v", err)
	}
	defer func() { _ = store.Close() }()

	engine, err := store.CreateNewEngine("save-1", "autosave-seed")
	if err != nil {
		t.Fatalf("CreateNewEngine() error: %v", err)
	}
	defer engine.Close()

	if _, err := engine.Apply("buy_worker", map[string]any{"workerLevel": float64(1)}); err != nil {
		t.Fatalf("Apply(buy_worker) error: %v", err)
	}

	game := NewGame(engine, store, "save-1", "level_1")
	game.handleLuaEvents([]any{
		map[string]any{"type": "autosave_requested", "reason": "periodic"},
	})

	loaded, err := store.LoadEngine("save-1")
	if err != nil {
		t.Fatalf("LoadEngine() error after autosave: %v", err)
	}
	defer loaded.Close()

	workers, err := loaded.Read("get_workers", nil)
	if err != nil {
		t.Fatalf("Read(get_workers) error: %v", err)
	}
	list, _ := workers["workers"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected the autosave to have persisted the purchased worker, got %d workers", len(list))
	}
}

func TestSaveNowIsNoOpWithoutStore(t *testing.T) {
	engine, err := luaengine.New()
	if err != nil {
		t.Fatalf("luaengine.New() error: %v", err)
	}
	defer engine.Close()
	if err := engine.NewGame("seed"); err != nil {
		t.Fatalf("NewGame() error: %v", err)
	}

	game := NewGame(engine, nil, "", "level_1")
	if err := game.SaveNow(); err != nil {
		t.Fatalf("SaveNow() with no store configured should be a no-op, got error: %v", err)
	}
}
