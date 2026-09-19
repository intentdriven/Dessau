---
id: itd-2609091707499248
slug: gropius-can-record-every-prompt-and-answer-and-every-client
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Gropius can record every prompt and answer, and every client is told so first. Alice turns recording on in the server's settings; nothing turns it on by itself. From then on the first answer any client receives opens with a notice that this server records every prompt and answer, every response carries the same fact, and the models list says so, so no client can talk to the server without having been told. The recording lives in Alice's own account on the Mac, readable by nobody else, until she turns it off.

## Press Release

> _The heading above is the maintainer's dictation of 2026-09-09, kept as it was
> spoken. The paragraphs below are the corrected press release: three of the
> heading's claims do not survive the review of 2026-09-19 and are restated here
> as what the product can honestly say. See `## Review 2026-09-19`._

Alice opens Settings and turns on a transcript of what her server serves. Under
the switch a fixed list says exactly what that writes down: for every request
this server handles, the whole prompt and the whole answer, in her own account's
data folder, in plain text, bounded by a size she sets. Nothing turns it on by
itself and nothing leaves the Mac. Until she turns it on, her server writes what
it writes today — counts and timings if she has those on, and nothing of what
anybody typed.

With it on, Gropius says so on every path a client can see. The models list
carries the fact on each model, beside the context length and the chat flag. A
header on every answer — including the refusals Gropius writes itself — says the
same thing, whether or not the install has an API key. Bob, using the Gropius
chat client, sees it in the client: his paired client reads the fact and puts it
in front of him before he types. Bob using someone else's client sees whatever
that client chooses to show him, which may be nothing: Gropius can make the fact
unmissable to a program, and it can hold its own client to showing a person, but
it cannot make a program it did not write say anything to anybody.

The transcript is Alice's, in her own account, mode 0600, and it is not the
statistics store — that store still carries no prompt, no answer, no key and no
client address, and the tests that hold it to that stay green. Switching the
transcript off stops new records and leaves the ones already written where they
are; a separate, deliberate **Delete transcript** is what removes them, on the
same terms as **Clear records** beside it.

## Why This Matters

An operator who runs a server for other people has one question the product
cannot currently answer: what is actually being sent to it. The per-model debug
draft (itd-2609062346072707) answers it for one model until that model server
restarts, in the model server's own log — the right tool for "this model is
answering strangely". This is the other shape: every model, until it is turned
off, in a store the operator can keep, search and delete.

The boundary it sits inside was settled before any observability feature was
designed, and this intent is the first thing to ask for a hole in it.
adr-2609061503319212 says local telemetry "never records prompt text,
completions, API keys"; adr-2609061610102325 says the gateway will "never log,
retain, count, index or otherwise keep any prompt content" it reads, and that
any further reading is a new decision superseding it. Recording is retention of
both sides of the conversation. It cannot ship as an addition to either record —
it needs two superseding decisions, and the reason it is worth them is the
disclosure: a server that records and says so on every path is a different thing
from one that records quietly.

Evidence: the readers' list held by `internal/archtest/prompt_content_test.go`,
which already forces a second reader of either side of a conversation to be
added by name in a diff someone reviews.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Hold

Held on 2026-09-09 by the maintainer at interview, after the design review of
the draft: the promise "every client is told first, and no client can talk to
the server without having been told" cannot be made honestly by a stateless
OpenAI-compatible server. There is no session it can know: the one API key is
shared by every client and loopback clients carry none; a remote address hides
everyone behind a proxy and reconnects per request; a user-agent is
client-chosen; an acknowledgement field is a client-controlled opt-out; and an
"already told" memory is unbounded state keyed on client strings inside the
gateway, a declared trust boundary. The only stateless rule is a notice on
every answer, which the maintainer declined for now. Also found: prepending
text to an answer is a new right to WRITE generated content that the
prompt-content boundary record does not grant, so the mode needs a superseding
decision record naming two readers and one writer; the store must be its own,
never the statistics store; and "readable by nobody else" is false under the
shared-cache mode, where the serving account would hold every local account's
prompts. Lifted when a client-side contract exists that can show a person the
notice, or the maintainer adopts the every-answer rule. The per-model debug
draft itd-2609062346072707 stays its own record meanwhile.

**Still held after the review of 2026-09-19.** The review found no way round the
original ground and found more that only the maintainer can settle; the open
questions below are what the hold now waits on. One part of the lifting
condition has arrived since the hold was written: itd-2609182357325215 and
adr-2609182357322050 give Gropius's own client a pairing contract a test can
hold, so a client-side contract that can show a person the notice now exists —
for Gropius's client, and for no other.

