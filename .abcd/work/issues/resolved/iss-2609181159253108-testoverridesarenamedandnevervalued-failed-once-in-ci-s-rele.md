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
resolution: "Not a timing flake: TestOverridesAreNamedAndNeverValued marshalled the whole record and searched the text for the client's values, one of which was the three digits 777, and the record's arrival stamp is a wall clock. From 1789777000 to 1789777999 UTC — 2026-09-19, sixteen minutes — every run read the 777 inside the timestamp as the client's max_completion_tokens, which is why it failed on both pushes to main and passed on every rerun outside the window. The stamp is now asserted as a stamp and then excluded from the search, and the sentinels are long enough not to be spelled by accident (max_completion_tokens 54321, top_p 0.98765). Reproduced 5/5 at epoch 1789777298 and watched pass 20/20 at 1789777375, inside the same window."
impact: internal
---

TestOverridesAreNamedAndNeverValued failed once in CI's release-configuration test job on PR 64 (run 35339539628) and passed on the pull-request run of the same head and on the rerun; a records-only PR, so the test's own inputs did not change. Suspected timing or ordering flake; establish by running it with -count and -race under load.

## Grounds

- pursued: we expect this test to stop failing because the only field whose digits nobody chooses is no longer searched and the remaining sentinels are five or more characters; wrong if some other recorded figure comes to spell 54321 or 0.98765, or if excluding the stamp lets a client's value reach it unnoticed
