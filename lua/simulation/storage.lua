local balance = require("config.balance")
local resourceConfig = require("config.resources")

local M = {}

local function next_id(state, kind)
  state.nextIds[kind] = state.nextIds[kind] + 1
  return kind .. "_" .. (state.nextIds[kind] - 1)
end

function M.storages_for_resource(state, resourceId)
  local list = {}
  for _, storage in pairs(state.storages) do
    if storage.resourceId == resourceId then
      list[#list + 1] = storage
    end
  end
  return list
end

function M.total_capacity(state, resourceId)
  local total = 0
  for _, storage in ipairs(M.storages_for_resource(state, resourceId)) do
    total = total + storage.capacity
  end
  return total
end

function M.total_stored(state, resourceId)
  local total = 0
  for _, storage in ipairs(M.storages_for_resource(state, resourceId)) do
    total = total + storage.storedAmount
  end
  return total
end

function M.available_capacity(state, resourceId)
  return M.total_capacity(state, resourceId) - M.total_stored(state, resourceId)
end

--- Deposits up to `amount` of resourceId into available storages (first-fit).
--- Returns the amount actually stored; the remainder must stay wherever it came from.
function M.deposit_resource(state, resourceId, amount)
  local remaining = amount
  for _, storage in ipairs(M.storages_for_resource(state, resourceId)) do
    if remaining <= 0 then
      break
    end
    local free = storage.capacity - storage.storedAmount
    local take = math.min(free, remaining)
    storage.storedAmount = storage.storedAmount + take
    remaining = remaining - take
  end
  return amount - remaining
end

--- Withdraws up to `amount` of resourceId from storages (first-fit). Returns amount withdrawn.
function M.withdraw_resource(state, resourceId, amount)
  local remaining = amount
  for _, storage in ipairs(M.storages_for_resource(state, resourceId)) do
    if remaining <= 0 then
      break
    end
    local take = math.min(storage.storedAmount, remaining)
    storage.storedAmount = storage.storedAmount - take
    remaining = remaining - take
  end
  return amount - remaining
end

function M.buy_storage(state, resourceId)
  local resource = resourceConfig.byId[resourceId]
  if not resource then
    return nil, { code = "unknown_resource", message = "Unknown resource: " .. tostring(resourceId) }
  end
  local cost = balance.storage_purchase_cost(resource)
  if state.money < cost then
    return nil, { code = "insufficient_funds", message = "Not enough money to buy storage" }
  end
  state.money = state.money - cost
  local id = next_id(state, "storage")
  local storage = {
    id = id,
    resourceId = resourceId,
    level = 1,
    capacity = balance.storage_capacity(resource, 1),
    storedAmount = 0,
  }
  state.storages[id] = storage
  return storage, nil
end

function M.upgrade_storage(state, storageId)
  local storage = state.storages[storageId]
  if not storage then
    return nil, { code = "storage_not_found", message = "Storage not found: " .. tostring(storageId) }
  end
  local cost = balance.storage_upgrade_cost(storage)
  if state.money < cost then
    return nil, { code = "insufficient_funds", message = "Not enough money to upgrade storage" }
  end
  local resource = resourceConfig.byId[storage.resourceId]
  state.money = state.money - cost
  storage.level = storage.level + 1
  storage.capacity = balance.storage_capacity(resource, storage.level)
  return storage, nil
end

return M
