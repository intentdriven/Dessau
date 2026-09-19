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
---

internal/runtime/footprint_test.go's TestExecProcessReportsAFootprintWhileAliveAndNoneAfter fails intermittently under load ('a live process reports 0'): it starts a real process and asserts the production footprint reader returns a positive figure, and that reader shells out to top, which under a loaded machine (a -race -count=10 run of the package) can return nothing in time. It passes ten times in isolation. The assertion needs to tolerate a reading the machine could not take, or the reader needs a retry.
