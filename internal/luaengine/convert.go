package luaengine

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// goToLua converts Go values (as produced by json-like map[string]any/[]any
// payloads) into Lua values for building command/query tables.
func goToLua(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case int:
		return lua.LNumber(val)
	case int64:
		return lua.LNumber(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case []string:
		tbl := L.NewTable()
		for i, item := range val {
			tbl.RawSetInt(i+1, lua.LString(item))
		}
		return tbl
	case []any:
		tbl := L.NewTable()
		for i, item := range val {
			tbl.RawSetInt(i+1, goToLua(L, item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, item := range val {
			tbl.RawSetString(k, goToLua(L, item))
		}
		return tbl
	default:
		return lua.LNil
	}
}

// luaToGo converts a Lua value returned from engine.apply/engine.read into plain
// Go values (map[string]any, []any, primitives) so callers never need to touch
// gopher-lua types directly.
func luaToGo(v lua.LValue) any {
	switch val := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(val)
	case lua.LNumber:
		return float64(val)
	case lua.LString:
		return string(val)
	case *lua.LTable:
		return luaTableToGo(val)
	default:
		return nil
	}
}

func luaTableToGo(tbl *lua.LTable) any {
	n := tbl.Len()
	isArray := n > 0
	fields := map[string]any{}

	tbl.ForEach(func(k, v lua.LValue) {
		if kn, ok := k.(lua.LNumber); ok {
			i := int(kn)
			if float64(i) != float64(kn) || i < 1 || i > n {
				isArray = false
			}
		} else {
			isArray = false
		}
		fields[luaKeyToString(k)] = luaToGo(v)
	})

	if isArray {
		arr := make([]any, n)
		for i := 1; i <= n; i++ {
			arr[i-1] = luaToGo(tbl.RawGetInt(i))
		}
		return arr
	}

	if n == 0 && len(fields) == 0 {
		// Lua can't distinguish {} from an empty map; treat as an empty list,
		// which is the common case for engine result tables like `events`.
		return []any{}
	}

	return fields
}

func luaKeyToString(k lua.LValue) string {
	if s, ok := k.(lua.LString); ok {
		return string(s)
	}
	return fmt.Sprintf("%v", k)
}
