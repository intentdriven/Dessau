---
id: adr-2609201008476813
slug: local-telemetry-may-record-prompt-text-and-completions-only
status: accepted
date: 2026-09-20
supersedes: adr-2609061503319212
superseded_by: null
related_intents: [itd-2609091707499248, itd-2609062346072707, itd-2609091715089488]
related_rfcs: []
related_adrs: [adr-2609061503319212, adr-2609201008470380, adr-2609201008477513, adr-2609061610102325, adr-2609111126115848, adr-2609181004167097]
---

# ADR-2609201008476813: Local telemetry may record prompt text and completions only in the transcript store, never in statistics, and only while the operator's switch is on

**Supersedes:** [adr-2609061503319212](2609061503319212-no-public-telemetry-local-telemetry-only-as-a-strict-opt-in.md),
in whole. Its public-telemetry decision and its opt-in rule are restated below
unchanged; what moves is one clause of one sentence.

## Context

adr-2609061503319212 fixed the observability posture on 2026-09-06, before any
observability feature was designed, so that every later intent would inherit
the same boundary instead of re-litigating it. It decided three things: no
public telemetry, ever; local telemetry offered only on that Mac, for the
operator of that Mac; and local telemetry as a strict opt-in, off by default.
It then added a clause that has held every design since: "Even when on, local
telemetry never records prompt text, completions, API keys, or anything the
existing error redaction removes."

That clause is unqualified, and its own definition of local telemetry —
"recorded on that Mac, for the operator of that Mac, and shown in Gropius's own
control panel or logs" — is exactly what two features decided on 2026-09-20
are. itd-2609091707499248 keeps every prompt and answer in a store the operator
switches on. itd-2609062346072707 raises one model server's own log level for
one run, and the pinned server writes both bodies in full at that level. Each is
local, each is the operator's, each is shown or read on this Mac. The clause
speaks to both head-on.

That is the larger of the two contradictions the 2026-09-19 review of the
transcript intent found, and it named it precisely: the hold on that draft
pointed only at the gateway's prompt-content record and missed this one, which
is the decision that set the posture in the first place. Both are superseded on
2026-09-20, by records linked in both directions.

What has *not* changed is why the clause was written. The research pass behind
the superseded record — on anonymous telemetry, and on what comparable local
LLM tools do — found that users of such tools treat even an update check as
suspect, that every opt-out design surveyed produced a backlash, and that a
random install id is pseudonymous personal data. None of that is about the
operator reading their own server's traffic on their own disk. All of it is
about anything leaving the machine, and none of it moves here.

The statistics store is the other thing that does not move. Its promise is
stated in its own reference page — "There is nothing else: no prompt, no
answer, no key, no client address" — and three byte-scanning tests hold it. The
whole design of both new features turns on their being a different kind of
thing from that store.

## Decision

**No public telemetry, ever. Unchanged, and restated in full.** Nothing about
usage, hardware, errors, models or configuration leaves the machine to the
project, to a vendor or to any third party. This covers crash reports, update
checks that carry metadata, and the telemetry of child processes and libraries
Gropius runs. The outbound connections Gropius makes are exactly those needed to
fetch models and to provision its runtime; the opt-in bridge of
adr-2609181004167097 is the one path by which a conversation may leave the Mac,
and it carries only what a person deliberately typed into a platform they chose,
reporting nothing to the project or a vendor.

**Local telemetry is offered only locally, and only as a strict opt-in.**
Unchanged: recorded on that Mac, for the operator of that Mac, shown in
Gropius's own control panel or logs; off by default, with the switch in
Settings. Until the operator turns something on, Gropius records nothing beyond
the operational logs it already writes.

**Local telemetry never records prompt text, completions or API keys — with
exactly two exceptions, each of which the operator switches on deliberately and
each of which has its own record.**

1. **The transcript store**, under
   [adr-2609201008470380](2609201008470380-the-gateway-may-retain-both-sides-of-a-conversation-in-a-tra.md).
   Both sides of a conversation, retained only while the operator's switch is
   on, in the operator's own account, in its own store, disclosed on every
   answer by an unconditional header and per entry in the models list, with a
   size cap, a delete, a per-model exception list, and refused outright under
   the shared-cache install.
2. **The per-model diagnostic log**, under
   [adr-2609201008477513](2609201008477513-a-deliberately-invoked-per-model-diagnostic-may-write-prompt.md).
   One model, one process lifetime, armed by the operator in the panel, written
   by the model server itself at a level Gropius raises and does not read,
   bounded in size, with the previous run's file kept — and with the client not
   told, which that record states as a cost rather than designing around it.

