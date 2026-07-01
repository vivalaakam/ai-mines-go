-- Default rulesConfig. Open design decisions (REQUIREMENTS.md §43) are config-driven, not hard-coded.
return {
  allowWorkerReassignmentDuringShift = false,
  orderAllocationMode = "priority_based", -- "priority_based" | "proportional"
}
