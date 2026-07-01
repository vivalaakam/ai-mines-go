local M = {}

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

return M
