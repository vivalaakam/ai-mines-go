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

// PauseButton is the always-visible top-right corner button that opens the
// pause menu. Exported so internal/app can hit-test clicks against the same
// rect this package draws.
var PauseButton = image.Rect(ScreenWidth-78, 6, ScreenWidth-8, 30)

// Pause menu modal layout. Continue/Exit buttons are exported for internal/app
// hit-testing; pauseModalRect/confirmExitModalRect are layout-only.
var (
	pauseModalRect       = image.Rect(ScreenWidth/2-130, ScreenHeight/2-100, ScreenWidth/2+130, ScreenHeight/2+100)
	PauseContinueButton  = image.Rect(pauseModalRect.Min.X+20, pauseModalRect.Min.Y+95, pauseModalRect.Min.X+125, pauseModalRect.Min.Y+135)
	PauseExitButton      = image.Rect(pauseModalRect.Max.X-125, pauseModalRect.Min.Y+95, pauseModalRect.Max.X-20, pauseModalRect.Min.Y+135)
	confirmExitModalRect = image.Rect(ScreenWidth/2-120, ScreenHeight/2-50, ScreenWidth/2+120, ScreenHeight/2+50)
	ConfirmYesButton     = image.Rect(confirmExitModalRect.Min.X+15, confirmExitModalRect.Max.Y-34, confirmExitModalRect.Min.X+105, confirmExitModalRect.Max.Y-10)
	ConfirmNoButton      = image.Rect(confirmExitModalRect.Max.X-105, confirmExitModalRect.Max.Y-34, confirmExitModalRect.Max.X-15, confirmExitModalRect.Max.Y-10)
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
	drawHireButton(screen, vm, money)
	drawWorkersPanel(screen, vm)
}

// sidebarPadding is the inset between the sidebar's opaque background edges
// and the text/buttons drawn inside it.
const sidebarPadding = 12

// sidebarX0 is the sidebar column's left edge in screen space.
const sidebarX0 = MapWidth

// drawSidebar paints the full-height, opaque info column (REQUIREMENTS.md UX:
// the game field and the resources/orders HUD must not blend into each other,
// so this is a solid background, not a translucent overlay on the map) and
// stacks the player summary, resources, orders, and recent order-event log
// inside it. Sections report back the next free y so they never overlap
// regardless of how many resources/orders/events are actually present.
func drawSidebar(screen *ebiten.Image, vm ViewModel) {
	vector.FillRect(screen, float32(sidebarX0), 0, SidebarWidth, ScreenHeight, color.RGBA{22, 22, 26, 255}, false)
	vector.StrokeRect(screen, float32(sidebarX0), 0, 1, ScreenHeight, 1, color.RGBA{60, 60, 68, 255}, false)

	x := sidebarX0 + sidebarPadding
	// The top-right corner is reserved for the Pause button (drawn last), so
	// the sidebar content starts below it instead of at the very top.
	y := sidebarPadding + 26

	y = drawPlayerSummary(screen, vm, x, y)
	y = drawResourcesPanel(screen, vm, x, y)
	y = drawOrdersPanel(screen, vm, x, y)
	drawOrderEventLog(screen, vm, x, y)

	drawPauseButton(screen, vm)
}

// drawPauseButton renders the top-right corner button that opens the pause
// menu. Hidden while the pause overlay is up (the overlay covers it).
func drawPauseButton(screen *ebiten.Image, vm ViewModel) {
	if vm.PauseMenu != nil {
		return
	}
	b := PauseButton
	vector.FillRect(screen, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), color.RGBA{50, 50, 55, 255}, false)
	vector.StrokeRect(screen, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), 1, color.RGBA{120, 120, 130, 255}, false)
	ebitenutil.DebugPrintAt(screen, "Pause", b.Min.X+16, b.Min.Y+6)
}

// drawPauseOverlay dims the screen and draws the Paused menu (Continue/Exit),
// plus the nested exit-confirmation dialog when requested. Drawn last so it
// sits above every other layer, including the merge modal.
func drawPauseOverlay(screen *ebiten.Image, vm ViewModel) {
	p := vm.PauseMenu
	if p == nil {
		return
	}
	vector.FillRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight), color.RGBA{0, 0, 0, 170}, false)

	r := pauseModalRect
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.RGBA{30, 30, 30, 235}, false)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 2, color.RGBA{255, 255, 255, 255}, false)
	ebitenutil.DebugPrintAt(screen, "Paused", r.Min.X+(r.Dx()-len("Paused")*6)/2, r.Min.Y+12)
	ebitenutil.DebugPrintAt(screen, "ESC / Start to resume", r.Min.X+38, r.Min.Y+44)

	drawModalButton(screen, PauseContinueButton, "Continue", color.RGBA{60, 130, 60, 255}, p.Gamepad && p.PauseSel == 0)
	drawModalButton(screen, PauseExitButton, "Exit", color.RGBA{130, 60, 60, 255}, p.Gamepad && p.PauseSel == 1)

	if p.ConfirmExit {
		drawConfirmExitModal(screen, p)
	}
}

