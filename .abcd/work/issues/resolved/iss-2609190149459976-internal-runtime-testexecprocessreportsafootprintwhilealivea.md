---
schema_version: 1
id: "iss-2609190149459976"
slug: "internal-runtime-testexecprocessreportsafootprintwhilealivea"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "running the full go test -race ./... suite for the OPINIONS pointer test (iss-2609190042447070)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "Duplicate of iss-2609190040226948 and iss-2609190208243645: the live-footprint test assumed a quiet Mac, and Footprint's Output waited on a killed listing's pipe past its own bound; both fixed by PR 111 (retry within a budget, loud skip when every listing overran; cmd.WaitDelay so an abandoned listing is no reading)."
impact: internal
---

internal/runtime TestExecProcessReportsAFootprintWhileAliveAndNoneAfter is flaky: it fails roughly one run in three with 'a live process reports 0'. The test starts /bin/sleep and immediately asks execProcess.Footprint, which shells out to /usr/bin/top -l 1 with a footprintTimeout; on a loaded Mac that listing can exceed the timeout or come back before the freshly forked child has a MEM figure, and Footprint returns 0 for both cases, which the test reads as a live process reporting nothing. It should be a test that cannot be false-red: give the child a moment or retry the read a bounded number of times before failing, and distinguish 'top did not answer' from 'top said zero' so the failure names which happened. Pre-existing on origin/main; reproduced three times on a branch whose diff touches only internal/archtest and the ledger.

## Deferral 2026-09-19

Left for the maintainer rather than fixed in the same change. It waits on two
things this lane cannot settle. First, `internal/runtime` is a declared trust
boundary — subprocess management — so a change to `Footprint` or its timeout
needs the adversarial security review the conventions require, which is a pass
of its own and not one to bolt onto a test-only branch about the OPINIONS
pointers. Second, the decision it needs is what a footprint of zero from a live
process should MEAN: `Footprint` currently collapses "top did not answer in
time" and "top answered zero" into the same `0`, and whoever fixes the flake has
to choose between widening the return (an error or a sentinel the pool reads)
and keeping the collapse and making only the test patient. That is a design
call about the sampler's contract, not a test tweak, and it belongs to the
maintainer.

## Grounds

- pursued: we expect the seven lanes that saw 'a live process reports 0' under a load average above 400 to have recorded one defect; wrong if any of them fails once the fix is in
