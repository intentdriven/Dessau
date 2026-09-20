---
id: itd-2609201824151758
slug: alice-sees-how-often-each-model-s-follow-up-questions-are-cl
spec_id: spc-2609201835161251
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201338120342, itd-2609061521102742]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Alice sees how often each model's follow-up questions are clicked: the Statistics tab counts them per model per day

## Press Release

Alice opens the Statistics tab and, beside each model's requests for the
day, sees how many of them began as a follow-up question the model itself
had offered and a person clicked. A model whose follow-ups are clicked
often is offering the right next question; one whose follow-ups are never
clicked is not, and Alice can see that over the weeks, in the same daily
summary that keeps request counts and token totals after the detail is
gone. What makes the count is one small mark on the request Dessau Chat
sends when Bob clicks an offered question: the server reads the mark,
counts it against the model it was answered by, and stores nothing else
about it. No question text, no client name, nothing beyond one more count
in the record every account on the Mac can already read. A request typed
by hand carries no mark and counts as it always has.

## Why This Matters

Follow-up questions (itd-2609201338120342) are a bet that a small local
model can name the next question well. The only honest way to see whether
the bet pays, per model, is to count how often people take the question it
offered, and the server's statistics store already counts requests per
model per day and folds them into a durable summary. One more count in that
store, keyed the same way, is the smallest thing that answers the question,
and it is what would show the follow-ups intent's own mechanism claim wrong.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect one
count per model per day to show whether a model's follow-ups are worth
clicking because clicks are the one signal the record can hold without
holding any text. What would show this wrong: the count staying near zero
for every model while people report using the follow-ups.

## Scope Conditions

Confirmed by the maintainer at the 2026-09-20 interview:

- The mark is a request header the gateway reads, counts and strips; it is <!-- cond: cond-2609201835162201 -->
  never forwarded to the model server, and the body stays byte-identical to
  a typed request.
- Counted per model per day in the statistics store and folded into the <!-- cond: cond-2609201835168357 -->
  daily summary; no question text and no client name are stored with it.
- Shown in the Statistics tab beside the request count and on the model's <!-- cond: cond-2609201835161046 -->
  card in My Models.
- The Discord bridge and the on-device model send no mark. <!-- cond: cond-2609201835167792 -->
- Any client may set the header; it moves a count and nothing else. <!-- cond: cond-2609201835162997 -->

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked
and accepted:

- Given Bob clicks an offered follow-up, when Dessau Chat sends it, then the
  request carries the follow-up header and nothing else about the click.
- Given a request with the header, when the gateway records its statistics,
  then the model's daily count of clicked follow-ups goes up by one, the
  header is stripped before the request is proxied, and no question text,
  client name or address is stored with it.
- Given the Statistics tab, when Alice reads a model's day, then she sees
  the clicked follow-ups beside the request count, and the daily summary
  keeps the figure after the detail is removed.
- Given a request typed by hand, when it is recorded, then the follow-up
  count is untouched.
- Given a mark on a request from a client the server does not know, when
  it is recorded, then it counts like any other, since the count is coarse
  by construction and names nobody.
- Given the Discord bridge and the on-device model, when a person answers,
  then no mark is sent, since neither offers follow-ups on a server.
- Given My Models, when Alice opens a model's card, then the running figure
  is shown there too.

## Open Questions

Both were put to the maintainer at the 2026-09-20 interview and are
decided: a request header the gateway reads, counts and strips; shown in the
Statistics tab and on the model's card.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the follow-ups intent's own claim is only testable by counting clicks per model; wrong if the count is unreadable or nobody looks at it
