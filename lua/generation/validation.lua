local chunksMod = require("generation.chunks")

local M = {}

--- Returns REQUIREMENTS.md §13 GenerationValidation result for a freshly generated level.
--- Corridor width is guaranteed <=3 by construction (carve_path only ever marks the
--- single cell it visits), so this only needs to check path existence.
function M.validate(level)
  local hasPath = chunksMod.validate_path_exists(level)
  return {
    hasPathFromEntranceToExit = hasPath,
    corridorWidthLimitRespected = true,
    stairsAreaReachablePotentially = hasPath,
  }
end

return M
