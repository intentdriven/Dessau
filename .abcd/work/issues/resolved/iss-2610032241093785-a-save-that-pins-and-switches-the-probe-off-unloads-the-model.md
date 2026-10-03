---
schema_version: 1
id: "iss-2610032241093785"
slug: "a-save-that-pins-and-switches-the-probe-off-unloads-the-model"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the probe-pins fix (iss-2609211754251373)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Apply the pins before the idle-job switches in SetConfig, so a run stopped by the save meets the pin at its stop's unload."
resolution: "SetConfig applies Pool.SetPinned before applyIdleJobs; TestASaveThatPinsAndSwitchesTheProbeOffKeepsTheModel watched to fail first."
impact: fix
resolved_by:
  commit: "0c5dc7e"
---

One settings save that pins the model the context probe is measuring and switches the probe off unloads that model: SetConfig applies the idle-job switches (applyIdleJobs, which cancels the run and waits for it) before it applies the pins (Pool.SetPinned), so the stopped run's own unload reads the old pin set and stops the model the operator has just pinned, leaving it pinned and not loaded. Every time, not a race.
