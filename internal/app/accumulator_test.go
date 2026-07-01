package app

import "testing"

func TestTickAccumulatorFiresEveryNAdvances(t *testing.T) {
	acc := NewTickAccumulator(60)
	fired := 0
	for range 180 {
		if acc.Advance() {
			fired++
		}
	}
	if fired != 3 {
		t.Fatalf("expected 3 game ticks over 180 frames at 60 updates/tick, got %d", fired)
	}
}

func TestTickAccumulatorResetDropsProgress(t *testing.T) {
	acc := NewTickAccumulator(10)
	for range 9 {
		acc.Advance()
	}
	acc.Reset()
	for range 9 {
		if acc.Advance() {
			t.Fatalf("did not expect a tick before reaching 10 advances after reset")
		}
	}
	if !acc.Advance() {
		t.Fatalf("expected the 10th advance after reset to fire a tick")
	}
}
