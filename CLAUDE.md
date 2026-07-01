# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Status

This repository currently contains only specification documents (`AGENTS.md`, `REQUIREMENTS.md`) — no Go source, `go.mod`, `Makefile`, or Lua files exist yet. Treat `AGENTS.md` and `REQUIREMENTS.md` as the authoritative architectural contract for whatever code gets written here.

## Project Context

Idle mining / exploration game (inspired by Deep Corp and Gold & Goblins) to be built with:

- **Go** + **Ebitengine** for windowing, game loop, input, camera, and rendering
- **Lua** for all gameplay mechanics and simulation
- **SQLite** for structured save data

## The Core Architectural Rule

**All gameplay mechanics, state mutations, simulation rules, validation, economy, generation, workers, orders, storage, visibility, reachability, and tick processing must live in Lua.** Go/Ebitengine are infrastructure and presentation only — they must never become a source of gameplay rules. If a piece of logic answers "what should happen according to the game rules?", it belongs in Lua, not Go.

## Layered Architecture

1. **Go Ebitengine App** (`/internal/app`) — window, main loop, input, camera, calls into Lua, invokes persistence on autosave/manual save.
2. **Go Render/UI Layer** (`/internal/render`) — draws map/workers/storage/orders UI from `read`-derived view-models only; holds local hover/selection/camera state; never mutates authoritative state.
3. **Lua Game Engine** (`/lua`) — the single source of truth: domain state, command validation, chunk/level generation, tick processing, mining, workers, merge, storage, orders, visibility, reachability, flood fill. Must be testable without Ebitengine.
4. **Go SQLite Persistence Adapter** (`/internal/persistence`) — runs migrations, converts between SQLite rows and Lua-compatible state, calls `export_state`/`load_state`. Contains **no** gameplay logic.

### The only allowed Go↔Lua interface

```lua
engine.apply(command)   -- the only mutation path; validates, mutates, returns { ok, events, patch } or { ok=false, error={code,message,details} }
engine.read(query)      -- strictly read-only; must never advance time, generate chunks, move workers, etc.
engine.export_state()   -- read-only snapshot for persistence
engine.load_state(state) -- used only by the persistence adapter to restore an engine
```

Go must branch on `error.code`, never parse `error.message`. No other system may directly mutate game state — not the render layer, not the SQLite adapter.

### Game time model

- 1 tick = 1 second. There are no shifts or phases: time advances continuously via `engine.apply({type="tick", ...})`, and purchases, merges, storage upgrades, worker (re)assignment, and order actions are valid at any time.
- The Ebitengine `Update()` loop must accumulate real frames into whole game ticks (see accumulator pattern in `AGENTS.md`/`REQUIREMENTS.md` §34) rather than simulating anything itself; `Draw()` must never mutate authoritative state.

### Persistence

Saves must use structured SQLite tables (`saves, levels, chunks, cells, cell_components, workers, storages, orders, order_requirements`), never a single JSON blob. Autosave is triggered periodically by the Go app layer (not Lua) on a fixed tick interval; Lua may still emit an `autosave_requested` event for other triggers, and the Go app layer is responsible for invoking the persistence adapter.

Full command/query lists, cell/worker/storage/order data models, map-generation rules (chunked, seeded, deterministic, 5×5 starting chunk area, 3×3 entrance/stairs zones), and worker merge/purchase formulas are specified in detail in `AGENTS.md` and `REQUIREMENTS.md` — consult them before implementing any of these systems.

## Required Checks Before Completing Any Task

Once Go/Lua code exists, run all applicable checks before considering a task done:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./...
golangci-lint run   # if configured
lua tests/run.lua   # or: make test-lua
stylua --check lua/ # if configured
```

Prefer a single `make check` once a Makefile exists. **Only commit if all checks pass**; report which checks were run and whether a commit was created. Do not skip tests silently, and do not move gameplay logic from Lua into Go as a workaround.

## Open Design Decisions

Several mechanics are intentionally unresolved in the spec (exact worker/order/storage cost formulas, cave-size limits, choice of Lua VM binding, SQLite migration framework). Do not hard-code a resolution to these unless the task explicitly asks you to decide one — prefer config-driven behavior.
