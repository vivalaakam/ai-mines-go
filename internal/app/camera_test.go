package app

import "testing"

func TestCameraClampKeepsMapGluedToScreenEdge(t *testing.T) {
	c := NewCamera()

	c.Move(-999999, -999999)
	c.Clamp(-64*24, -64*24, 96*24, 96*24, 1280, 720)
	if c.X != -64*24 || c.Y != -64*24 {
		t.Fatalf("expected camera clamped to map min edge, got X=%v Y=%v", c.X, c.Y)
	}

	c.Move(999999, 999999)
	c.Clamp(-64*24, -64*24, 96*24, 96*24, 1280, 720)
	wantMaxX := 96*24 - 1280.0
	wantMaxY := 96*24 - 720.0
	if c.X != wantMaxX || c.Y != wantMaxY {
		t.Fatalf("expected camera clamped to map max edge, got X=%v Y=%v want X=%v Y=%v", c.X, c.Y, wantMaxX, wantMaxY)
	}
}

func TestCameraClampWhenMapSmallerThanViewport(t *testing.T) {
	c := NewCamera()
	c.Move(500, 500)
	c.Clamp(0, 0, 100, 100, 1280, 720)
	if c.X != 0 || c.Y != 0 {
		t.Fatalf("map smaller than viewport should pin camera to min edge, got X=%v Y=%v", c.X, c.Y)
	}
}
