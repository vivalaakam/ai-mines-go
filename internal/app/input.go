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

const (
	cameraPanSpeed = 6.0
	// stickDeadzone ignores centered-stick drift; stickPanSpeed scales a
	// full-deflection analog stick into per-frame camera pixels. Both are
	// calibration knobs — ponytail: tuned by feel, raise if panning feels slow.
	stickDeadzone  = 0.25
	stickPanSpeed  = cameraPanSpeed * 5
	stickZoomSpeed = 0.02
)

// syncGamepads tracks connected gamepad IDs, mirroring the ebiten gamepad
// example. Only the first standard-layout pad is read each frame.
func (g *Game) syncGamepads() {
	if g.gamepadIDs == nil {
		g.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	}
	g.gamepadIDsBuf = inpututil.AppendJustConnectedGamepadIDs(g.gamepadIDsBuf[:0])
	for _, id := range g.gamepadIDsBuf {
		g.gamepadIDs[id] = struct{}{}
	}
	for id := range g.gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			delete(g.gamepadIDs, id)
		}
	}
}

// pollInput collects keyboard/mouse and gamepad input into one frame snapshot.
// Gamepad contributions add to the keyboard values, so both work simultaneously.
func (g *Game) pollInput() InputState {
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

	g.pollGamepad(&s)
	return s
}

// pollGamepad reads the first connected standard-layout gamepad:
//   - left stick / D-pad → camera pan
//   - right stick vertical → zoom (up = zoom in)
//   - A button (RightBottom) → hire worker, same as clicking the HUD button
func (g *Game) pollGamepad(s *InputState) {
	for id := range g.gamepadIDs {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		lx := standardAxis(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		ly := standardAxis(id, ebiten.StandardGamepadAxisLeftStickVertical)
		s.CameraDX += lx * stickPanSpeed
		s.CameraDY += ly * stickPanSpeed

		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			s.CameraDX -= cameraPanSpeed
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			s.CameraDX += cameraPanSpeed
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			s.CameraDY -= cameraPanSpeed
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			s.CameraDY += cameraPanSpeed
		}

		ry := standardAxis(id, ebiten.StandardGamepadAxisRightStickVertical)
		s.ZoomDelta += -ry * stickZoomSpeed

		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) {
			s.HireWorkerClicked = true
		}
		break
	}
}

// standardAxis returns a deadzoned standard gamepad axis value in [-1, 1].
func standardAxis(id ebiten.GamepadID, axis ebiten.StandardGamepadAxis) float64 {
	v := ebiten.StandardGamepadAxisValue(id, axis)
	if v > -stickDeadzone && v < stickDeadzone {
		return 0
	}
	return v
}
