local genLevel = require("generation.level")
local reachabilityMod = require("simulation.reachability")
local visibilityMod = require("simulation.visibility")

local M = {}

local function err(code, message)
  return { code = code, message = message, details = {} }
end

local function next_id(state, kind)
  state.nextIds[kind] = state.nextIds[kind] + 1
  return kind .. "_" .. (state.nextIds[kind] - 1)
end

function M.create_next_level(state, fromLevelId)
  local fromLevel = state.levels[fromLevelId]
  if not fromLevel then
    return nil, err("level_not_found", "Level not found: " .. tostring(fromLevelId))
  end
  if not fromLevel.stairsReachable then
    return nil, err("stairs_not_reachable", "Stairs area is not reachable yet")
  end
  if fromLevel.nextLevelId then
    return state.levels[fromLevel.nextLevelId], nil
  end

  local id = next_id(state, "level")
  local level = genLevel.new_level(id, fromLevel.depth + 1, state.seedPhrase, state.generatorVersion)
  state.levels[id] = level
  fromLevel.nextLevelId = id

  reachabilityMod.recompute(level)
  visibilityMod.recompute(level)

  return level, nil
end

return M
