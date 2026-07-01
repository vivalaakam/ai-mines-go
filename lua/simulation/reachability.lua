local chunksMod = require("generation.chunks")
local cellsMod = require("generation.cells")

local M = {}

--- Full BFS recompute of accessibility from the level entrance over passable cells.
--- Lazily generates neighboring chunks whenever the flood fill reaches their border
--- (REQUIREMENTS.md §11). Re-running this after mining events lets newly emptied
--- deposit cells merge into the reachable area, per the "connected empty area
--- becomes reachable" rule.
function M.recompute(level)
  for _, cell in pairs(level.cells) do
    cell.accessibility = "unreachable"
  end

  local start = level.entranceCell
  local startKey = chunksMod.cell_key(start.x, start.y)
  local visited = { [startKey] = true }
  local queue = { { x = start.x, y = start.y } }
  local head = 1

  while head <= #queue do
    local node = queue[head]
    head = head + 1
    local cx, cy = chunksMod.global_to_chunk(node.x, node.y)
    chunksMod.ensure_chunk(level, cx, cy)
    local cell = level.cells[chunksMod.cell_key(node.x, node.y)]
    if cell and cellsMod.is_passable(cell) then
      cell.accessibility = "reachable"
      local neighbors = { { node.x + 1, node.y }, { node.x - 1, node.y }, { node.x, node.y + 1 }, { node.x, node.y - 1 } }
      for _, n in ipairs(neighbors) do
        local nx, ny = n[1], n[2]
        local nkey = chunksMod.cell_key(nx, ny)
        if not visited[nkey] then
          visited[nkey] = true
          local ncx, ncy = chunksMod.global_to_chunk(nx, ny)
          chunksMod.ensure_chunk(level, ncx, ncy)
          local ncell = level.cells[nkey]
          if ncell and cellsMod.is_passable(ncell) then
            queue[#queue + 1] = { x = nx, y = ny }
          end
        end
      end
    end
  end

  level.stairsReachable = false
  local stairsCell = level.cells[chunksMod.cell_key(level.stairsCell.x, level.stairsCell.y)]
  if stairsCell and stairsCell.accessibility == "reachable" then
    level.stairsReachable = true
  end
end

return M
