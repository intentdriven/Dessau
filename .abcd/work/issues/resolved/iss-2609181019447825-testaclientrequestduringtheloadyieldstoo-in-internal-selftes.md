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
resolution: "waitFor's patience is now the test binary's own deadline less a margin (floored at five seconds) rather than a fixed five, so a loaded build runner is reported as slow rather than as a broken loop; a new test holds the bound to the deadline. 30 runs of TestAClientRequestDuringTheLoadYieldsToo green under -race."
impact: internal
---

TestAClientRequestDuringTheLoadYieldsToo in internal/selftest flaked in the merge queue on PR 58 (run 35331651173, 'gave up waiting for a load to start' after 5 s under the release configuration); the second queue run passed the same code. The wait for the fake server's load to start is a fixed five seconds on a loaded runner.

## Grounds

- pursued: the wait now measures nothing about the machine — it exists only to turn a wedged loop into a named failure, and it is given whatever the run itself has. What would show it wrong: a wedged loop that now takes minutes to report instead of seconds, which is what -timeout is for.
