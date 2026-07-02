# AGENTS.md

## Project Context

This repository contains an idle mining / exploration game built with:

- **Go**
- **Ebitengine**
- **Lua** for all game mechanics and simulation logic
- **SQLite** for structured save storage

The project is intentionally split into separate layers. Code agents must preserve this separation.

The most important rule:

> All gameplay mechanics, state mutations, simulation rules, validation, economy, generation, workers, orders, storage, visibility, reachability, and tick processing must live in Lua.

Go and Ebitengine are infrastructure and presentation layers. They must not become the source of gameplay rules.

---

## Architecture Rules

### 1. The only allowed engine API is `apply` and `read`

The game engine must expose only two primary interaction methods:

```lua
engine.apply(command)
engine.read(query)
```

Equivalent Go wrappers may exist, but they must call into Lua.

`apply(command)` is used for state-changing actions.

`read(query)` is used for read-only access.

No other system may directly mutate game state.

---

### 2. `apply` is the only mutation path

All state changes must happen through Lua `engine.apply`.

This includes:

- ticking game time;
- assigning workers;
- stopping workers;
- buying workers;
- merging workers;
- buying storage;
- upgrading storage;
- accepting orders;
- declining orders;
- setting order priority;
- completing orders;
- generating chunks;
- unlocking levels;
- updating visibility;
- updating reachability;
- opening empty cave areas;
- requesting autosave.

Go code must not mutate the authoritative game state directly.

---

### 3. `read` must be read-only

`engine.read(query)` must never mutate state.

It must not:

- advance time;
- generate chunks;
- update visibility;
- update reachability;
- move workers;
- process mining;
- expire orders;
- complete orders;
- request autosave;
- modify storage;
- modify money;
- modify cells.

If a query would require a state change, it must be redesigned as an `apply` command.

---

## Layer Responsibilities

### Go + Ebitengine App Layer

Allowed responsibilities:

- start the application;
- create the Ebitengine window;
- run the Ebitengine update/draw loop;
- collect keyboard/mouse input;
- control camera state;
- load assets;
- call Lua `apply`;
- call Lua `read`;
- pass view-model data to renderers;
- handle events returned by Lua;
- call persistence adapter when autosave/manual save is required.

Forbidden responsibilities:

- calculating mining output;
- calculating worker progress;
- calculating storage behavior;
- calculating order delivery;
- generating cells/chunks;
- deciding resource contents;
- deciding visibility/reachability;
- deciding if a worker assignment is valid;
- deciding if a cell is complete;
- deciding if a level is unlocked;
- applying economy rules;
- applying merge rules.

If the logic answers “what should happen according to the game rules?”, it belongs in Lua.

---

### Render / UI Layer

Allowed responsibilities:

- draw map tiles;
- draw workers;
- draw selected cells;
- draw reachable/scouted/unknown states;
- draw storage UI;
- draw order UI;
- draw buttons;
- draw animations;
- keep local hover/selection/camera state.

Forbidden responsibilities:

- changing game state directly;
- caching authoritative state;
- modifying cells;
- moving workers directly;
- changing storage directly;
- completing orders directly;
- creating levels directly.

The render/UI layer must receive data through `read` and submit player intent through `apply`.

---

### Lua Engine Layer

Lua is the authoritative game engine.

Lua must contain:

- domain state;
- command validation;
- query handling;
- tick processing;
- mining calculation;
- worker logic;
- merge logic;
- storage logic;
- order logic;
- resource generation;
- chunk generation;
- visibility calculation;
- reachability calculation;
- flood fill of empty areas;
- level unlocking;
- autosave event generation;
- state export/import.

The Lua layer must be testable without Ebitengine.

---

### SQLite Persistence Adapter

SQLite is used for structured saves.

The persistence adapter is implemented in Go, but it must not contain gameplay logic.

Allowed responsibilities:

- run migrations;
- read structured tables;
- construct Lua-compatible state;
- create a Lua engine from saved state;
- request exported state from Lua;
- write structured state back to SQLite.

Forbidden responsibilities:

