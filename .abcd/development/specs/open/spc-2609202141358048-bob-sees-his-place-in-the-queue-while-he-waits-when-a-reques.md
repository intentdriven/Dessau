---
id: spc-2609202141358048
slug: bob-sees-his-place-in-the-queue-while-he-waits-when-a-reques
intent: itd-2609202108185678
origin: researcher-authored
production_mode: hand-written
---

# The queue notice: a client tags its request and asks the server where that request stands

## Summary

This spec delivers itd-2609202108185678: a client may tag a completion
request with an id of its own making, and while it waits may ask the gateway
where that request stands; the server answers about that request and no
other — which of the two waits it is in, and its place among the requests
ahead of it for that model — and Dessau Chat shows the answer as a line under
the message once a wait has lasted longer than a threshold in its Settings,
three seconds by default. A check-in is not a request: it lands in no
statistic by construction and is written to the operational log. A client
that never tags or asks gets answers byte-identical to today's. Impact:
additive.

This intent lands AFTER the fair turn (spc-2609202154054806): the ordering
rule that spec sets at the turn queue is what a place in that queue means,
and the place is read from the structure that spec builds.

## Scope

In: `internal/gateway/gateway.go` (reading the tag, binding it, the check-in
endpoint on the OpenAI routes), `internal/runtime/pool.go` (a ticket the
pool marks with the wait a request is in, and a `Place` reader);
`client/DessauChat/` (the tag, the poll, the line, the Settings control);
`internal/archtest` (the client pins); `docs/queue-notice.md` (new
reference), `docs/response-headers.md` (a pointer), `client/README.md` (the
setting). Out: the on-device model and the Discord bridge
(cond-2609202141358693); anyone else's request (cond-2609202141354159); any
promise about when the answer starts (cond-2609202141353671); the statistics
(cond-2609202141350621).

## Approach

**The tag.** One request header, `X-Dessau-Request`, whose value the client
chooses: 1 to 64 characters from `[A-Za-z0-9._-]`, opaque to the server. A
value outside that shape, or the header on any other route, is ignored — the
request is served as if untagged, never refused over it, so the completions
path's refusals are exactly what they were. Dessau Chat sends a fresh UUID
per request; any client may send anything of that shape. The tag is written
to the log only as its first eight characters, the way a fingerprint is, and
appears in no statistic. It never reaches the model server: the upstream
request is built from scratch with `Content-Type` and `Accept` alone, which
is why nothing has to be stripped.

**What "own" means.** A tag is bound to the scheduling identity of the
request that carried it, which is adr-2609202200197975's identity: for a
paired client the SPKI fingerprint `pairedOnly` sets
(`runtime.WithPairedIdentity`), and for an unpaired client the source
bucket `withAuth` sets (`runtime.WithSource`: the API key on a keyed install,
`""` for loopback). A check-in is answered only when the asking request's
identity equals the binding's. For a paired client that is one person; for
unpaired clients it is the class, because the server cannot tell two holders
of the shared key apart and does not pretend to — which is why the tag must
be unguessable and the docs page says so. Binding to the connection was
considered and rejected: the check-in must arrive on a second connection,
because the first is busy streaming. A tag nobody bound, a tag another
identity bound, and a check-in from a client the residency rule does not
entitle are all answered the same: `404 {"error": "no such request"}`.

**The binding.** `Gateway` gains `tags`, a mutex-guarded map from
`{identity, tag}` to `*runtime.Ticket`, written in `handleCompletions` once
the model has resolved and before `Acquire`, and removed by a `defer` when
the handler returns — so it is bounded by the requests in flight and holds
nothing after an answer ends. A second request from the same identity with
the same tag replaces the binding (the client reused its own id; the older
request is served regardless, because the tag decides nothing about
serving). `Ask` binds nothing: the bridge's requests are never tagged.

**The ticket, and the two waits read from the pool's own structures.**
`runtime.WithTicket(ctx, t *Ticket)` is a context tag beside `WithSource`,
`WithSoftHold`, `WithResidentOnly` and `WithPairedIdentity`, carried the way
they are because a LAN client cannot reach one. `acquire` marks the ticket,
under `p.mu` as it goes, with the wait the request is in:

- **`load`** — the intent's "waiting for the model to load, or for memory to
  come free for it": the request is parked in `p.waiters` for room (the
  room wait, oldest first), or is blocked on `e.ready` while its model
  loads. The two never overlap for one model: a waiter leaves `p.waiters`
  the moment the model has an entry. `loadWaiter` gains the folded `repoID`
  it is waiting for, so the place among "those waiting for that model" can
  be counted; `entry` gains `loading []*Ticket`, appended at `entered` and
  removed when the ticket leaves, so the wait on `e.ready` — today a channel
  every waiter shares with no order, the design review's finding — has an
  arrival order to count in.
