// Package persistence is the SQLite adapter. It only converts between the Lua
// engine's exported state and structured tables - it must never contain
// gameplay logic (AGENTS.md: "Forbidden responsibilities: changing game rules,
// recalculating mechanics, ...").
package persistence

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
)

type Adapter struct {
	db *sql.DB
}

// Open opens (creating if needed) a SQLite database file and ensures the
// structured save schema exists.
func Open(path string) (*Adapter, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}
	return &Adapter{db: db}, nil
}

func (a *Adapter) Close() error {
	return a.db.Close()
}

// CreateNewEngine starts a fresh Lua engine and immediately persists it under
// saveID, so a freshly created save always exists on disk.
func (a *Adapter) CreateNewEngine(saveID, seedPhrase string) (*luaengine.Engine, error) {
	engine, err := luaengine.New()
	if err != nil {
		return nil, err
	}
	if err := engine.NewGame(seedPhrase); err != nil {
		engine.Close()
		return nil, err
	}
	if err := a.SaveEngine(engine, saveID); err != nil {
		engine.Close()
		return nil, err
	}
	return engine, nil
}

// LoadEngine reads a structured save back into a freshly started Lua engine.
func (a *Adapter) LoadEngine(saveID string) (*luaengine.Engine, error) {
	state, err := a.loadState(saveID)
	if err != nil {
		return nil, err
	}
	engine, err := luaengine.New()
	if err != nil {
		return nil, err
	}
	if err := engine.LoadState(state); err != nil {
		engine.Close()
		return nil, fmt.Errorf("restoring lua state: %w", err)
	}
	return engine, nil
}

// SaveEngine exports the engine's current state and writes it into saveID's
// structured rows, replacing whatever was there before, in one transaction.
func (a *Adapter) SaveEngine(engine *luaengine.Engine, saveID string) error {
	state, err := engine.ExportState()
	if err != nil {
		return fmt.Errorf("exporting lua state: %w", err)
	}
	return a.saveState(saveID, state)
}
