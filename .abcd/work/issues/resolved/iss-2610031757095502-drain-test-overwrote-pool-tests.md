---
schema_version: 1
id: "iss-2610031757095502"
slug: "drain-test-overwrote-pool-tests"
severity: "major"
category: "lapse"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031317470004"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/drain_test.go"
remedy: "Check a test file's name is free before writing a new one; restore the file byte-identical and put the new test in its own file."
resolution: "drain_test.go restored byte-identical; the new test lives in update_drain_test.go."
impact: internal
resolved_by:
  commit: "edf79db"
---

The new pool drain test was written over internal/runtime/drain_test.go, an existing file of the same name, and took six budget-accounting tests with it (a replacement waiting for its victim to exit, the drained charge given back, and the bounds on a drain wait). Caught by the review before the branch was pushed. Never on main.
