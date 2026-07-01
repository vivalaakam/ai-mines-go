-- Deterministic PRNG (Park-Miller minimal standard LCG), self-contained in Lua so
-- generation determinism never depends on the Go host or its math/rand implementation.
-- All arithmetic stays below 2^53 so it is exact under Lua 5.1 double-precision numbers.
local MODULUS = 2147483647 -- 2^31 - 1 (Mersenne prime)
local MULTIPLIER = 48271

local RNG = {}
RNG.__index = RNG

local function mix(hash, value)
  if type(value) == "string" then
    for i = 1, #value do
      hash = (hash * 31 + value:byte(i)) % MODULUS
    end
    return hash
  end
  return (hash * 31 + (math.floor(value) % MODULUS)) % MODULUS
end

--- Hashes an arbitrary mix of strings/numbers into a seed in [1, MODULUS-1].
function RNG.hash_seed(...)
  local hash = 2166136261 % MODULUS
  for _, v in ipairs({ ... }) do
    hash = mix(hash, v)
  end
  if hash <= 0 then
    hash = hash + MODULUS - 1
  end
  return hash
end

function RNG.new(seed)
  local self = setmetatable({}, RNG)
  self.state = seed % MODULUS
  if self.state <= 0 then
    self.state = self.state + MODULUS - 1
  end
  return self
end

--- Returns a float in [0, 1).
function RNG:next()
  self.state = (self.state * MULTIPLIER) % MODULUS
  return self.state / MODULUS
end

--- Returns an integer in [minInclusive, maxInclusive].
function RNG:next_int(minInclusive, maxInclusive)
  local span = maxInclusive - minInclusive + 1
  return minInclusive + math.floor(self:next() * span)
end

--- Picks an index from a list of weights (>=0), proportionally. Returns nil if all weights are 0.
function RNG:weighted_index(weights)
  local total = 0
  for _, w in ipairs(weights) do
    total = total + w
  end
  if total <= 0 then
    return nil
  end
  local roll = self:next() * total
  local acc = 0
  for i, w in ipairs(weights) do
    acc = acc + w
    if roll < acc then
      return i
    end
  end
  return #weights
end

return RNG
