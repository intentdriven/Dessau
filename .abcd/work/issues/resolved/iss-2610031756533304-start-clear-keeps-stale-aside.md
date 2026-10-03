---
schema_version: 1
id: "iss-2610031756533304"
slug: "start-clear-keeps-stale-aside"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Keep an aside only when the model's folder does not hold a version that checks out."
resolution: "An aside beside a model folder that checks out is removed at start."
impact: fix
resolved_by:
  commit: "fbf4fa6"
---

The start's clear, as first changed on this branch, kept every aside copy, including the stale one a swap that finished and died before removing it leaves, so each such interruption would have left a full model copy in the staging folder for ever. Never on main.
