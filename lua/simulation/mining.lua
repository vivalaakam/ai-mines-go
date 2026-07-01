local storageMod = require("simulation.storage")
local cellsMod = require("generation.cells")

local M = {}

--- Processes one tick of mining for every actively-worked cell in a level.
--- ponytail: only iterates level.activeMiningCells (maintained by workers.lua)
--- instead of scanning every generated cell every tick.
function M.process_tick(state, level)
  local events = {}
  for cellKey in pairs(level.activeMiningCells) do
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
        for side, workerId in pairs(cell.assignedWorkers) do
          local worker = state.workers[workerId]
          if worker then
            worker.state = "idle"
          end
          cell.assignedWorkers[side] = nil
        end
        level.activeMiningCells[cellKey] = nil
        events[#events + 1] = { type = "cell_depleted", cellId = cellKey, levelId = level.id }
      end
    end
  end
  return events
end

return M
