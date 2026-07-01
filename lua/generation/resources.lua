local resourceConfig = require("config.resources")

local M = {}

--- Weight of a resource at a given depth (0 if not yet available at that depth).
local function weight_at_depth(resource, depth)
  local weight = 0
  for _, entry in ipairs(resource.generationWeightByDepth) do
    if depth >= entry.fromDepth then
      weight = entry.weight
    end
  end
  return weight
end

--- Resources available (weight > 0) at the given depth.
function M.available_at_depth(depth)
  local available = {}
  for _, r in ipairs(resourceConfig.list) do
    local w = weight_at_depth(r, depth)
    if w > 0 then
      available[#available + 1] = { resource = r, weight = w }
    end
  end
  return available
end

--- Resources newly unlocked exactly at this depth (min unlockDepth == depth).
function M.unlocked_at_depth(depth)
  local unlocked = {}
  for _, r in ipairs(resourceConfig.list) do
    if r.unlockDepth == depth then
      unlocked[#unlocked + 1] = r
    end
  end
  return unlocked
end

--- Picks one resource id available at the given depth using rng, or nil if none available.
function M.pick_resource(rng, depth)
  local available = M.available_at_depth(depth)
  if #available == 0 then
    return nil
  end
  local weights = {}
  for i, entry in ipairs(available) do
    weights[i] = entry.weight
  end
  local idx = rng:weighted_index(weights)
  if not idx then
    return nil
  end
  return available[idx].resource.id
end

return M