## Review 2026-09-19

Two hostile passes, design-and-feasibility and record-discipline, against the
draft, its siblings, the shipped statistics records, adr-2609061503319212,
adr-2609061610102325, adr-2609181004167097 and the readers' list in
`internal/archtest/prompt_content_test.go`.

### The notice's carrier

Four carriers exist. One is barred by the record; three carry the fact to a
program and none of them reaches a person.

1. **A response header** (`X-Gropius-Recording`). The precedent is built:
   `gropiusHeaders` already reserves `x-gropius-state` and `x-gropius-queue-time`
   as headers an upstream may not add to. It breaks no client, it covers the
   refusals Gropius composes itself, and it costs nothing per token on a stream.
   Two things do not carry over. `setWaitHeaders` writes its two headers only on
   an install that has a key, which would leave a keyless open LAN install — the
   one with the most strangers on it — told nothing; a disclosure header has to
   be unconditional, and that has to be decided rather than inherited. And a
   header must join `gropiusHeaders`, or an upstream can forge it into the
   relayed set.
2. **A text prefix on the answer.** Barred on three independent grounds. It is a
   write of generated content, strictly larger than any grant on either readers'
   list — the relay is admitted only to read *whether* a choice array is empty,
   "never what is inside one". It breaks clients concretely: a
   `response_format: json_object` answer stops being valid JSON, a tool-call
   answer has empty content and a `tool_calls` array so the prefix lands nowhere
   or corrupts it, `usage` stops matching the delivered text, and any client
   matching an answer exactly breaks. And on a stream the prefix chunk must be
   emitted before the upstream has been asked anything, which makes time to
   first token an observable lie — to the client, and to Gropius's own
   `FirstTokenMS`.
3. **A field in the models list** (`recording` per entry). The cheapest, and it
   fits a deliberate pattern: the listing already publishes `context_length`,
   `served_context`, `measured_context`, `pipeline_tag`, `tags` and `chat` as
   top-level extensions, and `chat` is documented as always present "because an
   absent key would be read as an older Gropius that cannot say either way" —
   word for word the argument for a recording flag, and exactly the per-model
   field itd-2609091715089488 needs. But `/v1/models` is optional: a curl and a
   pinned-model agent never call it, so it cannot carry "no client can talk to
   the server without having been told".
4. **A top-level body field** on the completion object and each SSE chunk. Not a
   write of generated content — it touches no `choices`, no `delta`, no message
   content — and the machinery exists, since the relay already rewrites the
   top-level `model` field on both paths. But a strictly-typed deserialiser that
   forbids unknown fields rejects it, it rides every chunk of every answer
   against the relay's line bound, and it does not cover the error bodies the
   gateway composes itself.

The finding underneath all four: the heading's promise is about a **person**
("every client is told so first"), and every available carrier is a fact about a
**program**. The promise separates into two claims that can each be held by a
test — the server publishes the fact on every path a program can see, and
Gropius's own paired client is held to showing it to a person — and one that
cannot be held at all, which is the third-party client. Restated in the press
release.

### Where the transcript lives — and the third rotating writer

The store must be its own. `docs/statistics-store-reference.md` states "There is
nothing else: no prompt, no answer, no key, no client address", and three
byte-scanning tests hold it (`TestNothingFromTheRequestReachesTheRecordOrTheLog`,
`...TheStore`, `...TheSummary`). Those tests stay green; the transcript is a
different store.

But a third store means a third size-rotating writer, and two is already an open
defect: `iss-2609091714393599` records that `internal/stats/store.go` and
`internal/applog`'s rotator both rotate by size, deferred because lifting a
shared primitive out of the statistics store "is a larger change to this
repository's durable data format than a log file justifies". A third instance
retires that excuse, and the standing rule is explicit — never add a third copy;
find the canonical home and extend it.

So the transcript store does not get its own rotator. Either it reuses
`internal/applog`'s, which is already the general one — predictable name, an
`os.Root`, `O_NOFOLLOW` and `O_NONBLOCK`, mode 0600, an fstat on the opened
handle, rotate by size, keep N — or the spec's first commit lands
`iss-2609091714393599` and both callers move onto the lifted primitive. Reuse is
the better default: a transcript needs none of the statistics store's UTC-day
file names, its summary fold (a summary of a transcript is a transcript) or its
months horizon (content is kept or it is not).

