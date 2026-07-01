local balance = require("config.balance")
local cellsMod = require("generation.cells")

local M = {}

local function err(code, message)
  return { code = code, message = message, details = {} }
end

local function next_id(state, kind)
  state.nextIds[kind] = state.nextIds[kind] + 1
  return kind .. "_" .. (state.nextIds[kind] - 1)
end

local function side_of(target, position)
  if position.x == target.x and position.y == target.y - 1 then
    return "north"
  end
  if position.x == target.x and position.y == target.y + 1 then
    return "south"
  end
  if position.x == target.x - 1 and position.y == target.y then
    return "west"
  end
  if position.x == target.x + 1 and position.y == target.y then
    return "east"
  end
  return nil
end

--- Releases a worker from whatever cell/position it currently occupies, without changing its state.
function M.detach_worker(state, worker)
  if worker.assignedLevelId then
    local level = state.levels[worker.assignedLevelId]
    if level then
      local targetCellId = worker.targetCellId
      local targetCell = targetCellId and level.cells[targetCellId]
      local positionCell = worker.positionCellId and level.cells[worker.positionCellId]
      if targetCell then
        for side, wid in pairs(targetCell.assignedWorkers) do
          if wid == worker.id then
            targetCell.assignedWorkers[side] = nil
          end
        end
        if next(targetCell.assignedWorkers) == nil then
          level.activeMiningCells[targetCellId] = nil
        end
      end
      if positionCell and positionCell.occupiedBy == worker.id then
        positionCell.occupiedBy = nil
      end
    end
  end
  worker.assignedLevelId = nil
  worker.targetCellId = nil
  worker.positionCellId = nil
  worker.assignmentMode = nil
end

function M.assign_worker(state, workerId, levelId, targetCellId, positionCellId, assignmentMode)
  local worker = state.workers[workerId]
  if not worker then
    return nil, err("worker_not_found", "Worker not found: " .. tostring(workerId))
  end
  if worker.state ~= "idle" then
    return nil, err("worker_not_idle", "Worker is not idle")
  end
  local level = state.levels[levelId]
  if not level then
    return nil, err("level_not_found", "Level not found: " .. tostring(levelId))
  end
  local targetCell = level.cells[targetCellId]
  local positionCell = level.cells[positionCellId]
  if not targetCell or not positionCell then
    return nil, err("cell_not_found", "Target or position cell not found")
  end
  if targetCell.kind ~= "deposit" or cellsMod.is_depleted(targetCell) then
    return nil, err("target_not_mineable", "Target cell is not a workable deposit")
  end
  if not cellsMod.is_passable(positionCell) then
    return nil, err("position_not_open", "Position cell is not open")
  end
  if positionCell.accessibility ~= "reachable" then
    return nil, err("position_not_reachable", "Position cell is not reachable")
  end
  if positionCell.occupiedBy and positionCell.occupiedBy ~= workerId then
    return nil, err("position_occupied", "Position cell is occupied by another worker")
  end
  local side = side_of(targetCell, positionCell)
  if not side then
    return nil, err("position_not_adjacent", "Position cell is not adjacent to target cell")
  end
  if targetCell.assignedWorkers[side] and targetCell.assignedWorkers[side] ~= workerId then
    return nil, err("side_occupied", "Another worker already mines the target from this side")
  end

  worker.state = "working"
  worker.assignedLevelId = levelId
  worker.targetCellId = targetCellId
  worker.positionCellId = positionCellId
  worker.assignmentMode = assignmentMode or "until_completed"
  positionCell.occupiedBy = workerId
  targetCell.assignedWorkers[side] = workerId
  level.activeMiningCells[targetCellId] = true
  return worker, nil
end

--- Stands an idle worker on positionCellId without mining anything - used to
--- keep a worker visible on the map (e.g. after a merge) when the spot it's
--- taking over wasn't an active mining assignment.
function M.place_idle_worker(state, worker, levelId, positionCellId)
  local level = state.levels[levelId]
  local cell = level and level.cells[positionCellId]
  if cell and not cell.occupiedBy then
    cell.occupiedBy = worker.id
    worker.positionCellId = positionCellId
    worker.assignedLevelId = levelId
  end
end

--- Assigns a worker to mine targetCellId without a pre-chosen position: tries
--- each adjacent cell in turn and uses the first one assign_worker accepts
--- (open, reachable, not already occupied/claimed from that side).
function M.assign_worker_to_nearest_cell(state, workerId, levelId, targetCellId, assignmentMode)
  local level = state.levels[levelId]
  if not level then
    return nil, err("level_not_found", "Level not found: " .. tostring(levelId))
  end
  local targetCell = level.cells[targetCellId]
  if not targetCell then
    return nil, err("cell_not_found", "Target cell not found: " .. tostring(targetCellId))
  end

  local neighborOffsets = { { 0, -1 }, { 0, 1 }, { -1, 0 }, { 1, 0 } }
  for _, off in ipairs(neighborOffsets) do
    local positionCellId = (targetCell.x + off[1]) .. "," .. (targetCell.y + off[2])
    local worker, assignErr = M.assign_worker(state, workerId, levelId, targetCellId, positionCellId, assignmentMode)
    if not assignErr then
      return worker, nil
    end
  end
  return nil, err("no_open_adjacent_cell", "No free reachable cell adjacent to the target deposit")
