---
schema_version: 1
id: "iss-2609190056083748"
slug: "testthehorizonisappliedwithoutanyrotation-while-it-is-runnin"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "running the full race suite while adding the Discord bridge"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/stats/store_test.go"
resolution: "The subtest now gives the store a clock it moves by hand (the package's testClock). It starts where the record does, so the horizon does not reach the record and the 5ms pruner cannot take the file away before the guard has seen it; the horizon is then applied by moving the clock two years on, which is the thing the subtest is about. The guard's meaning is unchanged, and the wait for the pruner is still a retry to a deadline."
impact: internal
---

TestTheHorizonIsAppliedWithoutAnyRotation/while_it_is_running in internal/stats/store_test.go failed once under a full 'go test -race ./...' run with 'nothing was written, so this test proves nothing', and passed on eight repeated runs of the test alone. The subtest depends on the store's background writer having flushed by the time it reads, and under the load of the whole suite on -race it does not always have; the same run logged 'the statistics store's writer did not answer a reading's flush in time'. It should wait for the write rather than assume it, or the guard should be a retry with a deadline.

## Deferral 2026-09-19

Found while running the full race suite for the Discord bridge
(itd-2609180959397172). It is a pre-existing timing assumption in a test this
lane does not touch, and the fix is a judgement about what that subtest means
to assert. Left for the maintainer so the bridge's diff stays the bridge's.

## Grounds

- pursued: we expect a subtest about a background pruner to control when the pruner has something to take, because reading the directory and hoping is a race the machine loses under load; wrong if the store ever read its clock once at open rather than per pass, which would make moving it afterwards do nothing and the subtest pass vacuously — it does not: pruneOnce computes its cutoff from w.now() every pass.
