# Game Design

See `REQUIREMENTS.md` at the repository root for the full game design specification
(map generation, workers, storage, orders, resources). This file is a stub
for design notes that diverge from or extend that specification during implementation.

## Orders

New orders are not refilled immediately after accept/decline. They are rolled on
`orderArrivalIntervalTicks` boundaries while the available pool is below
`maxAvailableOrders`. The roll is deterministic from seed and tick, but long
miss streaks are capped: after `orderArrivalMaxMisses` missed eligible rolls,
the next eligible boundary guarantees one new order.
