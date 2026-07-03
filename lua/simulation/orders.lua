local RNG = require("generation.rng")
local storageMod = require("simulation.storage")
local genResources = require("generation.resources")
local resourceConfig = require("config.resources")
local balance = require("config.balance")

local M = {}

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

local function base_price(resourceId)
  for _, r in ipairs(resourceConfig.list) do
    if r.id == resourceId then
      return r.basePrice
    end
  end
  return 1
end

local function available_count(state)
  local count = 0
  for _, order in pairs(state.orders) do
    if order.state == "available" then
      count = count + 1
    end
  end
  return count
end

--- Fully deterministic: seeded from seedPhrase + the order's own sequential id.
--- Each requirement carries its own pricePerUnit (rolled from the resource's
--- base price range); rewardMoney is the order's full value, paid out
--- incrementally as parts are shipped.
local function generate_order(state)
  local id = next_id(state, "order")
  local rng = RNG.new(RNG.hash_seed(state.seedPhrase, "order", id))
  local depth = current_max_depth(state)
  local requirementCount = rng:next_int(1, 2)
  local requirements = {}
  local usedResources = {}
  for _ = 1, requirementCount do
    local resourceId = genResources.pick_resource(rng, depth)
    if resourceId and not usedResources[resourceId] then
      usedResources[resourceId] = true
      requirements[#requirements + 1] = {
        resourceId = resourceId,
        requiredAmount = rng:next_int(10, 50),
        deliveredAmount = 0,
        pricePerUnit = balance.order_price_per_unit(rng, base_price(resourceId)),
      }
    end
  end
  if #requirements == 0 then
    requirements[1] = {
      resourceId = "stone",
      requiredAmount = 20,
      deliveredAmount = 0,
      pricePerUnit = balance.order_price_per_unit(rng, base_price("stone")),
    }
  end

  local rewardMoney = 0
  for _, req in ipairs(requirements) do
    rewardMoney = rewardMoney + req.requiredAmount * req.pricePerUnit
  end

  local order = {
    id = id,
    state = "available",
    rewardMoney = rewardMoney,
    expiresAtTick = state.gameTime.tick + state.rulesConfig.orderLifetimeTicks,
    acceptedAtTick = nil,
    priority = 1,
    requirements = requirements,
  }
  state.orders[id] = order
  return order
end

--- Seeds the available-order pool up to maxAvailableOrders. Only used for a
--- new game; afterwards orders arrive solely via the periodic arrival roll.
function M.replenish(state)
  local count = available_count(state)
  while count < state.rulesConfig.maxAvailableOrders do
    generate_order(state)
    count = count + 1
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

--- Ships up to `amount` of one requirement: withdraws from storage, advances
--- deliveredAmount, and pays the order's per-unit price for what was shipped.
--- Storage may hold a fractional amount of a resource (mining extracts
--- speed-weighted, non-integer shares per tick), but buyers only ever accept
--- whole units, so the withdrawal is clamped to floor(min(amount,
--- available)) - always an exact integer, leaving any fractional remainder
--- untouched in storage for a later shipment to pick up.
local function ship(state, order, req, amount, events)
  if amount <= 0 then
    return
  end
  local available = storageMod.total_stored(state, req.resourceId)
  local wholeAmount = math.floor(math.min(amount, available))
  if wholeAmount <= 0 then
    return
  end
  local withdrawn = storageMod.withdraw_resource(state, req.resourceId, wholeAmount)
  if withdrawn <= 0 then
    return
  end
  req.deliveredAmount = req.deliveredAmount + withdrawn
  local payment = withdrawn * (req.pricePerUnit or 0)
  state.money = state.money + payment
  events[#events + 1] = {
    type = "order_shipped",
    orderId = order.id,
    resourceId = req.resourceId,
    amount = withdrawn,
    payment = payment,
  }
end

local function deliver_available(state, order, events)
  for _, req in ipairs(order.requirements) do
    ship(state, order, req, req.requiredAmount - req.deliveredAmount, events)
  end
end

local function complete_order(state, order)
  -- Payment already happened per shipped part; completion is just a state flip.
  order.state = "completed"
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
    deliver_available(state, order, {})
    complete_order(state, order)
    return order, nil
  end

  -- Not fully coverable now: just mark accepted. Partial shipments happen on
  -- the orderShipmentIntervalTicks cadence in settle(), each part paid as it goes.
  order.state = "accepted"
  order.acceptedAtTick = state.gameTime.tick
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
  deliver_available(state, order, {})
  complete_order(state, order)
  return order, nil
end

--- Distributes available stock across accepted orders and pays per shipped part.
local function ship_accepted(state, events)
  local accepted = {}
  for _, order in pairs(state.orders) do
    if order.state == "accepted" then
      accepted[#accepted + 1] = order
    end
  end
  if #accepted == 0 then
    return
  end
  table.sort(accepted, function(a, b)
    if a.priority ~= b.priority then
      return a.priority > b.priority
    end
    return a.id < b.id
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
              ship(state, order, req, math.floor((req.requiredAmount - req.deliveredAmount) * share), events)
            end
          end
        end
      end
    end
  end

  -- Greedy pass by priority: in priority_based mode this is the whole
  -- allocation; in proportional mode it hands out the remainder the
  -- floor()-ed proportional shares left behind.
  for _, order in ipairs(accepted) do
    deliver_available(state, order, events)
  end

  for _, order in ipairs(accepted) do
    if requirements_met(order) then
      complete_order(state, order)
      events[#events + 1] = { type = "order_completed", orderId = order.id }
    end
  end
end

--- Runs every tick. Expires overdue available orders each tick; ships to
--- accepted orders every orderShipmentIntervalTicks; rolls a chance for a new
--- order every orderArrivalIntervalTicks (deterministic from seedPhrase+tick).
function M.settle(state)
  local events = {}
  local tick = state.gameTime.tick
  local rules = state.rulesConfig

  for _, order in pairs(state.orders) do
    if order.state == "available" and tick >= order.expiresAtTick then
      order.state = "expired"
      events[#events + 1] = { type = "order_expired", orderId = order.id }
    end
  end

  if tick % rules.orderShipmentIntervalTicks == 0 then
    ship_accepted(state, events)
  end

  if tick % rules.orderArrivalIntervalTicks == 0 and available_count(state) < rules.maxAvailableOrders then
    local rng = RNG.new(RNG.hash_seed(state.seedPhrase, "order_arrival", tostring(tick)))
    local arrival = rng:next() < rules.orderArrivalChance
    if not arrival then
      state.orderArrivalMisses = (state.orderArrivalMisses or 0) + 1
      arrival = state.orderArrivalMisses >= rules.orderArrivalMaxMisses
    end
    if arrival then
      state.orderArrivalMisses = 0
      local order = generate_order(state)
      events[#events + 1] = { type = "order_arrived", orderId = order.id }
    end
  end

  return events
end

return M
