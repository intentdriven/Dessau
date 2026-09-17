---
schema_version: 1
id: "iss-2609161712555136"
slug: "a-queued-measure-now-whose-model-fails-the-fit-check-is-drop"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "manual context-probe check 2026-09-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest.go"
---

A queued 'Measure now' whose model fails the fit check is dropped silently on every tick. selftest.Runner.tick asks Server.Fits for a due job's model and, when it is false, moves on with no held_by, no log line and the model left in probe_queue forever; the panel shows the model queued and the manual check saw job null, held_by null for 3 h 40 min. MeasureNow should refuse up front with the pool's too-large text (what served window or batched-requests count would fit), or the tick should report a held_by naming no room.
