local shiftsMod = require("simulation.shifts")
local storageMod = require("simulation.storage")
local resourceConfig = require("config.resources")

local M = {}

local function err(code, message)
  return { code = code, message = message, details = {} }
end

local handlers = {}

handlers["get_game_phase"] = function(state)
  return { phase = state.phase }
end

handlers["get_game_time"] = function(state)
  return {
    tick = state.gameTime.tick,
    shiftIndex = state.gameTime.shiftIndex,
    ticksIntoShift = state.gameTime.tick % shiftsMod.SHIFT_LENGTH,
  }
end

handlers["get_level_view"] = function(state, query)
  local level = state.levels[query.levelId]
  if not level then
    return nil, err("level_not_found", "Level not found: " .. tostring(query.levelId))
  end
  local viewport = query.viewport or { x = 0, y = 0, width = 40, height = 30 }

  local cellsOut = {}
  for x = viewport.x, viewport.x + viewport.width - 1 do
    for y = viewport.y, viewport.y + viewport.height - 1 do
      local cell = level.cells[x .. "," .. y]
      if cell then
        if cell.visibility == "scouted" then
          cellsOut[#cellsOut + 1] = {
            x = cell.x,
            y = cell.y,
            kind = cell.kind,
            visibility = cell.visibility,
            accessibility = cell.accessibility,
            components = cell.components,
            occupiedBy = cell.occupiedBy,
          }
        else
          cellsOut[#cellsOut + 1] = {
            x = cell.x,
            y = cell.y,
            kind = "unknown",
            visibility = "unknown",
            accessibility = "unreachable",
            components = {},
            occupiedBy = nil,
          }
        end
      end
    end
  end

  local workersOut = {}
  for _, worker in pairs(state.workers) do
    if worker.assignedLevelId == query.levelId then
      workersOut[#workersOut + 1] = worker
    end
  end

  return {
    levelId = query.levelId,
    depth = level.depth,
    stairsReachable = level.stairsReachable,
    nextLevelId = level.nextLevelId,
    cells = cellsOut,
    workers = workersOut,
  }
end

handlers["get_workers"] = function(state)
  local list = {}
  for _, worker in pairs(state.workers) do
    list[#list + 1] = worker
  end
  return { workers = list, highestUnlockedWorkerLevel = state.highestUnlockedWorkerLevel }
end

handlers["get_storage_state"] = function(state)
  local list = {}
  for _, storage in pairs(state.storages) do
    list[#list + 1] = storage
  end
  return { storages = list }
end

handlers["get_available_orders"] = function(state)
  local list = {}
  for _, order in pairs(state.orders) do
    if order.state == "available" then
      list[#list + 1] = order
    end
  end
  return { orders = list }
end

handlers["get_active_orders"] = function(state)
  local list = {}
  for _, order in pairs(state.orders) do
    if order.state == "accepted" then
      list[#list + 1] = order
    end
  end
  return { orders = list }
end

handlers["get_resources"] = function(state)
  local list = {}
  for _, resource in ipairs(resourceConfig.list) do
    list[#list + 1] = {
      id = resource.id,
      name = resource.name,
      rarity = resource.rarity,
      unlockDepth = resource.unlockDepth,
      basePrice = resource.basePrice,
      totalStored = storageMod.total_stored(state, resource.id),
      totalCapacity = storageMod.total_capacity(state, resource.id),
    }
  end
  return { resources = list }
end

handlers["get_player_summary"] = function(state)
  local workerCount = 0
  for _ in pairs(state.workers) do
    workerCount = workerCount + 1
  end
  return {
    money = state.money,
    phase = state.phase,
    highestUnlockedWorkerLevel = state.highestUnlockedWorkerLevel,
    workerCount = workerCount,
    gameTime = { tick = state.gameTime.tick, shiftIndex = state.gameTime.shiftIndex },
  }
end

handlers["get_shift_summary"] = function(state)
  local ticksIntoShift = state.gameTime.tick % shiftsMod.SHIFT_LENGTH
  return {
    shiftIndex = state.gameTime.shiftIndex,
    phase = state.phase,
    ticksIntoShift = ticksIntoShift,
    ticksRemainingInShift = shiftsMod.SHIFT_LENGTH - ticksIntoShift,
    shiftLength = shiftsMod.SHIFT_LENGTH,
  }
end

function M.dispatch(state, query)
  if type(query) ~= "table" or type(query.type) ~= "string" then
    return nil, err("invalid_query", "Query must be a table with a string 'type' field")
  end
  local handler = handlers[query.type]
  if not handler then
    return nil, err("unknown_query", "Unknown query type: " .. tostring(query.type))
  end
  local success, result, handlerErr = pcall(handler, state, query)
  if not success then
    return nil, err("internal_error", tostring(result))
  end
  if handlerErr then
    return nil, handlerErr
  end
  return result, nil
end

return M
