---
schema_version: 1
id: "iss-2610031757103220"
slug: "parallel-test-extra-place-update-race"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031317470004"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest.go"
remedy: "Treat ErrNotNow from holdMore as the refusal at the load is treated: nothing recorded."
resolution: "holdMore's ErrNotNow skips the run like a refusal at the load."
impact: fix
resolved_by:
  commit: "9bede54"
---

The self-test's parallel test takes its further places after the load, so a drain that starts between them still recorded the run as a failed load. Never on main.
