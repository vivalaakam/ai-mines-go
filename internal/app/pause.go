package app

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// openPause shows the pause menu and freezes the simulation. The tick
// accumulator is reset so resuming starts a fresh tick window instead of
// inheriting the partial count from before pausing.
func (g *Game) openPause() {
	g.paused = true
	g.confirmExit = false
	g.pauseSel = 0
	g.confirmSel = 1
	g.accumulator.Reset()
	g.showCursor()
}

// showCursor forces the OS cursor visible so mouse-driven pause buttons are
// usable even when the gamepad tile cursor had hidden it.
func (g *Game) showCursor() {
	if g.cursorHidden {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		g.cursorHidden = false
	}
}

// handlePauseInput drives the pause menu: ESC/Start/B resume, Up/Down (pad)
// move the highlight, A activates it, a mouse click hits the Continue/Exit
// buttons directly.
func (g *Game) handlePauseInput(input InputState) {
	g.showCursor()
	gp := input.Gamepad

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || gp.startBtn || gp.b {
		g.paused = false
		return
	}

	if gp.present {
		if g.listCD > 0 {
			g.listCD--
		}
		if move := g.listMove(gp); move != 0 && g.listCD == 0 {
			g.pauseSel += move
			if g.pauseSel < 0 {
				g.pauseSel = 0
			}
			if g.pauseSel > 1 {
				g.pauseSel = 1
			}
			g.listCD = listMoveInterval
		}
		if gp.a {
			g.activatePauseSel()
			return
		}
	}

	if g.pointer.justPressed {
		pt := image.Pt(g.pointer.pos.X, g.pointer.pos.Y)
		switch {
		case pt.In(render.PauseContinueButton):
			g.paused = false
		case pt.In(render.PauseExitButton):
			g.confirmExit = true
			g.confirmSel = 1
		}
	}
}

// activatePauseSel triggers the gamepad-highlighted pause button: Continue
// resumes, Exit opens the exit-confirmation dialog.
func (g *Game) activatePauseSel() {
	if g.pauseSel == 0 {
		g.paused = false
		return
	}
	g.confirmExit = true
	g.confirmSel = 1
}

// handleConfirmExitInput drives the "Save and quit?" dialog: ESC/B/No cancels
// back to the pause menu, A activates the highlighted button, a mouse click
// hits Yes/No directly. Yes saves the game then exits via ebiten.Termination.
func (g *Game) handleConfirmExitInput(input InputState) error {
	g.showCursor()
	gp := input.Gamepad

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || gp.b {
		g.confirmExit = false
		return nil
	}

	if gp.present {
		if g.listCD > 0 {
			g.listCD--
		}
		if move := g.listMove(gp); move != 0 && g.listCD == 0 {
			g.confirmSel += move
			if g.confirmSel < 0 {
				g.confirmSel = 0
			}
			if g.confirmSel > 1 {
				g.confirmSel = 1
			}
			g.listCD = listMoveInterval
		}
		if gp.a {
			return g.activateConfirmSel()
		}
	}

	if g.pointer.justPressed {
		pt := image.Pt(g.pointer.pos.X, g.pointer.pos.Y)
		switch {
		case pt.In(render.ConfirmYesButton):
			return g.quitConfirmed()
		case pt.In(render.ConfirmNoButton):
			g.confirmExit = false
		}
	}
	return nil
}

// activateConfirmSel triggers the gamepad-highlighted confirm button: Yes
// saves and quits, No returns to the pause menu.
func (g *Game) activateConfirmSel() error {
	if g.confirmSel == 0 {
		return g.quitConfirmed()
	}
	g.confirmExit = false
	return nil
}

// quitConfirmed persists the current game state and terminates the Ebitengine
// loop cleanly. ebiten.Termination is the sentinel error RunGame treats as a
// normal exit (not a failure), so main returns 0.
func (g *Game) quitConfirmed() error {
	if err := g.SaveNow(); err != nil {
		log.Printf("save before quit failed: %v", err)
	} else {
		log.Printf("save before quit completed")
	}
	return ebiten.Termination
}
