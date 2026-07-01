local RNG = require("generation.rng")
local cellsMod = require("generation.cells")

local M = {}

M.CHUNK_SIZE = 32
M.AREA_RADIUS = 2 -- 5x5 chunks centered on (0,0)
M.OBSTACLE_CHANCE = 0.12

function M.chunk_key(cx, cy)
  return cx .. "," .. cy
end

function M.cell_key(x, y)
  return x .. "," .. y
end

function M.global_to_chunk(x, y)
  return math.floor(x / M.CHUNK_SIZE), math.floor(y / M.CHUNK_SIZE)
end

local function chunk_seed(seedPhrase, depth, generatorVersion, cx, cy)
  return RNG.hash_seed(seedPhrase, depth, generatorVersion, cx, cy)
end

--- Fills raw obstacle/deposit content for one chunk (no path carving yet).
local function fill_chunk(level, cx, cy)
  local rng = RNG.new(chunk_seed(level.seedPhrase, level.depth, level.generatorVersion, cx, cy))
  local ox, oy = cx * M.CHUNK_SIZE, cy * M.CHUNK_SIZE
  for lx = 0, M.CHUNK_SIZE - 1 do
    for ly = 0, M.CHUNK_SIZE - 1 do
      local x, y = ox + lx, oy + ly
      local key = M.cell_key(x, y)
      if not level.cells[key] then
        local cell
        if rng:next() < M.OBSTACLE_CHANCE then
          cell = cellsMod.new_obstacle(x, y)
        else
          cell = cellsMod.new_deposit(x, y, level.depth, rng)
        end
        level.cells[key] = cell
      end
    end
  end
  level.generatedChunks[M.chunk_key(cx, cy)] = true
end

--- Forces a square area (size x size, centered on cx,cy) to a passable kind.
local function carve_area(level, centerX, centerY, size, kind)
  local half = math.floor(size / 2)
  for x = centerX - half, centerX + half do
    for y = centerY - half, centerY + half do
      local key = M.cell_key(x, y)
      local cell = level.cells[key]
      if cell then
        cell.kind = kind
        cell.components = {}
      end
    end
  end
end

--- Ensures a cell is diggable (turns obstacles into deposits) without opening it -
--- the player must still mine through, per REQUIREMENTS.md §9 ("к спуску нужно пробуриться").
local function ensure_diggable(level, rng, x, y)
  local key = M.cell_key(x, y)
  local cell = level.cells[key]
  if cell and cell.kind == "obstacle" then
    level.cells[key] = cellsMod.new_deposit(x, y, level.depth, rng)
  end
end

--- Biased random walk from (x1,y1) to (x2,y2) that guarantees every cell it visits
--- is diggable (REQUIREMENTS.md §13: "potential path"), never exceeding corridor
--- width 3 since only the single visited cell per step is touched.
local function carve_path(level, rng, x1, y1, x2, y2)
  local x, y = x1, y1
  local guard = 0
  local maxSteps = (math.abs(x2 - x1) + math.abs(y2 - y1)) * 6 + 50
  while (x ~= x2 or y ~= y2) and guard < maxSteps do
    guard = guard + 1
    ensure_diggable(level, rng, x, y)

    local dx, dy = 0, 0
    if rng:next() < 0.75 then
      -- move toward target
      if x ~= x2 and (y == y2 or rng:next() < 0.5) then
        dx = x < x2 and 1 or -1
      else
        dy = y < y2 and 1 or -1
      end
    else
      -- random cardinal wiggle
      local dir = rng:next_int(1, 4)
      if dir == 1 then
        dx = 1
      elseif dir == 2 then
        dx = -1
      elseif dir == 3 then
        dy = 1
      else
        dy = -1
      end
    end
    x, y = x + dx, y + dy
  end
  ensure_diggable(level, rng, x2, y2)
end

--- Generates the whole 5x5 starting chunk area eagerly (REQUIREMENTS.md §8/§9).
function M.generate_initial_area(level)
  local half = M.AREA_RADIUS
  for cx = -half, half do
    for cy = -half, half do
      fill_chunk(level, cx, cy)
    end
  end

  local rng = RNG.new(chunk_seed(level.seedPhrase, level.depth, level.generatorVersion, "stairs-pick", 0))
  local candidates = {}
  for cx = -half, half do
    for cy = -half, half do
      if not (cx == 0 and cy == 0) then
        candidates[#candidates + 1] = { cx = cx, cy = cy }
      end
    end
  end
  local stairsChunk = candidates[rng:next_int(1, #candidates)]

  local half32 = math.floor(M.CHUNK_SIZE / 2)
  local entranceCenter = { x = half32, y = half32 }
  local stairsCenter = { x = stairsChunk.cx * M.CHUNK_SIZE + half32, y = stairsChunk.cy * M.CHUNK_SIZE + half32 }

  carve_area(level, entranceCenter.x, entranceCenter.y, 3, "empty")
  carve_path(level, rng, entranceCenter.x, entranceCenter.y, stairsCenter.x, stairsCenter.y)
  carve_area(level, stairsCenter.x, stairsCenter.y, 3, "stairs_area")

  level.entranceCell = entranceCenter
  level.stairsCell = stairsCenter
  level.stairsChunk = { cx = stairsChunk.cx, cy = stairsChunk.cy }
end

--- Lazily generates a single chunk outside the initial area, carving a straight-ish
--- corridor from its west border to its east border so newly explored chunks stay
--- connectable. ponytail: heuristic, not a global connectivity solver.
function M.ensure_chunk(level, cx, cy)
  local key = M.chunk_key(cx, cy)
  if level.generatedChunks[key] then
    return
  end
  fill_chunk(level, cx, cy)
  local rng = RNG.new(chunk_seed(level.seedPhrase, level.depth, level.generatorVersion, "corridor", cx * 100000 + cy))
  local ox, oy = cx * M.CHUNK_SIZE, cy * M.CHUNK_SIZE
  local midY = oy + rng:next_int(4, M.CHUNK_SIZE - 5)
  carve_path(level, rng, ox, midY, ox + M.CHUNK_SIZE - 1, midY + rng:next_int(-3, 3))
end

--- BFS-based check that a potential path of passable-or-carveable cells exists.
--- Used as a post-generation self-check (REQUIREMENTS.md §13 validation).
function M.validate_path_exists(level)
  local start = level.entranceCell
  local goal = level.stairsCell
  local startKey = M.cell_key(start.x, start.y)
  local visited = { [startKey] = true }
  local queue = { { x = start.x, y = start.y } }
  local head = 1
  while head <= #queue do
    local node = queue[head]
    head = head + 1
    if node.x == goal.x and node.y == goal.y then
      return true
    end
    local neighbors = { { node.x + 1, node.y }, { node.x - 1, node.y }, { node.x, node.y + 1 }, { node.x, node.y - 1 } }
    for _, n in ipairs(neighbors) do
      local nx, ny = n[1], n[2]
      local nkey = M.cell_key(nx, ny)
      if not visited[nkey] then
        local cell = level.cells[nkey]
        if cell and cell.kind ~= "obstacle" then
          visited[nkey] = true
          queue[#queue + 1] = { x = nx, y = ny }
        end
      end
    end
  end
  return false
end

return M
