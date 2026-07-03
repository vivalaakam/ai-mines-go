package app

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputState is a snapshot of this frame's raw input, decoupled from ebiten so
// Update() logic could be tested without a graphics context if needed later.
type InputState struct {
	CameraDX, CameraDY float64
	ZoomDelta          float64
	Gamepad            gamepadInput
}

// pointerState is the mouse pointer for one frame. drag.go reads it instead of
// ebiten.CursorPosition()+mouse buttons directly. (The gamepad uses a separate
// cell-cursor + focus state machine in gamepad.go, not this pointer.)
type pointerState struct {
	pos          image.Point
	justPressed  bool
	justReleased bool
}

// gamepadInput is one frame of raw state from the first connected
// standard-layout gamepad. Sticks are deadzoned floats in [-1,1]; buttons are
// just-pressed edges except dpadUp/dpadDown which are held (so zoom/list repeat
// can be driven off them with a cooldown).
type gamepadInput struct {
	present                      bool
	leftX, leftY, rightX, rightY float64
	a, b, selectBtn, r2          bool
	dpadUp, dpadDown             bool
}

const (
	cameraPanSpeed    = 6.0
	stickDeadzone     = 0.25
	stickPanSpeed     = cameraPanSpeed * 5
	dpadZoomSpeed     = 0.02
	stickActThreshold = 0.5 // deflection above which a stick "acts" (move cursor / navigate list)
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

// pollInput collects keyboard/mouse and gamepad input into one frame snapshot
// and resolves the mouse pointer (g.pointer) for this frame.
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

	g.pollGamepad(&s)
	g.syncPointer()
	return s
}

// pollGamepad reads the first connected standard-layout gamepad into s.Gamepad
// and adds the right stick to camera pan (pan works in any focus mode so the
// player can look around while navigating menus).
func (g *Game) pollGamepad(s *InputState) {
	var gp gamepadInput
	for id := range g.gamepadIDs {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		gp.present = true
		gp.leftX = standardAxis(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		gp.leftY = standardAxis(id, ebiten.StandardGamepadAxisLeftStickVertical)
		gp.rightX = standardAxis(id, ebiten.StandardGamepadAxisRightStickHorizontal)
		gp.rightY = standardAxis(id, ebiten.StandardGamepadAxisRightStickVertical)
		s.CameraDX += gp.rightX * stickPanSpeed
		s.CameraDY += gp.rightY * stickPanSpeed

		gp.dpadUp = ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop)
		gp.dpadDown = ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom)
		gp.a = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom)
		gp.b = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight)
		gp.selectBtn = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterLeft)
		gp.r2 = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonFrontBottomRight)
		break
	}
	s.Gamepad = gp
}

// syncPointer snapshots the mouse into g.pointer for drag.go/update.go.
func (g *Game) syncPointer() {
	mx, my := ebiten.CursorPosition()
	g.pointer = pointerState{
		pos:          image.Pt(mx, my),
		justPressed:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		justReleased: inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft),
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
