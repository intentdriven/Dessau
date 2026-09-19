---
schema_version: 1
id: "iss-2609190032382732"
slug: "testoverridesarenamedandnevervalued-in-internal-gateway-meas"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running the full suite before opening the header PR"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/measurement_test.go"
---

TestOverridesAreNamedAndNeverValued in internal/gateway/measurement_test.go asserts that no number the client sent reaches the measurement record by searching the serialized record for the substrings 0.123456, 777 and 0.5. The record also carries its own timestamp, at, as a decimal Unix epoch, so the test fails whenever the current epoch second contains 777 as a substring — it failed on every run at epoch 1789777950 on 2026-09-19 and will do so again for roughly 1000 seconds in every 10000. This is the cause the open record iss-2609181159253108 asks to establish. The assertion should look at the fields the client's values could reach rather than at the whole serialized record.
