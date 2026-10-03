---
schema_version: 1
id: "iss-2610031010360026"
slug: "panel-calls-a-reservation-resident"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "investigation of a peer report about the memory line"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
remedy: "Word it as a reservation, for example 'memory budget: 76.8 GiB of 76.8 GiB reserved by 1 loaded model (weights 16.6 GiB)', print GiB for binary figures, and match the docs."
resolution: "The memory line now reads as a reservation by the models in memory with their weights, and the panel's binary figures are labelled GiB/MiB/KiB; docs/memory-budget.md matches."
impact: fix
resolved_by:
  commit: "73d511a"
---

The control panel calls a reserved charge 'resident'. The memory line reads 'memory: <x> of <y> budget resident' (internal/ui/static/app.js) where the figure is the sum of each loaded model's charge (weights x 1.2 plus the KV charge for its whole served window and every decode sequence, internal/gateway/control.go), and the default served window is derived to fill the budget, so one loaded model routinely reads as the whole budget resident while far less memory is in use; docs/memory-budget.md describes the budget against 'what is resident'. The figures are binary but printed as GB. Found 2026-10-03 from a peer's screenshot (76.8 GB of 76.8 GB with one 16.6 GB model loaded).
