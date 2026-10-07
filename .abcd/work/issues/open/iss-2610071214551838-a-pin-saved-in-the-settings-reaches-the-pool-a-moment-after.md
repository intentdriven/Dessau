---
schema_version: 1
id: "iss-2610071214551838"
slug: "a-pin-saved-in-the-settings-reaches-the-pool-a-moment-after"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "security review of the idle-unload pin race fix (iss-2610042033419572), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Call Pool.SetPinned before releasing cfgMu on every settings write that changes pins, after confirming that no path holding p.mu ever takes cfgMu (the pool's callbacks into the app, such as the served-window and budget reads), with a test that holds the lock order; if that order cannot be guaranteed, say so and leave the window documented."
---

A pin saved in the settings reaches the pool a moment after the configuration says it is pinned. The settings writers (internal/app/app.go, the save and the spelling adoption around line 1188) write the per-model settings under cfgMu, release it, then call Pool.SetPinned, which takes p.mu. An idle job's unload in that gap reads the pool's pin set, which does not yet hold the new pin, and stops a model the operator has just pinned. The window is microseconds wide; iss-2610042033419572 closed the larger window between the idle job's own check and its stop, and the security review of that fix named this remaining one.