The bound is a product number, not an engineering one. The log's 5 MB across 5
files holds a few thousand turns; a day of agent traffic wraps that inside an
hour, which makes the transcript a rolling window of the last few minutes and
useless for the purpose. Either the cap is large and named in Settings, with the
oldest record's date on the panel as the statistics store already does, or the
record says plainly that this is a bounded rolling window and not an archive.

### "Readable by nobody else"

False under the shared-cache mode, as the hold found, and the reason is
structural. One process serves every local account by singleton election.
adr-2609061610107154 already conceded the shape for statistics — the store
"belongs to the account that runs the serving process and holds the requests of
every local account that used it", with the docs saying so in one sentence. For
token counts a sentence is proportionate. For prompt text it is not: Carol's
conversations would sit in Alice's folder, readable by Alice and not by Carol.

Per-account transcripts are not available: the singleton does not run as the
requesting account, the API key is shared, and a loopback source address
attributes nothing. So it is refuse-while-shared, or record-and-say-so in the
Settings pane on the terms adr-2609181004167097 set for the bridge's egress
sentence — the switch unrenderable without the sentence, held by a test.

### The grant, and which decisions are superseded

The hold says the mode "needs a superseding decision record naming two readers
and one writer". That count is wrong under any carrier but the barred text
prefix: nothing new **writes** generated content. What is new is a **retainer**
of both sides — the one grant adr-2609061610102325 withholds by name: "never
log, retain, count, index or otherwise keep any prompt content the gateway
reads".

