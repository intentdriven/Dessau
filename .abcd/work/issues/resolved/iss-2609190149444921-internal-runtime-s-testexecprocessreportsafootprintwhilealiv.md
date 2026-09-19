---
schema_version: 1
id: "iss-2609190149444921"
slug: "internal-runtime-s-testexecprocessreportsafootprintwhilealiv"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running the full suite before opening the device-noun PR"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "Duplicate of iss-2609190040226948 and iss-2609190208243645: the live-footprint test assumed a quiet Mac, and Footprint's Output waited on a killed listing's pipe past its own bound; both fixed by PR 111 (retry within a budget, loud skip when every listing overran; cmd.WaitDelay so an abandoned listing is no reading)."
impact: internal
---

internal/runtime's TestExecProcessReportsAFootprintWhileAliveAndNoneAfter fails on a busy machine and takes exactly footprintTimeout (3s) to do it. Footprint() shells out to /usr/bin/top -l 1 under a context timeout and deliberately returns 0 when the listing does not answer in time, which is the documented production behaviour; the test asserts a live process reports a non-zero footprint with no allowance for that, so it reports 'a live process reports 0' whenever the machine is loaded enough for top to miss the deadline. Reproduced three times in a row on pristine origin/main (1e585ff) in a detached worktree with no changes, and it passed on the same tree when the machine was quieter, so it is the machine's load and not any branch. go test -race ./internal/... is a CI gate, so a loaded runner fails the build for a reason that is not in the diff. The test should either tolerate a timed-out reading or drive the reader through a seam that does not depend on how busy the host is.

## Grounds

- pursued: we expect the seven lanes that saw 'a live process reports 0' under a load average above 400 to have recorded one defect; wrong if any of them fails once the fix is in
