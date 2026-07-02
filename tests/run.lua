-- Lua mechanics test suite (REQUIREMENTS.md §39). Runs the engine without
-- Ebitengine, exercising both white-box generation modules and the public
-- engine.apply/read/export_state/load_state contract. Exits non-zero on any
-- failure so `make test-lua` is a real gate.
package.path = package.path .. ";./lua/?.lua"

local failures = 0

local function test(name, fn)
  local ok, err = pcall(fn)
  if ok then
    print("PASS: " .. name)
  else
    failures = failures + 1
    print("FAIL: " .. name .. " -- " .. tostring(err))
  end
end

local function assert_eq(actual, expected, label)
  if actual ~= expected then
    error(string.format("[%s] expected %s, got %s", label, tostring(expected), tostring(actual)), 2)
  end
end

--------------------------------------------------------------------------
-- Generation (white-box via generation.* modules)
--------------------------------------------------------------------------

local genLevel = require("generation.level")
local chunksMod = require("generation.chunks")
local validationMod = require("generation.validation")

test("chunk generation is deterministic for the same seed", function()
  local a = genLevel.new_level("l1", 1, "seed-abc", 1)
  local b = genLevel.new_level("l1", 1, "seed-abc", 1)
  assert_eq(a.entranceCell.x, b.entranceCell.x, "entrance x")
  assert_eq(a.entranceCell.y, b.entranceCell.y, "entrance y")
  assert_eq(a.stairsCell.x, b.stairsCell.x, "stairs x")
  assert_eq(a.stairsCell.y, b.stairsCell.y, "stairs y")
  for key, cellA in pairs(a.cells) do
    local cellB = b.cells[key]
    assert(cellB, "cell missing in second generation: " .. key)
    assert_eq(cellA.kind, cellB.kind, "cell kind at " .. key)
  end
end)

test("different seeds produce different stairs placement", function()
  local a = genLevel.new_level("l1", 1, "seed-one", 1)
  local b = genLevel.new_level("l1", 1, "seed-two", 1)
  local differs = a.stairsCell.x ~= b.stairsCell.x or a.stairsCell.y ~= b.stairsCell.y
  assert(differs, "expected different seeds to place stairs differently")
end)

test("entrance area is an open 3x3 pocket", function()
  local level = genLevel.new_level("l1", 1, "seed-entrance", 1)
  local ex, ey = level.entranceCell.x, level.entranceCell.y
  for x = ex - 1, ex + 1 do
    for y = ey - 1, ey + 1 do
      local cell = level.cells[chunksMod.cell_key(x, y)]
      assert(cell, "expected entrance cell to exist at " .. x .. "," .. y)
      assert_eq(cell.kind, "empty", "entrance cell kind at " .. x .. "," .. y)
    end
  end
end)

test("stairs area is a 3x3 stairs_area pocket in a non-central chunk", function()
  local level = genLevel.new_level("l1", 1, "seed-stairs", 1)
  assert(not (level.stairsChunk.cx == 0 and level.stairsChunk.cy == 0), "stairs chunk must not be the central chunk")
  local sx, sy = level.stairsCell.x, level.stairsCell.y
  for x = sx - 1, sx + 1 do
    for y = sy - 1, sy + 1 do
      local cell = level.cells[chunksMod.cell_key(x, y)]
      assert(cell, "expected stairs cell to exist at " .. x .. "," .. y)
      assert_eq(cell.kind, "stairs_area", "stairs cell kind at " .. x .. "," .. y)
    end
  end
end)

test("generator guarantees a potential path from entrance to stairs", function()
  local level = genLevel.new_level("l1", 1, "seed-path", 1)
  local validation = validationMod.validate(level)
  assert(validation.hasPathFromEntranceToExit, "expected a potential path from entrance to stairs")
  assert(validation.corridorWidthLimitRespected, "expected corridor width limit respected")
  assert(validation.stairsAreaReachablePotentially, "expected stairs area potentially reachable")
end)

--------------------------------------------------------------------------
-- Visibility / reachability
--------------------------------------------------------------------------

local reachabilityMod = require("simulation.reachability")
local visibilityMod = require("simulation.visibility")

