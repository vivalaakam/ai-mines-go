# Persistence

`internal/persistence` is the SQLite adapter (`modernc.org/sqlite`, pure Go, no cgo).
It contains no gameplay logic - it only converts between `luaengine.Engine`'s
exported/loaded state and structured rows.

## Schema

Tables (`internal/persistence/schema.go`, applied idempotently on `Open()`):

```
saves               one row per save: seed, generator/schema version, game time,
                     phase, money, next-id counters, rulesConfig
levels              per-level entrance/stairs coordinates and reachability flag
chunks              (save_id, level_id, cx, cy) - which chunks were generated
cells               per-cell kind/visibility/accessibility/occupant
cell_components     rock/resource components for deposit cells
workers             the global worker pool, including assigned_side (which
                     cardinal side of its target cell a working worker occupies -
                     the one piece of state that lives on the cell, not the worker,
                     in the Lua model)
storages            one row per storage
orders / order_requirements
```

No table stores the whole state as a JSON blob (REQUIREMENTS.md §28). A dedicated
migration framework is not used yet (open decision, REQUIREMENTS.md §43.5) - the
schema is applied via idempotent `CREATE TABLE IF NOT EXISTS` statements, which is
sufficient while the schema only grows additively.

## Adapter contract

```go
func Open(path string) (*Adapter, error)
func (a *Adapter) CreateNewEngine(saveID, seedPhrase string) (*luaengine.Engine, error)
func (a *Adapter) LoadEngine(saveID string) (*luaengine.Engine, error)
func (a *Adapter) SaveEngine(engine *luaengine.Engine, saveID string) error
```

`SaveEngine` replaces the save's previous rows in one transaction (`DELETE FROM
saves WHERE id = ?` cascades via `ON DELETE CASCADE` to every child table, then
re-inserts). `internal/app` calls `SaveEngine` from `handleLuaEvents` whenever Lua
returns an `autosave_requested` event (Lua itself never touches SQLite, per
REQUIREMENTS.md §30) and exposes `Game.SaveNow()` for manual saves.
