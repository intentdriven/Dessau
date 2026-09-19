---
id: itd-2609091715089488
slug: some-models-keep-no-transcript-even-while-recording-is-on-al
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091707499248]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Some models keep no transcript even while recording is on. Alice names one or more models on an exception list in the server's settings; conversations with those models are never written to the recording, and the client is told which is which. Stub, to be thought through with the recording mode it depends on.

## Press Release

Some models keep no transcript, even while recording is on.

Alice runs a Gropius server for her household and has turned recording on, so
that every prompt and every answer is written down on her Mac. Some of what
passes through that server is nobody's business but the person typing it. In
Settings she opens the model she keeps for those conversations and switches its
transcript off. From then on that model answers exactly as it did before, and
nothing it is asked and nothing it replies reaches the recording — not while the
model is excepted, and not afterwards either, because what was never written
cannot be produced later. Every other model on the server goes on being
recorded, and excepting one model never excepts another.

An exception is a fact about a model, not a mode Alice has to remember being in.
It is set per model, alongside the other things Gropius already does differently
for one model rather than for the machine, and it is off for every model until
she says otherwise. It is reachable from all three places a Gropius setting
lives: the control panel, `config.json`, and Go.

And it is published rather than hidden. A client asking Alice's server what
models it offers is told, for each one, whether a conversation with it is
recorded, so a client that wants to show a person which models keep no
transcript has the fact to show it from. What a given client then does with that
fact is the client's business: an ordinary OpenAI-compatible client reads a
model's name and ignores everything beside it, and Gropius cannot make software
display something it was not written to display.

> _Stub. The narrative above is as far as this record can honestly go. It
> describes the setting and its reach and stops where the recording mode it
> excepts from is still undefined: what an excepted conversation means when a
> client changes model mid-thread, what a bridged conversation can be promised,
> and what an unauthenticated client is told are open below, and are settled in
> itd-2609091707499248 first._

## Why This Matters

Some models keep no transcript even while recording is on. Alice names one or more models on an exception list in the server's settings; conversations with those models are never written to the recording, and the client is told which is which. Stub, to be thought through with the recording mode it depends on.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Hold

A stub, filed on 2026-09-09 at the maintainer's request to be thought through
later. It depends on the recording mode (itd-2609091707499248), which is held
on the same day because its notice guarantee cannot be made by a stateless
server; this record inherits that hold. Open when it is picked up: whether the
exception is a per-model flag on the unified `models` settings map; what a
client is told for an excepted model, given the parent's unresolved notice
rule; and whether an excepted model may be reached through a request that names
a recorded one.

## Review 2026-09-19

Two hostile passes over the stub: design and feasibility, then record
discipline. The passes were made separately and are reported separately. Every
finding either reports a constraint this repository already holds, or hands a
question to the maintainer; nothing here decides anything, which is why the
review adds no line to the decisions ledger.

### Design and feasibility

**The exception is not a list.** `config.Config` holds exactly one per-model map
and `internal/archtest/per_model_settings_test.go` holds it there — the project
already collapsed three such structures into one under iss-2609062213413447. A
`no_transcript` array in `config.json` fails that test on the day it is written.
The stored shape is a field on `config.ModelSettings`, off by default like every
other field there. The Hold asked this; it is answered. "List" survives as a
description of what the panel shows, never of what is stored.

**The three surfaces are compulsory rather than chosen.**
`internal/archtest/settings_surface_test.go` walks every json-tagged setting
path and requires the Settings pane's markup to name it and its script to post
it, or an entry in an exemption table that a spec holds at two entries and only
two. A per-model field lands inside that walk. There is no Go-first,
panel-later path for this setting, and buying an exemption would mean breaking
a spec.

**The panel's save can drop the flag, and it drops it the dangerous way.** The
panel rebuilds the whole per-model map from the configuration snapshot it last
loaded, and the server replaces what it holds with the result. A flag written by
hand into `config.json` — the third surface, hand-editable by design — after the
panel loaded is therefore discarded by the next save from that panel, silently.
Every per-model setting has this exposure; the others fail towards a slower
model. This one fails towards a transcript that was promised not to exist. The
setting has to fail closed: either the save path merges this field instead of
replacing the map, or a save over a configuration the panel has not re-read is
refused. Note the direction — the repository's standing hazard is a save
*refused* over an untouched setting, and this is the same wedge pointed the
other way, a save *accepted* that quietly loses one.

