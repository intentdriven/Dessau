---
id: adr-2609201008470380
slug: the-gateway-may-retain-both-sides-of-a-conversation-in-a-tra
status: accepted
date: 2026-09-20
supersedes: adr-2609061610102325
superseded_by: null
related_intents: [itd-2609091707499248, itd-2609091715089488, itd-2609061441310453]
related_rfcs: []
related_adrs: [adr-2609061610102325, adr-2609061503319212, adr-2609201008476813, adr-2609181004167097, adr-2609182357322050, adr-2609061610107154]
---

# ADR-2609201008470380: The gateway may retain both sides of a conversation in a transcript store the operator switches on, disclosed on every answer

**Supersedes:** [adr-2609061610102325](2609061610102325-the-gateway-may-rewrite-prompt-content-only-to-merge-system.md),
in whole. Everything that record decided about merging is restated below,
unchanged; what moves is the clause that withheld retention.

## Context

adr-2609061610102325 settled the gateway's relationship with prompt content on
2026-09-06. It granted exactly one reading, for exactly one purpose — merging a
request's system-role messages into a single leading one, for a model the
operator switched it on for — and it withheld everything else by name: "We will
never log, retain, count, index or otherwise keep any prompt content the gateway
reads for this purpose." It also said what to do when that was no longer enough:
"We will treat any further reading of prompt content by the gateway as a new
decision that supersedes this record." This is that decision.

itd-2609091707499248 asks for a transcript: with a switch on, every prompt a
client sends and every answer a model gives is kept on the operator's Mac, and
every client is told so. Alice's reasons are ordinary and they are not
observability in the sense the statistics store means — an agent that answered
oddly three hours ago, a prompt she wants to re-read, a record of what a shared
server was asked to do.

Counting the grant correctly matters, because the intent's own hold stated it
wrongly and the 2026-09-19 review corrected it. Under the carrier the maintainer
chose, nothing new **writes** generated content into a conversation and nothing
new **rewrites** a request. What is new is a **retainer** of both sides — the
one thing the superseded record withheld by name. That is a smaller grant than
"two readers and one writer" and a more dangerous one, because retention is what
turns a relay into a store.

Two further facts shaped the conditions below.

**A transcript is local telemetry by the other record's own definition.**
adr-2609061503319212 defines local telemetry as what is "recorded on that Mac,
for the operator of that Mac, and shown in Gropius's own control panel or logs",
and bars it unqualifiedly from recording prompt text and completions. That
record is superseded in parallel by
[adr-2609201008476813](2609201008476813-local-telemetry-may-record-prompt-text-and-completions-only.md),
which names this store as one of its two exceptions. Neither record stands
without the other, and they are linked in both directions.

**The shared-cache install cannot honour the promise.** One process serves every
local account by singleton election. adr-2609061610107154 already conceded the
shape for statistics — the store belongs to the account that runs the serving
process and holds the requests of every local account that used it — and for
token counts a sentence on a docs page is proportionate. For prompt text it is
not: Carol's conversations would sit in Alice's folder, readable by Alice and not
by Carol, and Carol cannot consent to that. Per-account transcripts are not
available, because the singleton does not run as the requesting account, the API
key is shared, and a loopback source address attributes nothing.

The decisions below are the maintainer's, taken at the 2026-09-20 planning
interview on itd-2609091707499248 and itd-2609091715089488.

## Decision

We will allow the gateway to **retain** both sides of a conversation — the
messages a client sent and the answer a model generated — in a transcript store
on the operator's Mac, under the conditions below. Each condition is held by a
test or by the review of the change that lands it.

**1 — Opt-in, off by default, global with per-model exceptions.** Nothing is
retained until the operator turns the transcript on. The switch, the exception
list, the size bound and the delete each exist in Go, in the web control panel
and in `config.json`, and a `config.json` save never fails over a setting the
operator did not touch: the per-model map is merged rather than replaced, for
every per-model field, so a hand edit made after the panel loaded survives the
next save.

