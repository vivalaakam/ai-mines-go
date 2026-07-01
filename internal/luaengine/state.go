package luaengine

// EngineError mirrors the {code, message, details} shape returned by Lua
// (REQUIREMENTS.md §38). Callers must branch on Code, never parse Message.
type EngineError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *EngineError) Error() string {
	return e.Code + ": " + e.Message
}

// ApplyResult mirrors the table returned by engine.apply.
type ApplyResult struct {
	OK             bool
	Events         []any
	Patch          map[string]any
	Data           any
	ProcessedTicks *float64
	RemainingTicks *float64
	Error          *EngineError
}

func parseEngineError(v any) *EngineError {
	m, ok := v.(map[string]any)
	if !ok {
		return &EngineError{Code: "internal_error", Message: "malformed error from engine"}
	}
	engErr := &EngineError{}
	if code, ok := m["code"].(string); ok {
		engErr.Code = code
	}
	if msg, ok := m["message"].(string); ok {
		engErr.Message = msg
	}
	if details, ok := m["details"].(map[string]any); ok {
		engErr.Details = details
	}
	return engErr
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return []any{}
}

func floatPtr(v any) *float64 {
	if f, ok := v.(float64); ok {
		return &f
	}
	return nil
}

func parseApplyResult(raw any) ApplyResult {
	m := asMap(raw)
	result := ApplyResult{}
	if ok, _ := m["ok"].(bool); ok {
		result.OK = true
		result.Events = asSlice(m["events"])
		result.Patch = asMap(m["patch"])
		result.Data = m["data"]
		result.ProcessedTicks = floatPtr(m["processedTicks"])
		result.RemainingTicks = floatPtr(m["remainingTicks"])
	} else {
		result.OK = false
		result.Error = parseEngineError(m["error"])
	}
	return result
}
