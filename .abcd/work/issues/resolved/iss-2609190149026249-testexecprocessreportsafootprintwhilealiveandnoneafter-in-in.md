---
schema_version: 1
id: "iss-2609190149026249"
slug: "testexecprocessreportsafootprintwhilealiveandnoneafter-in-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running the full suite for the sidebar card fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "Duplicate of iss-2609190040226948 and iss-2609190208243645: the live-footprint test assumed a quiet Mac, and Footprint's Output waited on a killed listing's pipe past its own bound; both fixed by PR 111 (retry within a budget, loud skip when every listing overran; cmd.WaitDelay so an abandoned listing is no reading)."
impact: internal
---

TestExecProcessReportsAFootprintWhileAliveAndNoneAfter in internal/runtime is flaky on a loaded Mac: it starts /bin/sleep and asserts execProcess.Footprint() is above zero, but the reader shells out to top, and when the machine is busy top returns no MEM figure in time, so the live process reports 0 and the test fails. It failed once in a full -race run and again under -count=2, and passed on a quieter run, with no Go change of any kind in the branch. Either the reader should be given a figure it can always obtain, or the test should tolerate a reading top declined to give and say so rather than failing.

## Grounds

- pursued: we expect the seven lanes that saw 'a live process reports 0' under a load average above 400 to have recorded one defect; wrong if any of them fails once the fix is in