**2 — The decision is made on the model that serves the request.** A request
whose message array carries an excepted model's earlier turns is retained whole,
under the serving model, and the docs say so plainly. The gateway has no
conversations, only requests; any other rule would require it to reason about a
history it does not hold.

**3 — Disclosed on every answer, unconditionally.** Every response the server
sends carries a recording header — its prefix is the family name, so the prefix
follows the family name — including Gropius's own refusals. It is joined to the
set of headers the gateway declares its own, which upstream values are dropped
from before the rest are merged, so a model server cannot forge it and cannot
sit a second value beside it. It is written on a keyless install too: the rule
that withholds wait headers from an unauthenticated network exists to keep other
clients' *residency* facts off it, and a statement about this server's own
retention reveals nothing about anybody else's traffic. The open keyless LAN
install is the one that most needs to disclose.

**4 — And in the models list, per entry.** Each entry of `/v1/models` carries a
`recording` field in the base entry every client sees, so a picker can draw the
per-model fact before a person chooses rather than after they ask. The header is
the guarantee, because a curl and a pinned-model agent never fetch the models
list; the field is the detail. Gropius's own paired client is held to showing a
person — the pairing contract of adr-2609182357322050 is what makes that
test-holdable — and a client Gropius did not write is told, not shown.

**5 — Its own store, in the operator's own account, never the statistics
store.** The transcript is a separate file opened through the existing rotating
writer, which already carries the rules a new writer would otherwise
re-implement — a rooted directory, `O_NOFOLLOW` and `O_NONBLOCK`, mode 0600 with
an fstat on the opened handle, rotate by size and keep N. No third size-rotating
writer is added. The statistics store is untouched and stays content-free: the
three byte-scanning tests that hold it pass with no edit, and an edit to any of
them is the evidence that the store was not separate after all.

**6 — Retention is a size cap, switching off keeps, and deletion is its own
act.** The cap is named in Settings and defaults to 512 MB, so the transcript is
an archive the operator can keep and search rather than a rolling window of
minutes; the oldest record's date is shown on the panel. Switching the
transcript off stops new records and leaves the existing ones where they are.
**Delete transcript** removes them, on the same terms as the statistics store's
**Clear records**.

**7 — Refused under the shared-cache install.** Where the install is shared the
switch is unavailable and the panel says why. The headline promise — the
transcript lives in the operator's own account — stays true as written rather
than being retracted in a sentence beside the switch.

**8 — Bridged turns are retained, with a notice in the channel.** A bridged
message runs the same admission, merge and observation as an HTTP request, so
with the transcript on it is retained. That path has no response object and no
header, so the bridge's first answer in a channel carries the notice in its
message text, where a text notice is honest and harmless. An excepted model is
omitted from the bridge's model listing, and a bridged request naming one is
refused in the channel with the reason.

**9 — The retainer is named on the readers' list.** The transcript reads both
sides of a conversation, so it appears by name in both lists of
`internal/archtest/prompt_content_test.go` — the prompt-content readers and the
generated-content readers — with the reason it is admitted, in a diff somebody
reviews. The spec names the writer explicitly rather than trusting the scan to
discover it.

**10 — The word is "transcript".** On every surface a person reads — Go, the
control panel, `config.json`, the docs and the client — because "recording" is
already the panel's word for the statistics store, whose defining promise is
that it keeps no content, and the two would otherwise sit on the same pane. The
wire carriers keep the names the interview gave them: the header, and the
models-list `recording` field, which are frozen identifiers.

**11 — Gropius's own traffic is not a transcript.** The pool's readiness probe,
the self-test and the context probe each compose their own conversations and
post them straight to a model server on loopback, never through the gateway.
Nothing they send or receive is retained. The client's built-in on-device model
is never retained and is never shown a notice, since no request leaves the
device and a notice there would be a lie in the other direction.

**What carries forward from the superseded record, restated and unchanged.**

- The gateway rewrites prompt content for exactly one purpose: merging all
  system-role messages of a request into a single leading system message, and
  only for models the operator has switched merging on for in Settings. For
  every other model, and for every other field, the gateway relays prompt
  content unchanged. Equality of relayed content is judged on decoded values,
  never bytes, because the gateway re-encodes the request it buffers. The merged
  message is the conversation's own first system message with its content
  replaced.
