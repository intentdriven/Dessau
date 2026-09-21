---
id: itd-2609211335097114
slug: a-stuck-model-never-holds-the-server-hostage-alice-asks-her
spec_id: spc-2609211753023984
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091301112705, itd-2609100457007827, itd-2609061441241254, itd-2609061441285238, itd-2609091412177263]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A stuck model never holds the server hostage: a real request pre-empts idle work

## Press Release

A stuck model never holds the server hostage.

Bob asks a question from his laptop on the network while the Dessau Server in
Alice's study is measuring a model in the background — loading it, even. He
gets his answer. The server notices that his request needs the memory its own
idle work is holding, stops that work at once, and loads the model he asked
for; the pause is a few seconds at most, and if the memory does not come back
within the server's own bound he is told, in the refusal, what held it and that
it was the server's work rather than the size of what he asked for. The
measurement the server abandoned loses nothing it had verified: it resumes
later from where it stood. What Bob never sees is a 503 he has to retry, and
what Alice never sees is a server that answers nobody for hours because it is
busy with a job it started itself.

Alice can see the same thing from the control panel: a model held by idle
work says which job holds it and for how long, and one click frees it, as it
does today. A model she has pinned is the one thing idle work cannot give up
on her behalf: a request that needs a pinned model's memory is refused,
naming the holder, and the pin stays.

## Why This Matters

On 2026-09-21 the live server answered 503 to every chat request for six hours
because its context probe had queued a model the runtime cannot load and
retried it thirty-two times, each attempt holding the whole memory budget
(iss-2609211334563318, iss-2609211334570516, iss-2609211334576018; fixed in
0.9.3 by PR 144, which makes such a load fail in seconds, keeps a failed model
out of idle work, restricts the probe to chat models, and names the holder in
the refusal). What 0.9.3 leaves in place is the contract underneath: a request
that needs memory an idle job holds is refused once and served on its retry
(itd-2609100457007827's audit; the self-test's watcher), and a model that is
still loading is never evicted at all, whoever holds it. The self-test and the
tool-call probe already yield to a real request through the pool's soft hold
(2026-09-19); the context probe cannot, because it drives the gateway like a
client and a client's hold is never soft (2026-09-11, 2026-09-19). So the
outage's shape — idle work holding a loading model while people wait — is
narrower now but still possible, and the one refusal it costs lands on chat
clients that do not retry. This intent closes that: idle work yields to a real
request, including while its model is loading, and the person asking is parked
for a bounded moment rather than refused.

## Mechanism

