---
schema_version: 1
id: "iss-2609202010357676"
slug: "a-resident-only-acquire-can-wait-on-another-caller-s-reload"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "scoped security re-review of the tool-call probe fixes (itd-2609201445423499), pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
---

A resident-only Acquire can wait on another caller's reload instead of returning ErrNotResident promptly. If the model is evicted and another caller starts reloading it between the probe's Resident pre-check and the pool's lookup, the WithResidentOnly caller waits on that load for up to ReadyTimeout rather than getting ErrNotResident; no load is launched and nothing is evicted, so it is a liveness edge, not a safety one. Suggested: re-check ResidencyLoaded inside the resident-only branch before waiting. Found by the Sonnet security re-review of the four fix commits (LOW, non-blocking); it touches internal/runtime and the record does not decide it, so it is captured rather than fixed in the lane.
