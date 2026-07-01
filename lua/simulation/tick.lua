local miningMod = require("simulation.mining")
local reachabilityMod = require("simulation.reachability")
local visibilityMod = require("simulation.visibility")
local shiftsMod = require("simulation.shifts")

local M = {}

local function err(code, message)
  return { code = code, message = message, details = {} }
end

--- Advances up to ticksPassed ticks, never crossing more than one shift boundary
--- in a single call (REQUIREMENTS.md §6). Returns {processedTicks, remainingTicks, events}.
function M.advance(state, ticksPassed)
  if state.phase ~= "shift_running" then
    return nil, err("not_shift_running", "Ticks can only be processed while a shift is running")
  end

  local ticksIntoShift = state.gameTime.tick % shiftsMod.SHIFT_LENGTH
  local ticksRemainingInShift = shiftsMod.SHIFT_LENGTH - ticksIntoShift
  local processed = math.min(ticksPassed, ticksRemainingInShift)
  local events = {}

  for _ = 1, processed do
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
  end

  if processed >= ticksRemainingInShift then
    local shiftEvents = shiftsMod.complete_shift(state)
    for _, e in ipairs(shiftEvents) do
      events[#events + 1] = e
    end
  end

  return {
    processedTicks = processed,
    remainingTicks = ticksPassed - processed,
    events = events,
  }, nil
end

function M.fast_forward_to_shift_end(state)
  if state.phase ~= "shift_running" then
    return nil, err("not_shift_running", "Cannot fast-forward outside a running shift")
  end
  local ticksIntoShift = state.gameTime.tick % shiftsMod.SHIFT_LENGTH
  local remaining = shiftsMod.SHIFT_LENGTH - ticksIntoShift
  return M.advance(state, remaining)
end

function M.start_next_shift(state)
  if state.phase ~= "shift_planning" then
    return nil, err("not_in_planning", "Can only start a shift from shift_planning")
  end
  state.phase = "shift_running"
  state.gameTime.shiftIndex = state.gameTime.shiftIndex + 1
  return { shiftIndex = state.gameTime.shiftIndex, events = { { type = "shift_started", shiftIndex = state.gameTime.shiftIndex } } }, nil
end

return M
