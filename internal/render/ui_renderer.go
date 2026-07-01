package render

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// HireWorkerButton is the screen-space (not world-space) clickable area for
// hiring a worker. Exported so internal/app can hit-test clicks against the
// exact same rectangle this package draws - one source of truth, no duplicated
// layout constants between the render and input layers.
var HireWorkerButton = image.Rect(8, 32, 230, 60)

func drawUI(screen *ebiten.Image, vm ViewModel) {
	money, _ := vm.PlayerSummary["money"].(float64)
	phase, _ := vm.PlayerSummary["phase"].(string)
	workerCount, _ := vm.PlayerSummary["workerCount"].(float64)

	line := fmt.Sprintf("money: %.0f | phase: %s | workers: %.0f", money, phase, workerCount)
	if vm.ShiftSummary != nil {
		ticksIntoShift, _ := vm.ShiftSummary["ticksIntoShift"].(float64)
		shiftLength, _ := vm.ShiftSummary["shiftLength"].(float64)
		line += fmt.Sprintf(" | shift: %.0f/%.0f", ticksIntoShift, shiftLength)
	}
	ebitenutil.DebugPrintAt(screen, line, 8, 8)

	drawHireButton(screen, vm, phase, money)
}

func drawHireButton(screen *ebiten.Image, vm ViewModel, phase string, money float64) {
	if vm.Workers == nil {
		return
	}
	level, _ := vm.Workers["nextPurchasableWorkerLevel"].(float64)
	cost, _ := vm.Workers["nextPurchaseCost"].(float64)

	enabled := phase == "shift_planning" && money >= cost
	fill := color.RGBA{60, 110, 60, 255}
	if !enabled {
		fill = color.RGBA{60, 60, 60, 255}
	}

	b := HireWorkerButton
	vector.FillRect(screen, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), fill, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Hire Lv%.0f worker ($%.0f)", level, cost), b.Min.X+6, b.Min.Y+6)
}
