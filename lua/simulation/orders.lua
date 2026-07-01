local RNG = require("generation.rng")
local storageMod = require("simulation.storage")
local genResources = require("generation.resources")

local M = {}
M.MAX_AVAILABLE_ORDERS = 3
M.ORDER_LIFETIME_TICKS = 1500

local function err(code, message)
  return { code = code, message = message, details = {} }
end

local function next_id(state, kind)
  state.nextIds[kind] = state.nextIds[kind] + 1
  return kind .. "_" .. (state.nextIds[kind] - 1)
end

local function current_max_depth(state)
  local max = 1
  for _, level in pairs(state.levels) do
    if level.depth > max then
      max = level.depth
    end
  end
  return max
end

--- Fully deterministic: seeded from seedPhrase + the order's own sequential id.
local function generate_order(state)
  local id = next_id(state, "order")
  local rng = RNG.new(RNG.hash_seed(state.seedPhrase, "order", id))
  local depth = current_max_depth(state)
  local requirementCount = rng:next_int(1, 2)
  local requirements = {}
  local usedResources = {}
  local totalValue = 0
  for _ = 1, requirementCount do
    local resourceId = genResources.pick_resource(rng, depth)
    if resourceId and not usedResources[resourceId] then
      usedResources[resourceId] = true
      local amount = rng:next_int(10, 50)
      requirements[#requirements + 1] = { resourceId = resourceId, requiredAmount = amount, deliveredAmount = 0 }
      totalValue = totalValue + amount
    end
  end
  if #requirements == 0 then
    requirements[1] = { resourceId = "stone", requiredAmount = 20, deliveredAmount = 0 }
    totalValue = 20
  end

  local order = {
    id = id,
    state = "available",
    rewardMoney = math.floor(totalValue * (1.5 + rng:next())),
    expiresAtTick = state.gameTime.tick + M.ORDER_LIFETIME_TICKS,
    acceptedAtTick = nil,
    priority = 1,
    requirements = requirements,
  }
  state.orders[id] = order
  return order
end

--- Tops up the available-order pool. Safe to call any time; a no-op if already full.
function M.replenish(state)
  local availableCount = 0
  for _, order in pairs(state.orders) do
    if order.state == "available" then
      availableCount = availableCount + 1
    end
  end
  while availableCount < M.MAX_AVAILABLE_ORDERS do
    generate_order(state)
    availableCount = availableCount + 1
  end
end

local function requirements_met(order)
  for _, req in ipairs(order.requirements) do
    if req.deliveredAmount < req.requiredAmount then
      return false
    end
  end
  return true
end

local function can_fulfill_now(state, order)
  for _, req in ipairs(order.requirements) do
    local needed = req.requiredAmount - req.deliveredAmount
    if needed > 0 and storageMod.total_stored(state, req.resourceId) < needed then
      return false
    end
  end
  return true
end

local function deliver_available(state, order)
  for _, req in ipairs(order.requirements) do
    local needed = req.requiredAmount - req.deliveredAmount
    if needed > 0 then
      local withdrawn = storageMod.withdraw_resource(state, req.resourceId, needed)
      req.deliveredAmount = req.deliveredAmount + withdrawn
    end
  end
end

local function complete_order(state, order)
  order.state = "completed"
  state.money = state.money + order.rewardMoney
end

function M.accept_order(state, orderId)
  local order = state.orders[orderId]
  if not order then
    return nil, err("order_not_found", "Order not found: " .. tostring(orderId))
  end
  if order.state ~= "available" then
    return nil, err("order_not_available", "Order is not available")
  end

  if can_fulfill_now(state, order) then
    deliver_available(state, order)
    complete_order(state, order)
    M.replenish(state)
    return order, nil
  end

  order.state = "accepted"
  order.acceptedAtTick = state.gameTime.tick
  deliver_available(state, order) -- deliver whatever is available now; rest via settle() on later ticks
  M.replenish(state)
  return order, nil
end

function M.decline_order(state, orderId)
  local order = state.orders[orderId]
  if not order then
    return nil, err("order_not_found", "Order not found: " .. tostring(orderId))
  end
  if order.state ~= "available" then
    return nil, err("order_not_cancellable", "Only available orders can be declined")
  end
  order.state = "declined"
  M.replenish(state)
  return order, nil
end

function M.set_order_priority(state, orderId, priority)
  local order = state.orders[orderId]
  if not order then
    return nil, err("order_not_found", "Order not found: " .. tostring(orderId))
  end
  if order.state ~= "accepted" and order.state ~= "available" then
    return nil, err("order_not_active", "Order priority can only be changed while available or accepted")
  end
  order.priority = priority
  return order, nil
end

function M.complete_order_immediately(state, orderId)
  local order = state.orders[orderId]
  if not order then
    return nil, err("order_not_found", "Order not found: " .. tostring(orderId))
  end
  if order.state ~= "accepted" then
    return nil, err("order_not_accepted", "Order must be accepted first")
  end
  if not can_fulfill_now(state, order) then
    return nil, err("order_requirements_not_met", "Not enough resources in storage")
  end
  deliver_available(state, order)
  complete_order(state, order)
  M.replenish(state)
  return order, nil
end

--- Runs every tick: allocates available resources to accepted orders per
--- rulesConfig.orderAllocationMode, expires overdue available orders, and
--- completes orders whose requirements are now fully met.
function M.settle(state)
  local events = {}

  for _, order in pairs(state.orders) do
    if order.state == "available" and state.gameTime.tick >= order.expiresAtTick then
      order.state = "expired"
      events[#events + 1] = { type = "order_expired", orderId = order.id }
    end
  end

  local accepted = {}
  for _, order in pairs(state.orders) do
    if order.state == "accepted" then
      accepted[#accepted + 1] = order
    end
  end
  table.sort(accepted, function(a, b)
    return a.priority > b.priority
  end)

  if state.rulesConfig.orderAllocationMode == "proportional" then
    -- Split available stock proportionally to remaining need, per contested resource.
    local neededByResource = {}
    for _, order in ipairs(accepted) do
      for _, req in ipairs(order.requirements) do
        local needed = req.requiredAmount - req.deliveredAmount
        if needed > 0 then
          neededByResource[req.resourceId] = (neededByResource[req.resourceId] or 0) + needed
        end
      end
    end
    for resourceId, totalNeeded in pairs(neededByResource) do
      local availableStock = storageMod.total_stored(state, resourceId)
      if availableStock > 0 then
        local share = math.min(1, availableStock / totalNeeded)
        for _, order in ipairs(accepted) do
          for _, req in ipairs(order.requirements) do
            if req.resourceId == resourceId then
              local needed = req.requiredAmount - req.deliveredAmount
              if needed > 0 then
                local allocate = math.floor(needed * share)
                local withdrawn = storageMod.withdraw_resource(state, resourceId, allocate)
                req.deliveredAmount = req.deliveredAmount + withdrawn
              end
            end
          end
        end
      end
    end
  else
    -- priority_based (default/MVP): highest priority orders get first claim.
    for _, order in ipairs(accepted) do
      deliver_available(state, order)
    end
  end

  for _, order in ipairs(accepted) do
    if requirements_met(order) then
      complete_order(state, order)
      events[#events + 1] = { type = "order_completed", orderId = order.id }
    end
  end

  M.replenish(state)
  return events
end

return M
