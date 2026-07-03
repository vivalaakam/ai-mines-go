package persistence

import "fmt"

// loadState reads saveID's structured rows back into the exact map[string]any
// shape lua/state.lua expects from engine.load_state.
func (a *Adapter) loadState(saveID string) (map[string]any, error) {
	row := a.db.QueryRow(`
		SELECT seed_phrase, generator_version, schema_version, tick, money,
			highest_unlocked_worker_level, next_worker_id, next_storage_id, next_order_id, next_level_id,
			order_allocation_mode
		FROM saves WHERE id = ?`, saveID)

	var (
		seedPhrase, orderAllocationMode                              string
		generatorVersion, schemaVersion, tick, highestUnlockedWorker int
		nextWorker, nextStorage, nextOrder, nextLevel                int
		money                                                        float64
	)
	if err := row.Scan(&seedPhrase, &generatorVersion, &schemaVersion, &tick, &money,
		&highestUnlockedWorker, &nextWorker, &nextStorage, &nextOrder, &nextLevel,
		&orderAllocationMode); err != nil {
		return nil, fmt.Errorf("loading save %s: %w", saveID, err)
	}

	state := map[string]any{
		"schemaVersion":              float64(schemaVersion),
		"generatorVersion":           float64(generatorVersion),
		"seedPhrase":                 seedPhrase,
		"gameTime":                   map[string]any{"tick": float64(tick)},
		"money":                      money,
		"highestUnlockedWorkerLevel": float64(highestUnlockedWorker),
		"rulesConfig": map[string]any{
			"orderAllocationMode": orderAllocationMode,
		},
		"nextIds": map[string]any{
			"worker":  float64(nextWorker),
			"storage": float64(nextStorage),
			"order":   float64(nextOrder),
			"level":   float64(nextLevel),
		},
	}

	levels, err := a.loadLevels(saveID)
	if err != nil {
		return nil, err
	}
	state["levels"] = levels

	workers, err := a.loadWorkers(saveID, levels)
	if err != nil {
		return nil, err
	}
	state["workers"] = workers

	storages, err := a.loadStorages(saveID)
	if err != nil {
		return nil, err
	}
	state["storages"] = storages

	orders, err := a.loadOrders(saveID)
	if err != nil {
		return nil, err
	}
	state["orders"] = orders

	return state, nil
}

