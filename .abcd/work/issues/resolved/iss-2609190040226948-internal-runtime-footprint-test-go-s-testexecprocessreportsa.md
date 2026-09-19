---
schema_version: 1
id: "iss-2609190040226948"
slug: "internal-runtime-footprint-test-go-s-testexecprocessreportsa"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running internal/runtime with -count=10 for the soft-hold change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
resolution: "The test is split: the deterministic half (a process that is done reports nothing, which runs no listing at all) is its own test and always runs, and the live half asks for a reading again within a 20s budget and, when every listing took longer than the 3s footprintTimeout, says so and skips rather than failing the reader for the load on the machine. The assertion itself is untouched: a reading that arrives must still be positive."
impact: internal
---

internal/runtime/footprint_test.go's TestExecProcessReportsAFootprintWhileAliveAndNoneAfter fails intermittently under load ('a live process reports 0'): it starts a real process and asserts the production footprint reader returns a positive figure, and that reader shells out to top, which under a loaded machine (a -race -count=10 run of the package) can return nothing in time. It passes ten times in isolation. The assertion needs to tolerate a reading the machine could not take, or the reader needs a retry.

## Grounds

- pursued: we expect a test of the real top reader to distinguish a wrong figure from a sample this Mac could not take, because only the first is the reader's fault; wrong if the skip starts firing on a quiet machine or in CI, which would mean the reader, not the load, cannot produce a reading in 20s.
