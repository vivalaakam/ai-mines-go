package persistence

import (
	"path/filepath"
	"strconv"
	"testing"
)

func openTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	a, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func TestCreateSaveLoadRoundTrip(t *testing.T) {
	a := openTestAdapter(t)

	engine, err := a.CreateNewEngine("save-1", "persist-seed")
	if err != nil {
		t.Fatalf("CreateNewEngine() error: %v", err)
	}
	defer engine.Close()

	buy, err := engine.Apply("buy_worker", map[string]any{"workerLevel": float64(1)})
	if err != nil || !buy.OK {
		t.Fatalf("buy_worker failed: err=%v result=%+v", err, buy)
	}

	if err := a.SaveEngine(engine, "save-1"); err != nil {
		t.Fatalf("SaveEngine() error: %v", err)
	}

	loaded, err := a.LoadEngine("save-1")
	if err != nil {
		t.Fatalf("LoadEngine() error: %v", err)
	}
	defer loaded.Close()

	workers, err := loaded.Read("get_workers", nil)
	if err != nil {
		t.Fatalf("Read(get_workers) error: %v", err)
	}
	list, _ := workers["workers"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected 1 worker after reload, got %d (%+v)", len(list), workers)
	}

	summary, err := loaded.Read("get_player_summary", nil)
	if err != nil {
		t.Fatalf("Read(get_player_summary) error: %v", err)
	}
	if summary["money"] != 50.0 {
		t.Fatalf("expected money=50 after reload (100 - 50 for worker), got %v", summary["money"])
	}
}

func TestOrderPricePerUnitSurvivesRoundTrip(t *testing.T) {
	a := openTestAdapter(t)

	engine, err := a.CreateNewEngine("save-1", "order-price-seed")
	if err != nil {
		t.Fatalf("CreateNewEngine() error: %v", err)
	}
	defer engine.Close()

	before := readFirstOrderRequirement(t, engine.Read)
	price, _ := before["pricePerUnit"].(float64)
	if price <= 0 {
		t.Fatalf("expected a positive pricePerUnit on a fresh order, got %v", before["pricePerUnit"])
	}

	loaded, err := a.LoadEngine("save-1")
	if err != nil {
		t.Fatalf("LoadEngine() error: %v", err)
	}
	defer loaded.Close()

	after := readFirstOrderRequirement(t, loaded.Read)
	if after["pricePerUnit"] != before["pricePerUnit"] {
		t.Fatalf("pricePerUnit changed across save/load: before=%v after=%v", before["pricePerUnit"], after["pricePerUnit"])
	}
}

func readFirstOrderRequirement(t *testing.T, read func(string, map[string]any) (map[string]any, error)) map[string]any {
	t.Helper()
	orders, err := read("get_available_orders", nil)
	if err != nil {
		t.Fatalf("Read(get_available_orders) error: %v", err)
	}
	list, _ := orders["orders"].([]any)
	if len(list) == 0 {
		t.Fatalf("expected available orders, got %+v", orders)
	}
	reqs, _ := list[0].(map[string]any)["requirements"].([]any)
	if len(reqs) == 0 {
		t.Fatalf("expected order requirements, got %+v", list[0])
	}
	return reqs[0].(map[string]any)
}

