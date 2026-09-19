---
schema_version: 1
id: "iss-2609190204371327"
slug: "testexecprocessreportsafootprintwhilealiveandnoneafter-in-in"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "running the full suite for an unrelated fix in internal/lifecycle"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "Duplicate of iss-2609190040226948 and iss-2609190208243645: the live-footprint test assumed a quiet Mac, and Footprint's Output waited on a killed listing's pipe past its own bound; both fixed by PR 111 (retry within a budget, loud skip when every listing overran; cmd.WaitDelay so an abandoned listing is no reading)."
impact: internal
---

TestExecProcessReportsAFootprintWhileAliveAndNoneAfter in internal/runtime/footprint_test.go fails intermittently on an otherwise clean tree: it starts /bin/sleep 30, immediately asks execProcess.Footprint() for the process's memory, and reports 'a live process reports 0' roughly two runs in three on an Apple Silicon Mac where top itself answers correctly for the same pid a moment later. The reader shells out to top, and a process that has only just been started has no figure to report yet, so the test is asserting on a value that is not there for the first fraction of a second. It is not a failure of the code under test; it makes go test -race ./... red at random and trains a reader to ignore a red suite. The test should wait for the figure, or assert the reader's contract without racing the process's own start-up.

## Grounds

- pursued: we expect the seven lanes that saw 'a live process reports 0' under a load average above 400 to have recorded one defect; wrong if any of them fails once the fix is in
