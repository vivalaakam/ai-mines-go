package luaengine

import (
	"log"

	lua "github.com/yuin/gopher-lua"
)

// registerHostFunctions installs the small set of technical host functions Lua
// is allowed to call (AGENTS.md: logging, serialization helpers - never gameplay
// logic). Currently just structured logging for debugging the running engine.
func registerHostFunctions(L *lua.LState) {
	host := L.NewTable()
	L.SetFuncs(host, map[string]lua.LGFunction{
		"log": func(L *lua.LState) int {
			log.Println("[lua]", L.CheckString(1))
			return 0
		},
	})
	L.SetGlobal("host", host)
}
