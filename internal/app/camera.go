package app

// Camera holds local view state only (REQUIREMENTS.md §Render/UI Layer allowed
// responsibilities: "keep local hover/selection/camera state"). It never touches
// authoritative game state.
type Camera struct {
	X, Y float64
	Zoom float64
}

func NewCamera() *Camera {
	return &Camera{Zoom: 1}
}

func (c *Camera) Move(dx, dy float64) {
	c.X += dx
	c.Y += dy
}

func (c *Camera) SetZoom(z float64) {
	if z < 0.25 {
		z = 0.25
	}
	if z > 4 {
		z = 4
	}
	c.Zoom = z
}

// Clamp keeps the camera from panning past the generated map, so the map
// never tears away from the screen edge leaving empty space. minX/minY/maxX
// are in the same world-pixel units as c.X/c.Y; viewportW/H is how many world
// pixels are visible on screen at the current zoom.
func (c *Camera) Clamp(minX, minY, maxX, maxY, viewportW, viewportH float64) {
	maxCamX := maxX - viewportW
	if maxCamX < minX {
		maxCamX = minX
	}
	maxCamY := maxY - viewportH
	if maxCamY < minY {
		maxCamY = minY
	}
	if c.X < minX {
		c.X = minX
	}
	if c.X > maxCamX {
		c.X = maxCamX
	}
	if c.Y < minY {
		c.Y = minY
	}
	if c.Y > maxCamY {
		c.Y = maxCamY
	}
}
