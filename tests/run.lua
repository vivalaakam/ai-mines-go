package.path = package.path .. ";./lua/?.lua"

local engine = require("engine")

local function assert_eq(actual, expected, label)
  if actual ~= expected then
    error(string.format("ASSERTION FAILED [%s]: expected %s, got %s", label, tostring(expected), tostring(actual)))
  end
end

engine.new_game("test-seed-123")

-- get_game_phase / get_game_time
local r = engine.read({ type = "get_game_phase" })
assert(r.ok, "read get_game_phase failed")
assert_eq(r.data.phase, "shift_planning", "initial phase")

-- level view has scouted entrance cells
local levelView = engine.read({ type = "get_level_view", levelId = "level_1", viewport = { x = 0, y = 0, width = 32, height = 32 } })
assert(levelView.ok, "get_level_view failed")
local scoutedCount = 0
for _, c in ipairs(levelView.data.cells) do
  if c.visibility == "scouted" then scoutedCount = scoutedCount + 1 end
end
print("scouted cells in chunk (0,0): " .. scoutedCount)
assert(scoutedCount > 0, "expected some scouted cells near entrance")

-- buy a worker
local buyResult = engine.apply({ type = "buy_worker", workerLevel = 1 })
assert(buyResult.ok, "buy_worker failed: " .. (buyResult.error and buyResult.error.message or ""))
local workerId = buyResult.data.id
print("bought worker: " .. workerId)

-- find a reachable position cell adjacent to a deposit cell near entrance
local workers = engine.read({ type = "get_workers" })
assert(workers.ok, "get_workers failed")

local targetCellId, positionCellId
for _, c in ipairs(levelView.data.cells) do
  if c.kind == "deposit" then
    local neighbors = {
      { c.x, c.y - 1 }, { c.x, c.y + 1 }, { c.x - 1, c.y }, { c.x + 1, c.y },
    }
    for _, n in ipairs(neighbors) do
      for _, oc in ipairs(levelView.data.cells) do
        if oc.x == n[1] and oc.y == n[2] and oc.accessibility == "reachable" and (oc.kind == "empty" or oc.kind == "stairs_area") then
          targetCellId = c.x .. "," .. c.y
          positionCellId = oc.x .. "," .. oc.y
        end
      end
    end
  end
end
assert(targetCellId, "could not find a minable deposit adjacent to a reachable open cell")
print("assigning worker " .. workerId .. " target=" .. targetCellId .. " position=" .. positionCellId)

local assignResult = engine.apply({
  type = "assign_worker_to_target_cell",
  workerId = workerId,
  levelId = "level_1",
  targetCellId = targetCellId,
  positionCellId = positionCellId,
  assignmentMode = "until_completed",
})
assert(assignResult.ok, "assign failed: " .. (assignResult.error and assignResult.error.message or ""))

-- start shift and tick a bunch
local startResult = engine.apply({ type = "start_next_shift" })
assert(startResult.ok, "start_next_shift failed: " .. (startResult.error and startResult.error.message or ""))

local totalTicks = 0
for i = 1, 400 do
  local tr = engine.apply({ type = "tick", ticksPassed = 1 })
  if not tr.ok then
    assert_eq(tr.error.code, "not_shift_running", "expected shift to already be over by iteration " .. i)
    break
  end
  totalTicks = totalTicks + 1
  for _, e in ipairs(tr.events) do
    if e.type == "shift_completed" then
      print("shift completed at loop iteration " .. i)
    end
  end
end

local phaseAfter = engine.read({ type = "get_game_phase" })
assert_eq(phaseAfter.data.phase, "shift_planning", "phase after 400 ticks (>300) should be back to planning")

local shiftSummary = engine.read({ type = "get_shift_summary" })
print("shiftIndex after run: " .. shiftSummary.data.shiftIndex)
assert_eq(shiftSummary.data.shiftIndex, 1, "shiftIndex should be 1 after first shift completes")

-- fast forward should fail outside shift_running
local ffFail = engine.apply({ type = "fast_forward_to_shift_end" })
assert(not ffFail.ok, "fast_forward should fail during planning")
assert_eq(ffFail.error.code, "not_shift_running", "fast forward error code")

-- storage / mining sanity: check some resource got stored or is progressing
local storageState = engine.read({ type = "get_storage_state" })
print("storages: " .. #storageState.data.storages)

local playerSummary = engine.read({ type = "get_player_summary" })
print("money: " .. playerSummary.data.money)

-- export/load roundtrip
local exported = engine.export_state()
engine.load_state(exported)
local phaseAfterReload = engine.read({ type = "get_game_phase" })
assert_eq(phaseAfterReload.data.phase, "shift_planning", "phase preserved after export/load roundtrip")

print("ALL SMOKE TESTS PASSED")

-- extra coverage: storage, orders, merge, create_next_level validation
engine.new_game("seed-2")

local availOrders = engine.read({ type = "get_available_orders" })
assert(availOrders.ok and #availOrders.data.orders > 0, "expected available orders")
local orderId = availOrders.data.orders[1].id
local decline = engine.apply({ type = "decline_order", orderId = orderId })
assert(decline.ok, "decline_order failed")

local b1 = engine.apply({ type = "buy_worker", workerLevel = 1 })
local b2 = engine.apply({ type = "buy_worker", workerLevel = 1 })
assert(b1.ok and b2.ok, "buy_worker for merge setup failed")

local buyStorage = engine.apply({ type = "buy_storage", resourceId = "stone" })
assert(buyStorage.ok or buyStorage.error.code == "insufficient_funds", "buy_storage errored unexpectedly: " .. (buyStorage.error and buyStorage.error.message or ""))
if buyStorage.ok then
  local upgrade = engine.apply({ type = "upgrade_storage", storageId = buyStorage.data.id })
  assert(upgrade.ok or upgrade.error.code == "insufficient_funds", "upgrade_storage errored unexpectedly: " .. (upgrade.error and upgrade.error.message or ""))
end
local merge = engine.apply({ type = "merge_workers", workerIds = { b1.data.id, b2.data.id } })
assert(merge.ok, "merge_workers failed: " .. (merge.error and merge.error.message or ""))
assert_eq(merge.data.level, 2, "merged worker level")

local badNextLevel = engine.apply({ type = "create_next_level", fromLevelId = "level_1" })
assert(not badNextLevel.ok, "create_next_level should fail before stairs reachable")
assert_eq(badNextLevel.error.code, "stairs_not_reachable", "create_next_level error code")

print("ALL EXTRA COVERAGE TESTS PASSED")
