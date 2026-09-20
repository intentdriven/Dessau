---
id: itd-2609091715089488
slug: some-models-keep-no-transcript-even-while-recording-is-on-al
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091707499248]
severity: major
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Some models keep no transcript even while recording is on. Alice names one or more models on an exception list in the server's settings; conversations with those models are never written to the recording, and the client is told which is which.

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

> _The narrative above described the setting and its reach while the
> recording mode it excepts from was undefined. The parent's questions were
> answered on 2026-09-20 (see `## Open Questions` below and
> `.abcd/work/DECISIONS.md`); what an excepted conversation means when a
> client changes model mid-thread, what a bridged conversation is promised, and
> what an unauthenticated client is told are now settled and carried in the
> Scope Conditions and Acceptance Criteria._

## Why This Matters

Some models keep no transcript even while recording is on. Alice names one or more models on an exception list in the server's settings; conversations with those models are never written to the recording, and the client is told which is which.

## Mechanism

Five claims, each falsifiable against a seam this repository already holds. The
answers of 2026-09-20 are what makes them statable; before them every *When*
named machinery the parent had not defined.

- **We expect the setting to reach the control panel and `config.json` without
  anybody remembering to put it there, because it is a field on
  `config.ModelSettings` and `internal/archtest/settings_surface_test.go` walks
  every json-tagged setting path** — requiring the Settings pane's markup to
  name it and the pane's script to post it, or an entry in an exemption table a
  spec holds at two entries and only two. The field cannot be added in Go alone,
  and there is no exemption to buy that does not break a spec. What would show
  this wrong: the field landing with that walk green and no panel control, which
  would mean the walk does not reach a per-model field and the enumeration half
  was the wrong half — the reading itd-2609081259493890's own mechanism claim
  already offers.
- **We expect no spelling of a model to escape the exception, because the match
  folds through `config.FoldRepoID`, the one rule the registry, the pool and the
  configuration all key by.** Folding on both sides of every join is why "at
  most one model per folded id" is a guarantee rather than a hope, so a raw
  string comparison is the only way one spelling could be recorded while another
  was excepted, and folding removes it. What would show this wrong: two
  spellings of one repo id reaching different recording decisions.
- **We expect a value the panel never drew to survive a save, because the save
  merges the per-model map field by field instead of replacing it.** The panel
  rebuilds its post body from the snapshot it last loaded; a field written into
  `config.json` by hand after that load is absent from the body, so replacing
  the map discards it and merging cannot. The direction matters: every per-model
  setting has this exposure and the others fail towards a slower model, while
  this one fails towards a transcript that was promised not to exist. What would
  show this wrong: an unrendered per-model field missing from the stored
  configuration after a panel-shaped save.
- **We expect the recording decision to be made on the model that serves the
  request, because that is the only decision a stateless gateway can make.** The
  gateway has requests, not conversations, and the request's `model` field is a
  load instruction it rewrites on every one of them; nothing it holds can say
  which model produced a turn already sitting in the message array it was handed.
  So a request carrying an excepted model's earlier turns as prior context is
  recorded whole under the serving model, and the docs say so rather than the
  product pretending otherwise. What would show this wrong: a store the gateway
  could consult to attribute prior turns without keeping conversation state —
  which is the parent's problem, not this record's, and the parent settled it the
  other way.
- **We expect a person to know a model's transcript state before choosing it,
  because the fact is in the base models-list entry every client reads and three
  surfaces render it.** The response header arrives with the answer, which is
  after the choice; the listing is read before it. An icon in the client's picker
  and on the panel's model cards, and a word in the bridge's `/model` prose, is
  what turns a field a program can read into something a person sees — and the
  honest limit stands: an ordinary OpenAI-compatible client reads a model's `id`
  and shows none of it, which Gropius cannot fix from the server. What would show
  this wrong: a person picking an excepted model from one of those three surfaces
  and learning its state only from the answer.

And one claim about the exception's edges rather than its mechanism: **we expect
the exception to hold against the per-model debug log because arming is refused,
not because a promise is made about one file's name.** A refusal at the moment
Alice arms debug logging on an excepted model is enforceable in one place; "the
prompts are in the log rather than in the recording" is the promise literally
kept and practically broken that itd-2609062346072707 would otherwise create.

## Scope Conditions

- **The exception is a field on `config.ModelSettings`, off by default.** Not a
  list, not a second map keyed by model id:
  `internal/archtest/per_model_settings_test.go` holds the count of per-model
  maps at one, and the project already paid a settings-file migration to get
  there (iss-2609062213413447). "List" describes what the panel shows, never
  what is stored. It excepts from the recording mode the parent defines, which
  under the shared-cache install is unavailable altogether — where there is no
  recording there is nothing to except from.
