---
schema_version: 1
id: "iss-2609181119347301"
slug: "selftest-waitbound-returns-five-seconds-even-when-less-than"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial review of fix/open-captures-2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest_test.go"
resolution: "waitBound now returns the floor or half of what is left of the deadline, whichever is less, so a short -timeout cannot produce a wait that outlives the run; the test reads the bound before the time left, since both come from a moving clock, and asserts against the same clamp."
impact: internal
---

selftest waitBound returns five seconds even when less than five seconds of the test binary's deadline is left, so a short -timeout makes the wait outlive the run it belongs to

## Grounds

- pursued: green at -timeout 4s, 30s and 10m, where it failed at 4s before. What would show it wrong: a wait that now gives up before a condition an idle Mac reaches in milliseconds.
