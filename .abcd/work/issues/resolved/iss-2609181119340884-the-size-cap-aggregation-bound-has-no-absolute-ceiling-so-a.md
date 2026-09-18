---
schema_version: 1
id: "iss-2609181119340884"
slug: "the-size-cap-aggregation-bound-has-no-absolute-ceiling-so-a"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "adversarial review of fix/open-captures-2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/stats/dashboard_bound_test.go"
resolution: "Added aggregateCeiling, a flat sixty-second second guard beside the multiple, for the regression the multiple cannot see: one inside parseLine, which moves yardstick and bound together. It is set far above the 1.4 s an idle Mac and the 4 s a loaded one take, so it decides nothing else."
impact: internal
---

the size-cap aggregation bound has no absolute ceiling, so a regression inside parseLine moves the yardstick and the bound together and is not caught
