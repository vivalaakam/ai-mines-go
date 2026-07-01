-- ponytail: exact numeric formulas are an open design decision (REQUIREMENTS.md §43.3).
-- These are simple placeholder curves, tunable later without touching engine code.
local M = {}

function M.worker_speed(level)
  return level * 1.0
end

function M.worker_purchase_cost(level)
  return 50 * level * level
end

function M.storage_purchase_cost(resourceConfig)
  return 40 + resourceConfig.basePrice * 5
end

function M.storage_upgrade_cost(storage)
  return 30 * (storage.level + 1) * (storage.level + 1)
end

function M.storage_capacity(resourceConfig, level)
  return resourceConfig.storageBaseCapacity * level
end

function M.max_purchasable_worker_level(highestUnlockedWorkerLevel)
  return math.max(1, highestUnlockedWorkerLevel - 2)
end

return M
