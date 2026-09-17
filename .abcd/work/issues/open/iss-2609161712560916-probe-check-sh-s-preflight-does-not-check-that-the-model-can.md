---
schema_version: 1
id: "iss-2609161712560916"
slug: "probe-check-sh-s-preflight-does-not-check-that-the-model-can"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual context-probe check 2026-09-16"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/evidence/2026-09-06-model-bench/"
---

probe-check.sh's preflight does not check that the model can fit the budget before it quits the menu-bar Gropius for hours. The 2026-09-16 run started against a snapshot whose warnings already said the budget was smaller than the smallest model, then polled a queue that could never start until the Mac froze. The preflight should read /api/state's warnings and the model's charge against the budget and refuse to start a model that cannot fit, naming the served window and batched-requests count that would.
