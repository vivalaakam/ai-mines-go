local tickMod = require("simulation.tick")
local workersMod = require("simulation.workers")
local ordersMod = require("simulation.orders")
local levelsMod = require("simulation.levels")

local M = {}

local function ok(result)
  result = result or {}
  return {
    ok = true,
    events = result.events or {},
    patch = result.patch or {},
    processedTicks = result.processedTicks,
    remainingTicks = result.remainingTicks,
    data = result.data,
  }
end

local function fail(errObj)
  return { ok = false, error = errObj }
end

local handlers = {}

handlers["tick"] = function(state, cmd)
  local result, err = tickMod.advance(state, cmd.ticksPassed or 1)
  if err then
    return fail(err)
  end
  return ok(result)
end

handlers["buy_worker"] = function(state, cmd)
  local worker, err = workersMod.buy_worker(state, cmd.workerLevel)
  if err then
    return fail(err)
  end
  if cmd.levelId then
    workersMod.deploy_worker(state, worker, cmd.levelId)
  end
  return ok({ data = worker })
end

-- Merging is usually triggered by dragging one worker onto another, and
-- dragged workers are commonly still busy mining - stop each one first (idle
-- is a merge precondition) and, if the merge itself is then rejected (e.g.
-- mismatched levels), resend each worker back to whatever it was doing
-- before the drag interrupted it, instead of leaving it stranded idle.
handlers["merge_workers"] = function(state, cmd)
  local workerIds = cmd.workerIds or {}
  local previousAssignments = {}
  for _, workerId in ipairs(workerIds) do
    local worker = state.workers[workerId]
    if worker and worker.state ~= "idle" then
      previousAssignments[workerId] = {
        levelId = worker.assignedLevelId,
        targetCellId = worker.targetCellId,
        positionCellId = worker.positionCellId,
        assignmentMode = worker.assignmentMode,
      }
      workersMod.stop_worker(state, workerId)
    end
  end

  local worker, err = workersMod.merge_workers(state, workerIds)
  if err then
    for workerId, prev in pairs(previousAssignments) do
      if prev.levelId and prev.targetCellId and prev.positionCellId then
        workersMod.assign_worker(
          state,
          workerId,
          prev.levelId,
          prev.targetCellId,
          prev.positionCellId,
          prev.assignmentMode
        )
      end
    end
    return fail(err)
  end
  return ok({ data = worker })
end

handlers["assign_worker_to_target_cell"] = function(state, cmd)
  local worker, err =
    workersMod.assign_worker(state, cmd.workerId, cmd.levelId, cmd.targetCellId, cmd.positionCellId, cmd.assignmentMode)
  if err then
    return fail(err)
  end
  return ok({ data = worker })
end

handlers["stop_worker"] = function(state, cmd)
  local worker, err = workersMod.stop_worker(state, cmd.workerId)
  if err then
    return fail(err)
  end
  return ok({ data = worker })
end

-- Drag-and-drop assignment target: only a deposit cell id is known (the drop
-- point), not a specific adjacent position, so this stops the worker first
-- (if it was busy elsewhere) and lets Lua pick the nearest free reachable
-- neighbor cell to mine from.
handlers["assign_worker_to_deposit"] = function(state, cmd)
  local worker = state.workers[cmd.workerId]
  if not worker then
    return fail({ code = "worker_not_found", message = "Worker not found: " .. tostring(cmd.workerId), details = {} })
  end
  if worker.state ~= "idle" then
    workersMod.stop_worker(state, cmd.workerId)
  end
  local assigned, err =
    workersMod.assign_worker_to_nearest_cell(state, cmd.workerId, cmd.levelId, cmd.targetCellId, cmd.assignmentMode)
  if err then
    return fail(err)
  end
  return ok({ data = assigned })
end

handlers["accept_order"] = function(state, cmd)
  local order, err = ordersMod.accept_order(state, cmd.orderId)
  if err then
    return fail(err)
  end
  return ok({ data = order })
end

handlers["decline_order"] = function(state, cmd)
  local order, err = ordersMod.decline_order(state, cmd.orderId)
  if err then
    return fail(err)
  end
  return ok({ data = order })
end

handlers["set_order_priority"] = function(state, cmd)
  local order, err = ordersMod.set_order_priority(state, cmd.orderId, cmd.priority)
  if err then
    return fail(err)
  end
  return ok({ data = order })
end

handlers["complete_order_immediately"] = function(state, cmd)
  local order, err = ordersMod.complete_order_immediately(state, cmd.orderId)
  if err then
    return fail(err)
  end
  return ok({ data = order })
end

handlers["create_next_level"] = function(state, cmd)
  local level, err = levelsMod.create_next_level(state, cmd.fromLevelId)
  if err then
    return fail(err)
  end
  return ok({ data = { levelId = level.id, depth = level.depth } })
end

--- Routes a command table to its handler. Never raises past this boundary:
--- an internal Lua error becomes a structured `internal_error` result instead
--- of propagating as a pcall failure the Go host would have to special-case.
function M.dispatch(state, command)
  if type(command) ~= "table" or type(command.type) ~= "string" then
    return fail({
      code = "invalid_command",
      message = "Command must be a table with a string 'type' field",
      details = {},
    })
  end
  local handler = handlers[command.type]
  if not handler then
    return fail({ code = "unknown_command", message = "Unknown command type: " .. tostring(command.type), details = {} })
  end
  local success, result = pcall(handler, state, command)
  if not success then
    return fail({ code = "internal_error", message = tostring(result), details = {} })
  end
  return result
end

return M
