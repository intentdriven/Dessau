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
---

TestExecProcessReportsAFootprintWhileAliveAndNoneAfter in internal/runtime asserts that a just-started /bin/sleep reports a footprint above zero. The reader shells out to top, which on a heavily loaded machine returns nothing in the time it is given, so the test fails while the rest of the suite passes and passes again when run on its own. A test of the reader should not depend on how busy the machine running it is: it should either give top a bound it can meet under load or treat an unanswered read as a skip rather than a failure.
