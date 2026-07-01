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
internal/persistence  SQLite adapter (see docs/persistence.md)
internal/application  Cross-cutting app-layer helpers (currently unused; reserved
                      for orchestration that doesn't belong in internal/app or
                      internal/persistence as the project grows)
lua/                  Authoritative Lua game engine (see engine.lua)
tests/run.lua         Lua mechanics test suite (run via `make test-lua`)
```