test("visibility reveals cells within radius 5 and hides far cells", function()
  local level = genLevel.new_level("l1", 1, "seed-vis", 1)
  reachabilityMod.recompute(level)
  visibilityMod.recompute(level)
  local ex, ey = level.entranceCell.x, level.entranceCell.y

  local nearCell = level.cells[chunksMod.cell_key(ex + 1, ey)]
  assert(nearCell, "expected neighboring cell to exist")
  assert_eq(nearCell.visibility, "scouted", "adjacent cell visibility")

  local farCell = level.cells[chunksMod.cell_key(ex + 20, ey)]
  if farCell then
    assert_eq(farCell.visibility, "unknown", "far cell (distance 20) visibility")
  end
end)

test("entrance pocket is reachable from itself", function()
  local level = genLevel.new_level("l1", 1, "seed-reach", 1)
  reachabilityMod.recompute(level)
  local ex, ey = level.entranceCell.x, level.entranceCell.y
  local cell = level.cells[chunksMod.cell_key(ex, ey)]
  assert_eq(cell.accessibility, "reachable", "entrance accessibility")
end)

test("mining a deposit into empty extends the reachable flood-filled area", function()
  local level = genLevel.new_level("l1", 1, "seed-flood", 1)
  reachabilityMod.recompute(level)
  local ex, ey = level.entranceCell.x, level.entranceCell.y
  -- the cell just outside the entrance pocket must currently be an unreached deposit
  local outside = level.cells[chunksMod.cell_key(ex - 2, ey)]
  assert(outside, "expected a cell just outside the entrance pocket")
  assert_eq(outside.accessibility, "unreachable", "cell outside entrance should start unreachable")

  outside.kind = "empty"
  outside.components = {}
  reachabilityMod.recompute(level)
  assert_eq(outside.accessibility, "reachable", "cell should become reachable once emptied and connected")
end)

--------------------------------------------------------------------------
-- Engine-level mechanics (black-box via engine.apply/read)
--------------------------------------------------------------------------

local engine = require("engine")

--- Finds a deposit cell adjacent to a reachable open cell in the given level view.
local function find_minable_pair(levelView)
  local byCoord = {}
  for _, c in ipairs(levelView.cells) do
    byCoord[c.x .. "," .. c.y] = c
  end
  for _, c in ipairs(levelView.cells) do
    if c.kind == "deposit" then
      local neighbors = { { c.x, c.y - 1 }, { c.x, c.y + 1 }, { c.x - 1, c.y }, { c.x + 1, c.y } }
      for _, n in ipairs(neighbors) do
        local nc = byCoord[n[1] .. "," .. n[2]]
        if nc and nc.accessibility == "reachable" and (nc.kind == "empty" or nc.kind == "stairs_area") then
          return c.x .. "," .. c.y, n[1] .. "," .. n[2], c
        end
      end
    end
  end
  error("could not find a minable deposit adjacent to a reachable open cell")
end

test("worker mines a deposit: rock never enters storage, resource does, cell becomes empty", function()
  engine.new_game("mining-mechanics-seed")
  local levelView = engine.read({
    type = "get_level_view",
    levelId = "level_1",
    viewport = { x = 0, y = 0, width = 32, height = 32 },
  }).data
  local targetId, positionId, targetCell = find_minable_pair(levelView)

  local resourceId
  for _, comp in ipairs(targetCell.components) do
    if comp.type == "resource" then
      resourceId = comp.resourceId
    end
  end
  assert(resourceId, "expected the target deposit to contain a resource component")

  local buy = engine.apply({ type = "buy_worker", workerLevel = 1 })
  assert(buy.ok, "buy_worker failed")

  local assign = engine.apply({
    type = "assign_worker_to_target_cell",
    workerId = buy.data.id,
    levelId = "level_1",
    targetCellId = targetId,
    positionCellId = positionId,
    assignmentMode = "until_completed",
  })
  assert(assign.ok, "assign failed: " .. (assign.error and assign.error.message or ""))

  local depleted = false
  for i = 1, 250 do
    local tr = engine.apply({ type = "tick", ticksPassed = 1 })
    assert(tr.ok, "tick failed: " .. (tr.error and tr.error.message or ""))
    for _, e in ipairs(tr.events) do
      if e.type == "cell_depleted" and e.cellId == targetId then
        depleted = true
      end
    end
    if depleted then
      break
    end
  end
  assert(depleted, "expected the deposit to fully deplete within 250 ticks")

  local storageState = engine.read({ type = "get_storage_state" }).data
  local totalStored = 0
  for _, s in ipairs(storageState.storages) do
    if s.resourceId == resourceId then
      totalStored = totalStored + s.storedAmount
    end
  end
  assert(totalStored > 0, "expected some of the mined resource to have reached storage")

  local view2 = engine.read({
    type = "get_level_view",
    levelId = "level_1",
    viewport = { x = 0, y = 0, width = 32, height = 32 },
  }).data
  for _, c in ipairs(view2.cells) do
    if c.x .. "," .. c.y == targetId then
      assert_eq(c.kind, "empty", "depleted cell kind")
    end
  end
end)

