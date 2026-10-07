---
schema_version: 1
id: "iss-2610042100405045"
slug: "the-control-panel-does-not-show-a-program-s-unload-when-a"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "fidelity review rcp-92f55f59b36b of itd-2610031024247803, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
remedy: "Show released unloads in the panel's Statistics tab beside evictions, with the caller kind, held by a UI test against the field Go publishes."
resolution: "The stats counters count released unloads by caller kind (ModelCounters.released); the Statistics tab shows them beside evictions, held by a UI test over the marshalled Go struct"
impact: additive
---

The control panel does not show a program's unload. When a client unloads a model through /api/models/unload, the statistics store records the reason released and the kind of caller (itd-2610031024247803, criterion 10), and the log line says so. But the panel's Statistics tab counts evictions only and never shows a released unload (internal/ui/static/app.js, around line 2430). An operator using the panel cannot see that a program unloaded a model, though Go records it. AGENTS.md treats a Go capability with no panel equivalent as a gap.
