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
	Gamepad            gamepadInput
}

// pointerState is the unified pointer for one frame. drag.go reads it instead
// of ebiten.CursorPosition()+mouse buttons directly. With a gamepad connected
// it sits at g.cursor (mouse + left stick drive one entity); without one it
// tracks the OS mouse cursor.
type pointerState struct {
	pos          image.Point
	justPressed  bool
	justReleased bool
}

// gamepadInput is one frame of raw state from the first connected
// standard-layout gamepad. Sticks are deadzoned floats in [-1,1]; buttons are
// just-pressed edges except dpadUp/dpadDown which are held (so zoom/list repeat
// can be driven off them with a cooldown). mouseClick echoes the mouse
// just-pressed edge so that, while a pad is connected, a mouse click acts on
// the highlighted tile through the same mapCursorAction path as A.
type gamepadInput struct {
	present                       bool
	leftX, leftY, rightX, rightY  float64
	a, b, selectBtn, startBtn, r2 bool
	mouseClick                    bool
	dpadUp, dpadDown              bool
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
	g.syncPointer(s.Gamepad)
	return s
}

// pollGamepad reads the first connected standard-layout gamepad into s.Gamepad
// and adds the right stick to camera pan (pan works in any focus mode so the
// player can look around while navigating menus). The left stick does NOT move
// the cursor here — that is focus-dependent (map vs list nav) and lives in
// gamepad.go; only the A just-released edge is captured here so the pad's A
// can feed the same press/release click flow as the mouse button.
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
		gp.startBtn = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight)
		gp.r2 = inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonFrontBottomRight)
		break
	}
	gp.mouseClick = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	s.Gamepad = gp
	g.gamepadPresent = gp.present
}

// syncPointer builds this frame's g.pointer. With a gamepad connected:
//   - mouse over the sidebar (x >= MapWidth): the mouse is a normal OS cursor
//     — g.pointer tracks it so sidebar buttons are clickable, the tile stays
//     parked (no tile highlight).
//   - otherwise (mouse over the map, or the stick last moved the tile): the
//     tile is the cursor — mouse motion snaps it to the cell under the mouse,
//     g.pointer is zeroed so drag.go / UI hit-testing defer to mapCursorAction.
//
// Without a gamepad g.pointer just tracks the OS mouse cursor and drag.go
// handles clicks/drags normally.
func (g *Game) syncPointer(gp gamepadInput) {
	mx, my := ebiten.CursorPosition()
	mousePos := image.Pt(mx, my)
	mousePressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	mouseReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	if gp.present {
		mouseMoved := mousePos != g.lastMousePos
		if mouseMoved {
			g.cursorFromMouse = true
		}
		g.lastMousePos = mousePos

		// Mouse over the sidebar or a button → normal OS cursor, tile parked,
		// so those controls are clickable.
		if g.cursorFromMouse && g.pointerOnUI(mx, my) {
			g.pointer = pointerState{pos: mousePos, justPressed: mousePressed, justReleased: mouseReleased}
			return
		}

		// Tile mode: snap the tile to the mouse cell on mouse motion, and zero
		// the pointer so drag.go / UI hit-testing defer to the tile.
		if mouseMoved {
			cx, cy := render.ScreenToCell(mx, my, g.renderCamera())
			g.cursorCellX, g.cursorCellY = cx, cy
			g.clampCursorCell()
			g.cursorInit = true
		}
		g.pointer = pointerState{}
		return
	}

	// No gamepad: pure mouse. Drop the tile so a reconnect re-inits it.
	g.cursorFromMouse = false
	g.cursorInit = false
	g.pointer = pointerState{
		pos:          mousePos,
		justPressed:  mousePressed,
		justReleased: mouseReleased,
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
