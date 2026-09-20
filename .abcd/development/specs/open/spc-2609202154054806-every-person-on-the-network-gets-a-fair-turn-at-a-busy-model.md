---
id: spc-2609202154054806
slug: every-person-on-the-network-gets-a-fair-turn-at-a-busy-model
intent: itd-2609202108180709
origin: researcher-authored
production_mode: hand-written
---

# The fair turn: a turn queue at every loaded model, paired clients by least debt, the unpaired class from what is left

## Summary

This spec delivers itd-2609202108180709: the next free turn at a loaded model
goes to the paired client that has had the fewest answers while other paired
clients waited, ties by arrival; unpaired clients are one class, served only
when no paired request is waiting for that model, in arrival order among
themselves, and refused with the existing `waited` state once they have
waited longer than the queue's maximum wait. The decision lives on the pool's
slot-admission path, where today a bare channel admits whoever Go wakes
first; it is always on; the debt is in memory only. What is written down is
the contention — one log line per deferral, and per model per minute how
many waited, for how long, and how many unpaired requests were deferred —
never the client. The identity it schedules on is adr-2609202200197975's.
Impact: breaking on the statistics store's shape (fields added to the
request line and the summary line; pre-1.0, no migration, an older line
reads its new fields as zero); additive everywhere else.

This intent lands BEFORE the queue notice (spc-2609202141358048), because
the ordering rule this spec sets is what a "place in line" means.

## Scope