test("depleting a deposit auto-continues the worker onto an adjacent deposit", function()
  local miningMod = require("simulation.mining")
  local resourceConfig = require("config.resources")

  local storages = {}
  for _, resource in ipairs(resourceConfig.list) do
    storages[resource.id] =
      { id = resource.id, resourceId = resource.id, level = 1, capacity = math.huge, storedAmount = 0 }
  end

  -- Worker stands at (0,0); deposit A is one tick from depleting to its west,
  -- deposit B (untouched) is to its north.
  local level = {
    id = "lvl",
    activeMiningCells = { ["-1,0"] = true },
    cells = {
      ["0,0"] = { x = 0, y = 0, kind = "empty", accessibility = "reachable", occupiedBy = "w1", assignedWorkers = {} },
      ["-1,0"] = {
        x = -1,
        y = 0,
        kind = "deposit",
        assignedWorkers = { east = "w1" },
        components = { { type = "rock", resourceId = nil, ratio = 1, initialAmount = 1, remainingAmount = 1 } },
      },
      ["0,-1"] = {
        x = 0,
        y = -1,
        kind = "deposit",
        assignedWorkers = {},
        components = {
          { type = "rock", resourceId = nil, ratio = 0.5, initialAmount = 50, remainingAmount = 50 },
          { type = "resource", resourceId = "stone", ratio = 0.5, initialAmount = 50, remainingAmount = 50 },
        },
      },
    },
  }
  local state = {
    storages = storages,
    levels = { lvl = level },
    workers = {
      w1 = {
        id = "w1",
        level = 1,
        speed = 1000,
        state = "working",
        assignedLevelId = "lvl",
        targetCellId = "-1,0",
        positionCellId = "0,0",
        assignmentMode = "until_completed",
      },
    },
  }

  local events = miningMod.process_tick(state, level)
  local depleted = false
  for _, e in ipairs(events) do
    if e.type == "cell_depleted" and e.cellId == "-1,0" then
      depleted = true
    end
  end
  assert(depleted, "expected deposit A to deplete in a single tick")
  assert_eq(level.cells["-1,0"].kind, "empty", "depleted deposit kind")

  local worker = state.workers.w1
  assert_eq(worker.state, "working", "worker should auto-continue instead of going idle")
  assert_eq(worker.targetCellId, "0,-1", "worker should have been reassigned to the adjacent deposit")
  assert_eq(level.cells["0,-1"].assignedWorkers.south, "w1", "worker should be mining deposit B from the south")
  assert(level.activeMiningCells["0,-1"], "deposit B should now be an active mining cell")
end)

