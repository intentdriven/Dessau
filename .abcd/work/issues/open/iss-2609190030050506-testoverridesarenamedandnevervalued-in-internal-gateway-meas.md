---
schema_version: 1
id: "iss-2609190030050506"
slug: "testoverridesarenamedandnevervalued-in-internal-gateway-meas"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "baseline test run before starting the client-pairing lane"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/measurement_test.go"
---

TestOverridesAreNamedAndNeverValued in internal/gateway/measurement_test.go asserts that the statistics record does not carry a client-supplied sampling value by searching the whole marshalled JSON record for the string form of that value (777). The record also carries an epoch timestamp, so the test fails whenever the current second happens to contain the digits 777 — observed failing at at=1789777795 on 2026-09-19 with no code change. The assertion should be scoped to the fields that could carry a value (the overrides array and the sampling fields), not to the whole record's bytes.
