---
id: itd-2609202108180709
slug: every-person-on-the-network-gets-a-fair-turn-at-a-busy-model
spec_id: spc-2609202154054806
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609182357325215, itd-2609061441285238]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every person on the network gets a fair turn at a busy model: when a model has more requests waiting than it can serve, Dessau Server gives the next free slot to the client that has had the fewest answers recently rather than to whoever asked first, so one client sending request after request cannot keep the others waiting. Bob and Carol each get an answer in turn even while a script on Alice's own Mac is working through a stack of files. Nothing changes when the model is idle: the first request in is served at once, and no client is slowed down while nobody else is waiting. A paired client is recognised as itself; an unpaired client on the network is told apart in memory only and never written down.

## Press Release

Bob and Carol each get an answer in turn from a busy model, even while a
script is working through a stack of files on the same server. Dessau Server
gives the next free turn at a loaded model to the paired client that has had
the fewest answers while others were waiting, so a client that keeps others
waiting builds up a debt and one that has been waiting is served next. A
client alone on an idle model is served at once, every time, and builds no
debt: nothing is slower while nobody else is waiting.

A paired client is one person, by its pairing. Unpaired clients on the network
are one class: they are told apart from nobody, no address is used or kept,
and they are served only when no paired request is waiting for that model. An
unpaired request that has waited longer than the queue allows is refused with
the wait stated, rather than left waiting for ever. Nothing about who was
served is written down: the debt lives in memory and is gone when the server
stops. What Alice can see is the contention itself — one line in the
operational log each time a request is held back for another, and, in the
statistics, how many requests waited at each model and for how long, naming no
client.

## Why This Matters

With one batched request as the default, a model serves one answer at a time,
and arrival order lets whoever asks most often keep everyone else waiting. The
server already knows which client is which for a paired client, and the
record has already decided that an unpaired client is not told apart from
another; this rule uses exactly that knowledge and nothing more. Counting only
the answers served while somebody else waited keeps the rule from slowing an
idle server, which is the case that matters most of the day.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview. We expect counting
only answers served while others waited to make turns fair without slowing an
idle server, because a client alone on the server builds no debt and a client
that keeps others waiting does; what would show this wrong: a paired client
still starved, or a busy client held back while nobody waits. We expect
serving paired clients first to keep scripts and unknown callers from crowding
out the people the server is for, because a person pairs once and a script
rarely does; what would show this wrong: the people we serve turning out to be
unpaired callers.

## Scope Conditions

- Turns at a loaded model: the rule decides who gets the next free turn at a <!-- cond: cond-2609202154055090 -->
  model that is in memory; the wait for a model to load, and eviction, keep
  their own rules (arrival order, eviction grace).
- Paired identity, nothing else: a paired client is one person by its pairing; <!-- cond: cond-2609202154055143 -->
  unpaired clients are one class and are not told apart from each other; no
  address is used or kept.
- The bridge and the on-device model are outside it: the Discord bridge's <!-- cond: cond-2609202154059706 -->
  requests are one paired client's; the on-device model has no queue.
- What is written down is the contention, never the client: the debt is in <!-- cond: cond-2609202154059000 -->
  memory only; the log line and the statistics name the model and the class
  (paired or unpaired), never a client's name or address.

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked and accepted:

- Given a loaded model with a free turn and requests waiting from two or more
  paired clients, when the turn is given, then it goes to the paired client
  with the fewest answers served while others waited; ties go by arrival.
- Given a paired client alone on an idle model, when it sends request after
  request, then each is served at once and it builds up no debt.
- Given an unpaired request and any paired request waiting for the same model,
  when a turn frees, then the paired request is served first; the unpaired one
  waits.
- Given an unpaired request that has waited longer than the queue allows, when
  its time is up, then it is refused with the existing `waited` state and
  queue time in the answer, rather than left waiting.
- Given no paired request is waiting, when an unpaired request arrives, then it
  is served as today, in arrival order among unpaired requests.
- Given the debt, when the server stops, then it is gone: it is kept in memory
  only, and no client's name or address is written for it anywhere.
- Given contention, when a request is held back for another, then the
  operational log carries one line naming the model, whether the held request
  was paired or unpaired, and how many were waiting — never a client's name.
- Given the statistics are on, when requests wait, then the store and the
  Statistics tab carry, per model per minute, how many waited, for how long,
  and how many unpaired requests were deferred, naming no client.
- Given the Discord bridge, when it sends requests, then they count as its one
  paired client's; given the on-device model, nothing here applies.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: with one batched request by default, arrival order lets one client monopolise a model; we expect paired-first with debt counted only under contention to keep every person's wait short, and it is wrong if a paired person still waits behind a burst from another
