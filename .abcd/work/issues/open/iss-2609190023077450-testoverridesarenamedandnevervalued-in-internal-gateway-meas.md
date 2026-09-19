---
schema_version: 1
id: "iss-2609190023077450"
slug: "testoverridesarenamedandnevervalued-in-internal-gateway-meas"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "running the full suite for the built-in context trim fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/measurement_test.go"
---

TestOverridesAreNamedAndNeverValued in internal/gateway/measurement_test.go checks that no sampling value the client sent reaches the statistics record by searching the whole marshalled record for the substrings "0.123456", "777" and "0.5". The record carries an "at" field holding the epoch second, so during every wall-clock window whose epoch second contains "777" the test fails on its own timestamp rather than on any leaked value (observed 2026-09-19 with at=1789777354). The check should read the named fields it cares about, or compare against the record's own typed values, rather than substring-matching the serialised record.
