local genLevel = require("generation.level")
local defaultRules = require("config.rules")
local reachabilityMod = require("simulation.reachability")
local visibilityMod = require("simulation.visibility")
local ordersMod = require("simulation.orders")
local resourceConfig = require("config.resources")

local M = {}
M.SCHEMA_VERSION = 1
M.GENERATOR_VERSION = 1

local function deep_copy(value)
  if type(value) ~= "table" then
    return value
  end
  local copy = {}
  for k, v in pairs(value) do
    copy[deep_copy(k)] = deep_copy(v)
  end
  return copy
end
M.deep_copy = deep_copy

--- Creates a brand-new game state with a freshly generated level 1.
function M.new_game(seedPhrase)
  local state = {
    schemaVersion = M.SCHEMA_VERSION,
    generatorVersion = M.GENERATOR_VERSION,
    seedPhrase = seedPhrase,
    gameTime = { tick = 0 },
    rulesConfig = deep_copy(defaultRules),
    money = 100,
    highestUnlockedWorkerLevel = 1,
    levels = {},
    workers = {},
    storages = {},
    orders = {},
    nextIds = { worker = 1, storage = 1, order = 1, level = 1 },
  }

  -- Every resource gets an uncapped storage from the start (no storage
  -- purchases/limits - REQUIREMENTS.md storage model is intentionally
  -- simplified here per product decision).
  for _, resource in ipairs(resourceConfig.list) do
    state.storages[resource.id] = {
      id = resource.id,
      resourceId = resource.id,
      level = 1,
      capacity = math.huge,
      storedAmount = 0,
    }
  end

  local level = genLevel.new_level("level_1", 1, seedPhrase, M.GENERATOR_VERSION)
  state.levels[level.id] = level
  reachabilityMod.recompute(level)
  visibilityMod.recompute(level)
  ordersMod.replenish(state)

  return state
end

--- Read-only serializable snapshot for the persistence adapter.
function M.export_state(state)
  return deep_copy(state)
end

--- Restores an engine's state from a persistence-adapter-provided snapshot.
--- ponytail: backfills uncapped storages for any resource missing one, plus
--- newer rulesConfig keys and per-requirement order prices, so saves written
--- before those fields existed still work.
function M.load_state(state)
  local copy = deep_copy(state)
  copy.storages = copy.storages or {}
  for _, resource in ipairs(resourceConfig.list) do
    if not copy.storages[resource.id] then
      copy.storages[resource.id] = {
        id = resource.id,
        resourceId = resource.id,
        level = 1,
        capacity = math.huge,
        storedAmount = 0,
      }
    end
  end

  copy.rulesConfig = copy.rulesConfig or {}
  for key, value in pairs(defaultRules) do
    if copy.rulesConfig[key] == nil then
      copy.rulesConfig[key] = deep_copy(value)
    end
  end

  local basePriceById = {}
  for _, resource in ipairs(resourceConfig.list) do
    basePriceById[resource.id] = resource.basePrice
  end
  for _, order in pairs(copy.orders or {}) do
    for _, req in ipairs(order.requirements or {}) do
      if not req.pricePerUnit or req.pricePerUnit <= 0 then
        req.pricePerUnit = basePriceById[req.resourceId] or 1
      end
    end
  end

  return copy
end

return M
