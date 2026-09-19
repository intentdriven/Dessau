---
schema_version: 1
id: "iss-2609190150313993"
slug: "testexecprocessreportsafootprintwhilealiveandnoneafter-in-in"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "running the full suite for the panel's grace placeholders"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/footprint_test.go"
---

TestExecProcessReportsAFootprintWhileAliveAndNoneAfter in internal/runtime fails whenever the whole suite runs at once (go test -race ./...), and passes every time the package is run alone. It fails in 3.01 s, which is footprintTimeout exactly (internal/runtime/launcher.go), so the reading that came back empty is /usr/bin/top not answering within three seconds on a Mac busy running the rest of the suite. Reproduced twice against a branch that touches only internal/ui, so it is not that change. Two things sit behind it: the test is load-sensitive and will flake in CI on a loaded runner, and a timed-out listing is read as a footprint of zero in production too, so a Mac under load charges nothing for a model server that is in fact resident. The test needs to stop depending on top answering within a fixed three seconds, and the zero-on-timeout reading needs a decision of its own: a listing that did not answer is no reading, which is not the same as a footprint of zero.
