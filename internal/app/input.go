package app

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// InputState is a snapshot of this frame's raw input, decoupled from ebiten so
// Update() logic could be tested without a graphics context if needed later.
type InputState struct {
	CameraDX, CameraDY float64
	ZoomDelta          float64
	HireWorkerClicked  bool
}

const cameraPanSpeed = 6.0

func PollInput() InputState {
	var s InputState
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		s.CameraDY -= cameraPanSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		s.CameraDY += cameraPanSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		s.CameraDX -= cameraPanSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		s.CameraDX += cameraPanSpeed
	}
	_, wheelY := ebiten.Wheel()
	s.ZoomDelta = wheelY * 0.1

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.HireWorkerClicked = image.Pt(x, y).In(render.HireWorkerButton)
	}

	return s
}
