local applyMod = require("apply")
local readMod = require("read")
local stateMod = require("state")

local engine = {}
local currentState = nil

--- Not part of the required public API (REQUIREMENTS.md §33) but needed by the
--- Go host to bootstrap a state before load_state exists on disk.
function engine.new_game(seedPhrase)
  currentState = stateMod.new_game(seedPhrase)
  return true
end

function engine.apply(command)
  return applyMod.dispatch(currentState, command)
end

--- Wraps read results in the same {ok, ...} envelope as apply so the Go host has
--- one error contract for both entry points (branches on error.code, never message).
function engine.read(query)
  local result, err = readMod.dispatch(currentState, query)
  if err then
    return { ok = false, error = err }
  end
  return { ok = true, data = result }
end

function engine.export_state()
  return stateMod.export_state(currentState)
end

function engine.load_state(state)
  currentState = stateMod.load_state(state)
  return true
end

_G.engine = engine
return engine