- **A model is matched folded**, through `config.FoldRepoID`, on every path that
  reads the exception, as the registry, the pool and the models listing already
  do.
- **An exception is settable on a model that has not been downloaded**, and
  bites from the first request that model ever serves. The panel already allows
  settings on a model that does not exist yet, and a flag that begins working
  after the first download is the kind of thing discovered after the first leak.
- **A request naming a model the registry does not hold is refused before
  anything is written**, so there is no unknown-model hole beside the exception.
- **An excepted model is not offered over a bridge.** The bridge's `/model`
  listing omits it and a bridged request naming it is refused in the channel
  with the reason, because adr-2609181004167097 admits the platform as a reader
  and keeper of the message and the answer: Gropius would write no transcript
  while Discord kept one, and a panel saying "no transcript" beside that model
  would state what the product cannot honour.
- **Arming the per-model debug log on an excepted model is refused**, with the
  reason, and both panels say so. A debug log of every prompt and completion is
  a transcript; the exception is about transcripts, not about one file's name.
- **A request served by a recorded model is recorded whole, including an
  excepted model's earlier turns carried as prior context**, and the
  user-facing documentation states this plainly rather than leaving it to be
  discovered. The exception is a fact about the model that serves, not about
  text that has passed through another one.
- **The client's built-in on-device model is outside this.** It reaches no
  server, so there is no server-side transcript to keep or to except from, and
  the picker's icon says nothing about it beyond what it already says.

## Acceptance Criteria

- **Given** a build of Gropius, **when** the configuration type is read,
  **then** `config.ModelSettings` carries the transcript field, zero-valued by
  default, and it is the only place the setting is stored.
  *Held by* `TestConfigHoldsExactlyOnePerModelMap`
  (`internal/archtest/per_model_settings_test.go`), which must stay green
  without a new entry in `notKeyedByModelID`.
- **Given** the whole set of json-tagged setting paths, **when**
  `internal/archtest/settings_surface_test.go` walks it, **then** the new
  setting path is named by the Settings pane's script and its control ids are
  present in the markup, with no new entry in `settingsPaneExemptions`.
  *Held by* that test, and by the two-entry bound spc-2609111941481833 puts on
  the exemption table.
- **Given** two models the server knows, **when** Alice opens the Settings
  pane, **then** each model's row carries its own transcript control and the
  posted body names the field for every model the form drew a box for.
  *Held by* the script half of the surface test plus a round-trip test in
  `internal/ui`, which is where wiring is proved.
- **Given** a `config.json` whose per-model map holds a per-model field the
  panel did not render — planted after the panel loaded its snapshot —
  **when** a panel-shaped settings save is posted, **then** the stored map still
  holds that field, for **every** field on `ModelSettings` and not only the
  transcript one.
  *Held by* a new save-path test in `internal/gateway` beside
  `TestSavingSettingsThatOmitsPerModelKeepsIt` and
  `TestASettingTheFormDoesNotOwnSurvivesASave`, written against the surviving
  field rather than against the merge, so a later rewrite of the save path
  cannot satisfy it by accident.
- **Given** a form Alice has not edited, **when** she saves it, **then** the
  save is accepted and nothing is refused over a setting she did not touch.
  *Held by* `TestASaveOfAnUneditedFormIsAccepted` and
  `TestACrossFieldRefusalNamesAChangedField`
  (`internal/gateway/control_untouched_test.go`), which must stay green: the
  merge closes this wedge in the direction of a silently lost setting without
  opening it in the direction of a refused save.
- **Given** a request served by a recorded model whose message array carries
  earlier turns from an excepted model, **when** it is answered, **then** the
  request is recorded whole under the serving model, and no part of it is
  redacted or refused on account of the prior turns.
  *Held by* a gateway test driving a mixed message array and asserting the
  recorded content, and by the sentence stating it on the recording
  documentation page.
- **Given** an exception set under one spelling of a repo id, **when** a request
  arrives under another spelling of the same id, **then** nothing is written.
  *Held by* a folded-match test on the pattern of
  `TestListModelsJoinsResidencyWhateverTheSpelling`.
- **Given** a model named in settings that has never been downloaded, **when**
  the exception is set and the model is later downloaded and served, **then**
  the save was accepted and the first request it ever serves writes nothing.
  *Held by* a settings test asserting acceptance for an unknown key on the terms
  `TestSettingsRejectsAPerModelKeyThatIsNotAModelID` already sets, plus a
  gateway test over the first served request; a request naming a model the
  registry does not hold stays refused, per `TestUnknownModelReturns404`.
