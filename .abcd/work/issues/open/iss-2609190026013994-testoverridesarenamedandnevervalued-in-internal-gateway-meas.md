---
schema_version: 1
id: "iss-2609190026013994"
slug: "testoverridesarenamedandnevervalued-in-internal-gateway-meas"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running the gateway suite while adding the statistics record's source class"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/measurement_test.go"
---

TestOverridesAreNamedAndNeverValued in internal/gateway/measurement_test.go asserts that no client-chosen value reaches a record by searching the record's whole encoded JSON for the strings 0.123456, 777 and 0.5. The encoding includes the record's own 'at' field, a unix second, so the test fails whenever the wall clock's epoch second contains the substring 777 — observed failing at at=1789777544 on 2026-09-19. It should search the record with the fields the client cannot influence excluded (or marshal a copy with At zeroed) rather than the whole object.

## Deferral 2026-09-19

Found while adding the statistics record's `source` class for the Discord
bridge (itd-2609180959397172). The defect is in a test that this lane does not
otherwise touch, and the fix is a change to what that test asserts — which is
a judgement about the assertion's intent, not about the bridge. Left for the
maintainer so the bridge's diff stays the bridge's. It needs no decision
beyond "search the record with `at` excluded", but it is somebody else's test
to narrow.
