# Engine API

The Lua engine (`lua/engine.lua`) exposes exactly four entry points to Go:

```lua
engine.new_game(seedPhrase)  -- bootstraps a fresh state (host convenience, not persisted)
engine.apply(command)        -- { ok, events, patch } | { ok=false, error={code,message,details} }
engine.read(query)           -- { ok, data } | { ok=false, error }
engine.export_state()        -- read-only snapshot for persistence
engine.load_state(state)     -- restores an engine from a snapshot
```

## Commands (`apply`)

`tick`, `buy_worker`, `merge_workers`,
`assign_worker_to_target_cell`, `stop_worker`, `buy_storage`, `upgrade_storage`,
`accept_order`, `decline_order`, `set_order_priority`, `complete_order_immediately`,
`create_next_level`.

See `lua/apply.lua` for the exact payload shape of each command and `REQUIREMENTS.md`
§36 for the canonical list.

## Queries (`read`)

`get_game_time`, `get_level_view`, `get_workers`, `get_storage_state`,
`get_available_orders`, `get_active_orders`, `get_resources`, `get_player_summary`.
See `lua/read.lua`.

## Error codes

Go must branch on `error.code`, never parse `error.message`. Known codes are defined
next to the Lua module that raises them (e.g. `worker_not_idle` in `lua/simulation/workers.lua`).
