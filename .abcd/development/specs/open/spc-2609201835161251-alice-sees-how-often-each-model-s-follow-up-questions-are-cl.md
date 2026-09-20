---
id: spc-2609201835161251
slug: alice-sees-how-often-each-model-s-follow-up-questions-are-cl
intent: itd-2609201824151758
origin: researcher-authored
production_mode: hand-written
---
# Clicked follow-ups counted per model per day

## Summary

This spec delivers itd-2609201824151758: one request header marks a chat
request that began as a clicked follow-up; the gateway counts it against
the model, strips it, and stores nothing else; the statistics store keeps
the count per model per day and folds it into the summary; the Statistics
tab and the model's card show it. Impact: additive. It lands after the
follow-ups intent (itd-2609201338120342), which is what sends the header.

## Scope

In: `client/DessauChat/` (the header on the request a click sends);
`internal/gateway` (reading, counting and stripping the header);
`internal/stats` (the field on the request record and the daily summary);
`internal/ui` (the Statistics tab column and the model card line);
`docs/request-statistics.md`, `docs/statistics-store-reference.md` and
`docs/response-headers.md`'s neighbour for request headers. Out: the
Discord bridge and the on-device model, which send no mark; any text or
client identity, which the store never holds.

## Approach

**The header.** `X-Dessau-Follow-Up: 1`, set by Dessau Chat only on the
request that sends a clicked follow-up (cond: the mark is a header). Any
other value, or the header on any other request, is ignored except that it
is still stripped. The client sets it in the same place it adds the style
text to the newest turn, so the body is byte-identical to a typed request.

**The gateway.** In the proxy path, before the request is forwarded, the
gateway reads the header into a boolean on the per-request context it
already carries for statistics, and deletes it from the outgoing header
map beside the existing `X-Dessau-*` handling, so the model server never
sees it. A test sends the header and asserts the upstream request carries
none of it and the record carries the flag; a second sends a typed request
and asserts the flag is false. Any client may set it (cond: it moves a
count and nothing else), so no key, pairing or client identity is consulted.

**The store.** The request record gains one boolean, `follow_up`, beside its
outcome class; the daily summary gains `follow_up_count` per model, summed
the way the outcome counts are, and kept past the detail's removal the way
the other summary figures are (cond: per model per day, folded into the
summary). The record still holds no prompt text and no client name; the
store's existing schema test is extended for the field, and the summary
test for the fold.

**The panel.** The Statistics tab's per-model row gains a "Follow-ups"
column beside the request count, read from the same snapshot the row
reads; the model's card in My Models gains one line with the running
figure for the current month, from the summary. Both are held by the
existing panel tests that pin the statistics snapshot and the card's
lines. `config.json` gains nothing: the count follows the statistics
switch, off when statistics are off.

**Docs.** The request-statistics page says what the header is and that any
client may send it; the store reference documents the two fields; nothing
narrates the change.

## Acceptance Criteria, and what holds each

- The clicked follow-up's request carries the header and nothing else about
  the click: an architecture test over the Swift source that finds the
  header set on the follow-up send path and nowhere else.
- The gateway counts it, strips it, stores no text or client name: gateway
  tests with a fake upstream asserting the forwarded headers and the record.
- The Statistics tab shows it and the summary keeps it: the stats store's
  summary test and the panel snapshot test.
- A typed request leaves the count untouched: the second gateway test.
- An unknown client's mark counts like any other: a gateway test with no
  key and no pairing.
- No mark from the bridge or on-device: an architecture test that the
  header's name appears only in the client's follow-up path and the
  gateway.
- The model's card shows the figure: the panel's card test.

## Departures

None at the time of writing.
