package app

import (
	"image"
	"log"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// handleOrderButtonClick hit-tests a mouse press against the orders panel's
// Accept/Decline buttons (same index-based rects render draws) and sends the
// matching command to Lua. Returns true if the click landed on a button, so
// the caller can stop it from also being treated as a map click.
func (g *Game) handleOrderButtonClick(mx, my int) (bool, error) {
	pt := image.Pt(mx, my)
	for i, orderID := range g.lastAvailableOrderIDs {
		var command string
		switch {
		case pt.In(render.AvailableOrderAcceptButton(i)):
			command = "accept_order"
		case pt.In(render.AvailableOrderDeclineButton(i)):
			command = "decline_order"
		default:
			continue
		}
		result, err := g.engine.Apply(command, map[string]any{"orderId": orderID})
		if err != nil {
			return true, err
		}
		if !result.OK {
			log.Printf("%s rejected: %+v", command, result.Error)
		}
		return true, nil
	}
	return false, nil
}
