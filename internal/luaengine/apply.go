package luaengine

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// Apply sends a mutating command to the Lua engine (REQUIREMENTS.md §3: apply is
// the only mutation path). payload becomes the command table's fields alongside
// "type" = commandType.
func (e *Engine) Apply(commandType string, payload map[string]any) (ApplyResult, error) {
	L := e.state
	cmd := L.NewTable()
	cmd.RawSetString("type", lua.LString(commandType))
	for k, v := range payload {
		cmd.RawSetString(k, goToLua(L, v))
	}

	fn := e.engineTable.RawGetString("apply")
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, cmd); err != nil {
		return ApplyResult{}, fmt.Errorf("calling engine.apply(%s): %w", commandType, err)
	}
	ret := L.Get(-1)
	L.Pop(1)

	return parseApplyResult(luaToGo(ret)), nil
}
