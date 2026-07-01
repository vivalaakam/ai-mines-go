local genResources = require("generation.resources")

local M = {}

function M.new_empty(x, y, kind)
  return {
    x = x,
    y = y,
    kind = kind or "empty",
    visibility = "unknown",
    accessibility = "unreachable",
    components = {},
    occupiedBy = nil, -- workerId standing here
    assignedWorkers = {}, -- side -> workerId, workers mining this cell as target
  }
end

function M.new_obstacle(x, y)
  local cell = M.new_empty(x, y, "obstacle")
  return cell
end

--- Builds a deposit cell with a rock + resource mix appropriate for the given depth.
function M.new_deposit(x, y, depth, rng)
  local cell = M.new_empty(x, y, "deposit")
  local totalUnits = 100

  local rockRatio = 0.5 + rng:next() * 0.2 -- 50-70% rock
  local remainingRatio = 1 - rockRatio

  local resourceId = genResources.pick_resource(rng, depth)
  local components = {
    { type = "rock", resourceId = nil, ratio = rockRatio, initialAmount = totalUnits * rockRatio, remainingAmount = totalUnits * rockRatio },
  }

  if resourceId then
    components[#components + 1] = {
      type = "resource",
      resourceId = resourceId,
      ratio = remainingRatio,
      initialAmount = totalUnits * remainingRatio,
      remainingAmount = totalUnits * remainingRatio,
    }
  else
    -- No resource unlocked yet at this depth: the remainder is just more rock.
    components[1].ratio = 1
    components[1].initialAmount = totalUnits
    components[1].remainingAmount = totalUnits
  end

  cell.components = components
  return cell
end

function M.is_passable(cell)
  return cell.kind == "empty" or cell.kind == "stairs_area"
end

function M.is_depleted(cell)
  if cell.kind ~= "deposit" then
    return false
  end
  for _, c in ipairs(cell.components) do
    if c.remainingAmount > 0 then
      return false
    end
  end
  return true
end

return M