func (a *Adapter) loadLevels(saveID string) (map[string]any, error) {
	rows, err := a.db.Query(`
		SELECT id, depth, entrance_x, entrance_y, stairs_x, stairs_y, stairs_chunk_cx, stairs_chunk_cy,
			stairs_reachable, next_level_id
		FROM levels WHERE save_id = ?`, saveID)
	if err != nil {
		return nil, fmt.Errorf("querying levels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	levels := map[string]any{}
	type levelRow struct {
		id                              string
		depth, ex, ey, sx, sy, scx, scy int
		stairsReachable                 bool
		nextLevelID                     *string
	}
	var levelRows []levelRow
	for rows.Next() {
		var lr levelRow
		if err := rows.Scan(&lr.id, &lr.depth, &lr.ex, &lr.ey, &lr.sx, &lr.sy, &lr.scx, &lr.scy,
			&lr.stairsReachable, &lr.nextLevelID); err != nil {
			return nil, fmt.Errorf("scanning level: %w", err)
		}
		levelRows = append(levelRows, lr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, lr := range levelRows {
		var nextLevelID any
		if lr.nextLevelID != nil {
			nextLevelID = *lr.nextLevelID
		}
		level := map[string]any{
			"id":                lr.id,
			"depth":             float64(lr.depth),
			"entranceCell":      map[string]any{"x": float64(lr.ex), "y": float64(lr.ey)},
			"stairsCell":        map[string]any{"x": float64(lr.sx), "y": float64(lr.sy)},
			"stairsChunk":       map[string]any{"cx": float64(lr.scx), "cy": float64(lr.scy)},
			"stairsReachable":   lr.stairsReachable,
			"nextLevelId":       nextLevelID,
			"generatedChunks":   map[string]any{},
			"cells":             map[string]any{},
			"activeMiningCells": map[string]any{},
		}
		levels[lr.id] = level

		if err := a.loadChunks(saveID, lr.id, level); err != nil {
			return nil, err
		}
		if err := a.loadCells(saveID, lr.id, level); err != nil {
			return nil, err
		}
	}
	return levels, nil
}

func (a *Adapter) loadChunks(saveID, levelID string, level map[string]any) error {
	rows, err := a.db.Query(`SELECT cx, cy FROM chunks WHERE save_id = ? AND level_id = ?`, saveID, levelID)
	if err != nil {
		return fmt.Errorf("querying chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	generatedChunks := asMap(level["generatedChunks"])
	for rows.Next() {
		var cx, cy int
		if err := rows.Scan(&cx, &cy); err != nil {
			return fmt.Errorf("scanning chunk: %w", err)
		}
		generatedChunks[cellKey(cx, cy)] = true
	}
	return rows.Err()
}

func (a *Adapter) loadCells(saveID, levelID string, level map[string]any) error {
	rows, err := a.db.Query(`
		SELECT x, y, kind, visibility, accessibility, occupied_by
		FROM cells WHERE save_id = ? AND level_id = ?`, saveID, levelID)
	if err != nil {
		return fmt.Errorf("querying cells: %w", err)
	}
	defer func() { _ = rows.Close() }()

	cells := asMap(level["cells"])
	for rows.Next() {
		var x, y int
		var kind, visibility, accessibility string
		var occupiedBy *string
		if err := rows.Scan(&x, &y, &kind, &visibility, &accessibility, &occupiedBy); err != nil {
			return fmt.Errorf("scanning cell: %w", err)
		}
		var occupied any
		if occupiedBy != nil {
			occupied = *occupiedBy
		}
		cells[cellKey(x, y)] = map[string]any{
			"x": float64(x), "y": float64(y),
			"kind": kind, "visibility": visibility, "accessibility": accessibility,
			"occupiedBy":      occupied,
			"components":      []any{},
			"assignedWorkers": map[string]any{},
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return a.loadCellComponents(saveID, levelID, cells)
}

func (a *Adapter) loadCellComponents(saveID, levelID string, cells map[string]any) error {
	rows, err := a.db.Query(`
		SELECT cell_x, cell_y, type, resource_id, initial_amount, remaining_amount, ratio
		FROM cell_components WHERE save_id = ? AND level_id = ? ORDER BY cell_x, cell_y, idx`, saveID, levelID)
	if err != nil {
		return fmt.Errorf("querying cell_components: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var x, y int
		var compType string
		var resourceID *string
		var initialAmount, remainingAmount, ratio float64
		if err := rows.Scan(&x, &y, &compType, &resourceID, &initialAmount, &remainingAmount, &ratio); err != nil {
			return fmt.Errorf("scanning cell_component: %w", err)
		}
		cell := asMap(cells[cellKey(x, y)])
		var resID any
		if resourceID != nil {
			resID = *resourceID
		}
		components := cell["components"].([]any)
		cell["components"] = append(components, map[string]any{
			"type": compType, "resourceId": resID,
			"initialAmount": initialAmount, "remainingAmount": remainingAmount, "ratio": ratio,
		})
	}
	return rows.Err()
}

// loadWorkers also reconstructs the derived state cells/activeMiningCells need:
// each working worker's side entry in its target cell's assignedWorkers map.
func (a *Adapter) loadWorkers(saveID string, levels map[string]any) (map[string]any, error) {
	rows, err := a.db.Query(`
		SELECT id, level, speed, state, assigned_level_id, target_cell_id, position_cell_id,
			assignment_mode, assigned_side
		FROM workers WHERE save_id = ?`, saveID)
	if err != nil {
		return nil, fmt.Errorf("querying workers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	workers := map[string]any{}
	for rows.Next() {
		var id, state string
		var level int
		var speed float64
		var assignedLevelID, targetCellID, positionCellID, assignmentMode, assignedSide *string
		if err := rows.Scan(&id, &level, &speed, &state, &assignedLevelID, &targetCellID, &positionCellID,
			&assignmentMode, &assignedSide); err != nil {
			return nil, fmt.Errorf("scanning worker: %w", err)
		}

		worker := map[string]any{
			"id": id, "level": float64(level), "speed": speed, "state": state,
			"assignedLevelId": strPtrToAny(assignedLevelID), "targetCellId": strPtrToAny(targetCellID),
			"positionCellId": strPtrToAny(positionCellID), "assignmentMode": strPtrToAny(assignmentMode),
		}
		workers[id] = worker

		if assignedLevelID != nil && targetCellID != nil && assignedSide != nil {
			level := asMap(levels[*assignedLevelID])
			cells := asMap(level["cells"])
			cell := asMap(cells[*targetCellID])
			assignedWorkers := asMap(cell["assignedWorkers"])
			assignedWorkers[*assignedSide] = id
			activeMiningCells := asMap(level["activeMiningCells"])
			activeMiningCells[*targetCellID] = true
		}
	}
	return workers, rows.Err()
}

func strPtrToAny(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func (a *Adapter) loadStorages(saveID string) (map[string]any, error) {
	rows, err := a.db.Query(`
		SELECT id, resource_id, level, capacity, stored_amount FROM storages WHERE save_id = ?`, saveID)
	if err != nil {
		return nil, fmt.Errorf("querying storages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	storages := map[string]any{}
	for rows.Next() {
		var id, resourceID string
		var level int
		var capacity, storedAmount float64
		if err := rows.Scan(&id, &resourceID, &level, &capacity, &storedAmount); err != nil {
			return nil, fmt.Errorf("scanning storage: %w", err)
		}
		storages[id] = map[string]any{
			"id": id, "resourceId": resourceID, "level": float64(level),
			"capacity": capacity, "storedAmount": storedAmount,
		}
	}
	return storages, rows.Err()
}

func (a *Adapter) loadOrders(saveID string) (map[string]any, error) {
	rows, err := a.db.Query(`
		SELECT id, state, reward_money, expires_at_tick, accepted_at_tick, priority
		FROM orders WHERE save_id = ?`, saveID)
	if err != nil {
		return nil, fmt.Errorf("querying orders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	orders := map[string]any{}
	var orderIDs []string
	for rows.Next() {
		var id, state string
		var rewardMoney float64
		var expiresAtTick int
		var acceptedAtTick *int
		var priority int
		if err := rows.Scan(&id, &state, &rewardMoney, &expiresAtTick, &acceptedAtTick, &priority); err != nil {
			return nil, fmt.Errorf("scanning order: %w", err)
		}
		var accepted any
		if acceptedAtTick != nil {
			accepted = float64(*acceptedAtTick)
		}
		orders[id] = map[string]any{
			"id": id, "state": state, "rewardMoney": rewardMoney,
			"expiresAtTick": float64(expiresAtTick), "acceptedAtTick": accepted, "priority": float64(priority),
			"requirements": []any{},
		}
		orderIDs = append(orderIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, orderID := range orderIDs {
		if err := a.loadOrderRequirements(saveID, orderID, asMap(orders[orderID])); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

func (a *Adapter) loadOrderRequirements(saveID, orderID string, order map[string]any) error {
	rows, err := a.db.Query(`
		SELECT resource_id, required_amount, delivered_amount, price_per_unit
		FROM order_requirements WHERE save_id = ? AND order_id = ? ORDER BY idx`, saveID, orderID)
	if err != nil {
		return fmt.Errorf("querying order_requirements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	requirements := order["requirements"].([]any)
	for rows.Next() {
		var resourceID string
		var required, delivered, pricePerUnit float64
		if err := rows.Scan(&resourceID, &required, &delivered, &pricePerUnit); err != nil {
			return fmt.Errorf("scanning order_requirement: %w", err)
		}
		requirements = append(requirements, map[string]any{
			"resourceId": resourceID, "requiredAmount": required, "deliveredAmount": delivered,
			"pricePerUnit": pricePerUnit,
		})
	}
	order["requirements"] = requirements
	return rows.Err()
}
