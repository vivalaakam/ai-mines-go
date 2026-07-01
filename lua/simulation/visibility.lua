local chunksMod = require("generation.chunks")

local M = {}
M.RADIUS = 5

--- Marks every cell within RADIUS of a reachable/open cell as scouted, generating
--- neighboring chunks as the radius crosses chunk borders (REQUIREMENTS.md §11).
function M.recompute(level)
  local reachable = {}
  for _, cell in pairs(level.cells) do
    if cell.accessibility == "reachable" then
      reachable[#reachable + 1] = cell
    end
  end

  local radiusSq = M.RADIUS * M.RADIUS
  for _, rc in ipairs(reachable) do
    for dx = -M.RADIUS, M.RADIUS do
      for dy = -M.RADIUS, M.RADIUS do
        if dx * dx + dy * dy <= radiusSq then
          local x, y = rc.x + dx, rc.y + dy
          local cx, cy = chunksMod.global_to_chunk(x, y)
          chunksMod.ensure_chunk(level, cx, cy)
          local cell = level.cells[chunksMod.cell_key(x, y)]
          if cell then
            cell.visibility = "scouted"
          end
        end
      end
    end
  end
end

return M