test("storages exist for every resource from the start and never fill up", function()
  engine.new_game("full-storage-seed")
  local storageState = engine.read({ type = "get_storage_state" }).data
  assert(#storageState.storages > 0, "expected storages to be pre-created for every resource")
  for _, s in ipairs(storageState.storages) do
    assert(s.capacity == math.huge, "expected every storage to have unlimited capacity")
  end
end)

test("only one worker may occupy a given position cell", function()
  engine.new_game("occupancy-seed")
  local levelView = engine.read({
    type = "get_level_view",
    levelId = "level_1",
    viewport = { x = 0, y = 0, width = 32, height = 32 },
  }).data
  local targetId, positionId = find_minable_pair(levelView)

  local w1 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  local w2 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  assert(w1.ok and w2.ok, "buy_worker failed")

  local a1 = engine.apply({
    type = "assign_worker_to_target_cell",
    workerId = w1.data.id,
    levelId = "level_1",
    targetCellId = targetId,
    positionCellId = positionId,
    assignmentMode = "until_completed",
  })
  assert(a1.ok, "first assignment should succeed")

  local a2 = engine.apply({
    type = "assign_worker_to_target_cell",
    workerId = w2.data.id,
    levelId = "level_1",
    targetCellId = targetId,
    positionCellId = positionId,
    assignmentMode = "until_completed",
  })
  assert(not a2.ok, "second worker must not be able to share the same position cell")
  assert_eq(a2.error.code, "position_occupied", "expected position_occupied")
end)

test("tick advances gameTime.tick by exactly ticksPassed, with no phase gating", function()
  engine.new_game("tick-advance-seed")
  local before = engine.read({ type = "get_game_time" }).data.tick
  local tr = engine.apply({ type = "tick", ticksPassed = 5 })
  assert(tr.ok, "tick failed")
  assert_eq(tr.processedTicks, 5, "processed ticks")
  local after = engine.read({ type = "get_game_time" }).data.tick
  assert_eq(after, before + 5, "tick count after advancing")
end)

test("worker purchase respects highestUnlockedWorkerLevel - 2 formula", function()
  engine.new_game("purchase-formula-seed")
  local bad = engine.apply({ type = "buy_worker", workerLevel = 2 })
  assert(not bad.ok, "should not be able to buy level 2 worker before any level-2 worker is unlocked")
  assert_eq(bad.error.code, "worker_level_not_purchasable", "purchase formula error code")

  -- Force a level-4 unlock and enough money (fixture via export/load) to unlock buying level 2.
  local state = engine.export_state()
  state.highestUnlockedWorkerLevel = 4
  state.money = 100000
  engine.load_state(state)
  local ok = engine.apply({ type = "buy_worker", workerLevel = 2 })
  assert(
    ok.ok,
    "should be able to buy level 2 worker once highestUnlockedWorkerLevel is 4 (4-2=2): "
      .. (ok.error and ok.error.message or "")
  )
end)

test("buying a worker with a levelId deploys it onto the map immediately", function()
  engine.new_game("auto-deploy-seed")
  local bought = engine.apply({ type = "buy_worker", workerLevel = 1, levelId = "level_1" })
  assert(bought.ok, "buy_worker should succeed: " .. (bought.error and bought.error.message or ""))

  local workers = engine.read({ type = "get_workers" }).data
  local worker = workers.workers[1]
  assert(worker.positionCellId ~= nil, "freshly bought worker should have a positionCellId")
  assert_eq(worker.assignedLevelId, "level_1", "freshly bought worker should be attached to the level")
end)

test("orders: available order can be declined, and accepting with enough stock completes it immediately", function()
  engine.new_game("orders-seed")
  local orders = engine.read({ type = "get_available_orders" }).data.orders
  assert(#orders > 0, "expected available orders at game start")

  local declined = engine.apply({ type = "decline_order", orderId = orders[1].id })
  assert(declined.ok, "decline_order failed")

  -- Give ourselves a pile of every depth-1 resource, then accept an order that
  -- only needs those, expecting immediate completion.
  local state = engine.export_state()
  state.storages["storage_test"] =
    { id = "storage_test", resourceId = "stone", level = 1, capacity = 10000, storedAmount = 10000 }
  state.storages["storage_test_2"] =
    { id = "storage_test_2", resourceId = "coal", level = 1, capacity = 10000, storedAmount = 10000 }
  engine.load_state(state)

  local availableNow = engine.read({ type = "get_available_orders" }).data.orders
  assert(#availableNow > 0, "expected another available order to accept")
  local target = availableNow[1]
  local accept = engine.apply({ type = "accept_order", orderId = target.id })
  assert(accept.ok, "accept_order failed: " .. (accept.error and accept.error.message or ""))
  assert_eq(accept.data.state, "completed", "order with enough stock should complete immediately on accept")
end)

test("orders: declining does not refill the pool; new orders arrive on 100-tick boundaries", function()
  engine.new_game("orders-arrival-seed")
  local before = engine.read({ type = "get_available_orders" }).data.orders
  assert(#before > 0, "expected available orders at game start")
  for _, order in ipairs(before) do
    assert(engine.apply({ type = "decline_order", orderId = order.id }).ok, "decline_order failed")
  end
  assert_eq(#engine.read({ type = "get_available_orders" }).data.orders, 0, "pool must stay empty after declining")

  engine.apply({ type = "tick", ticksPassed = 99 })
  assert_eq(#engine.read({ type = "get_available_orders" }).data.orders, 0, "no arrivals off the 100-tick boundary")

  -- 10 arrival boundaries at 50% chance each, deterministic for this seed.
  engine.apply({ type = "tick", ticksPassed = 901 })
  assert(
    #engine.read({ type = "get_available_orders" }).data.orders > 0,
    "expected at least one order to arrive within 1000 ticks"
  )
end)

test("orders: accepted order ships partially every 50 ticks and pays per shipped unit", function()
  engine.new_game("orders-shipment-seed")
  local target = engine.read({ type = "get_available_orders" }).data.orders[1]
  local accept = engine.apply({ type = "accept_order", orderId = target.id })
  assert(accept.ok, "accept_order failed: " .. (accept.error and accept.error.message or ""))
  assert_eq(accept.data.state, "accepted", "with empty storages the order must stay accepted")

  local req = target.requirements[1]
  assert(req.pricePerUnit and req.pricePerUnit > 0, "requirement must carry a pricePerUnit")
  local partial = math.max(1, math.floor(req.requiredAmount / 2))
  local state = engine.export_state()
  state.storages[req.resourceId].storedAmount = partial
  engine.load_state(state)

  local moneyBefore = engine.read({ type = "get_player_summary" }).data.money
  engine.apply({ type = "tick", ticksPassed = 49 })
  local active = engine.read({ type = "get_active_orders" }).data.orders[1]
  assert_eq(active.requirements[1].deliveredAmount, 0, "no shipment before the 50-tick boundary")

  engine.apply({ type = "tick", ticksPassed = 1 })
  local shipped = engine.read({ type = "get_active_orders" }).data.orders[1]
  assert_eq(shipped.requirements[1].deliveredAmount, partial, "partial stock must be shipped on tick 50")
  local moneyAfter = engine.read({ type = "get_player_summary" }).data.money
  assert_eq(moneyAfter - moneyBefore, partial * req.pricePerUnit, "each shipped part is paid at the order's price")
end)

test("orders: fractional storage is shipped as a whole number, remainder stays in storage", function()
  engine.new_game("orders-fractional-seed")
  local target = engine.read({ type = "get_available_orders" }).data.orders[1]
  local accept = engine.apply({ type = "accept_order", orderId = target.id })
  assert(accept.ok, "accept_order failed: " .. (accept.error and accept.error.message or ""))

  -- Storage may legitimately hold a fractional amount (mining extracts a
  -- speed-weighted, non-integer share per tick) - the order must still only
  -- ever be delivered whole units, with the fractional remainder left behind.
  local req = target.requirements[1]
  local fractionalStock = 5.7
  local state = engine.export_state()
  state.storages[req.resourceId].storedAmount = fractionalStock
  engine.load_state(state)

  local moneyBefore = engine.read({ type = "get_player_summary" }).data.money
  engine.apply({ type = "tick", ticksPassed = 50 })

  local shipped = engine.read({ type = "get_active_orders" }).data.orders[1]
  assert_eq(shipped.requirements[1].deliveredAmount, 5, "only the whole-unit part of fractional stock is delivered")

  local storageState = engine.read({ type = "get_storage_state" }).data
  local remaining
  for _, s in ipairs(storageState.storages) do
    if s.resourceId == req.resourceId then
      remaining = s.storedAmount
    end
  end
  assert(
    math.abs(remaining - (fractionalStock - 5)) < 1e-9,
    "the fractional remainder must stay in storage, not vanish or get shipped: got " .. tostring(remaining)
  )

  local moneyAfter = engine.read({ type = "get_player_summary" }).data.money
  assert_eq(moneyAfter - moneyBefore, 5 * req.pricePerUnit, "payment is for the whole-unit amount actually shipped")
end)

test("orders: stock is split proportionally between accepted orders on shipment", function()
  engine.new_game("orders-proportional-seed")
  local state = engine.export_state()
  state.orders = {
    order_a = {
      id = "order_a",
      state = "accepted",
      rewardMoney = 200,
      expiresAtTick = 100000,
      acceptedAtTick = 0,
      priority = 1,
      requirements = { { resourceId = "stone", requiredAmount = 100, deliveredAmount = 0, pricePerUnit = 2 } },
    },
    order_b = {
      id = "order_b",
      state = "accepted",
      rewardMoney = 900,
      expiresAtTick = 100000,
      acceptedAtTick = 0,
      priority = 1,
      requirements = { { resourceId = "stone", requiredAmount = 300, deliveredAmount = 0, pricePerUnit = 3 } },
    },
  }
  state.storages.stone.storedAmount = 200
  engine.load_state(state)

  local moneyBefore = engine.read({ type = "get_player_summary" }).data.money
  engine.apply({ type = "tick", ticksPassed = 50 })

  local byId = {}
  for _, order in ipairs(engine.read({ type = "get_active_orders" }).data.orders) do
    byId[order.id] = order
  end
  assert_eq(byId.order_a.requirements[1].deliveredAmount, 50, "order_a gets its proportional share (100/400 of 200)")
  assert_eq(byId.order_b.requirements[1].deliveredAmount, 150, "order_b gets its proportional share (300/400 of 200)")
  local moneyAfter = engine.read({ type = "get_player_summary" }).data.money
  assert_eq(moneyAfter - moneyBefore, 50 * 2 + 150 * 3, "each order is paid at its own per-unit price")
end)

test("merge combines two idle same-level workers into one worker of the next level", function()
  engine.new_game("merge-seed")
  local b1 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  local b2 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  assert(b1.ok and b2.ok, "buy_worker failed")
  local merge = engine.apply({ type = "merge_workers", workerIds = { b1.data.id, b2.data.id } })
  assert(merge.ok, "merge_workers failed: " .. (merge.error and merge.error.message or ""))
  assert_eq(merge.data.level, 2, "merged worker level")
  local workers = engine.read({ type = "get_workers" }).data.workers
  assert_eq(#workers, 1, "expected the two source workers to be replaced by exactly one merged worker")
end)

test("merge stops busy workers and merges them (the drag-onto-another-worker case)", function()
  engine.new_game("busy-merge-seed")
  local levelView = engine.read({
    type = "get_level_view",
    levelId = "level_1",
    viewport = { x = 0, y = 0, width = 32, height = 32 },
  }).data

  local byCoord = {}
  for _, c in ipairs(levelView.cells) do
    byCoord[c.x .. "," .. c.y] = c
  end
  local used = {}
  local minePairs = {}
  for _, c in ipairs(levelView.cells) do
    if c.kind == "deposit" and #minePairs < 2 then
      local neighbors = { { c.x, c.y - 1 }, { c.x, c.y + 1 }, { c.x - 1, c.y }, { c.x + 1, c.y } }
      for _, n in ipairs(neighbors) do
        local key = n[1] .. "," .. n[2]
        local nc = byCoord[key]
        if
          nc
          and nc.accessibility == "reachable"
          and (nc.kind == "empty" or nc.kind == "stairs_area")
          and not used[key]
        then
          used[key] = true
          minePairs[#minePairs + 1] = { targetId = c.x .. "," .. c.y, positionId = key }
          break
        end
      end
    end
  end
  assert(#minePairs >= 2, "expected at least two independent minable positions for this seed")

  local b1 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  local b2 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  assert(b1.ok and b2.ok, "buy_worker failed")

  for i, worker in ipairs({ b1.data, b2.data }) do
    local p = minePairs[i]
    local assign = engine.apply({
      type = "assign_worker_to_target_cell",
      workerId = worker.id,
      levelId = "level_1",
      targetCellId = p.targetId,
      positionCellId = p.positionId,
      assignmentMode = "until_completed",
    })
    assert(assign.ok, "assign failed: " .. (assign.error and assign.error.message or ""))
  end

  for _, w in ipairs(engine.read({ type = "get_workers" }).data.workers) do
    assert_eq(w.state, "working", "expected both workers to be busy mining before the merge")
  end

  local merge = engine.apply({ type = "merge_workers", workerIds = { b1.data.id, b2.data.id } })
  assert(
    merge.ok,
    "merge_workers should succeed even if both workers were busy mining: "
      .. (merge.error and merge.error.message or "")
  )
  assert_eq(merge.data.level, 2, "merged worker level")
  assert_eq(
    merge.data.positionCellId,
    minePairs[2].positionId,
    "merged worker should take over the position of the worker it was dropped onto"
  )
  assert_eq(
    merge.data.state,
    "working",
    "merged worker should resume mining in place instead of vanishing from the map"
  )
end)

test("merge failure (mismatched levels) restores each worker's previous mining assignment", function()
  engine.new_game("busy-merge-mismatch-seed")
  local levelView = engine.read({
    type = "get_level_view",
    levelId = "level_1",
    viewport = { x = 0, y = 0, width = 32, height = 32 },
  }).data
  local targetId, positionId = find_minable_pair(levelView)

  local b1 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  assert(b1.ok, "buy_worker failed")
  local assign = engine.apply({
    type = "assign_worker_to_target_cell",
    workerId = b1.data.id,
    levelId = "level_1",
    targetCellId = targetId,
    positionCellId = positionId,
    assignmentMode = "until_completed",
  })
  assert(assign.ok, "assign failed: " .. (assign.error and assign.error.message or ""))

  -- Bump this worker to level 2 directly via the export/load fixture path so
  -- it mismatches the level-1 worker bought below.
  local state = engine.export_state()
  state.workers[b1.data.id].level = 2
  engine.load_state(state)

  local b2 = engine.apply({ type = "buy_worker", workerLevel = 1 })
  assert(b2.ok, "buy_worker failed")

  local merge = engine.apply({ type = "merge_workers", workerIds = { b1.data.id, b2.data.id } })
  assert(not merge.ok, "merge should fail for mismatched levels")
  assert_eq(merge.error.code, "worker_level_mismatch", "merge error code")

  local w1
  for _, w in ipairs(engine.read({ type = "get_workers" }).data.workers) do
    if w.id == b1.data.id then
      w1 = w
    end
  end
  assert(w1, "worker should still exist after a failed merge")
  assert_eq(w1.state, "working", "worker should resume mining after a failed merge")
  assert_eq(w1.targetCellId, targetId, "worker should resume its previous target cell after a failed merge")
end)

test("create_next_level is rejected until the stairs area is reachable", function()
  engine.new_game("next-level-seed")
  local bad = engine.apply({ type = "create_next_level", fromLevelId = "level_1" })
  assert(not bad.ok, "create_next_level should fail before stairs reachable")
  assert_eq(bad.error.code, "stairs_not_reachable", "create_next_level error code")
end)

test("export_state / load_state round-trip preserves money and worker count", function()
  engine.new_game("roundtrip-seed")
  engine.apply({ type = "buy_worker", workerLevel = 1 })
  local before = engine.read({ type = "get_player_summary" }).data
  local exported = engine.export_state()
  engine.load_state(exported)
  local after = engine.read({ type = "get_player_summary" }).data
  assert_eq(after.money, before.money, "money after round-trip")
  assert_eq(after.workerCount, before.workerCount, "worker count after round-trip")
end)

--------------------------------------------------------------------------

print(string.format("\n%d failure(s)", failures))
if failures > 0 then
  os.exit(1)
end
