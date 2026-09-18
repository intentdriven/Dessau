---
schema_version: 1
id: "iss-2609181119346098"
slug: "the-no-room-hold-is-suppressed-whenever-any-other-model-runs"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial review of fix/open-captures-2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest.go"
resolution: "Every due job is now asked rather than the loop stopping at the first that can run, so every non-fitting model is logged (once per model, by the existing ledger) and the hold is reported whether or not another run goes ahead: HeldBy is no_room and Due the held model even on a tick that measures something else. A new test measures one model repeatedly while another is held."
impact: fix
---

the no_room hold is suppressed whenever any other model runs that tick and only the first non-fitting model is remembered, so a queued model can stay invisible for as long as the self-test has work

## Grounds

- pursued: the new test was watched failing on the clearing behaviour it replaces. What would show it wrong: two jobs held at once, where only the first is named on Due while both are logged.