end

function M.stop_worker(state, workerId)
  local worker = state.workers[workerId]
  if not worker then
    return nil, err("worker_not_found", "Worker not found: " .. tostring(workerId))
  end
  M.detach_worker(state, worker)
  worker.state = "idle"
  return worker, nil
end

function M.merge_workers(state, workerIds)
  if #workerIds ~= 2 then
    return nil, err("merge_requires_two_workers", "Merge requires exactly two worker ids")
  end
  local w1 = state.workers[workerIds[1]]
  local w2 = state.workers[workerIds[2]]
  if not w1 or not w2 or w1.id == w2.id then
    return nil, err("worker_not_found", "One or both workers not found")
  end
  if w1.state ~= "idle" or w2.state ~= "idle" then
    return nil, err("worker_not_idle", "Both workers must be idle to merge")
  end
  if w1.level ~= w2.level then
    return nil, err("worker_level_mismatch", "Workers must be the same level to merge")
  end

  -- Detach before deleting so neither worker leaves a stale occupiedBy/
  -- assignedWorkers reference behind (a no-op for already-idle workers with
  -- no map placement).
  M.detach_worker(state, w1)
  M.detach_worker(state, w2)
  state.workers[w1.id] = nil
  state.workers[w2.id] = nil
  local newLevel = w1.level + 1
  local id = next_id(state, "worker")
  local worker = {
    id = id,
    level = newLevel,
    speed = balance.worker_speed(newLevel),
    state = "idle",
    assignedLevelId = nil,
    targetCellId = nil,
    positionCellId = nil,
    assignmentMode = nil,
  }
  state.workers[id] = worker
  if newLevel > state.highestUnlockedWorkerLevel then
    state.highestUnlockedWorkerLevel = newLevel
  end
  return worker, nil
end

function M.buy_worker(state, workerLevel)
  local maxLevel = balance.max_purchasable_worker_level(state.highestUnlockedWorkerLevel)
  if workerLevel < 1 or workerLevel > maxLevel then
    return nil, err("worker_level_not_purchasable", "Worker level is not purchasable yet")
  end
  local cost = balance.worker_purchase_cost(workerLevel)
  if state.money < cost then
    return nil, err("insufficient_funds", "Not enough money to buy this worker")
  end
  state.money = state.money - cost
  local id = next_id(state, "worker")
  local worker = {
    id = id,
    level = workerLevel,
    speed = balance.worker_speed(workerLevel),
    state = "idle",
    assignedLevelId = nil,
    targetCellId = nil,
    positionCellId = nil,
    assignmentMode = nil,
  }
  state.workers[id] = worker
  if workerLevel > state.highestUnlockedWorkerLevel then
    state.highestUnlockedWorkerLevel = workerLevel
  end
  return worker, nil
end

-- Places a freshly hired/merged idle worker on the map: if a reachable
-- mineable deposit is adjacent to an open cell, put the worker there and
-- start mining it immediately; otherwise just stand them at the entrance.
function M.deploy_worker(state, worker, levelId)
  local level = state.levels[levelId]
  local start = level and level.entranceCell
  if not level or not start or worker.state ~= "idle" then
    return
  end

  local neighborOffsets = { { 0, -1 }, { 0, 1 }, { -1, 0 }, { 1, 0 } }
  local visited = { [start.x .. "," .. start.y] = true }
  local queue = { { x = start.x, y = start.y } }

  while #queue > 0 do
    local pos = table.remove(queue, 1)
    local key = pos.x .. "," .. pos.y
    local cell = level.cells[key]
    if cell and cellsMod.is_passable(cell) and cell.accessibility == "reachable" then
      if not cell.occupiedBy then
        for _, off in ipairs(neighborOffsets) do
          local nkey = (pos.x + off[1]) .. "," .. (pos.y + off[2])
          local target = level.cells[nkey]
          if target and target.kind == "deposit" and not cellsMod.is_depleted(target) then
            local _, assignErr = M.assign_worker(state, worker.id, levelId, nkey, key, "until_completed")
            if not assignErr then
              return
            end
          end
        end
      end
      for _, off in ipairs(neighborOffsets) do
        local nx, ny = pos.x + off[1], pos.y + off[2]
        local nkey = nx .. "," .. ny
        if not visited[nkey] then
          visited[nkey] = true
          queue[#queue + 1] = { x = nx, y = ny }
        end
      end
    end
  end

  local startCell = level.cells[start.x .. "," .. start.y]
  if startCell and not startCell.occupiedBy then
    startCell.occupiedBy = worker.id
    worker.positionCellId = start.x .. "," .. start.y
    worker.assignedLevelId = levelId
  end
end

return M
