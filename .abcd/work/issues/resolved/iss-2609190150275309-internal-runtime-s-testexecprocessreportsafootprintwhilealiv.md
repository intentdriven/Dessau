---
schema_version: 1
id: "iss-2609190150275309"
slug: "internal-runtime-s-testexecprocessreportsafootprintwhilealiv"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "running the full suite for the sidebar-search fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "Duplicate of iss-2609190040226948 and iss-2609190208243645: the live-footprint test assumed a quiet Mac, and Footprint's Output waited on a killed listing's pipe past its own bound; both fixed by PR 111 (retry within a budget, loud skip when every listing overran; cmd.WaitDelay so an abandoned listing is no reading)."
impact: internal
---

internal/runtime's TestExecProcessReportsAFootprintWhileAliveAndNoneAfter fails on a loaded machine: Footprint() shells out to /usr/bin/top under a footprintTimeout, and a top that does not answer in time is reported as 0, which the test reads as a live process reporting no footprint. On a Mac at load average 400 the test failed on every one of six runs, on this branch and on a pristine origin/main checkout alike, so it is the environment and not any change. The production behaviour is deliberate (a listing that does not answer is no reading), but the test asserts a figure that depends on a three-second budget for an external process, so a busy CI runner can redden the suite for a reason that is not a defect. It should either tolerate a timed-out reading or exercise parseTopMem against a real listing without the deadline.

## Grounds

- pursued: we expect the seven lanes that saw 'a live process reports 0' under a load average above 400 to have recorded one defect; wrong if any of them fails once the fix is in
