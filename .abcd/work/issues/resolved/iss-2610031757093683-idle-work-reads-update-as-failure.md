---
schema_version: 1
id: "iss-2610031757093683"
slug: "idle-work-reads-update-as-failure"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031317470004"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/selftest.go"
remedy: "Map ErrUpdating to selftest.ErrNotNow (nothing recorded, the day not used) and to toolprobe.ErrGone."
resolution: "The self-test skips a model refused for now and the probe drops it as gone."
impact: fix
resolved_by:
  commit: "edf79db"
---

With the drain, a model being updated refuses every new Acquire with ErrUpdating, and the self-test recorded that as a failed load against a healthy model (using up its day) while the tool-call probe logged a recorded-nothing failure. Never on main.