- changing game rules;
- recalculating mechanics;
- completing simulation;
- generating gameplay outcomes;
- correcting state using gameplay assumptions.

Persistence must not be a single JSON blob. Tables must represent the domain structure.

---

## Required SQLite Structure

The save system must use structured tables.

Minimum required tables:

```sql
saves
levels
chunks
cells
cell_components
workers
storages
orders
order_requirements
```

JSON may be used only for secondary metadata, not as the primary save format.

The save must include:

- seed phrase;
- generator version;
- schema version;
- current tick;
- money;
- unlocked levels;
- unlocked resources;
- generated chunks;
- cells;
- cell components;
- visibility state;
- accessibility state;
- workers;
- worker assignments;
- storages;
- orders;
- order requirements;
- rule config.

---

## Game Mechanics Ownership

The following mechanics must be implemented in Lua only.

### Time

- 1 game tick = 1 second.
- Offline progress is not required for MVP.
- Game time advances only via:

```lua
engine.apply({
  type = "tick",
  ticksPassed = 1
})
```

Time is continuous: there is no shift boundary, no phase gating, and no
`remainingTicks` clamping. A single `tick` command simply advances
`state.gameTime.tick` by `ticksPassed` and runs mining/order-settlement logic
for each elapsed tick.

Purchases, merge, storage upgrades, order selection, order priority changes,
and worker (re)assignment are valid at any time — there is no planning phase
that gates them.

---

### Map Generation

Lua owns map generation.

Rules:

- levels are infinite;
- levels are generated lazily by chunks;
- chunk size for MVP is 32×32 cells;
- generation is deterministic by seed phrase;
- each level initially generates a 5×5 chunk area;
- the central chunk contains a 3×3 empty entrance area;
- a non-central chunk in the 5×5 area contains a 3×3 stairs area;
- the stairs area is hidden until scouted;
- the generator must guarantee a potential path from entrance to stairs;
- corridors between resource zones must not be wider than 3 cells;
- resource zones may be wider than 3 cells;
- large connected caves should be avoided.

Chunk generation must depend on:

```text
seedPhrase
levelDepth
chunkX
chunkY
generatorVersion
```

---

### Cells

Lua owns all cell state.

Required cell kinds:

```lua
"empty"
"deposit"
"obstacle"
"stairs_area"
```

A `deposit` cell contains components.

A fully depleted `deposit` cell becomes empty and passable.

Obstacles are indestructible.

There are no pits/chasms.

---

### Cell Components

Lua owns component processing.

A deposit cell can contain:

- rock;
- one or more resources.

Rock is processed like a component but does not go to storage.

Example:

```text
rock: 60%
stone: 25%
copper: 10%
gold: 5%
```

Resources go to their resource-specific storage if capacity is available.

If storage for one resource is full, that resource remains in the cell and can be mined later.

Other available components continue to be processed.

If only blocked resources remain, assigned workers become `blocked_by_storage`.

---

### Visibility and Reachability

Lua owns visibility and reachability.

Visibility radius:

```text
5 cells from every reachable/open cell
```

Lua must distinguish:

```lua
visibility = "unknown" | "scouted"
accessibility = "unreachable" | "reachable"
```

If a reachable empty area connects to a larger empty area, the full connected empty area becomes reachable.

If visibility or flood fill crosses chunk boundaries, Lua must request or trigger deterministic generation of the required neighboring chunks.

Go must not decide which cells are visible or reachable.

---

### Workers

Lua owns workers.

Workers are a global pool.

A worker can be assigned to any unlocked level if the target position is reachable.

A worker stands on an open neighboring cell, not on the target deposit cell.

Up to 4 workers may work on one target cell, one from each cardinal side.

One open position cell may contain only one worker.

Movement is instant if a path exists.

Merge rule:

```text
2 workers of the same level -> 1 worker of the next level
```

Merge is allowed only for idle workers.

Purchasable worker level:

```lua
maxPurchasableWorkerLevel = math.max(1, highestUnlockedWorkerLevel - 2)
```

Worker purchase and merge are allowed at any time.

---

### Storage

Lua owns storage rules.

