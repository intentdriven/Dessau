---
schema_version: 1
id: "iss-2609200823589977"
slug: "nothing-guards-home-in-the-test-suite-the-lifecycle-tests-wr"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "2026-09-19 autonomous sweep, recorded 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/lifecycle"
---

Nothing guards HOME in the test suite: the lifecycle tests wrote into the real ~/Library until PR 83 pointed HOME at a fixture, and any new test that resolves the account directory can do it again. A liveguard-style guard that fails a test which reads the real HOME is missing.
