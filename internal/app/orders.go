package app

import (
	"image"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// handleOrderButtonClick hit-tests a mouse press against the orders panel's
// Accept/Decline buttons (same index-based rects render draws) and sends the
// matching command to Lua. Returns true if the click landed on a button, so
// the caller can stop it from also being treated as a map click.
func (g *Game) handleOrderButtonClick(mx, my int) (bool, error) {
	pt := image.Pt(mx, my)
	for i := range g.lastAvailableOrderIDs {
		switch {
		case pt.In(render.AvailableOrderAcceptButton(i)):
			g.acceptOrderAtIndex(i)
			return true, nil
		case pt.In(render.AvailableOrderDeclineButton(i)):
			g.declineOrderAtIndex(i)
			return true, nil
		}
	}
	return false, nil
}