// drawConfirmExitModal draws the "Save and quit?" yes/no dialog on top of the
// pause menu.
func drawConfirmExitModal(screen *ebiten.Image, p *PauseMenu) {
	r := confirmExitModalRect
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.RGBA{20, 20, 20, 245}, false)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 2, color.RGBA{255, 255, 255, 255}, false)
	ebitenutil.DebugPrintAt(screen, "Save and quit?", r.Min.X+(r.Dx()-len("Save and quit?")*6)/2, r.Min.Y+16)
	drawModalButton(screen, ConfirmYesButton, "Yes", color.RGBA{60, 130, 60, 255}, p.Gamepad && p.ConfirmSel == 0)
	drawModalButton(screen, ConfirmNoButton, "No", color.RGBA{130, 60, 60, 255}, p.Gamepad && p.ConfirmSel == 1)
}

// drawModalButton fills a button rect, centers its label, and strokes a
// yellow highlight when the gamepad has it selected.
func drawModalButton(screen *ebiten.Image, r image.Rectangle, label string, fill color.Color, highlight bool) {
	vector.FillRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), fill, false)
	if highlight {
		vector.StrokeRect(screen, float32(r.Min.X)-2, float32(r.Min.Y)-2, float32(r.Dx())+4, float32(r.Dy())+4, 2, color.RGBA{255, 230, 0, 255}, false)
	}
	ebitenutil.DebugPrintAt(screen, label, r.Min.X+(r.Dx()-len(label)*6)/2, r.Min.Y+(r.Dy()-8)/2)
}

func drawPlayerSummary(screen *ebiten.Image, vm ViewModel, x, y int) int {
	money, _ := vm.PlayerSummary["money"].(float64)
	workerCount, _ := vm.PlayerSummary["workerCount"].(float64)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("money: %.0f | workers: %.0f", money, workerCount), x, y)
	return y + 20
}

// Orders panel layout, stacked inside the sidebar below the resources panel.
// MaxAvailableOrderRows caps how many available orders get drawn (and thus
// clickable buttons); internal/app hit-tests against the same index-based
// rects, so both sides stay in sync by construction.
const MaxAvailableOrderRows = 3

const (
	availableOrderBlockH = 50
	orderButtonW         = 62
	orderButtonH         = 16
)

// availableOrdersOriginY is set by drawOrdersPanel each frame so
// AvailableOrderAcceptButton/AvailableOrderDeclineButton (called by
// internal/app for hit-testing, and by drawOrdersPanel itself) always agree
// on where the available-orders list actually starts, even though that
// position depends on the (variable-length) resources panel drawn above it.
var availableOrdersOriginY = 224

// AvailableOrderAcceptButton is the clickable rect of the i-th drawn available
// order's Accept button - exported for internal/app hit-testing, same pattern
// as HireWorkerButton.
func AvailableOrderAcceptButton(i int) image.Rectangle {
	x := sidebarX0 + sidebarPadding
	y := availableOrdersOriginY + 16 + i*availableOrderBlockH + 30
	return image.Rect(x, y, x+orderButtonW, y+orderButtonH)
}

// AvailableOrderDeclineButton is the clickable rect of the i-th drawn
// available order's Decline button.
func AvailableOrderDeclineButton(i int) image.Rectangle {
	return AvailableOrderAcceptButton(i).Add(image.Pt(orderButtonW+8, 0))
}

// AvailableOrderRow is the full screen-space row (label + buttons) of the i-th
// drawn available order. Exported so internal/app can draw a gamepad
// selection highlight over the whole row, not just the buttons.
func AvailableOrderRow(i int) image.Rectangle {
	x := sidebarX0 + sidebarPadding
	y := availableOrdersOriginY + 16 + i*availableOrderBlockH
	return image.Rect(x, y, x+SidebarWidth-2*sidebarPadding, y+availableOrderBlockH)
}

// requirementsLine renders one order's requirements as e.g.
// "stone 12/40 @$2, coal 0/15 @$5" (delivered/required at per-unit price).
func requirementsLine(order map[string]any) string {
	reqs, _ := order["requirements"].([]any)
	line := ""
	for _, raw := range reqs {
		req, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		resourceID, _ := req["resourceId"].(string)
		required, _ := req["requiredAmount"].(float64)
		delivered, _ := req["deliveredAmount"].(float64)
		price, _ := req["pricePerUnit"].(float64)
		if line != "" {
			line += ", "
		}
		line += fmt.Sprintf("%s %.0f/%.0f @$%.0f", resourceID, delivered, required, price)
	}
	return line
}

