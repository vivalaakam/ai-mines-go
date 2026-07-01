package app

// TickAccumulator batches Ebitengine's per-frame Update() calls into whole
// 1-second game ticks (REQUIREMENTS.md §34: "Ebitengine may run at its normal
// update frequency, but game simulation uses 1-second ticks"). It has no
// Ebitengine dependency so the batching logic is unit-testable headlessly.
type TickAccumulator struct {
	updatesPerGameTick int
	counter            int
}

func NewTickAccumulator(updatesPerGameTick int) *TickAccumulator {
	if updatesPerGameTick < 1 {
		updatesPerGameTick = 1
	}
	return &TickAccumulator{updatesPerGameTick: updatesPerGameTick}
}

// Advance registers one Ebitengine frame and reports whether a full game tick
// has now accumulated (and resets the counter if so).
func (a *TickAccumulator) Advance() bool {
	a.counter++
	if a.counter >= a.updatesPerGameTick {
		a.counter = 0
		return true
	}
	return false
}

func (a *TickAccumulator) Reset() {
	a.counter = 0
}
