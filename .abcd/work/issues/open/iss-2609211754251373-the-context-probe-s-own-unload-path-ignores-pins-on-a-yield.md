---
schema_version: 1
id: "iss-2609211754251373"
slug: "the-context-probe-s-own-unload-path-ignores-pins-on-a-yield"
severity: "minor"
category: "bug"
source: "plan-review"
found_during: "planning interview itd-2609211335097114 (design review finding 7)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/contextprobe/probe.go"
---

The context probe's own unload path ignores pins: on a yield or a held step the probe calls unloadWaiting -> Pool.Unload (internal/contextprobe/probe.go), which unloads whatever it is given, while the eviction plan skips a pinned model before it looks at soft holds ('a pin still beats everything', 2026-09-19) and the probe intent says it never evicts a pinned model. So a pinned model under probe is freed by one path and protected by the other. The pin must win on both paths (decision 4 of itd-2609211335097114's interview); the fix is the probe's, not the pool's.