func TestSaveEngineOverwritesPreviousSave(t *testing.T) {
	a := openTestAdapter(t)

	engine, err := a.CreateNewEngine("save-1", "seed-a")
	if err != nil {
		t.Fatalf("CreateNewEngine() error: %v", err)
	}
	defer engine.Close()

	if _, err := engine.Apply("buy_worker", map[string]any{"workerLevel": float64(1)}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if err := a.SaveEngine(engine, "save-1"); err != nil {
		t.Fatalf("first SaveEngine() error: %v", err)
	}

	if _, err := engine.Apply("buy_worker", map[string]any{"workerLevel": float64(1)}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if err := a.SaveEngine(engine, "save-1"); err != nil {
		t.Fatalf("second SaveEngine() error: %v", err)
	}

	loaded, err := a.LoadEngine("save-1")
	if err != nil {
		t.Fatalf("LoadEngine() error: %v", err)
	}
	defer loaded.Close()

	workers, err := loaded.Read("get_workers", nil)
	if err != nil {
		t.Fatalf("Read(get_workers) error: %v", err)
	}
	list, _ := workers["workers"].([]any)
	if len(list) != 2 {
		t.Fatalf("expected 2 workers (no duplicate rows from re-save), got %d", len(list))
	}
}

func TestLoadEngineAfterSavePreservesWorkerAssignment(t *testing.T) {
	a := openTestAdapter(t)

	engine, err := a.CreateNewEngine("save-1", "assign-seed")
	if err != nil {
		t.Fatalf("CreateNewEngine() error: %v", err)
	}
	defer engine.Close()

	buy, err := engine.Apply("buy_worker", map[string]any{"workerLevel": float64(1)})
	if err != nil || !buy.OK {
		t.Fatalf("buy_worker failed: err=%v result=%+v", err, buy)
	}
	workerID, _ := buy.Data.(map[string]any)["id"].(string)

	levelView, err := engine.Read("get_level_view", map[string]any{
		"levelId":  "level_1",
		"viewport": map[string]any{"x": float64(0), "y": float64(0), "width": float64(32), "height": float64(32)},
	})
	if err != nil {
		t.Fatalf("Read(get_level_view) error: %v", err)
	}

	targetCellID, positionCellID := findMinableAdjacentPair(t, levelView)

	assign, err := engine.Apply("assign_worker_to_target_cell", map[string]any{
		"workerId": workerID, "levelId": "level_1",
		"targetCellId": targetCellID, "positionCellId": positionCellID,
		"assignmentMode": "until_completed",
	})
	if err != nil || !assign.OK {
		t.Fatalf("assign failed: err=%v result=%+v", err, assign)
	}

	if err := a.SaveEngine(engine, "save-1"); err != nil {
		t.Fatalf("SaveEngine() error: %v", err)
	}

	loaded, err := a.LoadEngine("save-1")
	if err != nil {
		t.Fatalf("LoadEngine() error: %v", err)
	}
	defer loaded.Close()

	workers, err := loaded.Read("get_workers", nil)
	if err != nil {
		t.Fatalf("Read(get_workers) error: %v", err)
	}
	list, _ := workers["workers"].([]any)
	w := list[0].(map[string]any)
	if w["state"] != "working" {
		t.Fatalf("expected reloaded worker to still be working, got state=%v", w["state"])
	}
	if w["targetCellId"] != targetCellID {
		t.Fatalf("expected reloaded worker targetCellId=%s, got %v", targetCellID, w["targetCellId"])
	}

	// Reloaded engine must still be able to progress the mining job: tick once
	// without error, proving activeMiningCells/assignedWorkers were
	// reconstructed, not just the worker's own fields.
	tickResult, err := loaded.Apply("tick", map[string]any{"ticksPassed": float64(1)})
	if err != nil {
		t.Fatalf("tick error: %v", err)
	}
	if !tickResult.OK {
		t.Fatalf("tick failed after reload: %+v", tickResult.Error)
	}
}

func findMinableAdjacentPair(t *testing.T, levelView map[string]any) (string, string) {
	t.Helper()
	cells, _ := levelView["cells"].([]any)

	type coord struct{ x, y float64 }
	byCoord := map[coord]map[string]any{}
	for _, raw := range cells {
		c := raw.(map[string]any)
		byCoord[coord{c["x"].(float64), c["y"].(float64)}] = c
	}

	for _, raw := range cells {
		c := raw.(map[string]any)
		if c["kind"] != "deposit" {
			continue
		}
		x, y := c["x"].(float64), c["y"].(float64)
		neighbors := []coord{{x, y - 1}, {x, y + 1}, {x - 1, y}, {x + 1, y}}
		for _, n := range neighbors {
			nc, ok := byCoord[n]
			if !ok {
				continue
			}
			if nc["accessibility"] == "reachable" && (nc["kind"] == "empty" || nc["kind"] == "stairs_area") {
				return coordID(x, y), coordID(n.x, n.y)
			}
		}
	}
	t.Fatal("could not find a minable deposit adjacent to a reachable open cell")
	return "", ""
}

func coordID(x, y float64) string {
	return strconv.Itoa(int(x)) + "," + strconv.Itoa(int(y))
}
