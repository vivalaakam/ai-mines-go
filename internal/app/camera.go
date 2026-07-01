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
	if c.X < 0 {
		c.X = 0
	}
	if c.Y < 0 {
		c.Y = 0
	}
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
