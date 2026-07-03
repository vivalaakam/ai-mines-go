package app

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/vivalaakam/ai-mines-go/internal/luaengine"
	"github.com/vivalaakam/ai-mines-go/internal/persistence"
	"github.com/vivalaakam/ai-mines-go/internal/render"
)

// TicksPerSecond is Ebitengine's default TPS; one game tick = one real second
// (REQUIREMENTS.md §6), so this is also updatesPerGameTick for the accumulator.
const TicksPerSecond = 60

// AutosaveIntervalTicks controls how often the game autosaves now that there
// are no more shift boundaries to trigger it.
const AutosaveIntervalTicks = 60

// Game is the Ebitengine entry point. It holds no gameplay state of its own -
// everything authoritative is fetched from the Lua engine via apply/read on
// demand (REQUIREMENTS.md: "Go must not mutate authoritative state directly").
type Game struct {
	engine         *luaengine.Engine
	store          *persistence.Adapter
	saveID         string
	camera         *Camera
	accumulator    *TickAccumulator
	levelID        string
	mapBounds      *MapBounds
	ticksSinceSave int

	// lastLevelView caches the level view Draw last fetched, so Update can
	// hit-test worker drag-and-drop against it without an extra engine.Read
	// (one frame stale, which is imperceptible for mouse interaction).
	lastLevelView     map[string]any
	draggingWorkerID  string
	pressPos          image.Point
	suppressNextClick bool

	// lastAvailableOrderIDs caches the available-order ids in the exact order
	// the orders panel drew them last frame, so Update can hit-test the
	// index-based Accept/Decline button rects against the right order.
	lastAvailableOrderIDs []string

	// viewsDirty is set after every successful engine.Apply (tick, buy, merge,
	// assign, order action, level creation) and cleared once Draw has
	// re-fetched the cached view-models. Between applies the engine state is
	// unchanged, so the 5 non-viewport reads are returned from cache instead
	// of doing 6 Lua round-trips every frame (the main source of allocation
	// churn and baseline memory). The level view is additionally re-fetched
	// when the camera viewport cell range changes.
	viewsDirty            bool
	cachedLevelView       map[string]any
	cachedPlayerSummary   map[string]any
	cachedWorkers         map[string]any
	cachedResources       map[string]any
	cachedAvailableOrders map[string]any
	cachedActiveOrders    map[string]any
	lastViewportX         float64
	lastViewportY         float64
	lastViewportW         float64
	lastViewportH         float64
	hasCachedViewport     bool

	// orderEventLog is a newest-first ring of human-readable order events
	// (arrived/shipped/expired/completed), shown in the sidebar so order
	// activity is visible in the UI itself, not just the application log.
	orderEventLog []string

	// selectedWorkerID and pendingMerge back the click-to-select "cut/paste"
	// gesture: click a worker to select it, click a deposit to move it there,
	// click a same-level worker to ask for merge confirmation (see drag.go).
	selectedWorkerID string
	pendingMerge     *PendingMerge

	// gamepadIDs tracks connected gamepads so pollInput can read the first
	// standard-layout one (see syncGamepads in input.go).
	gamepadIDs    map[ebiten.GamepadID]struct{}
	gamepadIDsBuf []ebiten.GamepadID

	// pointer is this frame's mouse pointer, consumed by update.go/drag.go for
	// click interactions. It is only used when no gamepad is connected; while a
	// pad is connected the highlighted tile is the single cursor and A/mouse
	// clicks act on it via gamepad.go's mapCursorAction, so g.pointer is zeroed
	// to keep drag.go / UI hit-testing from double-acting.
	pointer pointerState

	// cursorCell is the single map cursor: the highlighted tile. The left
	// stick steps it one cell at a time (gamepad.go); mouse motion snaps it to
	// the cell under the mouse — one entity driven by both inputs.
	// cursorFromMouse tracks whether the mouse (true) or the stick (false) last
	// drove it: over the sidebar the mouse becomes a normal OS cursor instead
	// (tileActive=false), so sidebar buttons are clickable; the stick always
	// reclaims the tile. gamepadPresent gates pad-mode behavior; cursorHidden
	// tracks the OS cursor mode to avoid re-setting it every frame. cursorCD
	// paces stick cell-stepping. lastMousePos detects mouse motion to snap.
	cursorCellX     float64
	cursorCellY     float64
	cursorInit      bool
	cursorCD        int
	cursorFromMouse bool
	tileActive      bool
	gamepadPresent  bool
	cursorHidden    bool
	lastMousePos    image.Point

	// Gamepad focus/list state (gamepad.go):
	//   focus         - which surface the pad drives: map, orders, or hire
	//   orderSel      - highlighted available-order index (orders focus)
	//   hireSel       - highlighted purchasable-worker index (hire focus)
	//   hireLevels    - cached purchasable levels while the hire panel is open
	//   listCD        - frames until a held up/down advances the list again
	focus      focusMode
	orderSel   int
	hireSel    int
	hireLevels []hireLevel
	listCD     int

	// Pause menu state (pause.go). paused stops the tick accumulator from
	// advancing (the simulation freezes); confirmExit is the nested "save and
	// quit?" dialog. pauseSel/confirmSel are the gamepad-highlighted button
	// indices (0=Continue/Yes, 1=Exit/No). Pause is app-layer UI state only -
	// it never reaches Lua.
	paused      bool
	confirmExit bool
	pauseSel    int
	confirmSel  int
}

// focusMode is which surface the gamepad currently drives. Mouse input is
// independent and always active regardless of focus.
type focusMode int

const (
	focusMap focusMode = iota
	focusOrders
	focusHire
)

// hireLevel is one buyable worker tier shown in the hire-select panel.
type hireLevel struct {
	Level float64
	Cost  float64
}

// PendingMerge holds a same-level worker pair awaiting the player's yes/no
// confirmation in the merge modal.
type PendingMerge struct {
	WorkerA, WorkerB string
	Level            int
}

// MapBounds is the known-generated extent of the current level, in world
// cell coordinates (inclusive). Populated from get_level_view's "bounds"
// field each Draw and used by Update to keep the camera from panning past
// the generated map into empty space.
type MapBounds struct {
	MinX, MinY, MaxX, MaxY float64
}

// NewGame wires an already-loaded/created engine to the Ebitengine loop. store
// and saveID may be left as nil/"" to run without autosave (e.g. in tests).
func NewGame(engine *luaengine.Engine, store *persistence.Adapter, saveID string, levelID string) *Game {
	return &Game{
		engine:      engine,
		store:       store,
		saveID:      saveID,
		camera:      NewCamera(),
		accumulator: NewTickAccumulator(TicksPerSecond),
		levelID:     levelID,
	}
}

// Layout returns a fixed logical resolution rather than echoing back
// outsideWidth/outsideHeight. Ebitengine then scales this logical canvas up
// to fill the actual window/fullscreen output (letterboxed, aspect-preserved),
// which is what makes the map/HUD/buttons scale with the screen instead of
// staying pinned to a small corner of a large fullscreen framebuffer.
// CursorPosition() already reports clicks in these same logical coordinates,
// so hit-testing (e.g. render.HireWorkerButton) needs no extra conversion.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return render.ScreenWidth, render.ScreenHeight
}
