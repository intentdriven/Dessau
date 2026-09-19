---
schema_version: 1
id: "iss-2609190023124431"
slug: "the-gateway-test-testoverridesarenamedandnevervalued-interna"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running the full suite for the model-search fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/measurement_test.go"
resolution: "Duplicate of iss-2609181159253108: TestOverridesAreNamedAndNeverValued substring-matched the sentinel 777 against the record's own Unix timestamp; fixed by PR 79 (the stamp is asserted as a stamp and excluded from the search, the sentinels widened)."
impact: internal
---

The gateway test TestOverridesAreNamedAndNeverValued (internal/gateway/measurement_test.go) asserts that no client-supplied sampling value appears anywhere in the marshalled measurement record by searching the whole JSON for the substrings "0.123456", "777" and "0.5". The record carries an epoch timestamp, so the test fails for everyone whose clock happens to sit in a window where the epoch contains one of those digit strings — it fails today on origin/main with at=1789777370, unchanged and without the branch's edits. The assertion should look at the fields that could carry a value (or a record marshalled with the timestamp zeroed), not at the raw JSON.

## Grounds

- pursued: we expect the four lanes that met the same failure in the same window to have recorded one defect; wrong if any of them names a distinct cause