- **`turn`** — the request is in `e.turns`, the priority queue
  spc-2609202154054806 builds where a bare `chan struct{}` was. The place is
  the request's rank under that spec's ordering rule: one plus the number of
  waiters `grantTurnsLocked` would serve before it, paired by least debt
  then arrival, unpaired after every paired one. This is what this spec
  adds to a wait that today has no position at all: nothing is computable
  on a channel, and nothing is added here that the fair turn did not build.
- **`none`** — the request holds a slot, or has been refused; the binding
  goes when the handler returns.

`Pool.Place(t *Ticket) (wait Wait, place int)` reads all of it under `p.mu`
in one pass over the structures — O(waiters) — and no ticket is ever
consulted on the admission path, so a request that carries none pays
nothing.

**The endpoint.** `GET /dessau/queue/{tag}`, registered in `routes()` beside
`/v1/models`, so it is wrapped by `withAuth` on the plain port and
`pairedOnly` on the TLS port and is reachable from the LAN on the same terms
as a completion. It is deliberately NOT on the control plane, which
`loopbackOnly` gates: the asker is a LAN client asking about its own
request. It answers under `entitled(r)`, the rule the wait headers and the
models list already share, so a fact withheld from one is withheld from the
other and no residency rule moves. The answer:

```
200 {"tag": "<tag>", "wait": "load" | "turn" | "none", "place": 3}
```

`place` is one-based and present for `load` and `turn`; the body carries no
model name and no count of anyone else — the place says only how many are
ahead. A place is where the request stands now (cond-2609202141353671); it
moves when a model is evicted, a request ahead is cancelled, or the fair
turn's debt changes the order, and the docs page says a client reads it as
a place and not as a promise.

**A check-in is not a request.** The handler calls no `observe`, so no
record is built and no statistic can count it — by construction, the way
the self-test's and the tool-call probe's routes count nothing, rather than
by an exemption a counter has to remember. It writes one line at the
detailed level, `queue check-in  model=<repo id> wait=<w> place=<n>
tag=<first 8>`, no more than once a second per binding, because a client
sets the rate and every per-request line this server writes is Debug
(iss-2609190200097326). A miss writes nothing: what would be logged is a
guess.

**Dessau Chat.** On a send to a served model, `ServerBackend` sets
`X-Dessau-Request` to a fresh UUID. A timer starts at the send; once the
threshold passes with no first token, the client asks
`GET /dessau/queue/{tag}` once a second — the residency poll's cadence — and
`AppModel.activity` gains `.waiting(wait, place)`; the line under the
message reads "Waiting for the model to load · 3rd in line", "Waiting for a
turn · next" for place 1, and goes at the first `.text` or `.reasoning`
event (`streamSpoke`). A non-200 answer keeps today's behaviour: the
residency poll in `startResidencyPoll` stays as the fallback for a server
that has no endpoint, and nothing is shown. A wait shorter than the
threshold shows nothing. The on-device model and the bridge never enter
this path: the tag and the poll live on `ServerBackend` alone.

**The setting.** Settings, Server section, under the API key: "Show my place
in the queue after" — a stepper from 1 to 30 seconds, default 3,
`@AppStorage("queueNoticeAfterSeconds")`; the caption says the line appears
only when a request has waited that long. `client/README.md` gains the
sentence.

**Byte identity.** A request without the header takes the code path it took
yesterday: `handleCompletions` reads the header once, and with none present
binds nothing and hands the pool a context with no ticket; the answer's
headers and body are built as before, and the tag adds no header to any
answer — tagged or not. The only new things on the wire are the client's own
header going in and the endpoint's answer coming out.

**Docs.** `docs/queue-notice.md` is a reference page: the header and its
shape, the endpoint, the three answers, what "own" means for a paired and an
unpaired client, that the tag must be unguessable, that a check-in counts in
no statistic and is logged at the detailed level, and that a place is not a
promise. `docs/response-headers.md` gains one sentence pointing at it for
the same fact while it is happening. The `LoadingComment` constant's note in
`gateway.go` stays as it is: still unemitted, still pinned.

## Acceptance Criteria, and what holds each

