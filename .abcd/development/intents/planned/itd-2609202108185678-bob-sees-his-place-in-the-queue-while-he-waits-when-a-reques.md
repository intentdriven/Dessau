---
id: itd-2609202108185678
slug: bob-sees-his-place-in-the-queue-while-he-waits-when-a-reques
spec_id: spc-2609202141358048
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609061441285238, itd-2609182357325215]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Bob sees his place in the queue while he waits: when a request cannot start at once because the model is busy or loading, Dessau Server tells the waiting client where it stands, and Dessau Chat shows it. A streaming answer carries the notice before its first token, as lines every standard client ignores and Dessau Chat reads, so Bob sees third in line and a queue that is moving rather than a spinner; the answer's headers carry the position beside the queue time it already carries, so any client can read it afterwards. Nothing is asked of a client that does not understand the notice, and the answer is byte-identical to what it was for one that does not stream.

## Press Release

Bob sends a question to a model that is busy answering someone else, or that
is not in memory yet. After a few seconds the spinner under his message gives
way to a line that says where he stands: "waiting for the model to load", then
"3rd in line", then "next", and it goes as the answer begins. He knows the
server has his request and that the line is moving, so he waits instead of
sending it again or switching models. A quick turn looks exactly as it does
today: the line appears only once a wait has lasted longer than a threshold in
Dessau Chat's settings, three seconds unless Bob changes it.

The place comes from the server. Dessau Chat tags each request with an id of
its own making and, while it waits, asks the server where that request stands;
the server answers about that request and about no other, so nobody learns of
anyone else's traffic. Any client can adopt the tag and ask the same question;
one that never does gets exactly the answers it gets today. The check-ins are
not requests: they count in no statistic, and they are written to the
operational log so Alice can see them if she looks.

## Why This Matters

With one batched request as the default, people on the same Mac wait behind
each other more often, and an unexplained spinner is what makes a person resend
a question or give up on a model. The server already knows the two waits a
request can be in — for a turn at a loaded model, and for the model to load or
for memory to come free — and their order; telling the person which wait and
what place costs nothing the server does not already hold. The existing
`X-Dessau-State` and `X-Dessau-Queue-Time` headers say what the wait was after
the fact; this is the same fact while it is happening.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect a person who
is told "3rd in line" to wait rather than resend or give up, because an
unexplained spinner is what makes them do both. What would show this wrong:
resends and cancellations during waits not dropping once the line appears.

## Scope Conditions

- Dessau Chat against Dessau Server: the notice is shown by Dessau Chat talking <!-- cond: cond-2609202141358693 -->
  to a Dessau Server; the on-device model and the Discord bridge show nothing
  (there is no queue on-device, and Discord has its own typing indicator).
- A client's own requests only: the server answers only about a request the <!-- cond: cond-2609202141354159 -->
  asking client tagged itself; no client learns about another's.
- A place, not a promise: the place is where the request stands now; it can <!-- cond: cond-2609202141353671 -->
  change when a model is evicted or a request ahead is cancelled, and nothing in
  it promises when the answer starts.
- Check-ins are out of the statistics and in the log: a check-in appears in no <!-- cond: cond-2609202141350621 -->
  request statistic, and is written to the operational log.

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked and accepted:

- Given Bob's request is waiting for a free turn at a loaded model, when his
  client asks where it stands, then the server says it is waiting for a turn
  and its place among the requests ahead of it for that model.
- Given Bob's request is waiting for the model to load, or for memory to come
  free for it, when his client asks, then the server says it is waiting for the
  model to load and its place among those waiting for that model.
- Given a request the asking client did not tag itself, when a client asks
  about it, then the server answers nothing about it, so no client learns of
  another's request.
- Given a wait longer than the threshold in Dessau Chat's settings (default
  three seconds), when Bob's answer has not started, then a line under his
  message shows which wait and his place, updates as it moves, and disappears
  when the answer starts; a shorter wait shows nothing.
- Given a check-in, when the server handles it, then it appears in no request
  statistic and is written to the operational log.
- Given the on-device model or the Discord bridge, when Bob waits, then no
  notice is shown.
- Given a client that never tags or asks, when it sends requests, then its
  answers are byte-identical to today's.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: with one batched request by default, people will wait behind each other more often; we expect a visible place in line to make that acceptable, and it is wrong if people still resend or switch models during waits