**"The client is told which is which" has no field to land in, and the field set
is pinned twice.** `handleListModels` publishes a closed set — id, object,
created, owned_by, chat, and the context figures — and two tests assert it
exactly, refusing both an unexpected field and a missing one. Adding a recording
flag is a deliberate widening of a pinned contract, not a free addition.

**And the listing has two audiences, which cuts the promise in half.** Residency
and activity go only to a client the install has admitted on its key, or to a
client on this Mac; an unentitled client — the LAN, on a keyless install — is
told id, object, created, owned_by and chat, and nothing else. Put the flag in
the entitled half and "the client is told which is which" is false for exactly
the anonymous clients most at risk. Put it in the unentitled half and an
unauthenticated stranger is handed the map of which models leave no trace, which
is the first thing anyone wanting to evade the recording would ask the server
for. Neither arm is obviously right.

**An OpenAI client does not read the field at all.** This is the parent's
feasibility crux, unchanged: a real OpenAI-compatible client reads a model's
`id` and ignores everything beside it. A boolean on the models list tells
software, not a person. The only surfaces that can tell a person are Gropius's
own client and the control panel, so the honest promise is that the fact is
published where a client can read it — not that any client shows it.

**The flag is per model; the promise is per conversation.** The gateway has no
conversations. It has requests, and the request's `model` field is a load
instruction it rewrites on every one of them. A client that changes model
mid-thread — ordinary behaviour — sends turn one to a recorded model and turn
two to an excepted one, and the turn after that carries the excepted model's
prompt and answer inside its own message array as prior context, under the
recorded model's name. A per-model flag applied per request records the excepted
text anyway. The exception has to be defined on whatever unit the recording
stores, and that unit has to be able to refuse a recorded turn that carries
excepted text. The parent decides this; this record cannot.

**The debug-logging draft defeats the exception.** itd-2609062346072707 puts
"every request sent to it and every answer it produced, prompts and completions
included" into one model's log until that model restarts. Switch it on for an
excepted model and the prompts are on disk regardless — in the log rather than
in the recording. The promise is then literally kept and practically broken. A
debug log of every prompt and completion is a transcript; the exception is about
transcripts, not about one file's name.

**Both features already sit outside the telemetry decision.**
adr-2609061503319212 says local telemetry never records prompt text or
completions even when it is on. The recording mode, the per-model debug log and
this exception all live on the far side of that sentence. The parent's Hold
already calls for a superseding record naming its readers and its writer; this
record inherits that need, and an exception is meaningless until the thing it
excepts from has a record permitting it to exist.

**A bridge makes "never written" a promise about one machine only.**
adr-2609181004167097 admits a bridge as a reader of the message and the answer,
and requires the pane taking the token to say that messages and answers pass
through the platform and are kept under its terms. If Bob talks to an excepted
model over the Discord bridge, Gropius writes no transcript and Discord keeps
one. A panel that says "no transcript" beside a model while a bridge is on
states something the product cannot honour. Either the flag's control carries
the bridge exclusion in the same pane — the discipline that ADR already imposes
on the token — or an excepted model is not offered over a bridge at all.

**The bridge also has no models list to be told by.** In Discord, Bob picks a
model with a slash command that lists the chat models the server offers. That
listing is the bridge's own prose, not the models-list JSON, so "the client is
told which is which" needs a second implementation there or Discord users are
told nothing. The bridge is already planned, so this record inherits the
obligation rather than setting it.

**The match folds, like every other model key here.** The registry, the pool and
the configuration all key on `config.FoldRepoID`, and the models listing folds
on both sides of its join for exactly this reason. An exception matched on the
raw string would let one spelling of a model be recorded while another is
excepted. It belongs in an acceptance criterion rather than in an implementer's
head.

**A model can be excepted before it is downloaded, and should be.** The panel
already allows settings on a model that does not exist yet. The alternative —
the flag biting only after the first download — is the kind of thing discovered
after the first leak. There is no unknown-model hole beside it: a request naming
a model the registry does not hold is refused before anything could be written.

