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
}

// pointerState is the unified mouse/gamepad pointer for one frame. drag.go and
// update.go read it instead of ebiten.CursorPosition()+mouse buttons directly,
// so a gamepad-driven cursor (left stick to move, A to click) feeds the exact
// same hit-testing the mouse does.
type pointerState struct {
	pos          image.Point
	justPressed  bool
	justReleased bool
	gamepad      bool // true when driven by the gamepad this frame (draw a cursor)
}

const (
	cameraPanSpeed = 6.0
	// stickDeadzone ignores centered-stick drift. stickPanSpeed scales a full
	// right-stick deflection into per-frame camera pixels. cursorSpeed does the
	// same for the left-stick virtual cursor. ponytail: calibration knobs.
	stickDeadzone = 0.25
	stickPanSpeed = cameraPanSpeed * 5
	cursorSpeed   = 10.0
	dpadZoomSpeed = 0.02
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
// and resolves the unified pointer (g.pointer) for this frame.
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

	aPressed, aReleased := g.pollGamepad(&s)
	g.syncPointer(aPressed, aReleased)
	return s
}

// pollGamepad reads the first connected standard-layout gamepad:
//   - right stick → camera pan
//   - left stick → virtual cursor
//   - D-pad up/down → zoom
//   - A button (RightBottom) → click edges (resolved into g.pointer by syncPointer)
//
// Returns the A button just-pressed/just-released edges for this frame.
func (g *Game) pollGamepad(s *InputState) (aPressed, aReleased bool) {
	for id := range g.gamepadIDs {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		rx := standardAxis(id, ebiten.StandardGamepadAxisRightStickHorizontal)
		ry := standardAxis(id, ebiten.StandardGamepadAxisRightStickVertical)
		s.CameraDX += rx * stickPanSpeed
		s.CameraDY += ry * stickPanSpeed

		lx := standardAxis(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		ly := standardAxis(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if lx != 0 || ly != 0 {
			if !g.cursorActive {
				g.cursor = image.Pt(render.ScreenWidth/2, render.ScreenHeight/2)
				g.cursorActive = true
			}
			g.cursor.X += int(lx * cursorSpeed)
			g.cursor.Y += int(ly * cursorSpeed)
			clampCursor(&g.cursor)
		}

		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			s.ZoomDelta += dpadZoomSpeed
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			s.ZoomDelta -= dpadZoomSpeed
		}

		aPressed = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom)
		aReleased = inpututil.IsStandardGamepadButtonJustReleased(id, ebiten.StandardGamepadButtonRightBottom)
		break
	}
	return aPressed, aReleased
}

// syncPointer resolves the frame's unified pointer from mouse + gamepad A edge.
// The gamepad cursor stays active until the mouse moves or clicks, so the two
// don't fight: whichever the player last touched wins.
func (g *Game) syncPointer(aPressed, aReleased bool) {
	mx, my := ebiten.CursorPosition()
	mousePos := image.Pt(mx, my)
	mousePressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	mouseReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	if mousePos != g.lastMousePos || mousePressed {
		g.cursorActive = false
	}
	if aPressed {
		g.cursorActive = true
	}

	pos := mousePos
	if g.cursorActive {
		pos = g.cursor
	}
	g.lastMousePos = mousePos
	g.pointer = pointerState{
		pos:          pos,
		justPressed:  mousePressed || aPressed,
		justReleased: mouseReleased || (aReleased && g.cursorActive),
		gamepad:      g.cursorActive,
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

func clampCursor(p *image.Point) {
	if p.X < 0 {
		p.X = 0
	}
	if p.X > render.ScreenWidth {
		p.X = render.ScreenWidth
	}
	if p.Y < 0 {
		p.Y = 0
	}
	if p.Y > render.ScreenHeight {
		p.Y = render.ScreenHeight
	}
}
