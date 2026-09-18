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
deferred_after: "v0.7.0"
deferral_reason: "The 0.7.0 cut ships the macOS 27 client; the silent drop is in the self-test runner the release does not touch, and its fix is the next item of the probe lane (recorded 2026-09-18)."
resolution: "A due job whose model fails Server.Fits is no longer passed over in silence: the tick reports it as held_by no_room with due naming the model, and says once per model in the log that the measurement is held because the model does not fit the memory budget. The panel already renders held_by and now words the new reason; docs/context-probe.md names it among what holds a run. The HTTP API is unchanged — a new value on a field that already exists. The deferral recorded for v0.7.0 is discharged."
impact: fix
---

A queued 'Measure now' whose model fails the fit check is dropped silently on every tick. selftest.Runner.tick asks Server.Fits for a due job's model and, when it is false, moves on with no held_by, no log line and the model left in probe_queue forever; the panel shows the model queued and the manual check saw job null, held_by null for 3 h 40 min. MeasureNow should refuse up front with the pool's too-large text (what served window or batched-requests count would fit), or the tick should report a held_by naming no room.

## Grounds

- pursued: a runner test queues a job whose model the fake server says does not fit, and holds the status to no_room and the model, the job to never having run, and the log to one line however many ticks pass. What would show it wrong: a hold reported while a self-test run is also due that tick, which the loop still lets run and reports on the next tick instead.
