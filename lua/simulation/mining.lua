local storageMod = require("simulation.storage")
local cellsMod = require("generation.cells")
local workersMod = require("simulation.workers")

local M = {}

local neighborOffsets = { { 0, -1 }, { 0, 1 }, { -1, 0 }, { 1, 0 } }

--- After a worker's deposit depletes, keep it working by assigning it to a
--- still-workable deposit on one of the four cells adjacent to where it
--- already stands, without moving it. Leaves the worker idle if none of its
--- neighbors have anything left to mine.
local function try_auto_continue(state, level, worker)
  local position = worker.positionCellId and level.cells[worker.positionCellId]
  if not position then
    return
  end
  for _, off in ipairs(neighborOffsets) do
    local targetCellId = (position.x + off[1]) .. "," .. (position.y + off[2])
    local targetCell = level.cells[targetCellId]
    if targetCell and targetCell.kind == "deposit" and not cellsMod.is_depleted(targetCell) then
      local assigned =
        workersMod.assign_worker(state, worker.id, level.id, targetCellId, worker.positionCellId, worker.assignmentMode)
      if assigned then
        return
      end
    end
  end
end

--- Processes one tick of mining for every actively-worked cell in a level.
--- ponytail: only iterates level.activeMiningCells (maintained by workers.lua)
--- instead of scanning every generated cell every tick.
function M.process_tick(state, level)
  local events = {}
  -- Snapshot the keys before iterating: try_auto_continue below can add a
  -- newly-assigned cell to level.activeMiningCells mid-loop, and mutating a
  -- table while pairs() is iterating it is undefined behavior in Lua.
  local cellKeys = {}
  for cellKey in pairs(level.activeMiningCells) do
    cellKeys[#cellKeys + 1] = cellKey
  end
  for _, cellKey in ipairs(cellKeys) do
    local cell = level.cells[cellKey]
    if not cell or cell.kind ~= "deposit" or next(cell.assignedWorkers) == nil then
      level.activeMiningCells[cellKey] = nil
    else
      local totalSpeed = 0
      local workerIds = {}
      for _, workerId in pairs(cell.assignedWorkers) do
        local worker = state.workers[workerId]
        if worker then
          totalSpeed = totalSpeed + worker.speed
          workerIds[#workerIds + 1] = workerId
        end
      end

      local processable = {}
      local processableRatioSum = 0
      local hasRemaining = false
      for _, comp in ipairs(cell.components) do
        if comp.remainingAmount > 0 then
          hasRemaining = true
          local canProcess = comp.type ~= "resource" or storageMod.available_capacity(state, comp.resourceId) > 0
          if canProcess then
            processable[#processable + 1] = comp
            processableRatioSum = processableRatioSum + comp.ratio
          end
        end
      end

      if #processable == 0 then
        if hasRemaining then
          for _, workerId in ipairs(workerIds) do
            state.workers[workerId].state = "blocked_by_storage"
          end
        end
      else
        for _, workerId in ipairs(workerIds) do
          if state.workers[workerId].state == "blocked_by_storage" then
            state.workers[workerId].state = "working"
          end
        end
        for _, comp in ipairs(processable) do
          local share = totalSpeed * (comp.ratio / processableRatioSum)
          local amount = math.min(share, comp.remainingAmount)
          if comp.type == "resource" then
            local stored = storageMod.deposit_resource(state, comp.resourceId, amount)
            comp.remainingAmount = comp.remainingAmount - stored
          else
            comp.remainingAmount = comp.remainingAmount - amount
          end
        end
      end

      if cellsMod.is_depleted(cell) then
        cell.kind = "empty"
        cell.components = {}
        local vacatedWorkerIds = {}
        for side, workerId in pairs(cell.assignedWorkers) do
          local worker = state.workers[workerId]
          if worker then
            worker.state = "idle"
            worker.targetCellId = nil
            vacatedWorkerIds[#vacatedWorkerIds + 1] = workerId
          end
          cell.assignedWorkers[side] = nil
        end
        level.activeMiningCells[cellKey] = nil
        events[#events + 1] = { type = "cell_depleted", cellId = cellKey, levelId = level.id }

        for _, workerId in ipairs(vacatedWorkerIds) do
          local worker = state.workers[workerId]
          if worker and worker.state == "idle" then
            try_auto_continue(state, level, worker)
          end
        end
      end
    end
  end
  return events
end

return M
