---
schema_version: 1
id: "iss-2609190018027384"
slug: "the-self-test-s-watcher-mistakes-the-loop-s-own-acquisition"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "landing PRs 70 and 71, 2026-09-19"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest.go"
resolution: "The run now claims its place in the pool before it asks for it and drops the claim only after it is released: watch.claim sets the hold count and a new loading flag together, run() claims 1 before Server.Acquire and the parallel test claims spec.Parallel before holdMore, and busy() reads the 'my own load may be among the waiters' signal off hold.loading rather than off count == 0. TestTheLoopDoesNotYieldToItsOwnAcquisitions reproduces the load window deterministically with a 150ms acquireDelay and no client; TestAnIdleMacGetsEveryModelMeasuredAndLeftAsFound now fails with t.Fatalf rather than panicking on a short set."
impact: fix
---

The self-test's watcher mistakes the loop's own acquisition for a client's request and ends the run it is watching. internal/selftest/selftest.go raises the run's hold count (watch.setHold) only AFTER Server.Acquire returns, but the pool counts a load as in flight from the moment Acquire is called: runtime.Pool.Acquire increments e.inFlight before it waits for the model to become ready, and Pool.Residency reports a loading entry. So for the whole of a cold load the model under test shows one in-flight request that busy() cannot attribute to the loop (h.count is still 0), the watcher cancels the run, and it is recorded as yielded with no tests — on a real Mac every cold-load self-test run is lost this way. The same window reopens at each extra acquisition of the parallel test, between holdMore returning and setHold(1+extra): a run that reaches it is cut short with Tests [pp512 tg128] and outcome yielded. That is what fails TestAnIdleMacGetsEveryModelMeasuredAndLeftAsFound and TestALoadThatFailsIsRecordedByClassAndTheLoopMovesOn on the GitHub macOS runner (runs 35407955690 and 35408299438), kicking record-only PRs out of the merge queue. The hold must be claimed before the acquisition is asked for and dropped only after it is released, and the 'my own load is parked' signal that busy() reads off count == 0 has to become a field of its own. Secondary defect in the test: TestAnIdleMacGetsEveryModelMeasuredAndLeftAsFound indexes run.Tests[2] after reporting that the set is short, so a short set panics with index out of range instead of failing with a message.

## Grounds

- pursued: we expect the self-test to complete a cold-load run because the pool's in-flight count for the model under test is now fully attributed to the loop from before Acquire until after release; wrong if the pool ever reports a client's request on that model in a way the count cannot separate, or if a claim raised before the acquisition hides a client that arrives in the same window for longer than one poll