Nothing else. An exception is a decision with a record, not a precedent to
reason from: a third place that records prompt text or a completion needs its
own ADR, and this one is where a future reader looks to find that there are two.

**The statistics store stays content-free, and that is armed rather than
asserted.** Neither exception touches it. The three byte-scanning tests that
hold its promise today pass with no edit —
`TestNothingFromTheRequestReachesTheRecordOrTheLog`,
`TestNothingFromTheRequestReachesTheStore` and
`TestNothingFromTheRequestReachesTheSummary`. An edit to any
of the three is the evidence that an exception leaked into the store it was
supposed to stay out of. The token counts stay what they are — content-free by
construction, and a different kind of thing from the answer they count.

**API keys and the existing redaction are unchanged.** Nothing this record
permits is a licence to record a key. The transcript retains what a client sent
in its messages; the diagnostic's file is the model server's, which redacts
nothing, and both records say plainly that a key a client puts inside a message
is written verbatim. Every other path — error messages, snapshots, the panel,
`config show`, the statistics store — redacts as it does today.

**The word is "transcript", on every surface a person reads.** Go, the control
panel, `config.json`, the docs and the client. "Recording" is already the
panel's word for the statistics store, whose defining promise is that it keeps
no content; shipping a store of prompts under that word would put the two on the
same pane, one word apart, meaning opposite things. Only the frozen wire
carriers — the disclosure header and the models-list field — keep the names the
2026-09-20 interview gave them.

## Alternatives Considered

1. **Supersede with two named exceptions (chosen).** The posture that matters —
   nothing leaves the Mac — is restated in force, and the two places where
   content is now kept are named in the record a future reader reaches first,
   each pointing at the decision that governs it.
2. **Leave this record alone and let the two feature ADRs speak for
   themselves.** Cheapest, and each feature record does state its own boundary.
   Rejected for the reason the diagnostic precedent gives: the exemption would
   live in the feature records while the ratified posture record went on saying
   prompt text is never recorded, so a reader who was not in the room could not
   tell which was in force — which is the whole job of the record.
3. **Narrow rather than supersede, as the diagnostic record narrows this one.**
   Rejected for the transcript: a narrowing reads as an edge case, and a store
   that keeps every prompt and every answer is not an edge of "never records
   prompt text". The sentence has to be rewritten, which means a superseding
   record.
4. **Decline both features and keep the clause absolute.** Genuinely
   attractive: it is the promise the product is easiest to explain by.
   Rejected by the maintainer at the 2026-09-20 interview, and rejected on
   merit too — an operator who needs to see what a client sent will put a
   logging proxy in front of the server, which keeps the same bytes with no
   disclosure header, no exception list, no size cap and no delete.
5. **One exception rather than two: build the transcript and let the diagnostic
   read it.** Considered in the reading of the two intents together. Rejected:
   the transcript holds what Gropius relays, and the diagnostic exists precisely
   to show what the *model server* received and generated, at the moment the two
   might differ. A transcript cannot answer the question the diagnostic is armed
   for.

## Consequences

- adr-2609061503319212 is marked superseded by this record and changes only its
  status fields; its body is unedited and its reasoning about public telemetry
  is restated here rather than living in two places.
- The two feature records are the load-bearing ones for their own conditions.
  This record names them and does not repeat their conditions, so a change to
  either is made in one place.
- The docs sentence beside the statistics switch — what turning it on records
  and where — now has two siblings, and each must say what its own switch keeps,
  in its own words, on its own pane. A single page that explains "telemetry" as
  one thing would now be wrong three ways.
- The architecture test that refuses any outbound host other than the ones the
  product needs is unaffected and stays the instrument for the half of this
  record that did not move.
- **What this costs, stated plainly.** The project's simplest sentence — "your
  prompts are never recorded" — is no longer true, and no wording recovers it.
  What is true, and what every user-facing page must now say instead, is
  narrower and longer: nothing about a prompt ever leaves the Mac; nothing is
  kept unless the operator deliberately switched it on; where it is kept is
  named, bounded and deletable; the transcript tells every client on every
  answer; and the diagnostic does not, which is why it is one model, for one
  run, armed by hand. A promise that needs five clauses is a worse promise than
  one that needs none, and that is the price of both features.
- GitHub release download counts and user-initiated diagnostics attached to
  issues remain the project's only sources of install and failure data. That
  cost is unchanged and is accepted again here.
