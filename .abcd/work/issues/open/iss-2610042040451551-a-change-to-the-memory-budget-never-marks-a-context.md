---
schema_version: 1
id: "iss-2610042040451551"
slug: "a-change-to-the-memory-budget-never-marks-a-context"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "independent review of fix/probe-resume-clamps-window, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Re-judge the measurements in SetConfig after the pool's budget is applied, so the provenance reads the budget the save put in force; test by changing MaxResidentBytes through SetConfig and asserting the saved figure reads stale budget."
---

A change to the memory budget never marks a context measurement stale until the next start. SetConfig re-judges every measurement with refreshStaleness before it applies the new budget to the pool with SetMemoryBudget, and the provenance it judges by reads the budget from the pool, so the save is judged against the old budget. Verified in review: a figure stamped with a budget of 82463372083 bytes and a budget of 41231686041 in force reads stale empty, while StaleAgainst on the provenance in force says budget.