func orderProgress(order map[string]any) (delivered, required float64) {
	reqs, _ := order["requirements"].([]any)
	for _, raw := range reqs {
		req, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		r, _ := req["requiredAmount"].(float64)
		d, _ := req["deliveredAmount"].(float64)
		required += r
		delivered += d
	}
	return delivered, required
}

// drawOrdersPanel shows incoming (available) orders with Accept/Decline
// buttons and active (accepted) orders with per-resource delivery progress,
// starting at (x,y) and returning the next free y.
func drawOrdersPanel(screen *ebiten.Image, vm ViewModel, x, y int) int {
	availableOrdersOriginY = y
	ebitenutil.DebugPrintAt(screen, "Orders:", x, y)

	var tick float64
	if vm.AvailableOrders != nil {
		tick, _ = vm.AvailableOrders["tick"].(float64)
		available, _ := vm.AvailableOrders["orders"].([]any)
		for i, raw := range available {
			if i >= MaxAvailableOrderRows {
				break
			}
			order, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			by := y + 16 + i*availableOrderBlockH
			reward, _ := order["rewardMoney"].(float64)
			expires, _ := order["expiresAtTick"].(float64)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("$%.0f  expires in %.0ft", reward, expires-tick), x, by)
			ebitenutil.DebugPrintAt(screen, requirementsLine(order), x, by+14)

			acc := AvailableOrderAcceptButton(i)
			vector.FillRect(screen, float32(acc.Min.X), float32(acc.Min.Y), float32(acc.Dx()), float32(acc.Dy()), color.RGBA{60, 130, 60, 255}, false)
			ebitenutil.DebugPrintAt(screen, "Accept", acc.Min.X+8, acc.Min.Y+1)

			dec := AvailableOrderDeclineButton(i)
			vector.FillRect(screen, float32(dec.Min.X), float32(dec.Min.Y), float32(dec.Dx()), float32(dec.Dy()), color.RGBA{130, 60, 60, 255}, false)
			ebitenutil.DebugPrintAt(screen, "Decline", dec.Min.X+5, dec.Min.Y+1)
		}
	}

	ay := y + 16 + MaxAvailableOrderRows*availableOrderBlockH + 6
	ebitenutil.DebugPrintAt(screen, "Active orders:", x, ay)
	ay += 16

	var active []any
	if vm.ActiveOrders != nil {
		active, _ = vm.ActiveOrders["orders"].([]any)
	}
	if len(active) == 0 {
		ebitenutil.DebugPrintAt(screen, "(none)", x, ay)
		return ay + 16
	}
	barW := SidebarWidth - 2*sidebarPadding - 110
	for _, raw := range active {
		order, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		delivered, required := orderProgress(order)
		ratio := 0.0
		if required > 0 {
			ratio = delivered / required
		}
		id, _ := order["id"].(string)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s  %.0f%%", id, ratio*100), x, ay)
		barX, barY := x+110, ay+3
		vector.FillRect(screen, float32(barX), float32(barY), float32(barW), 8, color.RGBA{50, 50, 55, 255}, false)
		vector.FillRect(screen, float32(barX), float32(barY), float32(float64(barW)*ratio), 8, color.RGBA{80, 160, 80, 255}, false)
		ay += 14
		ebitenutil.DebugPrintAt(screen, requirementsLine(order), x, ay)
		ay += 16
		if ay > ScreenHeight-80 {
			break
		}
	}
	return ay
}

// orderEventLogMaxLines caps how many recent order events (accepted,
// shipped, arrived, expired, completed) are shown, so the sidebar's bottom
// section has a fixed, predictable height.
const orderEventLogMaxLines = 6

// drawOrderEventLog makes order events explicit in the sidebar itself
// (arrivals, shipments and their payment, expirations) instead of only being
// visible in the application log - vm.OrderEventLog is newest-first.
func drawOrderEventLog(screen *ebiten.Image, vm ViewModel, x, y int) {
	ebitenutil.DebugPrintAt(screen, "Recent order events:", x, y)
	y += 16
	if len(vm.OrderEventLog) == 0 {
		ebitenutil.DebugPrintAt(screen, "(none yet)", x, y)
		return
	}
	lines := vm.OrderEventLog
	if len(lines) > orderEventLogMaxLines {
		lines = lines[:orderEventLogMaxLines]
	}
	for _, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, x, y)
		y += 14
	}
}

// drawResourcesPanel lists every unlocked resource's stored amount, starting
// at (x,y) and returning the next free y. Storage is uncapped, so no
// capacity is shown.
func drawResourcesPanel(screen *ebiten.Image, vm ViewModel, x, y int) int {
	ebitenutil.DebugPrintAt(screen, "Resources:", x, y)
	y += 16

	list, _ := vm.Resources["resources"].([]any)
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
	return y + 10
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
