package persistence

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// -- small accessors over the map[string]any/[]any shape luaToGo produces --

func str(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func strOrNil(m map[string]any, key string) any {
	if v, ok := m[key].(string); ok {
		return v
	}
	return nil
}

func num(m map[string]any, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func numOrNil(m map[string]any, key string) any {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return nil
}

func boolean(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func tbl(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return map[string]any{}
}

func arr(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return []any{}
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func parseIntPair(s string) (int, int, bool) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	x, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	y, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return x, y, true
}

func cellKey(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

// saveState writes the exported Lua state into saveID's structured rows,
// replacing any previous save with the same id, inside one transaction.
func (a *Adapter) saveState(saveID string, state map[string]any) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op once Commit succeeds

	if _, err := tx.Exec(`DELETE FROM saves WHERE id = ?`, saveID); err != nil {
		return fmt.Errorf("clearing previous save: %w", err)
	}

	gameTime := tbl(state, "gameTime")
	rules := tbl(state, "rulesConfig")
	nextIDs := tbl(state, "nextIds")

	_, err = tx.Exec(`
		INSERT INTO saves (
			id, seed_phrase, generator_version, schema_version, tick, money,
			highest_unlocked_worker_level, next_worker_id, next_storage_id, next_order_id, next_level_id,
			order_allocation_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		saveID, str(state, "seedPhrase"), int(num(state, "generatorVersion")), int(num(state, "schemaVersion")),
		int(num(gameTime, "tick")), num(state, "money"),
		int(num(state, "highestUnlockedWorkerLevel")), int(num(nextIDs, "worker")), int(num(nextIDs, "storage")),
		int(num(nextIDs, "order")), int(num(nextIDs, "level")),
		str(rules, "orderAllocationMode"),
	)
	if err != nil {
		return fmt.Errorf("inserting save: %w", err)
	}

	if err := saveLevels(tx, saveID, tbl(state, "levels")); err != nil {
		return err
	}
	if err := saveWorkers(tx, saveID, tbl(state, "workers"), tbl(state, "levels")); err != nil {
		return err
	}
	if err := saveStorages(tx, saveID, tbl(state, "storages")); err != nil {
		return err
	}
	if err := saveOrders(tx, saveID, tbl(state, "orders")); err != nil {
		return err
	}

	return tx.Commit()
}

func saveLevels(tx *sql.Tx, saveID string, levels map[string]any) error {
	for levelID, raw := range levels {
		level := asMap(raw)
		entrance := tbl(level, "entranceCell")
		stairs := tbl(level, "stairsCell")
		stairsChunk := tbl(level, "stairsChunk")

		_, err := tx.Exec(`
			INSERT INTO levels (id, save_id, depth, entrance_x, entrance_y, stairs_x, stairs_y,
				stairs_chunk_cx, stairs_chunk_cy, stairs_reachable, next_level_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			levelID, saveID, int(num(level, "depth")),
			int(num(entrance, "x")), int(num(entrance, "y")),
			int(num(stairs, "x")), int(num(stairs, "y")),
			int(num(stairsChunk, "cx")), int(num(stairsChunk, "cy")),
			boolean(level, "stairsReachable"), strOrNil(level, "nextLevelId"),
		)
		if err != nil {
			return fmt.Errorf("inserting level %s: %w", levelID, err)
		}

		for chunkKey := range tbl(level, "generatedChunks") {
			cx, cy, ok := parseIntPair(chunkKey)
			if !ok {
				continue
			}
			if _, err := tx.Exec(`INSERT INTO chunks (save_id, level_id, cx, cy) VALUES (?, ?, ?, ?)`,
				saveID, levelID, cx, cy); err != nil {
				return fmt.Errorf("inserting chunk %s/%s: %w", levelID, chunkKey, err)
			}
		}

		for cellKeyStr, rawCell := range tbl(level, "cells") {
			cell := asMap(rawCell)
			x, y, ok := parseIntPair(cellKeyStr)
			if !ok {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO cells (save_id, level_id, x, y, kind, visibility, accessibility, occupied_by)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				saveID, levelID, x, y, str(cell, "kind"), str(cell, "visibility"), str(cell, "accessibility"),
				strOrNil(cell, "occupiedBy"),
			); err != nil {
				return fmt.Errorf("inserting cell %s/%s: %w", levelID, cellKeyStr, err)
			}

			for idx, rawComp := range arr(cell, "components") {
				comp := asMap(rawComp)
				if _, err := tx.Exec(`
					INSERT INTO cell_components (save_id, level_id, cell_x, cell_y, idx, type, resource_id,
						initial_amount, remaining_amount, ratio)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					saveID, levelID, x, y, idx, str(comp, "type"), strOrNil(comp, "resourceId"),
					num(comp, "initialAmount"), num(comp, "remainingAmount"), num(comp, "ratio"),
				); err != nil {
					return fmt.Errorf("inserting component %s/%s[%d]: %w", levelID, cellKeyStr, idx, err)
				}
			}
		}
	}
	return nil
}

// saveWorkers also recovers each working worker's mining "side" (north/south/
// east/west) by searching its target cell's assignedWorkers map, since the
// worker record itself doesn't carry that - the cell does.
func saveWorkers(tx *sql.Tx, saveID string, workers map[string]any, levels map[string]any) error {
	for workerID, raw := range workers {
		worker := asMap(raw)
		side := findAssignedSide(levels, worker)

		_, err := tx.Exec(`
			INSERT INTO workers (id, save_id, level, speed, state, assigned_level_id, target_cell_id,
				position_cell_id, assignment_mode, assigned_side)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			workerID, saveID, int(num(worker, "level")), num(worker, "speed"), str(worker, "state"),
			strOrNil(worker, "assignedLevelId"), strOrNil(worker, "targetCellId"), strOrNil(worker, "positionCellId"),
			strOrNil(worker, "assignmentMode"), side,
		)
		if err != nil {
			return fmt.Errorf("inserting worker %s: %w", workerID, err)
		}
	}
	return nil
}

func findAssignedSide(levels map[string]any, worker map[string]any) any {
	levelID, ok := worker["assignedLevelId"].(string)
	if !ok || levelID == "" {
		return nil
	}
	targetCellID, ok := worker["targetCellId"].(string)
	if !ok || targetCellID == "" {
		return nil
	}
	workerID, _ := worker["id"].(string)

	level := asMap(levels[levelID])
	cell := asMap(tbl(level, "cells")[targetCellID])
	for side, wid := range tbl(cell, "assignedWorkers") {
		if s, ok := wid.(string); ok && s == workerID {
			return side
		}
	}
	return nil
}

func saveStorages(tx *sql.Tx, saveID string, storages map[string]any) error {
	for storageID, raw := range storages {
		storage := asMap(raw)
		if _, err := tx.Exec(`
			INSERT INTO storages (id, save_id, resource_id, level, capacity, stored_amount)
			VALUES (?, ?, ?, ?, ?, ?)`,
			storageID, saveID, str(storage, "resourceId"), int(num(storage, "level")),
			num(storage, "capacity"), num(storage, "storedAmount"),
		); err != nil {
			return fmt.Errorf("inserting storage %s: %w", storageID, err)
		}
	}
	return nil
}

func saveOrders(tx *sql.Tx, saveID string, orders map[string]any) error {
	for orderID, raw := range orders {
		order := asMap(raw)
		if _, err := tx.Exec(`
			INSERT INTO orders (id, save_id, state, reward_money, expires_at_tick, accepted_at_tick, priority)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			orderID, saveID, str(order, "state"), num(order, "rewardMoney"),
			int(num(order, "expiresAtTick")), numOrNil(order, "acceptedAtTick"), int(num(order, "priority")),
		); err != nil {
			return fmt.Errorf("inserting order %s: %w", orderID, err)
		}

		for idx, rawReq := range arr(order, "requirements") {
			req := asMap(rawReq)
			if _, err := tx.Exec(`
				INSERT INTO order_requirements (save_id, order_id, idx, resource_id, required_amount, delivered_amount)
				VALUES (?, ?, ?, ?, ?, ?)`,
				saveID, orderID, idx, str(req, "resourceId"), num(req, "requiredAmount"), num(req, "deliveredAmount"),
			); err != nil {
				return fmt.Errorf("inserting order requirement %s[%d]: %w", orderID, idx, err)
			}
		}
	}
	return nil
}
