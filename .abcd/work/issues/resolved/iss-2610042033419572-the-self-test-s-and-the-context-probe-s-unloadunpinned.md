---
schema_version: 1
id: "iss-2610042033419572"
slug: "the-self-test-s-and-the-context-probe-s-unloadunpinned"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "independent review of fix/selftest-honours-pins, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/contextprobe.go"
remedy: "Add a pool-level UnloadUnpinned in internal/runtime that checks the pin under the same hold of p.mu as the stop and refuses a pinned model with a sentinel the idle jobs map to their own ErrPinned; route unloadUnpinned through it. internal/runtime is a trust boundary, so the change needs an adversarial security review."
resolution: "Pool.UnloadUnpinned reads the pin under the same hold of p.mu as the stop; both idle jobs route through it and map runtime.ErrPinned to their own ErrPinned"
impact: fix
---

The self-test's and the context probe's unloadUnpinned (internal/app/contextprobe.go) checks the pin with isPinned and then stops the model with Pool.Unload; each takes the pool's lock separately, so a pin saved in the instant between the check and the stop is lost: the idle job unloads a model a person has just pinned, leaving it pinned and not loaded.