- **Given** a models list fetched by an unentitled client — the LAN on a keyless
  install — **when** each entry is read, **then** every entry carries the
  `recording` field, in the base entry beside `chat` rather than in the entitled
  half.
  *Held by* both pinned-field-set tests updated to name it:
  `TestListModelsPublishesContextLengthUnderBothNames` and
  `TestListModelsCarriesNoResidencyWithoutAnAPIKey`
  (`internal/gateway/gateway_test.go`), and by
  `TestModelsListReferenceDocumentsEveryFieldServed`, which holds
  `docs/models-list.md` to the field set served.
- **Given** a server offering one excepted and one recorded model, **when**
  Alice opens the Gropius chat client's model picker and the control panel's
  Models tab, **then** each model shows an icon of its transcript state, read
  from the models-list field and carrying an accessible label, so the state is
  visible before the model is chosen.
  *Held by* an architecture test over `client/GropiusChat/Picker.swift` on the
  pattern of `internal/archtest/chat_client_native_test.go`, by the client's
  Swift unit target (the 2026-09-20 two-tier client-test decision), and by a
  node test over `internal/ui/static` for the model card.
- **Given** the Discord bridge on and one model excepted, **when** Bob runs
  `/model` bare and then names the excepted model, **then** the listing omits
  it and the request is refused in the channel with the reason.
  *Held by* tests in `internal/bridge/discord` beside the existing
  `modelCommand` tests, one per arm.
- **Given** an excepted model, **when** Alice arms per-model debug logging on it
  (itd-2609062346072707), **then** the arming is refused with the reason, and
  both the debug control and the transcript control say so before she tries.
  *Held by* a control-endpoint test in `internal/gateway` for the refusal and a
  panel markup test for the two sentences.
- **Given** the user-facing documentation, **when** the recording page and
  `docs/models-list.md` are read, **then** they state: the setting is per model
  and off by default, matched folded, settable before download, published in the
  models list to every client, shown as an icon in the client and the panel and
  as a word in the bridge, refused over a bridge, refused as a debug-log target,
  that a request carrying an excepted model's earlier turns is recorded whole
  under the serving model, and that an ordinary OpenAI-compatible client
  displays none of it.
  *Held by* `TestModelsListReferenceDocumentsEveryFieldServed` for the field and
  by a hand check with the docs-currency reviewer before the release that ships
  this — recorded on the record, since prose completeness is not a test.

## Hold

A stub, filed on 2026-09-09 at the maintainer's request to be thought through
later. It depends on the recording mode (itd-2609091707499248), which is held
on the same day because its notice guarantee cannot be made by a stateless
server; this record inherits that hold. Open when it is picked up: whether the
exception is a per-model flag on the unified `models` settings map; what a
client is told for an excepted model, given the parent's unresolved notice
rule; and whether an excepted model may be reached through a request that names
a recorded one.

**The hold is lifted on 2026-09-20, with the parent's.** itd-2609091707499248's
six questions were answered at the 2026-09-20 interview and this record's eight
in the same sitting, so the recording mode now has a store, a record shape, a
notice rule and a stated answer to the served-versus-named case — the three
structural unknowns that made a criterion here unwritable. Every question above
and under Open Questions is answered, each in `.abcd/work/DECISIONS.md` under
2026-09-20. What remains is planning, not a decision.

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

Every question below is answered. They are kept as asked, each with the answer
and where it is recorded, so that a later reader sees what was open and what
closed it rather than a section that was quietly emptied.

What this record needed from the recording intent (itd-2609091707499248) before
it could be planned. None of these could be answered here; each is a property of
the recording mode, and this record could only inherit the answer.

- **What unit the recording stores**, and therefore what an exception is applied
  to. A per-request row makes "conversations with those models are never
  written" untrue as soon as a client changes model mid-thread; a
  per-conversation unit needs a definition of a conversation that a stateless
  gateway can hold.
  **Answered 2026-09-20:** the decision is made per request, on the model that
  serves it, because the gateway has requests and not conversations
  (`.abcd/work/DECISIONS.md`, 2026-09-20, itd-2609091715089488); the parent's
  store is a size-capped statistics store, default 512 MB (same date,
  itd-2609091707499248 Questions 1–3).
- **Whether a recorded request may carry an excepted model's text as prior
  context**, and if not, what the recording does with the turn that carries it —
  refuse it, redact it, or record it whole.
  **Answered 2026-09-20:** recorded whole, under the serving model, and the docs
  say so plainly (`.abcd/work/DECISIONS.md`, 2026-09-20,
  itd-2609091715089488). Mechanism claim four and the sixth acceptance criterion
  carry it.
- **What "never written" is scoped to.** The recording only, or every place a
  prompt can reach disk. Until the parent names its store and its writer, the
  exception has no boundary to be an exception to.
  **Answered 2026-09-20:** every transcript the product knows of — hence the
  refused debug arm and the bridge exclusion (`.abcd/work/DECISIONS.md`,
  2026-09-20, itd-2609091715089488, points 2 and 3).