- Merging is still the only *rewrite*. This record grants retention, not a
  second rewrite, and not a writer of generated content.
- No reader of prompt content exists beyond those the readers' list admits: the
  merge, the bridge under adr-2609181004167097 condition 4, the transcript
  retainer added here, and the entries that compose Gropius's own requests
  rather than reading a client's.
- Any further reading, rewriting or retention of prompt content by the gateway
  is a new decision that supersedes this record.

## Alternatives Considered

1. **Retain, under the eleven conditions above (chosen).** The operator gets the
   archive the intent promises; every client is told in a place a program can
   see; the store is separate, bounded, deletable and refused where the promise
   cannot hold; and the grant is named in the one test that makes a second
   retainer hard to add quietly.
2. **Decline: the relay stays a relay and keeps nothing.** The cleanest posture,
   and the one the superseded record wrote. Rejected by the maintainer's ask —
   and the alternative in practice is worse, because an operator who wants a
   transcript badly enough runs a logging proxy in front of the server, with no
   disclosure header, no exception list, no cap and no delete.
3. **Narrow the old record instead of superseding it**, the way the diagnostic
   record narrows the telemetry one. Rejected: the clause that moves is the
   never-retain clause, which is the heart of that record rather than an edge of
   it, and the record itself said further reading supersedes it. Narrowing would
   leave a ratified sentence promising something untrue.
4. **Disclose in the answer's body — a text prefix on every reply, or a field on
   every chunk.** Both were put and both declined: the text prefix corrupts
   structured output and tool calls, and a body field breaks clients that
   validate the OpenAI shape. The header is the carrier that costs no client
   anything.
5. **Per-model switch only, or global switch only.** Per-model only makes the
   common case — record everything for a week — a chore over every model;
   global only leaves no way to keep one model off the disk, which is the whole
   of itd-2609091715089488. Global with exceptions was chosen.
6. **Record under the shared-cache install, with a retraction sentence beside
   the switch**, on the terms adr-2609181004167097 condition 5 sets for the
   bridge's egress. Rejected: a headline that needs a retraction beside it is
   the wrong headline, and Carol cannot consent to Alice keeping her
   conversations.

## Consequences

- adr-2609061610102325 is marked superseded by this record and changes only its
  status fields. Its merging decision is in force and is read from here, where
  it is restated, rather than from two places.
- adr-2609201008476813 supersedes the telemetry record in parallel and names
  this store as one of its two exceptions. The two are linked in both
  directions; neither is complete alone.
- Easier: Alice can read back what a client actually sent, three hours later,
  without arming anything on a model or restarting it — which is the difference
  between this and the per-model diagnostic of
  [adr-2609201008477513](2609201008477513-a-deliberately-invoked-per-model-diagnostic-may-write-prompt.md).
  The two write very nearly the same bytes and differ in scope, in bound and in
  who is told; the intents say so, because left unsaid someone builds one
  believing they have built the other.
- Harder: a new store on every surface; a delete affordance on all three; a
  disclosure obligation on every response path, including the one that has no
  response object; and, for the first time, the Swift client carries a promise
  rather than a convenience — it is the only surface that can show a *person*
  the notice.
- Obligations: an adversarial security review of the change that lands it, the
  gateway being a trust boundary; the three byte-scanning tests green with no
  edit; a test that the header cannot be forged by an upstream answer and is
  present on a keyless install; a test that the switch cannot be rendered under
  a shared-cache install; an architecture test plus a Swift case holding the
  notice in front of Bob before he types; and entries on both readers' lists
  naming the retainer and why.
- **What this costs, stated plainly.** A server whose promise was that it kept
  nothing can now keep everything, and the only thing standing between a client
  and a permanent record of what it said is a switch in somebody else's panel
  and a header that client may never read. That is why the disclosure is
  unconditional rather than conditional, why the exception list exists, why the
  shared install is refused rather than annotated, and why the word is not the
  one already spent on a store that keeps no content.
