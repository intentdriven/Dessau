---
schema_version: 1
id: "iss-2609181159253108"
slug: "testoverridesarenamedandnevervalued-failed-once-in-ci-s-rele"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "PR 64 CI, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/measurement_test.go"
---

TestOverridesAreNamedAndNeverValued failed once in CI's release-configuration test job on PR 64 (run 35339539628) and passed on the pull-request run of the same head and on the rerun; a records-only PR, so the test's own inputs did not change. Suspected timing or ordering flake; establish by running it with -count and -race under load.
