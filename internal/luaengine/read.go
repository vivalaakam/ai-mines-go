package luaengine

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// Read sends a read-only query to the Lua engine (REQUIREMENTS.md §3: read must
// never mutate state). Returns the query's `data` payload on success.
func (e *Engine) Read(queryType string, payload map[string]any) (map[string]any, error) {
	L := e.state
	query := L.NewTable()
	query.RawSetString("type", lua.LString(queryType))
	for k, v := range payload {
		query.RawSetString(k, goToLua(L, v))
	}

	fn := e.engineTable.RawGetString("read")
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, query); err != nil {
		return nil, fmt.Errorf("calling engine.read(%s): %w", queryType, err)
	}
	ret := L.Get(-1)
	L.Pop(1)

	m := asMap(luaToGo(ret))
	if ok, _ := m["ok"].(bool); ok {
		return asMap(m["data"]), nil
	}
	return nil, parseEngineError(m["error"])
}

// ExportState returns a serializable snapshot of the current game state for the
// persistence adapter (REQUIREMENTS.md §33: export_state must not mutate state).
func (e *Engine) ExportState() (map[string]any, error) {
	fn := e.engineTable.RawGetString("export_state")
	if err := e.state.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}); err != nil {
		return nil, fmt.Errorf("calling engine.export_state(): %w", err)
	}
	ret := e.state.Get(-1)
	e.state.Pop(1)
	return asMap(luaToGo(ret)), nil
}

// LoadState restores the engine's state from a persistence-adapter-provided
// snapshot, as previously produced by ExportState.
func (e *Engine) LoadState(state map[string]any) error {
	L := e.state
	tbl := goToLua(L, state)
	fn := e.engineTable.RawGetString("load_state")
	return L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, tbl)
}
