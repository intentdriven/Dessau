---
schema_version: 1
id: "iss-2609112100298761"
slug: "testtimetofirsttokenisthefirstchunkabouttheanswer-in-interna"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "merge queue for PR 45, 2026-09-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/stats_test.go"
resolution: "The fake server now paces the first word after a role-only preamble like any other chunk — it used to send both together, so the test could not tell a figure taken at the preamble from one taken at the first word by anything but machine speed — and the test sets the two 40 ms and 340 ms apart instead of 40 and 40. 30 runs green under -race."
impact: internal
---

TestTimeToFirstTokenIsTheFirstChunkAboutTheAnswer in internal/gateway/stats_test.go failed once in a merge-queue CI run (merge_group run for PR 45) and passed on the pull-request and push runs of the same commit and locally; a timing assertion on the first-token measurement under a loaded runner. Same shape as the other timing flakes (iss-2609091705185072, iss-2609091950213211, iss-2609111104112814): needs an ordering the fake controls rather than a wall-clock bound.

## Grounds

- pursued: the old assertion held nothing — with the gateway mutated to time the first word-bearing chunk it still passed, while the new one fails at 343 ms. What would show it wrong: a runner on which a 40 ms chunk is not delivered within 300 ms.
