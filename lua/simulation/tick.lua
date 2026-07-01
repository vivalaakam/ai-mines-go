local miningMod = require("simulation.mining")
local reachabilityMod = require("simulation.reachability")
local visibilityMod = require("simulation.visibility")
local ordersMod = require("simulation.orders")

local M = {}

--- Advances ticksPassed ticks; no shift boundary, no phase gating.
function M.advance(state, ticksPassed)
  local events = {}

  for _ = 1, ticksPassed do
    state.gameTime.tick = state.gameTime.tick + 1
    for _, level in pairs(state.levels) do
      local minedEvents = miningMod.process_tick(state, level)
      for _, e in ipairs(minedEvents) do
        events[#events + 1] = e
      end
      if #minedEvents > 0 then
        reachabilityMod.recompute(level)
        visibilityMod.recompute(level)
      end
    end
    local orderEvents = ordersMod.settle(state)
    for _, e in ipairs(orderEvents) do
      events[#events + 1] = e
    end
  end

  return {
    processedTicks = ticksPassed,
    events = events,
  }, nil
end

return M
