local chunksMod = require("generation.chunks")
local validationMod = require("generation.validation")

local M = {}

--- Constructs and eagerly generates a fresh level's initial 5x5 chunk area.
--- Shared by state.lua (first level) and simulation/levels.lua (create_next_level)
--- so both paths produce an identically-shaped level table.
function M.new_level(id, depth, seedPhrase, generatorVersion)
  local level = {
    id = id,
    depth = depth,
    seedPhrase = seedPhrase,
    generatorVersion = generatorVersion,
    cells = {},
    generatedChunks = {},
    bounds = nil,
    activeMiningCells = {},
    entranceCell = nil,
    stairsCell = nil,
    stairsChunk = nil,
    stairsReachable = false,
    nextLevelId = nil,
  }
  chunksMod.generate_initial_area(level)

  local validation = validationMod.validate(level)
  assert(validation.hasPathFromEntranceToExit, "generator invariant violated: no path from entrance to stairs")

  return level
end

return M
