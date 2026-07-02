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

// mergeModalRect/MergeModalYesButton/MergeModalNoButton lay out the
// merge-confirmation modal. Yes/No buttons are exported so internal/app can
// hit-test clicks against the exact rectangles this package draws.
var (
	mergeModalRect      = image.Rect(ScreenWidth/2-110, ScreenHeight/2-40, ScreenWidth/2+110, ScreenHeight/2+40)
	MergeModalYesButton = image.Rect(mergeModalRect.Min.X+10, mergeModalRect.Max.Y-32, mergeModalRect.Min.X+95, mergeModalRect.Max.Y-8)
	MergeModalNoButton  = image.Rect(mergeModalRect.Max.X-95, mergeModalRect.Max.Y-32, mergeModalRect.Max.X-10, mergeModalRect.Max.Y-8)
)

// drawMergeModal shows the "merge these two workers?" confirmation prompt
// triggered by click-selecting a worker and then clicking another worker of
// the same level (see internal/app/drag.go handleWorkerClick).
func drawMergeModal(screen *ebiten.Image, vm ViewModel) {
	if vm.MergeConfirm == nil {
		return
	}
	r := mergeModalRect
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.RGBA{30, 30, 30, 230}, false)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 2, color.RGBA{255, 255, 255, 255}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Merge two Lv%d workers?", vm.MergeConfirm.Level), r.Min.X+10, r.Min.Y+10)

	yes := MergeModalYesButton
	vector.FillRect(screen, float32(yes.Min.X), float32(yes.Min.Y), float32(yes.Dx()), float32(yes.Dy()), color.RGBA{60, 130, 60, 255}, false)
	ebitenutil.DebugPrintAt(screen, "Yes", yes.Min.X+28, yes.Min.Y+8)

	no := MergeModalNoButton
	vector.FillRect(screen, float32(no.Min.X), float32(no.Min.Y), float32(no.Dx()), float32(no.Dy()), color.RGBA{130, 60, 60, 255}, false)
	ebitenutil.DebugPrintAt(screen, "No", no.Min.X+32, no.Min.Y+8)
}

func drawUI(screen *ebiten.Image, vm ViewModel) {
	money, _ := vm.PlayerSummary["money"].(float64)
	workerCount, _ := vm.PlayerSummary["workerCount"].(float64)

	line := fmt.Sprintf("money: %.0f | workers: %.0f", money, workerCount)
	ebitenutil.DebugPrintAt(screen, line, 8, 8)

	drawHireButton(screen, vm, money)
	drawWorkersPanel(screen, vm)
	drawResourcesPanel(screen, vm)
}

// drawResourcesPanel lists every unlocked resource's stored amount,
// anchored to the top-right so it never overlaps the worker roster on
// the left. Storage is uncapped, so no capacity is shown.
func drawResourcesPanel(screen *ebiten.Image, vm ViewModel) {
	if vm.Resources == nil {
		return
	}
	list, _ := vm.Resources["resources"].([]any)
	if len(list) == 0 {
		return
	}

	x, y := ScreenWidth-220, 8
	ebitenutil.DebugPrintAt(screen, "Resources:", x, y)
	y += 16

	for _, raw := range list {
		resource, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		stored, _ := resource["totalStored"].(float64)
		if stored == 0 {
			continue
		}
		name, _ := resource["name"].(string)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s: %.0f", name, stored), x, y)
		y += 14
	}
}

func drawHireButton(screen *ebiten.Image, vm ViewModel, money float64) {
	if vm.Workers == nil {
		return
	}
	level, _ := vm.Workers["nextPurchasableWorkerLevel"].(float64)
	cost, _ := vm.Workers["nextPurchaseCost"].(float64)

	enabled := money >= cost
	fill := color.RGBA{60, 110, 60, 255}
	if !enabled {
		fill = color.RGBA{60, 60, 60, 255}
	}

	b := HireWorkerButton
	vector.FillRect(screen, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), fill, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Hire Lv%.0f worker ($%.0f)", level, cost), b.Min.X+6, b.Min.Y+6)
}

// workersPanelMaxRows caps how many worker rows the roster prints so it can
// never grow past the screen even with a large pool.
const workersPanelMaxRows = 14

var workersPanelOrigin = image.Pt(8, 68)

// drawWorkersPanel lists every worker in the global pool (REQUIREMENTS.md
// §17), not just the ones currently mining. Idle/just-purchased workers have
// no positionCellId yet, so drawWorkers (worker_renderer.go) never places them
// on the map - this roster is the only place they're visible until assigned.
func drawWorkersPanel(screen *ebiten.Image, vm ViewModel) {
	if vm.Workers == nil {
		return
	}
	list, _ := vm.Workers["workers"].([]any)
	if len(list) == 0 {
		return
	}

	x, y := workersPanelOrigin.X, workersPanelOrigin.Y
	ebitenutil.DebugPrintAt(screen, "Workers:", x, y)
	y += 16

	shown := list
	overflow := 0
	if len(shown) > workersPanelMaxRows {
		overflow = len(shown) - workersPanelMaxRows
		shown = shown[:workersPanelMaxRows]
	}

	for _, raw := range shown {
		worker, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := worker["id"].(string)
		level, _ := worker["level"].(float64)
		state, _ := worker["state"].(string)

		clr := workerLevelColor(int(level))
		vector.FillRect(screen, float32(x), float32(y+2), 8, 8, clr, false)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s  Lv%.0f  %s", id, level, state), x+14, y)
		y += 14
	}
	if overflow > 0 {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("... +%d more", overflow), x+14, y)
	}
}
