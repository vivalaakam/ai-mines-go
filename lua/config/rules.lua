-- Default rulesConfig. Open design decisions (REQUIREMENTS.md §43) are config-driven, not hard-coded.
return {
  orderAllocationMode = "proportional", -- "priority_based" | "proportional"
  orderArrivalIntervalTicks = 100, -- new orders may arrive when tick % interval == 0
  orderArrivalChance = 0.5, -- probability of an arrival on each interval boundary
  orderArrivalMaxMisses = 3, -- guaranteed arrival after this many missed boundary rolls
  orderShipmentIntervalTicks = 50, -- accepted orders are shipped/paid on this cadence
  maxAvailableOrders = 3, -- cap on simultaneously available (unaccepted) orders
  orderLifetimeTicks = 1500, -- how long an available order stays before expiring
}