Each storage stores exactly one resource type.

There may be multiple storages for the same resource.

Capacities for the same resource are summed.

Storage can be bought and upgraded at any time.

If a resource has no free capacity, that resource is not mined.

---

### Orders

Lua owns order logic.

Orders can require one or several resources. Each requirement carries its own
`pricePerUnit`, rolled deterministically from the resource's base-price range
when the order is generated; `rewardMoney` is the order's full value
(`sum(requiredAmount * pricePerUnit)`).

New orders arrive periodically: on every tick divisible by
`orderArrivalIntervalTicks` (default 100) there is an `orderArrivalChance`
(default 0.5) roll — deterministic from seedPhrase + tick — for one new order,
as long as fewer than `maxAvailableOrders` (default 3) are available. A new
game seeds the pool once; accepting/declining an order does not instantly
refill it.

Order states:

```lua
"available"
"accepted"
"completed"
"expired"
"declined"
```

If the player starts an order, it cannot be cancelled.

If all required resources are available, an order can complete immediately.

Otherwise the accepted order is shipped in parts: on every tick divisible by
`orderShipmentIntervalTicks` (default 50) the engine ships whatever stock is
available toward accepted orders (if there is anything to ship) and pays
`amount * pricePerUnit` for each shipped part immediately. An order completes
when all requirements are fully delivered — there is no extra lump payment on
completion.

When multiple active orders require the same resource, the engine must support allocation mode:

```lua
"priority_based"
"proportional"
```

The default is:

```lua
"proportional"
```

(proportional shares of remaining need; the rounding remainder is handed out
by priority).

Order selection and priority changes are allowed at any time.

---

## Required Lua API

The Lua engine must expose at least:

```lua
engine.apply(command)
engine.read(query)
engine.export_state()
engine.load_state(state)
```

`export_state` must not mutate state.

`load_state` should be used only by the Go persistence adapter when creating/restoring the engine.

---

## Required Commands

Minimum required command types:

```lua
-- time
{ type = "tick", ticksPassed = 1 }

-- workers
{ type = "buy_worker", workerLevel = 1 }
{ type = "merge_workers", workerIds = { "w1", "w2" } }
{ type = "assign_worker_to_target_cell", workerId = "...", levelId = "...", targetCellId = "...", positionCellId = "...", assignmentMode = "until_completed" }
{ type = "stop_worker", workerId = "..." }

-- storage
{ type = "buy_storage", resourceId = "stone" }
{ type = "upgrade_storage", storageId = "..." }

-- orders
{ type = "accept_order", orderId = "..." }
{ type = "decline_order", orderId = "..." }
{ type = "set_order_priority", orderId = "...", priority = 1 }
{ type = "complete_order_immediately", orderId = "..." }

-- levels
{ type = "create_next_level", fromLevelId = "..." }
```

All validation is performed in Lua.

---

## Required Queries

Minimum required query types:

```lua
{ type = "get_game_time" }
{ type = "get_level_view", levelId = "...", viewport = { x = 0, y = 0, width = 40, height = 30 } }
{ type = "get_workers" }
{ type = "get_storage_state" }
{ type = "get_available_orders" }
{ type = "get_active_orders" }
{ type = "get_resources" }
{ type = "get_player_summary" }
```

Queries must not mutate state.

---

## Apply Result Format

`apply` must return structured results.

Success:

```lua
{
  ok = true,
  events = {},
  patch = {}
}
```

Failure:

```lua
{
  ok = false,
  error = {
    code = "worker_not_idle",
    message = "Worker is not idle",
    details = {}
  }
}
```

Go must rely on `error.code`, not on parsing `message`.

---

## Ebitengine Loop Rules

Ebitengine may run at its normal update frequency, but game simulation uses 1-second ticks.

The Go app must use an accumulator.

Pseudo-flow:

```go
func (g *Game) Update() error {
    g.handleInput()

    g.updateAccumulator++

    if g.updateAccumulator >= g.updatesPerGameTick {
        g.updateAccumulator = 0

        result := g.luaEngine.Apply(Command{
            Type: "tick",
            Payload: map[string]any{
                "ticksPassed": 1,
            },
        })

        g.handleLuaEvents(result.Events)
    }

    return nil
}
```

