---
id: spc-2609201007367486
slug: some-models-keep-no-transcript-even-while-recording-is-on-al
intent: itd-2609091715089488
origin: researcher-authored
production_mode: hand-written
---

# Some models keep no transcript, even while recording is on

## Summary

This spec delivers itd-2609091715089488: one field on `config.ModelSettings`,
`no_transcript`, off by default, which takes a model out of the recording the
parent intent (itd-2609091707499248, spec spc-2609201007360569) defines. The
exception is matched folded, settable on a model that has not been downloaded,
decided on the model that **serves** each request, published per entry in the
models list every client can read, shown as an icon wherever a person picks a
model and as a word in the bridge's `/model` prose, refused over the Discord
bridge, and refused as a target for per-model debug logging
(itd-2609062346072707). One seam changes for everybody: the settings save
merges the per-model map field by field instead of replacing it, so a value
the panel never drew survives the next save — for every field on
`ModelSettings`, not only this one. `impact: additive`; `severity: major`,
because the failure mode is prompts on disk that were promised not to exist.

The recording store, the global switch, the unconditional response header and
the models-list entry those three arrive with belong to the parent and are
referenced here, never respecified. Two ADRs carry the boundary this work sits
on the far side of — adr-2609201008470380 and adr-2609201008476813, written in
the parent's lane, superseding adr-2609061503319212 and adr-2609061610102325.

## Scope

