---
schema_version: 1
id: "iss-2609190153283787"
slug: "testexecprocessreportsafootprintwhilealiveandnoneafter-in-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running the full race suite for the bubble-colour fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "Duplicate of iss-2609190040226948 and iss-2609190208243645: the live-footprint test assumed a quiet Mac, and Footprint's Output waited on a killed listing's pipe past its own bound; both fixed by PR 111 (retry within a budget, loud skip when every listing overran; cmd.WaitDelay so an abandoned listing is no reading)."
impact: internal
---

TestExecProcessReportsAFootprintWhileAliveAndNoneAfter in internal/runtime asserts that a just-started /bin/sleep reports a footprint above zero. The reader shells out to top, which on a heavily loaded machine returns nothing in the time it is given, so the test fails while the rest of the suite passes and passes again when run on its own. A test of the reader should not depend on how busy the machine running it is: it should either give top a bound it can meet under load or treat an unanswered read as a skip rather than a failure.

## Grounds

- pursued: we expect the seven lanes that saw 'a live process reports 0' under a load average above 400 to have recorded one defect; wrong if any of them fails once the fix is in
