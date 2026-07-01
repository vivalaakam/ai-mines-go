# Architecture

See `AGENTS.md` and `REQUIREMENTS.md` at the repository root for the authoritative
architecture rules (layering, the `apply`/`read` boundary, ownership of gameplay
mechanics). This document tracks implementation-specific notes as the codebase grows.

## Current layout

```
cmd/mining-game       Go entry point
internal/app          Ebitengine window, Update/Draw loop, input, camera, accumulator
internal/render       Map/worker/UI rendering from Lua view-models
internal/luaengine    Go<->Lua bridge (runtime, apply, read, bindings)
internal/persistence  SQLite adapter (not yet implemented)
internal/application  Cross-cutting app-layer helpers (autosave orchestration, etc.)
lua/                  Authoritative Lua game engine (see engine.lua)
```

Persistence (SQLite adapter, migrations) is not implemented yet.
