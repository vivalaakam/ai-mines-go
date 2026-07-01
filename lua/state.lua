local genLevel = require("generation.level")
local defaultRules = require("config.rules")
local reachabilityMod = require("simulation.reachability")
local visibilityMod = require("simulation.visibility")
local ordersMod = require("simulation.orders")

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
function M.load_state(state)
  return deep_copy(state)
end

return M
