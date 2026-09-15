---
id: adr-2609121450000000
slug: the-statistics-store-gains-a-fifth-record-kind-a-footprint
status: accepted
date: 2026-09-12
supersedes: adr-2609090716413337
superseded_by: null
related_intents: [itd-2609091712141073, itd-2609061521102742]
related_rfcs: []
related_adrs: [adr-2609090716413337, adr-2609061610107154, adr-2609061503319212]
---

# ADR-2609121450000000: The statistics store gains a fifth record kind, a footprint reading, and the request and load records gain the facts that decide a served window and a concurrency

## Context

adr-2609090716413337 ratified four record kinds: a request line, a load, a
removal with one of seven reasons, and a settings record. The usage
measurement intent (itd-2609091712141073) needs the facts that decide a
served window and a decode concurrency — how prompts sat against the window,
how many requests were in flight, which sampling values were in force and
whether the client overrode them, and what the model server's memory did —
and one of those, the memory, is not a fact about any request. It is a
reading of a process, taken on a clock, and it needs a record of its own or
the figure exists only as whatever value a completing request happened to
see.

This decision was taken by the agent on the maintainer's lane assignment of
2026-09-12, under the instruction to decide autonomously and record; the
record-kinds decision is architecture-shaping, which is why this is an ADR
and not a line.

## Decision

We supersede adr-2609090716413337 on the record kinds, and adopt five:

1. **A request line**, as before, gaining: the declared and served windows
   the request was judged against, Gropius's own estimate of the prompt's
   size (written for a refused request too), the model's in-flight count at
   admission, the names — never the values — of the sampling parameters the
   client set, and the server's footprint as last sampled before completion.
2. **A load**, as before, gaining the sampling values the server was launched
   with.
3. **A removal**, unchanged.
4. **A footprint reading** — `kind: "footprint"` — the model's repo id, when,
   and the server process's resident memory in bytes, taken every thirty
   seconds for each running server and reported through the pool's observer.
5. **A settings record**, unchanged.

Every new numeric is bounded at the recorder as the windows already are: a
window past the registry's bound, a count past any batch, a footprint past
any Mac, or an override name outside the closed set is dropped, never
repaired. The override names are the one field derived from a client's body,
and the reference page says so.

## Alternatives Considered

1. **Carry the footprint only on the request record.** Rejected: the figure
   would exist only at the moments requests completed, which on a quiet
   server is never, and the intent's criterion asks for a periodic record.
2. **A separate file for the samples.** Rejected: the store is one
   collection path under one switch and one retention rule
   (adr-2609061610107154), and a second file would need a second of each.
3. **Store the client's sampling values.** Rejected: adr-2609061503319212
   draws the line at content the client sent; a name says which knob was
   touched and nothing of what was asked.

## Consequences

- `internal/stats.StoreKinds` names seven kinds of line (five records plus
  the two summary lines), and the reference page's table is held to it.
- The summary a day folds into ignores footprint readings: a mean over a
  day is not a figure anyone would size a setting from, and the series is
  what the view draws.
- The record size constant is re-measured over a record carrying every
  field.