And two decisions are superseded, not one. The hold names
adr-2609061610102325. It does not name adr-2609061503319212, whose clause is
unqualified — "Even when on, local telemetry never records prompt text,
completions, API keys" — and whose own definition of local telemetry
("recorded on that Mac, for the operator of that Mac, and shown in Gropius's own
control panel or logs") is exactly what a transcript is. That is the head-on
contradiction, and it is the larger of the two: it is the decision that fixed the
posture before any observability feature was designed, precisely so later intents
would not re-litigate it.

### The bridge, and the paths that are not the gateway's

- **The Discord bridge.** `spc-2609181018499470` routes a bridged message through
  an in-process `gateway.Ask` that runs "the same admission, merge and
  observation as an HTTP request would", so with the transcript on, Bob's Discord
  messages land in Alice's transcript. There is no response object and no header
  on that path — the answer is an edited Discord message. Either bridged turns
  are excluded, or the notice goes into the Discord message text (honest there:
  no JSON mode to break), or the bridge's egress sentence gains a second clause.
- **The client's built-in model.** Gropius's own chat client can answer on the
  device without the server (`client/GropiusChat/Backends.swift`,
  `Picker.swift`). That conversation is never recorded and must not be shown a
  recording notice, or the client lies in the other direction.
- **Gropius's own requests.** The pool's readiness probe, the self-test and the
  context probe each compose their own conversations and post them straight to a
  model server on loopback, never through the gateway — the shipped condition
  `cond-2609061822382803` says so and the audit found it survived. They are not
  recorded, and that belongs in Scope Conditions rather than being discovered.

### Retention, deletion, and one word that is already taken

The precedent is built and visible. The panel says: "Switching recording off
stops new records and leaves these ones where they are; **Clear records** is what
removes them." So the heading's "until she turns it off" reads as a deletion the
house precedent does not perform; a transcript needs the delete affordance more
than the statistics store does, and it needs it on all three surfaces.

And the word is taken. The control panel already calls the statistics store
"recording". A transcript shipped under that word sits on the same pane as the
feature whose defining promise is that it records no content. It needs its own
word — transcript — on every surface.

### Three surfaces

The switch, the exception list (itd-2609091715089488), the size bound and the
delete must each exist in Go, in the web panel and in `config.json`, and a
`config.json` save must never be refused over a setting the operator did not
touch. The Swift client's obligation is different in kind from every feature
before it: it is the only surface that can show a **person** the notice, so for
the first time the client carries a promise rather than a convenience, and the
pairing contract (adr-2609182357322050) is where a test can hold it.

### Record discipline

- The press release was the seed prompt and "Why This Matters" was the heading
  pasted back. Both rewritten above, against the shape of itd-2609061521082551.
- Mechanism, Scope Conditions and Acceptance Criteria stay unfilled: every
  acceptance bullet turns on the carrier and the retention rule, which are the
  maintainer's. Parts of Scope Conditions are writable now (Gropius's own
  loopback callers; the client's built-in model; the pinned mlx-lm server) and
  are named above so they are not lost.
- Carol was unused and is the persona the shared-cache finding needs; she is used
  for it above.
- `severity: minor` is wrong for a feature that writes every prompt and answer to
  disk and supersedes two ADRs. Recommended `severity: major`, left unchanged
  here because the frontmatter is the maintainer's dictation. `impact: additive`
  is right: a new store, off by default, no stored format changed.
- itd-2609062346072707 writes very nearly the same bytes and differs only in
  scope and in who reads them. The difference is now stated in "Why This
  Matters", because left unsaid someone builds one and believes they have built
  the other.
- There is no `## Grounds` section. One is required before this can be planned.

## Open Questions

Six, and each is the maintainer's: the first is a product and compatibility
choice, the rest are promises only they can make or narrow. The draft stays a
draft until they are answered.

1. **What carries the notice?**
   - (a) An unconditional response header on every answer, plus the models-list
     field. (b) The header, the models-list field, and a top-level body field on
     the completion and each chunk. (c) The models-list field alone. (d) A text
     prefix on the answer.
   - Recommendation: **(a)**. It covers every answer including Gropius's own
     refusals, costs nothing per token, breaks no client, and needs no new grant
     to write generated content. (d) is barred by adr-2609061610102325 and breaks
     JSON-mode and tool-call clients outright; (b) adds bytes to every chunk and
     a real chance of rejection by a strict deserialiser for a fact (a) already
     carries; (c) is silent to every client that never lists models.
   - The part that is genuinely theirs: **(a) tells a program, not a person.**
     Adopting it means the heading's "no client can talk to the server without
     having been told" becomes "every client is told, in a place a program can
     see; Gropius's own client is held to showing a person". If that narrowing
     is not acceptable, the intent does not ship in any form.

2. **Is the notice header written on a keyless install?**
   - (a) Always. (b) Only when the install has a key, matching `setWaitHeaders`.
   - Recommendation: **(a)**. The existing rule withholds *residency* facts from
     an unauthenticated LAN because they say who is busy and when. A disclosure
     is the opposite kind of fact: the install that most needs to make it is the
     open one. But it breaks a symmetry the codebase currently keeps, so it is
     worth saying out loud rather than inheriting.

3. **What is the retention rule, and does switching off delete?**
   - (a) A size cap in Settings, off-switch stops and keeps, a separate **Delete
     transcript** removes — the statistics store's own shape. (b) Off-switch
     deletes. (c) A size cap and a days horizon, both in Settings.
   - Recommendation: **(a)**, with a default cap large enough that the transcript
     is not a rolling window of minutes — the number is theirs. A switch that
     silently destroys data is worse than one that does not, and the panel
     already teaches this exact shape beside it. (b) makes the heading's "until
     she turns it off" literally true at the cost of a destructive switch.

4. **Is the transcript per model or global?**
   - (a) Global switch plus a per-model exception list — itd-2609091715089488 as
     written. (b) Per model only, no global switch. (c) Global only, no
     exceptions, and itd-2609091715089488 is dropped.
   - Recommendation: **(a)**. It is what the dependent draft already assumes, the
     models-list field carries it per entry for free, and it is the only shape in
     which the notice can differ per model without the client having to guess.
     Note the consequence the dependent draft flags and does not resolve: a
     request that names a recorded model but is served by an excepted one, or the
     reverse, has to have a stated answer — the models-list field is the place it
     becomes visible.

5. **Under the shared-cache install mode, does Gropius record at all?**
   - (a) Refuse: the switch is unavailable, and the panel says why. (b) Record,
     with the pane carrying a sentence saying that the serving account holds every
     local account's conversations, the switch unrenderable without it and held by
     a test — the bridge's egress rule.
   - Recommendation: **(a)**. The headline promise is "readable by nobody else",
     and Carol cannot consent to Alice keeping her conversations. A product that
     must retract its own headline in a sentence has picked the wrong default.
     (b) is defensible and is the cheaper build.

6. **Are bridged conversations recorded, and how is a Discord user told?**
   - (a) Bridged turns are excluded from the transcript. (b) Recorded, and the
     bridge's first answer in a channel carries the notice in the message text.
     (c) Recorded, and the Settings egress sentence gains a clause — nothing is
     said in the channel.
   - Recommendation: **(b)**. The bridge is the one surface where writing the
     notice into the message text is both honest and harmless: there is no JSON
     mode and no tool-call shape to corrupt, and the person on the other end is a
     person rather than a program. (c) tells Alice and never tells Bob, which is
     the failure this whole intent exists to avoid.

Beyond the six, two things are settled by the review rather than open, and are
recorded so they are not re-opened as questions: the transcript is its own store
and never the statistics store, and it reuses the existing rotating writer rather
than adding a third.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