### Record discipline

**A stub cannot be planned before the intent it depends on.** `builds_on` names
a draft that is held, and held for a structural reason rather than a scheduling
one: its notice guarantee cannot be made honestly by a stateless server.
Planning this record first would specify an exception to something with no
definition — no store, no record shape, no notice rule. It stays a draft.

**The hold is inherited in prose and nowhere else.** An intent's frontmatter has
no held state, so every record in `drafts/` looks alike to the tooling and
nothing mechanical stops this one being planned once somebody fills in the
criteria. The Hold heading is the only guard there is.

**The itd-1 placeholder is left empty deliberately.** Every criterion this
record could write has a *When* naming machinery the parent has not defined.
Criteria written now would be written against an imagined parent and would be
wrong. The same verdict covers Mechanism and Scope Conditions: the one claim
this record can make alone — that the setting is a field on `ModelSettings`, and
that the surface test then compels its panel control — is a claim about where a
setting lives, not about whether the feature works, and it is recorded here
instead. `None stated.` would be false in all three sections: these claims are
not considered and declined, they are considered and blocked.

**Personas are clean.** Alice is the operator, as everywhere else. Bob appears
in this review only where the bridge intent already puts him, on the far side of
Discord. Carol is unused and there is no reason to conscript her. The maintainer
is they/them throughout.

**The filing status sits inside the promise.** "Stub, to be thought through with
the recording mode it depends on" is captured quoted text, so it is in the
heading and repeated in Why This Matters, where it reads as part of what the
product promises. Correcting it means rewriting the heading, which changes the
slug, which is in the filename. Flagged and not done: the heading is the
maintainer's dictated capture. Nothing outside this file references the id
today, so the rename is cheap whenever they want it.

**Severity is arguable and left alone.** `impact: additive` is right — a new
setting, off by default, nothing existing changing shape. `severity: minor` is
arguable, because the feature's whole job is to keep prompts off disk and a bug
in it writes prompts that were promised not to exist. As a stub it ships
nothing, so the field is left as the maintainer set it.

## Open Questions

What this record needs from the recording intent (itd-2609091707499248) before
it can be planned. None of these can be answered here; each is a property of the
recording mode, and this record can only inherit the answer.

- **What unit the recording stores**, and therefore what an exception is applied
  to. A per-request row makes "conversations with those models are never
  written" untrue as soon as a client changes model mid-thread; a
  per-conversation unit needs a definition of a conversation that a stateless
  gateway can hold.
- **Whether a recorded request may carry an excepted model's text as prior
  context**, and if not, what the recording does with the turn that carries it —
  refuse it, redact it, or record it whole.
- **What "never written" is scoped to.** The recording only, or every place a
  prompt can reach disk. Until the parent names its store and its writer, the
  exception has no boundary to be an exception to.
- **What a client is told, at all**, given the parent's unresolved notice rule.
  If the parent settles on a per-answer notice, an excepted model's answers carry
  what instead — a different notice, or none, which is itself a signal.
- **Whether the superseding decision record the parent needs** covers this
  record's reader and writer too, or whether the exception needs its own.

Decisions that are the maintainer's, not a reviewer's or an implementer's.

- **Which audience of the models list is told.** The entitled half keeps the
  fact from anonymous LAN clients, who are the ones the promise is most for; the
  unentitled half publishes to an unauthenticated network which models leave no
  trace. It is a privacy trade with no neutral option.
- **What happens when an excepted model has debug logging switched on**
  (itd-2609062346072707): the exception refuses the logging, the logging takes
  the model off the exception, or the two are declared orthogonal and both panels
  say so plainly.
- **What an excepted model is promised over a bridge** (adr-2609181004167097):
  the control states the bridge exclusion beside the switch, or an excepted model
  is not offered over a bridge at all.
- **Whether the panel's save must fail closed for this field** — merging it
  rather than replacing the per-model map, or refusing a stale save — and
  whether that rule applies only here or to every per-model setting.
- **Whether `severity: minor` still fits** once the feature is real, given that
  its failure mode is prompts on disk that were promised not to be there.
- **Whether the heading is rewritten** to drop the filing status from the
  promise, accepting the slug and filename change that follows.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