| # | Criterion | Held by |
|---|---|---|
| 1 | Waiting for a turn: the check-in says `turn` and the place among the requests ahead for that model | `TestACheckInOnARequestWaitingForATurnSaysTurnAndItsPlace` in `internal/gateway` (batch of one, three tagged requests, the places 1, 2 and 3 under the fair turn's order); `TestPlaceInTheTurnQueueFollowsTheGrantOrder` in `internal/runtime` |
| 2 | Waiting for the model to load or for room: the check-in says `load` and the place among those waiting for that model | `TestACheckInOnARequestWaitingForItsModelSaysLoadAndItsPlace` (one parked for room, one on the load, another model's waiter not counted); `TestPlaceAmongRoomWaitersCountsOnlyThoseForTheSameModel` in `internal/runtime` |
| 3 | A request the asker did not tag itself is answered with nothing about it | `TestACheckInOnAnotherClientsTagIsAnsweredNotFound` (paired A's tag asked by paired B, by a keyed plain-port client and unbound: three identical 404 bodies) |
| 4 | Past the threshold the line shows the wait and the place, moves, and goes at the first token; a shorter wait shows nothing | `TestChatClientTagsItsRequestsAndAsksAboutThem` in `internal/archtest` (the Swift source's header name, path and default threshold held to the gateway's constants); `QueueNoticeAppearsAfterTheThresholdAndGoesAtTheFirstToken` in the client's UI test tier (iss-2609181116225273) against a fake server that answers the endpoint |
| 5 | A check-in appears in no statistic and is written to the log | `TestACheckInCountsInNoStatisticAndWritesOneLogLine` (statistics on, ten check-ins, the view's request count unchanged, one Debug line per second per binding with the model and the place and no full tag) |
| 6 | No notice for the on-device model or the bridge | `TestChatClientShowsNoQueueNoticeForTheOnDeviceModel` in `internal/archtest` (the poll and the header live on `ServerBackend` only); `TestTheBridgeBindsNoTag` in `internal/gateway` (an `Ask` leaves `tags` empty) |
| 7 | A client that never tags or asks gets byte-identical answers | `TestAnUntaggedRequestIsAnsweredByteForByteAsBefore` (a golden of headers and body against the fake upstream, streamed and not) and `TestATaggedRequestsAnswerCarriesNothingNew` (the same golden with the header present) |

Every test is watched red before the change; the runtime tests use the
pool's injectable clock.

## Security

The change touches `internal/gateway` (network input) and `internal/runtime`
and gets the adversarial review before the pull request. What it answers:

- **A LAN client asking about tags it guesses.** A miss costs a map lookup
  and returns the one 404; a hit needs the asker to hold the binding's
  identity AND the tag. For a paired client the identity is the handshake's
  and cannot be borrowed. Among holders of the shared key the tag is the
  only secret, so the shape allows 64 characters and Dessau Chat uses 122
  random bits; what a lucky guess learns is a place number for a model it
  is not told the name of.
- **Enumeration.** There is no listing; the 404 is identical for unbound,
  foreign and unentitled; timing differs by one map lookup, not by a pool
  walk, because the walk happens only after a hit.
- **The tag's bound.** 64 characters of a closed alphabet: no log injection,
  no unbounded map keys; a malformed value binds nothing and is served.
- **Rate of check-ins.** Each hit is one `p.mu` pass bounded by
  `MaxLoadWaiters + MaxQueueDepth`, the same order of work the models list
  does per request; a flood is the existing per-request cost of any GET on
  the port and is refused by nothing new. Logging is once a second per
  binding, so a flood cannot write at its own rate.
- **The residency rule.** The endpoint answers under `entitled(r)`, so an
  open server tells a LAN client nothing it did not already withhold.
- **Nothing persisted.** The binding is in memory for the life of the
  handler; the ticket is the pool's; neither reaches the store or the
  configuration.

## Verification

- `make test`, `gofmt -l .` empty, `go vet ./...` clean; the client built
  with `client/build.sh` and its UI tier green.
- The named tests green, each watched red first;
  `TestChatClientRecognizesTheGatewaysLoadingComment` still green, since the
  comment's pin is untouched.
- A `make run` hand check on the shipping line: statistics on, a model at a
  batch of one, two Dessau Chat instances; the second's line appears after
  three seconds saying "Waiting for a turn · next", goes when its answer
  begins; `/api/stats` shows two requests and no more; the detailed log shows
  the check-in lines with eight characters of the tag.
- The security review before the pull request; a docs-currency review of
  `docs/queue-notice.md` and `docs/response-headers.md`.

## Out of scope

- **SSE comment lines.** The interview chose the check-in over an in-body
  notice: a comment line before the first token commits the answer to 200
  before `Acquire` returns, after which a refusal can no longer be a clean
  503 (the design review's most severe finding), and it would need an ADR
  for a channel inside the response body. The reserved `LoadingComment`
  marker stays declared and unemitted, its note in `gateway.go` unchanged,
  and the client keeps recognising it.
- **Headers after the fact.** No position header is added to the answer;
  `X-Dessau-State` and `X-Dessau-Queue-Time` stay what they are. The intent's
  title carries the seed's sentence about headers; the criteria confirmed at
  the interview carry none, and this spec follows the criteria.
- **The on-device model and the bridge.** No queue on-device; the bridge has
  its own typing indicator and binds nothing.
- **A live queue view on the control panel.** The panel shows the
  contention from the fair turn's statistics; a per-request live view is not
  a setting and not a gap.
- **The order itself.** What a place means is spc-2609202154054806's.

## Departures

None at the time of writing.
