---
schema_version: 1
id: "iss-2610031756541470"
slug: "aside-left-picks-first-not-newest"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Pick the aside whose attempt (the base-36 start time in its name) is latest."
resolution: "asideLeft picks the newest aside by its attempt time."
impact: fix
resolved_by:
  commit: "fbf4fa6"
---

When the model's folder was missing, the swap put back the first aside copy the directory listing returned, in no particular order, rather than the newest; with more than one left, an older version could be restored.
