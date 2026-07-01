package app

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// Draw reads view-models from the Lua engine and hands them to the render
// package. It never touches authoritative state (REQUIREMENTS.md: "Go must not
// mutate authoritative state in Draw").
func (g *Game) Draw(screen *ebiten.Image) {
	viewportX := math.Floor(g.camera.X / render.TileSize)
	viewportY := math.Floor(g.camera.Y / render.TileSize)

	levelView, err := g.engine.Read("get_level_view", map[string]any{
		"levelId": g.levelID,
		"viewport": map[string]any{
			"x":      viewportX,
			"y":      viewportY,
			"width":  float64(render.ViewportCellsWide),
			"height": float64(render.ViewportCellsTall),
		},
	})
	if err != nil {
		log.Printf("get_level_view failed: %v", err)
		return
	}

	playerSummary, err := g.engine.Read("get_player_summary", nil)
	if err != nil {
		log.Printf("get_player_summary failed: %v", err)
		return
	}

	shiftSummary, err := g.engine.Read("get_shift_summary", nil)
	if err != nil {
		log.Printf("get_shift_summary failed: %v", err)
		return
	}

	workers, err := g.engine.Read("get_workers", nil)
	if err != nil {
		log.Printf("get_workers failed: %v", err)
		return
	}

	render.Draw(screen, render.ViewModel{
		Camera:        render.Camera{X: g.camera.X, Y: g.camera.Y, Zoom: g.camera.Zoom},
		LevelView:     levelView,
		PlayerSummary: playerSummary,
		ShiftSummary:  shiftSummary,
		Workers:       workers,
	})
}
