---
schema_version: 1
id: "iss-2609190114530680"
slug: "adr-2609091239058072-rule-1-says-gateway-control-settingsmu"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/2609091239058072-the-app-s-four-locks-have-one-order-the-settings-handler-s-a.md"
resolution: "Recorded as a dated decision line: the lock order is unchanged and the invariant holds, what no longer describes the callers is the record's account of why holding it across SetConfig is safe. The consequence that was real — a stalled body wedging every save — is closed by reading before locking."
impact: internal
---

adr-2609091239058072 rule 1 says gateway.Control.settingsMu 'is taken only by the settings handler, and only from an HTTP handler goroutine'. The pairing work adds two more callers, PairHandler and handleRevoke, and one of them is an unauthenticated network endpoint rather than the loopback-only control plane. The lock ORDER is unchanged — settingsMu then saveMu, nothing below takes either — so the invariant holds; the record's account of why it is safe to hold across SetConfig no longer describes its callers. An ADR is never edited once ratified, so this needs a superseding record or, at least, the decision line that says what changed.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