Go must not process simulation directly in `Update`.

Go must not mutate authoritative state in `Draw`.

---

## Autosave Rules

Lua does not write to SQLite.

Lua no longer emits autosave-related events. Instead, the Go app layer
(`internal/app/update.go`) tracks a periodic tick counter and triggers
autosave itself every `AutosaveIntervalTicks` ticks (see
`internal/app/game.go`), independent of anything Lua returns.

Manual save is also allowed via UI command at any time.

---

## Required Project Documents

The repository must contain:

```text
AGENTS.md
docs/architecture.md
docs/game-design.md
docs/engine-api.md
docs/persistence.md
```

When changing architecture, update the relevant docs in the same task.

---

## Required Tests

### Lua Mechanics Tests

Required coverage:

- deterministic chunk generation by seed;
- same seed produces same generated area;
- entrance area is 3×3;
- stairs area is 3×3;
- stairs area is in a non-central chunk of the initial 5×5 area;
- potential path from entrance to stairs exists;
- visibility reveals 5 cells from reachable/open cells;
- empty connected areas flood-fill correctly;
- chunk generation continues across visibility/flood-fill boundaries;
- large caves are avoided or limited;
- mixed cell components are processed proportionally;
- rock disappears and does not go to storage;
- resources go to correct storage;
- full storage blocks only that resource;
- blocked resource is not lost;
- cell becomes passable only after all components are depleted;
- up to 4 workers can mine one target cell from different sides;
- two workers cannot occupy the same position cell;
- worker movement requires reachability;
- worker merge is 2-to-1;
- worker purchase level follows `highestUnlockedWorkerLevel - 2`;
- order completion works;
- order allocation works.

### Go Integration Tests

Required coverage:

- Lua runtime starts;
- Go can call `apply`;
- Go can call `read`;
- Lua errors map to structured Go errors;
- Go app layer's periodic autosave counter triggers the persistence adapter;
- SQLite adapter saves structured state;
- SQLite adapter loads structured state;
- loaded state creates an equivalent Lua engine;
- render layer consumes view-model only;
- Go app does not contain domain mechanics.

---

## Required Checks Before Completing Any Task

Before finishing a task, the agent must run all applicable checks.

Minimum Go checks:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

If `golangci-lint` is configured:

```bash
golangci-lint run
```

Minimum Lua checks:

```bash
lua tests/run.lua
```

or:

```bash
make test-lua
```

If StyLua is configured:

```bash
stylua --check lua/
```

Preferred project-level command:

```bash
make check
```

`make check` should include:

- Go formatting;
- Go lint/vet;
- Go tests;
- Go race tests if practical;
- Go build;
- Lua formatting;
- Lua tests.

---

## Commit Rule

After completing a task:

1. Run all required checks.
2. If checks pass, commit changes to the current branch.
3. If any check fails, do not commit.
4. Report which checks were run.
5. Report whether a commit was created.
6. If a check cannot be run, explain why.

A task is not complete until checks are run and the result is reported.

---

## Strict Prohibitions

Agents must not:

- move gameplay mechanics from Lua to Go;
- mutate game state outside `engine.apply`;
- make `read` mutate state;
- let UI/render write authoritative state;
- let SQLite adapter apply game rules;
- store the whole save as one JSON blob;
- bypass Lua validation from Go;
- parse human-readable error messages for logic;
- commit if checks fail;
- silently skip tests;
- introduce circular dependencies between app, render, Lua engine, and persistence.

---

## Open Design Decisions

The following decisions are intentionally unresolved:

1. Exact worker speed formula.
2. Exact resource amount formula.
3. Exact order reward formula.
4. Exact worker purchase cost formula.
5. Exact storage upgrade cost formula.
8. Exact cave-size limits.
9. Exact Lua VM library for Go.
10. Exact SQLite migration framework.

Do not hard-code unresolved decisions unless the task explicitly resolves them.

Prefer config-driven behavior where possible.