In: `internal/config` (the field, its reader, the merge helper);
`internal/gateway` (the merge-save in `applySettings`, the transcript
decision at the point the serving model is known, the `recording` value on the
models-list entry, the refused debug arm); `internal/ui` (the per-model
transcript control, the model card's icon, the two sentences about the debug
arm, the panel-side tests that evaluate `app.js`); `internal/ui/static`
(`app.js`, `index.html`); `internal/bridge/discord` (the `/model` listing's
omission and word, the refusal in the channel); `client/DessauChat`
(`DessauChat.swift`'s models-list decode, `Picker.swift`'s icon);
`internal/archtest` (the client architecture test); `docs/` (the recording
page's section, the `docs/models-list.md` row, the client README line); the
changelog.

Out: everything under **Out of scope** below.

## Approach

### The field

`config.ModelSettings` gains one field beside `MergeSystemMessages`, `Pinned`,
`Sampling` and `ServedContext`:

```go
// NoTranscript takes this model out of the recording: while it is set,
// nothing this model is asked and nothing it answers is written to the
// transcript store, whether or not the machine-wide switch is on. Off
// unless the operator sets it for this model.
NoTranscript bool `json:"no_transcript,omitempty"`
```

A field, not a list and not a second map: `TestConfigHoldsExactlyOnePerModelMap`
holds the count of per-model maps at one and the project already paid a
settings-file migration to get there (iss-2609062213413447, cond-2609201007367769).
The zero value is "recorded", so a configuration written before this change
means what it meant.

Because the field is json-tagged, `internal/archtest/settings_surface_test.go`
walks it as `models.*.no_transcript` and requires the segment `no_transcript`
to be named by the Settings pane's script, with every control id that script
binds present in the markup — or an entry in `settingsPaneExemptions`, which
spc-2609111941481833 holds at two entries and only two. There is no exemption
to buy, so the panel control is compulsory rather than chosen. The walk's own
comment says what it cannot do: a per-model setting reaches the panel through
a generated row, so the markup half does not see the row and the wiring is
proved by the round-trip test in `internal/ui` instead.

**Reading it, folded.** A reader on `config.Config`, on the precedent of
`Config.ServedContext`:

```go
// NoTranscript reports whether this model is excepted from the recording.
func (c Config) NoTranscript(repoID string) bool
```

It takes the exact key first, then every key that folds alike under
`config.FoldRepoID`, and **any** folded match that carries the exception wins.
`ServedContext` resolves the same ambiguity by taking the largest figure; the
same reasoning — deterministic, and never smaller than what the operator asked
for — here means failing closed, because the operator asked for nothing to be
written. Duplicate spellings are refused on the settings path
(`ValidateModelKeys`) and dropped on the file path (`sanitizeModels`), so a map
holding two reached the reader some other way; the reader answers the same on
every run of the same binary either way (cond-2609201007366437).

Every path that decides whether to write goes through this reader, never
through a raw index into `cfg.Models`.

**Settable before download.** Nothing on the settings path requires the model
to exist: `TestSettingsRejectsAPerModelKeyThatIsNotAModelID` refuses a key that
is not shaped like a repo id and accepts one that is, and the exception is read
on the first request the model ever serves rather than at download time
(cond-2609201007362675). A request naming a model the registry does not hold is
refused before anything could be written, as `TestUnknownModelReturns404`
already holds (cond-2609201007367665).

### The merge-save, for every per-model field

`Control.applySettings` (`internal/gateway/control.go`) today does this:

```go
if namesModels(raw) {
    incoming.Models = nil
}
if err := json.Unmarshal(raw, &incoming); err != nil { … }
```

The panel rebuilds the whole per-model map from the snapshot it last loaded, so
a field written into `config.json` by hand after that load is absent from the
posted body and the `nil` above discards it, silently. Every per-model setting
has this exposure; the others fail towards a slower model, and this one fails
towards a transcript that was promised not to exist.

The replacement merges, and merges for every field on `ModelSettings`
(2026-09-20 decision, point 4):

1. Decode the body's `models` object into `map[string]json.RawMessage`.
2. The merged map holds **exactly the keys the body names** — so clearing every
   box for a model still removes that model, which is the rule
   `TestSavingSettingsThatOmitsPerModelKeepsIt` and the panel's own
   `modelSettings()` are written around, and `namesModels` still decides
   whether the body speaks about the map at all.
3. For each posted key, find the stored entry — exact key first, then folded —
   encode it as `map[string]json.RawMessage`, overwrite the keys the posted
   object names, and decode the result back into `ModelSettings`.

Merging at the **encoded-object level** rather than by unmarshalling the posted
object into the stored struct is the load-bearing detail: `encoding/json`
decodes into an existing struct in place, so decoding a posted `"sampling": {}`
into a stored entry would merge the nested object too and an override the
operator had just cleared would come back. Overwriting whole keys means a
posted key replaces that field entirely, and only a key the body never names —
`no_transcript`, hand-written after the panel's load — survives. The technique
is the one `changedSettings` already uses through `encodedSettings`, in the same
file.

The opposite wedge is kept shut by construction. `changedSettings` compares the
two configurations as encoded values, so a field the merge preserved is
byte-identical before and after and appears among no save's changes:
`TestASaveOfAnUneditedFormIsAccepted` and
`TestACrossFieldRefusalNamesAChangedField` stay green, and a cross-field
refusal still names a field this save actually changed. AGENTS.md records that
a save refused over a setting nobody touched is a wedge this repository has
built three times; this change closes the other direction without reopening it.

### The serving-model decision

The gateway has requests, not conversations, and the request's `model` field is
a load instruction it rewrites on every one of them (gateway.go: "The
load-bearing rewrite"). The transcript writer the parent's spec defines is
therefore called **after** the pool has handed back the model that will answer
— beside the existing per-model read on that path — and the decision is
`cfg.NoTranscript(servingModel)`:

- If the serving model is excepted, nothing about the request or the answer
  reaches the store: no row, no truncated row, no metadata row carrying the
  prompt's shape. What was never written cannot be produced later.
- If the serving model is recorded, the request is recorded **whole**, under
  the serving model, including turns an excepted model produced that the client
  carried back as prior context. Nothing is redacted, nothing is refused on
  account of them (cond-2609201007364849). Dessau cannot attribute a turn in a
  message array it was handed to the model that produced it without keeping
  conversation state it deliberately does not keep, and pretending otherwise
  would be a promise it could not hold.

The user-facing documentation states this plainly rather than leaving it to be
discovered; see **The documentation**.

### What the models list says

The parent's spec introduces `recording` on each models-list entry. This spec
owns two things about it, and specifies no more of it than that.

**Its value is the per-model truth.** `recording` is `true` for an entry only
when the machine-wide switch is on **and** that model is not excepted;
otherwise `false`. The folded reader above is what the handler asks, on the
same folded join `handleListModels` already performs for residency.

**It sits in the base half.** `handleListModels` publishes two classes of fact:
what a model **is** — `chat`, the context figures — to every client, and what
this Mac is **doing** — `state`, `in_flight`, `last_used`, `pinned` — only to a
client the install has admitted on its key or one on this Mac. `recording` is
the first class: it goes beside `chat`, to the unentitled LAN client on a
keyless install too (2026-09-20 decision, point 1). The two pinned field-set
tests, `TestListModelsPublishesContextLengthUnderBothNames` and
`TestListModelsCarriesNoResidencyWithoutAnAPIKey`, are widened to name it —
deliberately, since both refuse an unexpected field and a missing one — and
`TestModelsListReferenceDocumentsEveryFieldServed` holds `docs/models-list.md`
to the set served.

Whichever of this spec and spc-2609201007360569 lands first carries the field
into those tests; the second adds only its half of the value rule, and the
criterion below is satisfied when both halves are in.

### The icon, the card, and the bridge's word

A boolean on a models list tells software. Three surfaces turn it into
something a person sees before they choose (2026-09-20 addition).

**The chat client.** `ModelsResponse.Model` in `DessauChat.swift` gains
`let recording: Bool?` beside `let chat: Bool?`, and the connect path keeps the
folded ids whose `recording` is `false` in a published set beside `chatModels`.
`Picker.swift`'s model rows draw the state beside each model: a chosen-state
`Label` today, with the transcript icon added as a trailing view, its
`accessibilityLabel` reading "keeps no transcript" or "recorded" in words
rather than in a symbol's name. A server that says nothing — an older Dessau,
or one whose entry carries no `recording` — draws no icon at all, because an
absent fact is not a promise. The client's built-in on-device answerer reaches
no server, so it carries no icon and the picker says nothing new about it
(cond-2609201007366686).

**The control panel.** The model cards in `app.js` are built from the control
snapshot, not from `/v1/models`, so the card computes the same value from the
snapshot it already holds — the switch and `state.config.models[id]` — in one
named helper, and shows the same icon with the same words. That parity is a
rule stated in two places rather than one fact read twice, and this spec says
so rather than implying the card reads the served field.

**The bridge.** `modelList()` in `internal/bridge/discord/commands.go` renders
the models on offer; excepted models are not on offer at all (below), so the
prose gains one word for the ordinary case — each listed model is a recorded
one while the switch is on, and the listing says so plainly in the line it
already prints.

### The bridge omits and refuses

adr-2609181004167097 admits the platform as a reader and keeper of the message
and the answer. Dessau would write no transcript while Discord kept one, and a
panel saying "no transcript" beside that model would state what the product
cannot honour. So an excepted model is not offered over a bridge at all
(cond-2609201007365982, 2026-09-20 decision, point 3):

- `modelCommand`'s listing — bare `/model` and the list appended to a refusal —
  is filtered through the folded reader, so an excepted model never appears.
- A `/model` naming an excepted model, by repo id or short name, is refused in
  the channel with the reason: that the model keeps no transcript on this
  server and a message sent through Discord would be kept by Discord.
- A channel already pointed at a model that is excepted afterwards is refused
  at the next message with the same reason, because the reader is asked per
  request rather than at the time the channel chose.

### The refused debug arm

itd-2609062346072707 puts every request sent to one model and every answer it
produced, prompts and completions included, into that model's log. A debug log
of every prompt and completion is a transcript; the exception is about
transcripts, not about one file's name. So arming debug logging on an excepted
model is refused, at the control endpoint, with the reason
(cond-2609201007364816, and the debug intent's own cond-2609201007351874).

Both panels say so **before** Alice tries: the debug control on a model card
carries the sentence that a model with its transcript switched off cannot be
armed, and the transcript control carries the sentence that switching it on
takes the model off debug logging's list. The refusal is enforced in one place
— the arming endpoint — rather than being a promise about which file a prompt
lands in.

The reverse direction is deliberately not symmetrical: excepting a model that
is currently armed refuses nothing at save time. The arming endpoint is where
the two meet, and the debug intent's own mark is consumed at the model's next
launch, so the refusal there is what makes the exception hold; the transcript
control's sentence is what tells Alice the armed model will stop being armed.
The debug intent's lane owns the mark's clearing.

### The documentation

- The recording page the parent's spec creates (`docs/transcript.md`) gains a
  section on exceptions: per model and off by default, matched folded, settable
  before a model is downloaded, published in the models list to every client,
  shown as an icon in the chat client and the panel and as a word in the
  bridge, not offered over a bridge, refused as a debug-log target, and — in
  its own short paragraph — that a request served by a recorded model is
  recorded whole, including earlier turns from an excepted model that the
  client carried back as prior context.
- `docs/models-list.md` gains the `recording` row in the field table, in the
  base half above the entitled fields, saying what `false` means and that it is
  always present.
- The client README's feature list gains one line: the picker shows which
  models keep no transcript.
- The honest limit is stated once, on the recording page: an ordinary
  OpenAI-compatible client reads a model's `id` and displays none of this, and
  Dessau cannot fix that from the server.

British English, present tense, one Diátaxis type per page.

## How each criterion is held

Every row is a test watched to fail before the change and pass after, or a hand
check recorded on the shipping decision line. Criteria are numbered in the
order itd-2609091715089488 states them.

| # | Criterion | Held by |
| --- | --- | --- |
| 1 | The field is on `ModelSettings`, zero by default, stored nowhere else | `TestConfigHoldsExactlyOnePerModelMap` (`internal/archtest/per_model_settings_test.go`), green with no new entry in `notKeyedByModelID` |
| 2 | The settings-surface walk names the path | `internal/archtest/settings_surface_test.go`, script half, path `models.*.no_transcript`, no new entry in `settingsPaneExemptions` |
| 3 | Each model's row carries its own control and the posted body names it | a round-trip test in `internal/ui` evaluating `modelSettings()` and the new row's collectors through `evalPanelValue`, beside the existing per-model settings test |
| 4 | An unrendered per-model field survives a panel-shaped save, for **every** field | a new save-path test in `internal/gateway` beside `TestSavingSettingsThatOmitsPerModelKeepsIt` and `TestASettingTheFormDoesNotOwnSurvivesASave`: plant a field in the stored map after the snapshot, post a panel-shaped body, assert the surviving field — written against the survival, not against the merge, and table-driven over every field on `ModelSettings` so a field added later is covered |
| 5 | An unedited form is accepted; a cross-field refusal names a changed field | `TestASaveOfAnUneditedFormIsAccepted` and `TestACrossFieldRefusalNamesAChangedField` (`internal/gateway/control_untouched_test.go`), which must stay green |
| 6 | A recorded model's request carrying excepted prior turns is recorded whole under the serving model | a gateway test driving a mixed message array and asserting the recorded content and its model, plus the paragraph on the recording page |
| 7 | An exception set under one spelling bites under another | a folded-match test on the pattern of `TestListModelsJoinsResidencyWhateverTheSpelling`, plus a unit test of `Config.NoTranscript` over exact, folded and duplicate keys |
| 8 | Settable on a model that has never been downloaded; the first request it serves writes nothing | a settings test on the terms `TestSettingsRejectsAPerModelKeyThatIsNotAModelID` sets, plus a gateway test over the first served request; `TestUnknownModelReturns404` stays green |
| 9 | Every models-list entry carries `recording`, in the base entry beside `chat` | `TestListModelsPublishesContextLengthUnderBothNames` and `TestListModelsCarriesNoResidencyWithoutAnAPIKey` widened to name it, and `TestModelsListReferenceDocumentsEveryFieldServed` |
| 10 | The icon in the client's picker and on the panel's model cards, with an accessible label | an architecture test over `client/DessauChat/Picker.swift` and `DessauChat.swift` on the pattern of `internal/archtest/chat_client_native_test.go`; the client's Swift unit target (2026-09-20 two-tier client-test decision) over the decode and the folded set; a test in `internal/ui` evaluating the card's helper for both states |
| 11 | The bridge's `/model` omits an excepted model and refuses one named | two tests in `internal/bridge/discord` beside the existing `modelCommand` tests, one per arm, plus one for a channel excepted after it chose |
| 12 | Arming debug logging on an excepted model is refused, and both panels say so | a control-endpoint test in `internal/gateway` for the refusal and its reason; a panel markup test for the two sentences |
| 13 | The documentation states all of it | `TestModelsListReferenceDocumentsEveryFieldServed` for the field; a hand check with the docs-currency reviewer before the release that ships this, recorded on the shipping decision line, since prose completeness is not a test |

Scope conditions, and where each is discharged:

| Condition | Where |
| --- | --- |
| cond-2609201007367769 | **The field** — one field on `ModelSettings`, off by default |
| cond-2609201007366437 | **The field**, "Reading it, folded" — `Config.NoTranscript` |
| cond-2609201007362675 | **The field**, "Settable before download" |
| cond-2609201007367665 | **The field**, "Settable before download" — `TestUnknownModelReturns404` |
| cond-2609201007365982 | **The bridge omits and refuses** |
| cond-2609201007364816 | **The refused debug arm** |
| cond-2609201007364849 | **The serving-model decision** and **The documentation** |
| cond-2609201007366686 | **The icon, the card, and the bridge's word** — the client's on-device answerer |

## Security

This is a trust-boundary change: `internal/config` parses the field,
`internal/gateway` decides on it at the network edge, and an adversarial review
is owed before it lands.

**The failure mode is a save that silently drops the flag.** The dangerous
outcome is not a refused save or a broken panel; it is a configuration in which
`no_transcript` was set, a subsequent panel save wrote the file without it, and
Alice has no signal at all that a model she excepted is being recorded again.
That is why the fourth criterion is written against the surviving value rather
than against the merge, and why it is table-driven over every field: a later
rewrite of the save path must not be able to satisfy it by accident. It is also
why the reader fails closed on duplicate folded keys, and why the exception is
read per request rather than cached per channel or per conversation.

**The map of excepted models is disclosed on purpose.** `recording` goes to the
unentitled half, which means an unauthenticated client on the LAN of a keyless
install can ask the server which models leave no trace. That is the maintainer's
decision of 2026-09-20 (point 1), taken with the trade named: disclosure is the
fact that must reach everyone the promise is for, the anonymous LAN clients are
exactly the ones least able to learn it any other way, and the parent's
unconditional per-answer header already reveals the same fact per model to
anyone who sends one request. Withholding it from the listing would hide it
from the people it protects while hiding it from nobody who asked twice.

**What the exception does not cover, and says so.** It is a promise about
transcripts this product writes on this Mac. It does not reach a bridge's
platform — hence the omission and the refusal rather than a caveat in a pane —
and it does not reach the model server's own stdout at a level Alice set by
hand outside the arming path; the recording page says both plainly rather than
letting "never written" be read as a promise about every file on the Mac.

**The refusal's words.** The bridged refusal and the debug-arm refusal name the
reason, which discloses that a model is excepted. Both are answers to a request
that named the model, on surfaces the fact is already published to, so neither
adds a disclosure the models list has not already made.

## Verification

- `make test` green, `gofmt -l .` empty, `go vet ./...` clean.
- Every test named above watched red before the change and green after; the
  merge-save test planted against the unmerged save path first.
- The client's Swift unit target for the decode and the folded set; the
  architecture test for the picker.
- Adversarial security review of the diff before it is presented
  (`internal/config`, `internal/gateway`).
- Hand checks, recorded on the shipping decision line: except a model in the
  panel and confirm the icon changes on the card and in the chat client's
  picker; hand-edit `config.json` to add a per-model field the panel does not
  draw, save an untouched form from the panel, and confirm the field is still
  there; ask an excepted model a question over the bridge and confirm the
  refusal; try to arm debug logging on it and confirm the refusal and both
  sentences; run the docs-currency reviewer over the recording page and
  `docs/models-list.md`.

## Out of scope

- **The recording store itself** — its shape, its size cap, its retention and
  its unavailability under the shared-cache install belong to
  itd-2609091707499248 and spc-2609201007360569.
- **The response header** — its name, its prefix and its unconditional presence
  are the parent's; this spec reads nothing from it and changes nothing about
  it.
- **The global switch** and its panel control, likewise the parent's.
- **The debug log itself** — the mark, its launch-time consumption and its
  clearing belong to itd-2609062346072707; only the refusal at the arming
  endpoint is here.
- **Per-conversation exceptions**, an exception applied to text rather than to
  the model that serves it, and any attribution of prior turns to the model
  that produced them: refused by the 2026-09-20 answer and by the gateway
  holding no conversation state.
- **Migration.** Pre-1.0, the zero value is the old behaviour and no
  compatibility shim is written.

## Falsifiers

From the intent's Mechanism claims, each of these reopens the record rather
than being patched: the field landing with the settings-surface walk green and
no panel control; two spellings of one repo id reaching different recording
decisions; an unrendered per-model field missing from the stored configuration
after a panel-shaped save; a person picking an excepted model from the client's
picker, the panel's cards or the bridge's listing and learning its state only
from the answer.

## Departures

Recorded at the close, rendered from the implementer and fix reports and the reviews.

- **The transcript decision is not called on the completions path at the point the serving model is known.** The parent (itd-2609091707499248 / spc-2609201007360569) has not landed on this HEAD: there is no internal/transcript package, no retainer sink on relayOptions, no recording header and no machine-wide switch. Gateway.recorded(cfg, servingModel) holds the rule (switch AND not excepted, read through Config.NoTranscript) and its comment names the call site — after g.pool.Acquire hands back the serving model, beside the existing per-model read on that path. Calling it there today would be a dead call with nothing to consume the answer. The two gateway tests of criteria 6 and 8 (a mixed message array recorded whole under the serving model; the first served request of an excepted model writing nothing) need the store and are owed to the integration with the parent.
- **Options.TranscriptOn func() bool was added to the gateway as the seam for the machine-wide switch; nil reads as off.** The switch is the parent's (config.Transcript). Without a seam the models-list value could only ever be false and neither arm of the value rule could be tested. It is the same shape the debug lane used for TranscriptExcepted; the parent's integration replaces g.transcriptOn() with cfg.Transcript in one line and drops the option.
- **Control.TranscriptExcepted (the debug lane's seam) was removed; handleDebugLog reads c.App.Config().NoTranscript(req.Model) directly.** The seam's own comment said it stood in until the per-model field existed. Reading the configuration is one fewer thing for part 2's app wiring to set and the refusal is live the moment the exception is saved (TestExceptingAModelInSettingsRefusesItsNextDebugArm). The closed debug spec and shipped intent still cite the seam by name; those records are history and were not edited.
- **One row was added to docs/models-list.md (the `recording` row) although docs are part 2's.** TestModelsListReferenceDocumentsEveryFieldServed in internal/gateway — my package — holds the page to the field set served and went red the moment the field landed; the row is what the spec's criterion 9 names. Part 2 owns every other docs change.
- **internal/lifecycle/testdata/config_show.json gained the no_transcript line.** `dessau config show --json` flattens every per-model field and its golden pins the contract; the terminal is the third surface and the field belongs there. A fixture update, not a feature.
- **docs/transcript.md was created by this record rather than gained as a section of a page the parent created.** The spec says the recording page the parent's spec creates gains a section on exceptions; the parent (itd-2609091707499248 / spc-2609201007360569) has not landed on this HEAD, so there was no page to add to and the panel's Transcript control links the page by name. The page is the how-to for the exception; its opening sentence describes the server's transcript 'while it is on', which is the parent's switch, on the same footing as the `recording` row part 1 wrote in docs/models-list.md. The parent's integration adds its own sections (turning it on, the cap, deletion, the shared-cache refusal) to the same page.
- **The panel's card helper reads the machine-wide switch as state.config.transcript, a key this HEAD does not carry.** The spec has the card compute the models-list value from the snapshot's switch and the per-model map in one helper; the parent's spec names the field `transcript` on config.Config, so that is the key transcriptState reads, and an absent key reads as off — the same nil-is-off rule part 1 gave gateway.Options.TranscriptOn. Until the parent lands every card and every picker row reads 'keeps no transcript', which is what is true: nothing is recorded.
- **The picker's two label strings live in Picker.swift, not in the Foundation-free TranscriptState.swift the unit tier compiles.** Part 1's TestChatClientPickerShowsTheTranscriptStateWithWords requires the literals in Picker.swift. The unit tier covers what the spec assigns it — the decode's absent case and the folded set — and the words are held by the archtest; putting them in both files would have been a second copy of one fact.
- **The form posts every per-model box explicitly (a clear box as false, a blank window as 0, a removed sampling override as {}), which rewrites applyModelSwitch/applyModelNumber and the expectations of two existing internal/ui tests.** Part 1's note_for_next named this as REQUIRED: the merge-save reads an absent key as 'keep', so the old delete-the-key form could never switch a per-model setting off. The server drops a zero entry, so the zeros store nothing; TestASaveOfAnUneditedFormIsAccepted stays green because changedSettings compares encoded configurations. Criterion 3's 'the posted body names the field for every model the form drew a box for' is this.
- **docs/discord-bridge.md gained one bullet under Limits that the spec's docs list does not name.** The bridge's `/model` omission and refusal are user-facing changes to the bridge, and AGENTS.md's definition of done reflects user-facing changes in docs/; one sentence pointing at the transcript page rather than a second account of the rule.
- **Not applied from review:** ruthless (high): /v1/models reports recording:false for every model, so an excepted and a recorded model are indistinguishable, falsifying criteria 9 and 10. — Applied as a capture, not a code change. The value is the spec's rule — recording = switch on AND not excepted — and on a tree with no switch nothing is recorded, so false for every model is the truth, not a bug in the reader; the client and the panel draw exactly that truth. What is wrong is the tree, not the branch: the parent has not landed, which the record decides must happen first. Changing the value rule to publish the per-model exception alone would contradict the 2026-09-20 decision (point 1) and part 1's tests, and would tell a client a model was recorded when nothing is. iss-2609211218478273 names the dependency and, being major, refuses the release cut while open.
- **Not applied from review:** ruthless: gate the CHANGELOG/docs additions that claim the transcript feature works until the parent lands. — Not rewritten. The changelog entry sits under [Unreleased] and cannot ship while the major capture is open; docs/transcript.md, the panel's Transcript hint and the models-list row (part 1's) are written for the tree the ordering decision produces, and rewriting them to say 'there is no transcript' and back again when the parent lands would be two edits to reach the state one landing gives. The gate is the capture on the release cut and the docs-currency hand check already owed; the finding's own proposed fix ('do not merge until the parent has landed') is the loop's to honour and is recorded in the capture for it.
- **Not applied from review:** ruthless note: postedModels only partially handles two differently-cased top-level "models" keys in one body. — Not raised as a finding by the reviewer and not reachable from the panel; part 1's code in internal/config (a trust boundary) that the record does not decide — left as the reviewer left it, noted here so it is not lost.
- **Captured, not fixed:** iss-2609211218478273
