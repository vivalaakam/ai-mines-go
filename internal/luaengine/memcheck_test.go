package luaengine

import (
	"fmt"
	"runtime"
	"testing"
)

// TestMemoryTickOnlyPath is a regression guard against unbounded growth in the
// authoritative Lua engine's hot path (tick + settle). It runs many ticks with
// no reads and no autosave (the shape of an Ebitengine run while the window is
// unfocused/hidden: Update runs, Draw is skipped) and asserts the Go heap does
// not climb. The previously-leaking culprit was state.orders accumulating
// terminal orders forever; orders.settle now prunes them.
func TestMemoryTickOnlyPath(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if err := e.NewGame("memcheck"); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)

	const batches = 40
	const perBatch = 60
	for b := 0; b < batches; b++ {
		for i := 0; i < perBatch; i++ {
			if _, err := e.Apply("tick", map[string]any{"ticksPassed": float64(1)}); err != nil {
				t.Fatal(err)
			}
		}
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if b%10 == 0 {
			fmt.Printf("tick=%4d  HeapAlloc=%.1f MB  NumGC=%d\n", b*perBatch, float64(m.HeapAlloc)/1e6, m.NumGC)
		}
	}
	runtime.GC()
	var end runtime.MemStats
	runtime.ReadMemStats(&end)
	growth := int64(end.HeapAlloc) - int64(base.HeapAlloc)
	fmt.Printf("tick path growth after %d ticks: %.2f MB (base=%.1fMB end=%.1fMB)\n",
		batches*perBatch, float64(growth)/1e6, float64(base.HeapAlloc)/1e6, float64(end.HeapAlloc)/1e6)
	if growth > 5_000_000 {
		t.Errorf("tick path leaked >5MB: %.2f MB", float64(growth)/1e6)
	}
}
