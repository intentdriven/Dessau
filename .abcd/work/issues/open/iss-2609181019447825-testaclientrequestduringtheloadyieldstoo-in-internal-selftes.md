---
schema_version: 1
id: "iss-2609181019447825"
slug: "testaclientrequestduringtheloadyieldstoo-in-internal-selftes"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "PR 58 merge queue, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest_test.go"
---

TestAClientRequestDuringTheLoadYieldsToo in internal/selftest flaked in the merge queue on PR 58 (run 35331651173, 'gave up waiting for a load to start' after 5 s under the release configuration); the second queue run passed the same code. The wait for the fake server's load to start is a fixed five seconds on a loaded runner.
