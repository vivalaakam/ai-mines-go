// Package luaengine is the only bridge between Go and the authoritative Lua game
// engine. It knows nothing about gameplay rules - it just marshals commands/queries
// in and results out (AGENTS.md: "the only allowed engine API is apply and read").
package luaengine

import (
	"bytes"
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"

	assets "github.com/vivalaakam/ai-mines-go"
)

// Engine wraps a single Lua VM running the embedded lua/engine.lua module.
type Engine struct {
	state       *lua.LState
	engineTable *lua.LTable
}

// New starts a fresh Lua VM, wires up the embedded-filesystem module loader and
// host bindings, and requires the top-level lua/engine.lua module.
func New() (*Engine, error) {
	L := lua.NewState()
	registerRequireLoader(L)
	registerHostFunctions(L)

	e := &Engine{state: L}
	tbl, err := e.requireModule("engine")
	if err != nil {
		L.Close()
		return nil, fmt.Errorf("loading lua engine: %w", err)
	}
	e.engineTable = tbl
	return e, nil
}

// Close releases the underlying Lua VM.
func (e *Engine) Close() {
	e.state.Close()
}

// NewGame bootstraps a brand-new game state seeded by seedPhrase.
func (e *Engine) NewGame(seedPhrase string) error {
	fn := e.engineTable.RawGetString("new_game")
	return e.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, lua.LString(seedPhrase))
}

func (e *Engine) requireModule(name string) (*lua.LTable, error) {
	requireFn := e.state.GetGlobal("require")
	if err := e.state.CallByParam(lua.P{Fn: requireFn, NRet: 1, Protect: true}, lua.LString(name)); err != nil {
		return nil, err
	}
	ret := e.state.Get(-1)
	e.state.Pop(1)
	tbl, ok := ret.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("module %q did not return a table", name)
	}
	return tbl, nil
}

// registerRequireLoader replaces the global `require` with one that resolves
// dotted module names (e.g. "simulation.tick") against files embedded from the
// repository's lua/ directory, so the shipped binary has no external file
// dependency and Go/Lua always run the exact same source.
func registerRequireLoader(L *lua.LState) {
	cache := map[string]lua.LValue{}
	L.SetGlobal("require", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		if cached, ok := cache[name]; ok {
			L.Push(cached)
			return 1
		}

		path := "lua/" + strings.ReplaceAll(name, ".", "/") + ".lua"
		data, err := assets.LuaFS.ReadFile(path)
		if err != nil {
			L.RaiseError("module not found: %s (%s)", name, err)
			return 0
		}

		fn, err := L.Load(bytes.NewReader(data), name)
		if err != nil {
			L.RaiseError("error loading module %s: %s", name, err)
			return 0
		}

		L.Push(fn)
		if err := L.PCall(0, 1, nil); err != nil {
			L.RaiseError("error running module %s: %s", name, err)
			return 0
		}

		result := L.Get(-1)
		L.Pop(1)
		if result == lua.LNil {
			result = lua.LTrue
		}
		cache[name] = result
		L.Push(result)
		return 1
	}))
}
