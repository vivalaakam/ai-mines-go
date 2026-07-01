local ordersMod = require("simulation.orders")
local workersMod = require("simulation.workers")

local M = {}
M.SHIFT_LENGTH = 300 -- ticks (REQUIREMENTS.md §6)

--- Ends the current shift: releases shift-scoped worker assignments, settles
--- orders, requests autosave, and moves the game into shift_planning.
function M.complete_shift(state)
  local events = {}

  for _, worker in pairs(state.workers) do
    if worker.assignmentMode == "shift_task" and worker.state ~= "idle" then
      workersMod.detach_worker(state, worker)
      worker.state = "idle"
    end
  end

  events[#events + 1] = { type = "shift_completed", shiftIndex = state.gameTime.shiftIndex }

  local orderEvents = ordersMod.process_shift_end(state)
  for _, e in ipairs(orderEvents) do
    events[#events + 1] = e
  end

  state.phase = "shift_planning"
  events[#events + 1] = { type = "autosave_requested", reason = "shift_completed" }
  events[#events + 1] = { type = "shift_planning_started" }
  return events
end

return M
