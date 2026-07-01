package render

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

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
}
