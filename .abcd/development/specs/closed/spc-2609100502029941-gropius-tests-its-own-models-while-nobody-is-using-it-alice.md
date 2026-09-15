---
id: spc-2609100502029941
slug: gropius-tests-its-own-models-while-nobody-is-using-it-alice
intent: itd-2609100457007827
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The idle-time model self-test

## Summary

An opt-in loop that measures every downloaded model on the operator's own
Mac while nobody is using it, with the same short set of tests, and writes
one line of figures per run to a bounded file. It delivers
itd-2609100457007827: the switch on three surfaces, the loop, the standard
set, the results file, and the two docs pages. It does not deliver a panel
view of the results, a fold of the results into the statistics store, or any
setting derived from the figures; those are captured as follow-ups
(iss-2609100515475279, iss-2609100515478018) and the held draft
itd-2609091712142715.

## Scope

- `internal/config`: `Config.SelfTest` (`self_test`, bool, omitempty, off
  by default, no repair and no refusal since a bool has no bad value) and
  `Paths.SelfTest`, the results directory beside the statistics store under
  the account's own directory.
- `internal/selftest`: the loop (`Runner`), the standard set and the request
  builder (`request.go`, allow-listed by name in
  `internal/archtest/prompt_content_test.go` on both sides of the content
  boundary), and the bounded results file (`results.go`).
- `internal/app`: `App.SelfTest`, the `selfTestServer` adapter, and
  `applySelfTest` called from `New`, `SetConfig` and `Close`.
- `internal/ui`: one checkbox on the Settings pane, posted as `self_test`
  and filled from it, with its sync test.
- `docs/self-test.md` (how-to), `docs/self-test-reference.md` (reference),
  a README bullet, a changelog entry; `internal/archtest` holds the reference
  page to the record's fields.

## Approach

**The switch.** `self_test` follows the statistics switch's shape: a bool the
panel posts and the file carries; `SetConfig` applies it live by calling
`SelfTest.SetEnabled`, and `New` applies the saved value at construction.
Off cancels a run in progress and returns once the loop is gone; `Close`
does the same before the pool closes, so the pool's close never waits on a
release that is on its way.

**Idleness, from outside the pool.** The runner never holds an app lock and
adds no mechanism to the pool (adr-2609091239058072). It reads
`Pool.Residency` (per-model in-flight and last-used), `Pool.Waiting` and
`App.Downloading` through the adapter on every tick. Quiet means: nothing in
flight, nothing waiting, nothing downloading, and every resident model's last
use older than five minutes. A run in progress polls the same view every
quarter of a second, subtracting its own one hold on the model under test;
any other in-flight request, any waiter, or a rise in the pool's count of
loads refused for want of room (`Residency.Refusals`, the one seam added to
the pool: with eviction grace off a refused client waits nowhere else
visible) cancels the run's context. The watcher runs from before `Acquire`,
so a cold load yields too, and the pool tears down an abandoned load. The
pool has no preemption, so this is the whole of the yielding: the self-test's
own request or load is cancelled, the run is written as `yielded`, and the
release follows within the poll interval plus the cancellation. A
pool-side preemptible hold, which would spare the refused client its one
503, is captured as a follow-up rather than built.

**Loading through the pool, never by eviction.** `Acquire`, tagged with its
own source for the load-waiter queue, never a second launcher: the budget,
the pins, the served window and the KV charge are the pool's own. Before
asking, the runner checks `Server.Fits`, which the app answers from the
residency snapshot and `chargeOf`: a model that would need another evicted
is left due for a later tick, so a Mac whose working model stays warm all
day keeps it. The parallel test takes the decode concurrency's worth of
acquisitions on the resident model before it sends, so the pool's semaphore
bounds the batch and a client arriving on the model queues where the watcher
sees it. A model the self-test loaded is unloaded afterwards through
`Pool.Unload`, unless the run yielded, in which case the client's request
decides what stays; a model that was resident is left resident.

**The set.** `pp512` (a fixed ~512-token English prompt, one token back),
`tg128` (one sentence, 128 tokens back) and `tg128xN` (the same, N at once,
N the decode concurrency, omitted at one), plus the load time from `Acquire`.
Requests go straight to the upstream the pool hands back — the model server
on loopback, with the exact `ModelArg` the pool reports — as the gateway does
after its own Acquire, so they land in no request statistic and need no API
key. Streaming with `include_usage` always written; the first chunk carrying
text is the first token, and the server's usage event gives the counts, or
the chunk count does when the server sends none, and the result says which.

**The file.** `selftest/results.jsonl`, 0600 in a 0700 directory, opened
with `O_NOFOLLOW`, appended to until the next line would pass 4 MiB and then
truncated: a single bounded file, deliberately not a third rotating writer
(iss-2609091714393599). One `Run` per line; `Test` records inside it. A
failed run carries a class, never the error's text. On start the loop seeds
its last-tested map from the file so a restart does not begin the cycle
over; a model is measured again after a day, the stalest first.

## How it satisfies the acceptance criteria

| Criterion | Where it is held |
| --- | --- |
| Off: nothing loaded, no file | `TestOffLoadsNothingAndWritesNothing`, `TestTheSelfTestFollowsItsSwitch` (app) |
| On and idle: acquired through the pool, the set runs, one line appended | `TestAnIdleMacGetsEveryModelMeasuredAndLeftAsFound` |
| Unload what it loaded, leave what was there | the same, and `TestAModelThatWasResidentIsLeftResident` |
| A client request yields the run, releases the model, records `yielded` | `TestAClientRequestYieldsTheRun`, with a bound on how long the yield takes |
| Busy Mac: nothing runs | `TestNothingRunsWhileTheMacIsBusy` (four cases) |
| Every model fresh: nothing runs | the tail of `TestAnIdleMacGets…`, and `TestAResultStandsForADay…` |
| The line's fields, and no prompt or answer text | `TestAResultCarriesNoPromptAndNoAnswer`, `TestALoadThatFails…` (no error text), and the archtest docs test |
| Switch off mid-run cancels and releases | `TestSwitchingOffStopsARunAndReleasesTheModel` |
| The cap: file started again, newest kept | `TestTheResultsFileIsPrivateAndBounded` |
| The panel offers the switch under the file's key | `TestThePaneIsWiredToTheSelfTestSwitch` (ui) |
| The docs pages | `TestTheSelfTestReferenceNamesEveryField`, `TestTheSelfTestPagesAreLinkedAndSayWhatIsNeverWritten` (archtest) |

## Not in scope, and why

- A results view in the panel: the switch is the accessible surface this
  change needs; the figures are a file the reference page names. Captured.
- Folding the results into the statistics store: its schema is per-request
  and its writer is in another lane; captured, to move with the writer
  unification.
- A line on the posture page saying the self-test is on: the page's lane is
  another session's; captured as iss-2609100515484516.
- Any setting chosen from the figures: itd-2609091712142715, held.
