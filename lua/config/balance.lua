-- ponytail: exact numeric formulas are an open design decision (REQUIREMENTS.md §43.3).
-- These are simple placeholder curves, tunable later without touching engine code.
local M = {}

function M.worker_speed(level)
  return level * 1.0
end

function M.worker_purchase_cost(level)
  return 50 * level * level
end

function M.max_purchasable_worker_level(highestUnlockedWorkerLevel)
  return math.max(1, highestUnlockedWorkerLevel - 2)
end

--- Per-unit order price, rolled from the resource's base price range.
function M.order_price_per_unit(rng, basePrice)
  return rng:next_int(basePrice, basePrice * 3)
end

return M