- **What a client is told, at all**, given the parent's unresolved notice rule.
  If the parent settles on a per-answer notice, an excepted model's answers carry
  what instead — a different notice, or none, which is itself a signal.
  **Answered 2026-09-20:** the notice is an unconditional response header, whose
  name carries the family-name prefix, plus a `recording` field per models-list
  entry, so the header states the truth for the model that answered and the
  listing states it per model before anything is asked
  (`.abcd/work/DECISIONS.md`, 2026-09-20, itd-2609091707499248 Question 1).
- **Whether the superseding decision record the parent needs** covers this
  record's reader and writer too, or whether the exception needs its own.
  **Answered 2026-09-20:** the parent's two superseding ADRs cover it — one over
  adr-2609061503319212 and one over adr-2609061610102325, each linked in both
  directions — and this record adds none of its own
  (`.abcd/work/DECISIONS.md`, 2026-09-20, itd-2609091707499248 Questions 4–6).

Decisions that were the maintainer's, not a reviewer's or an implementer's.

- **Which audience of the models list is told.** The entitled half keeps the
  fact from anonymous LAN clients, who are the ones the promise is most for; the
  unentitled half publishes to an unauthenticated network which models leave no
  trace. It is a privacy trade with no neutral option.
  **Answered 2026-09-20:** the unentitled half — the base entry every client
  sees, beside `chat` — because disclosure is the fact that must reach everyone,
  and the per-request header already reveals it per model
  (`.abcd/work/DECISIONS.md`, 2026-09-20, itd-2609091715089488, point 1). The
  ninth acceptance criterion pins it to both field-set tests.
- **What happens when an excepted model has debug logging switched on**
  (itd-2609062346072707): the exception refuses the logging, the logging takes
  the model off the exception, or the two are declared orthogonal and both panels
  say so plainly.
  **Answered 2026-09-20:** the exception refuses the arming, with the reason,
  and both panels say so (`.abcd/work/DECISIONS.md`, 2026-09-20,
  itd-2609091715089488, point 2).
- **What an excepted model is promised over a bridge** (adr-2609181004167097):
  the control states the bridge exclusion beside the switch, or an excepted model
  is not offered over a bridge at all.
  **Answered 2026-09-20:** not offered at all — the bridge's `/model` listing
  omits it and a bridged request naming it is refused in the channel with the
  reason (`.abcd/work/DECISIONS.md`, 2026-09-20, itd-2609091715089488,
  point 3).
- **Whether the panel's save must fail closed for this field** — merging it
  rather than replacing the per-model map, or refusing a stale save — and
  whether that rule applies only here or to every per-model setting.
  **Answered 2026-09-20:** the save merges, and for every per-model field rather
  than for this one, at the cost of one seam (`.abcd/work/DECISIONS.md`,
  2026-09-20, itd-2609091715089488, point 4). The fourth and fifth acceptance
  criteria hold both directions of the wedge.
- **Whether `severity: minor` still fits** once the feature is real, given that
  its failure mode is prompts on disk that were promised not to be there.
  **Answered 2026-09-20:** it does not; `severity: major`
  (`.abcd/work/DECISIONS.md`, 2026-09-20, itd-2609091715089488). The frontmatter
  carries it.
- **Whether the heading is rewritten** to drop the filing status from the
  promise, accepting the slug and filename change that follows.
  **Answered 2026-09-20:** rewritten (`.abcd/work/DECISIONS.md`, 2026-09-20,
  itd-2609091715089488). The dropped sentence sits past the slug's
  sixty-character truncation, so in the event the derived slug is unchanged and
  the filename with it — the change the answer accepted turned out not to be
  owed.

And one addition the maintainer made rather than a question they answered.

- **A model's transcript state is shown as an icon wherever a person picks a
  model** — the Gropius chat client's picker and the control panel's model
  cards, and as a word in the bridge's `/model` prose — read from the models-list
  field, so the person knows before choosing rather than after asking
  (`.abcd/work/DECISIONS.md`, 2026-09-20). This is the client-side half of "the
  client is told which is which", and it answers the review's finding that a
  boolean on a models list tells software rather than a person: it does not make
  a third-party client display anything, and it does make Gropius's own three
  surfaces display it. Mechanism claim five and the tenth acceptance criterion
  carry it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

**Answers 2026-09-20:** unentitled models-list field; the exception refuses a debug arm; not offered over a bridge; merge-save for every per-model field; severity major; heading rewritten; recording decided on the serving model, prior context recorded whole. Recorded in `.abcd/work/DECISIONS.md`.

**Addition 2026-09-20:** the transcript state is shown as an icon wherever a person picks a model (client picker, panel model cards; a word in the bridge's `/model` listing). Recorded in `.abcd/work/DECISIONS.md`.