In: `internal/runtime/pool.go` (the turn queue, the debt, the unpaired
bound, the log line, a context tag for the paired identity);
`internal/gateway/pairing.go` (`pairedOnly` sets the tag), `ask.go` (the
bridge's identity), `gateway.go` (wait headers on the new refusal);
`internal/app` (feeding the pool the queue's maximum wait whether or not
grace is on); `internal/stats` (two request-line fields, three summary
fields, a per-model-per-minute contention rollup in the view);
`internal/ui` (a "Waiting" table on the Statistics tab);
`internal/archtest` (the no-address and no-persistence tests);
`docs/fair-turn-explained.md` (new), `docs/response-headers.md`,
`docs/request-statistics.md`, `docs/statistics-store-reference.md`,
`docs/logging.md`, `docs/eviction-grace.md`, `docs/pairing-explained.md`.
Out: the wait for room and the wait for a load, which keep their oldest-first
rule (cond-2609202154055090); any identity but a pairing (cond-2609202154055143);
the on-device model and the bridge's own queue (cond-2609202154059706);
`config.json`, which gains no setting.

## Approach

**Where the decision lives.** Today a request that has its model ready
claims a turn with `case e.sem <- struct{}{}` (`pool.go`, the slot select
after `<-ready`); `e.sem` is a buffered channel whose capacity is the batch
the model server decodes, and its queue is whatever order Go wakes blocked
senders in — the design review's finding that the layer has no scheduler.
The channel is replaced by two things on `entry`: `running int`, bounded by
the batch capacity `e.sem` carried, and `turns`, an ordered set of
`turnWaiter{arrived time.Time; who identity; signal chan struct{}}` — the
priority queue. A request whose model is ready joins `turns` under `p.mu`;
`grantTurnsLocked(e)` runs under the same lock whenever a slot frees
(`releaseSlot`) or a waiter joins, hands each free slot to the waiter the
rule below picks, and signals it. The wait stays off the lock and bounded by
the caller's context, as today. `MaxQueueDepth` keeps its meaning: it bounds
`inFlight`, which counts `running` and `turns` together, and the `ErrBusy`
refusal at admission is untouched.

**The identity the pool receives.** A new context tag,
`runtime.WithPairedIdentity(ctx, id string)`, beside `WithSource`,
`WithSoftHold` and `WithResidentOnly`, carried in the context for the reason
those are: a LAN client cannot reach one, so a caller can only ever name
itself. `pairedOnly` sets it to the SPKI fingerprint the handshake proved,
where it already sets `WithSource("client:"+spki)`; `Gateway.Ask` sets it to
a fixed identity of the server's own for the bridge (`bridge:discord`), so
the bridge's requests are one paired client's (criterion 9). Every other
acquisition — the shared key on the plain port, loopback, a keyless LAN, the
self-test and the tool-call probe — carries none and is the unpaired class.
`WithSource` is not reused for this because its value is the load-waiter
queue's key and includes the API key, which is not an identity and must not
become one. The pool reads no address, anywhere: `RemoteAddr` is not a name
the package knows.

**The debt rule.** `entry.debt map[string]int` — per paired client, per
model, on the model's own entry. Per model rather than across models,
because the thing contended for is a turn at one model: Alice's script
working through a stack of files on a small model says nothing about who
should be served next at a large one, and a cross-model debt would let a
person's heavy use of one model put them behind a newcomer at another where
nobody had waited. It is bounded by construction: at most `config.MaxClients`
(64) pairings plus the bridge can hold a key. When `grantTurnsLocked` gives a
turn to a paired client's request while a request of a DIFFERENT paired
client is in `turns` for that model, the served client's debt rises by one;
served with nobody else's request waiting, it does not (criterion 2). The
debt is a fact about one episode of contention: it starts at zero when the
first request has to wait and is cleared for the model when `turns` empties.
A debt carried across an idle stretch would make a person who used the model
alone an hour ago wait behind a newcomer now — the "busy client held back
while nobody waits" case the intent names as its own falsification — and
within an episode the rule already holds: a waiting client whose debt is
lowest is served as soon as every client served ahead of it has been counted
once, so a paired person is passed over at most once per other paired client
before their turn comes. That is the answer to the design review's
starvation-the-other-way finding. The debt lives in memory only: it is on
the entry, gone when the model leaves memory and when the pool closes, and no
type outside `internal/runtime` can reach it (criterion 6).

**The order, stated once.** When a slot frees for model M:

1. If any paired request is in `turns`, the turn goes to the paired request
   whose client has the least debt at M; among equal debts, the earliest
   `arrived` — the moment the request joined `turns` (criterion 1).
2. Otherwise the earliest-arrived unpaired request (criterion 5).

An unpaired request is therefore never served while a paired one waits
(criterion 3), and a paired request alone on an idle model finds a free slot
without joining `turns` at all, which is today's fast path unchanged
(criterion 2). Nothing in it slows an idle model: `turns` is only consulted
when `running` is at the batch.

**The unpaired cap.** An unpaired request in `turns` for longer than the
queue's maximum wait is refused (criterion 4). The bound is the figure the
operator already has for "how long a request may wait before it is refused":
`eviction_max_wait_sec`, read through `Config.MaxWaitSeconds()` (default 300
s, ceiling an hour). Today the pool only receives it while grace is on
(`evictionGraceFor` returns zeros otherwise), so the pool gains
`PoolOptions.MaxTurnWait` with a live `SetMaxTurnWait`, and `internal/app`
feeds it `MaxWaitSeconds()` at start and at every save regardless of the
grace switch; the config comment and `docs/eviction-grace.md` say the figure
now also bounds an unpaired turn wait. The refusal is a new
`*runtime.TurnWaitError{Waited time.Duration}` that unwraps to `ErrBusy`
(class `busy` at the gateway, 503), and `handleCompletions` sets the wait
headers for it as it does for `NoRoomError` — one `WaitedFor() time.Duration`
interface the two refusals share, so the answer says `X-Dessau-State: waited`
and the time. A paired request has no such bound: it is never starved by the
rule, only by the client's own context. `config.json` gains nothing, so the
settings-surface tests are unaffected in both directions.

**The log line.** Each time `grantTurnsLocked` gives a turn to one request
while passing over another — a paired request outranked by debt, or an
unpaired request held for a paired one — it writes, off the lock like every
other pool line, one line at the detailed level:

```
held a request back for another   model=<repo id> held=paired|unpaired waiting=<n> rule=debt|paired_first
```

— the model, the class of the held request, how many were in `turns`, and
which rule applied; no fingerprint, no source, no name (criterion 7). The
detailed level because it is a per-request line and every per-request line
this server writes is Debug (iss-2609190200097326). So that the sparse level
still shows the contention, one Info line per model per episode — "requests
are waiting for a turn at a model", model and count — rate-limited by the
existing `logEvery` pattern; `docs/logging.md`'s "never written" list needs
no change and the page gains the two lines.

**Statistics.** The request line gains `turn_wait_ms` (the part of
`queue_wait_ms` spent in `turns`; `queue_wait_ms` keeps its meaning as the
whole) and `deferred` (how many times this request was passed over under the
paired-first rule — non-zero only for an unpaired request, so "how many
unpaired requests were deferred" is the count of lines with `deferred > 0`
and no class field is needed). `AcquireStats` gains the two figures and the
observer copies them. The recorder gains a per-model-per-minute
`Contention` rollup in the view — `{minute, model, waited, wait_ms,
deferred}`, kept for `RollupWindow` like `Rollup` — and the daily summary
gains `turn_wait_ms_total`, `waited_requests` and `deferred_total` per model
so the figures outlive the detail's drop. The store carries the facts per
request with `model` and `at`, from which the per-model-per-minute figures
fold; no new record kind is added, so adr-2609121450000000 stands. Every new
numeric is bounded at the recorder as the windows are; the byte-scanning
tests are extended for the fields; the record still names no client
(criterion 8, the store half). `internal/archtest`'s
`TestTheStatisticsPageNamesEveryFieldTheStoreWrites` holds the two docs
pages to the new fields by construction.

**The Statistics tab.** A "Waiting" table on the Statistics tab, beside the
request table: one row per model with requests that waited in the window,
the mean and longest turn wait, and unpaired requests deferred, drawn from
the contention rollup; the History view's per-day tables gain the same three
figures from the summary (criterion 8, the panel half). Nothing is settable,
so the three-surfaces obligation is met by the table alone.

**Docs.** `docs/fair-turn-explained.md` is a new explanation page — why a
turn queue rather than arrival order, why paired people first, why the debt
counts only under contention, what an unpaired client on a shared Mac should
expect — rather than an edit to `docs/eviction-grace-explained.md`, whose
"served oldest first" describes the room queue and stays true.
`docs/pairing-explained.md` gains a sentence: pairing is also what puts a
person ahead of scripts at a busy model. `docs/response-headers.md` lists
the new 503 among those that carry the headers.

## Acceptance Criteria, and what holds each

| # | Criterion | Held by |
|---|---|---|
| 1 | Two or more paired clients waiting: the turn goes to the least debt, ties by arrival | `TestTheTurnGoesToThePairedClientWithTheLeastDebtAndTiesGoByArrival` in `internal/runtime` (batch of one, three paired identities, the grant order asserted) |
| 2 | A paired client alone on an idle model is served at once and builds no debt | `TestAPairedClientAloneOnAnIdleModelIsServedAtOnceAndBuildsNoDebt` (request after request, `turns` never joined, debt stays zero) |
| 3 | An unpaired request waits while any paired request waits for the same model | `TestAPairedRequestIsServedBeforeAnUnpairedOneWaitingForTheSameModel` |
| 4 | An unpaired request past the queue's wait is refused with `waited` and the time | `TestAnUnpairedRequestPastTheQueuesWaitIsRefusedWithTheWaitStated` in `internal/runtime` and `TestARefusedTurnWaitCarriesTheWaitHeaders` in `internal/gateway` (503, `X-Dessau-State: waited`, the time) |
| 5 | No paired request waiting: unpaired requests are served in arrival order | `TestWithNoPairedRequestWaitingUnpairedRequestsAreServedInArrivalOrder` |
| 6 | The debt is in memory only and names no client anywhere | `TestTheDebtGoesWithTheModelAndWithThePool` in `internal/runtime`; `TestNoSchedulingStateReachesTheDiskOrTheLog` in `internal/archtest` (the debt type is referenced only in `internal/runtime`; `RemoteAddr` appears nowhere in that package; the fingerprint and source strings appear in no log call the pool makes) |
| 7 | One log line per deferral naming the model, the class and the count, never a client | `TestHoldingARequestBackWritesOneLineNamingTheModelTheClassAndTheCount` (a captured logger; the line's keys asserted; the identity string asserted absent) |
| 8 | Per model per minute: how many waited, for how long, unpaired deferred — store and tab, naming no client | `TestContentionIsCountedPerModelPerMinute` and `TestARecordHoldsOnlyCountsTimingsAndTheModel` (extended) in `internal/stats`; `TestTheSummaryKeepsTheContentionTotals`; `TestTheWaitingTableShowsTheContentionPerModel` in `internal/ui`; the docs-to-code pin in `internal/archtest` |
| 9 | The bridge's requests are one paired client's; the on-device model is outside | `TestTheBridgesRequestsAreOnePairedClients` in `internal/gateway` (an `Ask` acquisition carries the bridge identity, and two of them share one debt); the on-device model never reaches the server, held already by the client's architecture tests |

Every test is watched red before the change, as the definition of done
requires; the runtime tests use the pool's injectable clock so no wall-clock
wait is paid.

## Security

The change is inside two trust boundaries (`internal/gateway`,
`internal/runtime`) and gets the adversarial review before the pull request.
What it must answer:

- **A client forging a pairing identity.** The identity is set only by
  `pairedOnly`, from the certificate the TLS handshake verified against the
  paired set, and by `Ask` for the bridge; it travels in the context, which
  no header reaches. The plain port's `withAuth` sets none, so a plain-port
  request carrying any header claims nothing. A test on the plain port
  asserts a request with a forged `client:` source string in a header lands
  in the unpaired class.
- **A burst of unpaired requests.** Cannot take a turn from a paired person
  at all; can fill `turns` up to `MaxQueueDepth`, at which point the existing
  `ErrBusy` refusal applies to everyone, paired included — that bound is
  unchanged by this spec and is the same lever it was. Each unpaired waiter
  is bounded in time by the queue's wait, so the goroutines and bodies a
  burst pins are released at that bound rather than at the client's leisure.
- **A burst from one paired client.** Builds debt from the first turn another
  paired client waits for, and is then served at most once per other paired
  client per round; it cannot lower anyone else's place.
- **Starvation of unpaired requests is deliberate and bounded.** They wait
  behind paired requests for at most the queue's maximum wait and are then
  refused with the wait stated; the docs page says so, and the class is what
  curl on a shared Mac is in.
- **Nothing persisted.** The debt is on the entry; the archtest holds that the
  type reaches no store, and the store's byte-scan holds the new fields to
  numbers.

## Verification

- `make test`, `gofmt -l .` empty, `go vet ./...` clean.
- The nine named tests green, each watched red first.
- `TestThePlainPortDoesNotMoveWhenAClientPairs` still green: the plain port's
  admission is untouched.
- A `make run` hand check on the shipping line: statistics on, one paired
  client and one unpaired script against a model at a batch of one; the
  script's requests wait while the person's are answered, the Waiting table
  shows the model with the counts, the detailed log shows the deferral lines
  with no fingerprint, `stats-*.jsonl` shows `turn_wait_ms` and `deferred`
  and no client anywhere.
- The security review of `internal/gateway` and `internal/runtime` before
  the pull request; a docs-currency review of the seven pages.

## Out of scope

- Address identity of any kind (adr-2609202200197975).
- A Settings switch: the fair turn is always on. There is nothing to turn off
  on an idle model, and a switch would put a fairness rule in the operator's
  hands as a policy.
- The wait for room and the wait for a load: both stay oldest first
  (cond-2609202154055090); the debt does not reach into them.
- Decay of the debt by time: it is cleared by the end of an episode, not by a
  clock.
- A per-client rate limit (captured as iss-2609202200256227, declined here).

## Departures

None at the time of writing.