We expect a real request to be served rather than refused while an idle job
holds the memory because the pool already parks a pre-empting client and tears
down a soft-held ready model for it (the self-test's yield since 2026-09-19),
so the work is to extend two existing seams — the probe takes its hold through
the pool with the same soft-hold tag, and the eviction plan may name a loading
entry whose only holder is soft — rather than to add a scheduler. We are wrong
if a torn-down load leaves the pool's accounting or the model's measurement
inconsistent (a partial figure published, a charge not released, a child not
reaped within the drain bound), or if the park routinely exceeds the drain
bound because a child mid-prefill ignores SIGTERM, in which case the promise
degrades to today's refuse-then-retry.

## Scope Conditions

- One serving process per Mac, the pool's memory budget as the only arbiter of what fits; Apple Silicon, the pinned mlx-lm runtime. <!-- cond: cond-2609211753023388 -->
- Idle work only: the context probe, the self-test and the tool-call probe are the holders a real request may pre-empt; a hold taken for a client's request is never pre-empted, and no network client can declare or trigger a soft hold. <!-- cond: cond-2609211753029766 -->
- A pinned model is never freed for a request, whoever holds it; the client is refused naming the holder. <!-- cond: cond-2609211753021602 -->
- The park is bounded by the pool's drain limit (derived from the child stop timeouts, stated in the docs); past it the client is refused naming the holder, not left waiting. <!-- cond: cond-2609211753022201 -->
- A yield leaves no partial figure: a yielded probe resumes from its last verified size, a yielded self-test records a yield, not a failure. <!-- cond: cond-2609211753029769 -->
- The silent-child readiness residual (a child that answers `/health`, writes no traceback and never answers a completion) is outside this intent — its own capture. <!-- cond: cond-2609211753026059 -->

## Acceptance Criteria

- **Given** the context probe holding a model, **when** a request needs that
  memory, **then** the probe's hold is a soft hold taken through the pool and
  the pool pre-empts it exactly as it does the self-test's.
  *Held by* a pool test on the probe's hold tag, and an architecture test that
  no soft-hold tag is reachable from a gateway request.
- **Given** an idle job's model still loading and a request that needs its
  memory, **when** the request arrives, **then** the loading child is torn
  down, the request is parked, and it is served without a 503.
  *Held by* a pool test driving a slow fake load.
- **Given** a parked request, **when** the teardown exceeds the drain bound,
  **then** the request is refused naming the holder and the bound; the bound
  is derived from the stop timeouts and stated in `docs/`.
  *Held by* a pool test with a child that ignores SIGTERM, and a docs test on
  the figure.
- **Given** a pinned model an idle job is using, **when** a request needs its
  memory, **then** the pin stays resident and the request is refused naming
  the holder; the probe's own unload never unpins.
  *Held by* a pool test and a probe test.
- **Given** a probe step or self-test run that is pre-empted, **when** it
  yields, **then** no partial figure is published, the probe resumes from its
  last verified size, and the self-test records a yield.
  *Held by* the existing yield tests, widened to the pre-emption path.
- **Given** a client's own request holding a model, **when** another request
  needs the memory, **then** nothing changes from today: a client's hold is
  never pre-empted.
  *Held by* the existing eviction tests staying green.
- **Given** the panel, **when** an idle job holds a model, **then** the card
  names the job and for how long, and Unload frees it as today.
  *Held by* a node test over the card.
- **Given** `docs/context-probe.md` and `docs/self-test.md`, **when** read,
  **then** they say a real request pre-empts idle work, the park bound, and
  that a pin is never freed.
  *Held by* the docs-currency hand check on the shipping line.

## Decisions at the planning interview (2026-09-21)

Maintainer's answers, each recorded in `.abcd/work/DECISIONS.md`:

1. **Scope: the full residual.** A real request is served, not refused-then-
   retried, while any idle job holds the memory — including while the job's
   model is still loading.
2. **Mechanism: a pool-side soft hold for the probe.** The probe holds its
   model through the pool with the self-test's soft-hold tag, then drives the
   gateway path as before. This supersedes the "keeps internal/runtime
   untouched" clause of the 2026-09-11 harness decision; the harness shape
   (the gateway path, the gateway's bounds) stands.
3. **A loading model held only by idle work is torn down** for a real request.
   This reverses "never evict a model that is still loading" (2026-08-02) for
   idle-only holders and for nothing else: a client's load and a pin are
   untouched.
4. **The pin wins.** A pinned model is never freed for a request; the client
   is refused naming the holder. The probe's own unload path, which ignores
   pins today, is a bug captured separately.
5. **The park is bounded by the runtime's drain limit**, derived from the stop
   timeouts, never invented; past it the client is refused naming the holder.
6. **The silent-child residual is its own capture**, flagged as reversing the
   step-floor decision of 2026-09-21 if its fix needs to.
7. **Impact additive, severity minor**: clients only gain; after 0.9.3 the
   residual is one refusal-then-retry.

Typed links: refines itd-2609091301112705 (answers its open question on
whether the yielding seam becomes real pre-emption); refines
cond-2609100502034205 of itd-2609100457007827 ("the pool has no preemption",
narrowed a second time); supersedes one clause of the 2026-09-11 decision and
one rule of 2026-08-02 as above.

## Open Questions

- Whether the pool's drain bound as derived today (about twenty seconds with
  eviction grace off) is acceptable as the stated park, or the stop timeouts
  are tightened for a soft-held child; the spec's design review proposes,
  from measurements on the scratch root.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect that once idle work yields to real requests, nobody on the network need know the server measures models in the background; we are wrong if the box's statistics after the run still show refusals attributed to an idle holder, or if pre-emption makes the probe's figures unstable
