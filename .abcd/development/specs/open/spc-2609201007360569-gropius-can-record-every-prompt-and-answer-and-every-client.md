---
id: spc-2609201007360569
slug: gropius-can-record-every-prompt-and-answer-and-every-client
intent: itd-2609091707499248
origin: researcher-authored
production_mode: hand-written
---
# The transcript store, and the disclosure on every answer

## Summary

This spec delivers itd-2609091707499248: a **transcript** — with one switch on,
every prompt a client sends and every answer a model gives is written to a file
of its own in the operator's account, mode 0600, on the existing rotating
writer, bounded by a size cap she sets and removed only by a deliberate
**Delete transcript**. It is off by default and nothing else turns it on. With
it on, the server says so on every path a program can see: a recording header on
every answer including the refusals Gropius composes itself, and a per-entry
field on the models list. Gropius's own chat client is held to putting that in
front of Bob before he types; the Discord bridge writes it into the first answer
of each channel. Under the shared-cache install the switch is unavailable and
the pane says why. The statistics store is untouched and stays content-free.

The decisions this implements are the maintainer's of 2026-09-20, ratified as
adr-2609201008470380 (superseding adr-2609061610102325) and adr-2609201008476813
(superseding adr-2609061503319212). `impact: additive`; a new store, off by
default, no stored format changed.

## Scope

**In.** A new package `internal/transcript` (the record, the writer, the
deletion); `internal/config` (the switch, the cap and its floor, the store's
directory, the shared-root refusal); `internal/gateway` (the disclosure header
and the one wrapper that writes it; the models-list field; the seam that hands
the relay's bytes to the retainer; `Ask`, so a bridged turn is written on the
same path); `internal/app` (opening and closing the store on a settings apply,
the refusal at start under a shared root); `internal/bridge/discord` (the
per-channel notice in the editor's first message); `internal/ui/static` (the
Transcript fieldset, the cap, the oldest-record line, **Delete transcript**);
`internal/archtest` (the retainer named on both readers' lists; the third
bounded writer; the no-third-rotator scan; the client's notice and icon);
`client/GropiusChat` (the composer notice, the picker icon, the decoded field);
`client/tests` (the notice rule's own checks); `docs/` (a how-to and a reference
page, four pages updated); `README.md`; the changelog.

**Out.** The per-model exception and everything that hangs off it — the
`/model` listing's omission, the refusal of a debug arm on an excepted model,
the panel's model-card icon — all of which belong to the sibling
itd-2609091715089488. Per-model debug logging (itd-2609062346072707). Any route
that reads the transcript back over the network. Per-account transcripts under
the shared root. A days horizon, a summary fold, and any migration of an
existing file (pre-1.0). Telling a person who uses a client Gropius did not
write (cond-2609201007369925).

## Approach

### 1. The store

A new package, `internal/transcript`. It is the only place in the tree that
reads a message's fields for this purpose, and it is named on both readers'
lists (§5).

**Where.** `config.TranscriptDir(root)` returns a `transcript` directory under
`accountDir(root)`, beside `StatsDir` and the self-test's results and under the
same rule: this account's own record, never the group-writable shared root. The
directory is created 0700; the file is `transcript.jsonl`, mode 0600, created
and reopened through `applog.OpenRotator` — which already carries the four rules
a new writer would otherwise re-implement (an `os.Root` on the directory,
`O_NOFOLLOW`, `O_NONBLOCK`, and an fstat on the opened handle that refuses a
file carrying bits outside 0600). The self-test's results file is the pattern to
copy: `applog.RotateOptions` with `Dir`, `Name`, `MaxBytes`, `Keep` and a
`Rotated` callback that says in the log that the transcript reached its cap and
was started again.

**The cap, and how it maps onto the rotator.** `transcript_max_bytes` is the
whole store's ceiling. The rotator bounds one file and keeps N, so the store
opens it with `Keep: 5` — the log's own figure — and `MaxBytes` the cap divided
by that. `DefaultTranscriptMaxBytes` is 512 MB (cond-2609201007361219);
`MinTranscriptMaxBytes` is 16 MB, the floor below which five files are a window
of minutes rather than the archive the cap exists to be;
`MaxTranscriptMaxBytes` is 10 GB. The three constants sit beside
`DefaultStatsMaxBytes`/`MinStatsMaxBytes`/`MaxStatsMaxBytes` and are repaired
the same way: a figure outside the range in a hand-edited file is repaired to
the default and named in the repaired list, never a refusal to start.

**The record.** One JSON line per request, written once, after the answer has
finished or failed:

| Field | What it is |
| --- | --- |
| `v` | the record format's version, first on the line, as the statistics store's records carry it |
| `at` | when the request arrived, in whole UTC seconds |
| `model` | the repo id that **served** the request — never the name the client sent (adr-2609201008470380, decision 2) |
| `source` | `http` or `bridge`: the same two fixed classes `stats.Source` declares, and nothing else |
| `streamed` | whether the client asked for the answer as a stream |
| `class` | how it ended, on the statistics record's own fixed classes, so a refusal and a cut answer are legible |
| `prompt` | the whole message array as the client sent it |
| `merged` | present and true when Gropius merged the request's system messages for this model, so the record says the prompt was rewritten without keeping a second copy |
| `answer` | the whole of the generated text, assembled from what was relayed |
| `prompt_tokens`, `completion_tokens` | the model server's own counts when it gave them |

**What is not on the line:** no API key, no client address, no port, no
user-agent. adr-2609182357322050 condition 6 permits a paired client's *name*
where the log already names it; the transcript writes none, because the operator
can identify a client from the paired list and a name beside every conversation
is a second fact this store was not asked for. Adding one is a later decision.

**No truncation, and why none is needed.** "The whole prompt and the whole
answer" is a promise, so nothing on the line is elided. It is still bounded,
because both sides were already bounded before this store existed: the gateway
reads a request body under `maxRequestBody` and ends an answer at
`maxStreamLine`. The longest line the store can write is therefore the sum of
two ceilings the gateway already enforces, and the cap bounds the file
regardless.

**The seam, and why it is drawn there.** The gateway does not gain a reading.
It hands `internal/transcript` bytes it already holds — the buffered request
body, and each relayed event or the buffered non-streamed body — through a sink
on `relayOptions` beside `observing`; the transcript package is the only one
that names `messages`, `content`, `choices` or `delta` in order to compose the
record. `internal/gateway/gateway.go`'s existing grant ("reads whether an event
carries a choice, never what is inside one") is therefore unchanged, which is
what keeps this a **retainer** rather than a second reader inside the relay.
Writes go through a buffered queue drained on the store's own goroutine, so a
slow disk delays no client; a write that fails is logged once and the answer is
unaffected.

**Nothing is read back over the network.** No endpoint serves a record. The
panel is given figures only — the oldest record's date, the bytes, the file
count — on the model of `stats.StoreStatus`, carried on the loopback snapshot as
`transcript_store`. Alice reads the file with her own tools.

### 2. The switch, the cap, and the three surfaces

`config.Config` gains `Transcript bool` (`transcript`) and
`TranscriptMaxBytes int64` (`transcript_max_bytes`) — the spelling
`stats_max_bytes` already set. `Default()` leaves `Transcript` false and stores
the default cap, so switching it on in Settings is one tick rather than a tick
and a number. Nothing else sets it: not `log_level`, not `statistics`, not a
per-model field.

**The panel.** A `Transcript` fieldset of its own in Settings — never inside the
Request statistics fieldset, because the two make opposite promises and the
whole point of the separate word is that they cannot be read as one
(cond-2609201007361803). It carries `id="setTranscript"`, `id="setTranscriptMB"`,
`id="transcriptStoreLine"` and `id="transcriptDelete"`; `app.js` posts
`transcript` and `transcript_max_bytes` and reads the figures back from
`transcript_store`. The oldest-record line is rendered from that snapshot in the
words the statistics store already uses — "Records from … onwards" — and never
from a constant. No entry is added to `settingsPaneExemptions`.

**The shared-cache refusal.** `config.IsSharedRoot` is known before the pane is
rendered. Where it is true the snapshot carries `transcript_refused` with its
reason and the fieldset renders the switch unavailable with the reason in the
same fieldset: one process serves every local account, so Carol's conversations
would land in Alice's folder and Carol cannot consent to that. The control
cannot be rendered without the sentence, on the terms adr-2609181004167097
condition 5 sets and in the shape `TestTheBridgeSwitchCarriesTheEgressSentence`
holds — the test walks back from the control to its nearest `<fieldset>` and
requires the words there, so a sentence that drifts three screens away fails.
The panel is not the only gate: `config.json` is hand-editable, so a server
started under a shared root with `transcript: true` opens no store, records
nothing, and says so once in the log.

**A save is never refused over an untouched setting.** The cap is repaired
rather than refused (§1), and the switch is a boolean. The per-model half of
this criterion is the sibling's seam: `applySettings` today sets
`incoming.Models = nil` when the body names `models`, so a model the panel never
rendered loses its settings on the next save. adr-2609201008470380 decision 1
makes that a merge for every per-model field, and itd-2609091715089488 owns the
change. This spec depends on it and names the same save-path test; until it
lands, that half of criterion 9 is red.

### 3. The disclosure header

One header, named once here: the **family-prefixed recording header** —
`Recording` under the same `X-<family>-` prefix that `State` and `Queue-Time`
already carry. The prefix follows the family name (the 2026-09-20 ledger line on
naming), so this spec spells the suffix and not the prefix.

- **Unconditional.** It is written on every answer the API surface sends,
  whether or not the install has a key. The rule that withholds the wait headers
  from an unauthenticated network exists to keep other clients' *residency*
  facts off it; a statement about this server's own retention says nothing about
  anybody else's traffic, and the open keyless install is the one that most
  needs to disclose (adr-2609201008470380, decision 3).
- **Where it is written.** In one wrapper, outermost of `routes()`, applied in
  both `Handler()` and `TLSHandler()`. Outermost is the point: `withAuth` writes
  its own 401 and 403 before any route is reached, and a wrapper inside it would
  miss them. Every exit below it — the 400s, the 404, the 413, the 408, the
  500, the 502, the 504 and each 503 — is served through `writeError` under that
  wrapper and carries the header by construction, so a refusal path added later
  carries it without anyone remembering.
- **Its value.** `on` or `off`, always exactly one. The wrapper writes the
  server's own state before the handler runs; the completions handler overwrites
  it with this request's fact once the serving model is resolved and before the
  status is written, so an answer from a model the sibling has excepted says
  `off` while the rest say `on`. A request refused before it named a model keeps
  the server-wide value, which is the honest answer to a question about a
  conversation that never happened.
- **Unforgeable.** Its folded name joins `gropiusHeaders`, so
  `copyResponseHeaders` drops an upstream value before merging the rest — the
  mechanism that already keeps the two wait headers Gropius's own word. The
  folded lookup gains its entry in `internal/archtest/repo_id_fold_test.go`.
- **Not on the control plane.** The loopback panel is the operator's own
  surface, not a client's, and it renders the state in words. The header is an
  API-surface fact and the wrapper is mounted there only.

### 4. The models list

`handleListModels` writes `recording` on every entry, beside `chat` and before
the residency join, for the reason `chat` is always present: an absent key reads
as an older server that cannot say either way. It is in the base entry, so a
keyless LAN client sees it and `assertNoResidency` still passes unchanged
(adr-2609201008470380, decision 4). The header remains the guarantee, because a
curl and a pinned-model agent never fetch the listing
(cond-2609201007367978).

Two pinned field sets widen with it: `TestModelsListReferenceDocumentsEveryFieldServed`,
which holds `docs/models-list.md`'s `## Fields` table to exactly the fields
served in both directions, and `TestChatClientReadsTheCategoryTheGatewayPublishes`,
which pins each field on both sides — the client's `let recording: Bool?` against
`entry["recording"] =` in the gateway. The client cannot draw the icon from a
field it throws away: `connect()` today reduces the decoded entries to
`[String]`, so it gains a published per-model table keyed by model id, which the
picker reads.

### 5. The readers' list, and the rotating writers

`internal/transcript/` is added **by name** to both lists in
`internal/archtest/prompt_content_test.go` — `promptContentReaders` and
`generatedContentReaders` — each with its reason: the retainer
adr-2609201008470380 decision 9 admits, which reads both sides of a conversation
because composing the record *is* the retaining. The spec names it rather than
leaving the scan to discover it, precisely because the scan's stated blind spot
is a reader that goes through a type declared in a listed file.

No third rotating writer appears. The transcript is a caller of
`applog.OpenRotator`, as the self-test's results file is. Two tests hold that:
`TestEveryBoundedWriterRefusesAPlantedFile` gains a third entry, "the
transcript", so the planted-file discipline is proved for it; and a new
`TestTheTreeHoldsNoThirdSizeRotatingWriter` scans shipping Go for the rotation
shape — a size check against a cap plus a rename of numbered files — and admits
exactly `internal/applog/rotate.go` (the primitive) and `internal/stats/store.go`
(the specialisation the store's durable format needs), each with its reason, in
the shape the readers' lists use. That test is what keeps
`iss-2609091714393599` a two-copy defect rather than a three-copy one.

### 6. Delete transcript

`POST /api/transcript/delete` on the loopback control plane, in the shape of
`handleClearStats` and for its reasons: it takes no path from the request, acts
on the store's resolved directory and only on the file names the store itself
writes, works whether the switch is on or off, and leaves the switch exactly as
it found it. A failure logs the reason and answers 500 with a body that does not
name the directory. `App.DeleteTranscript` is the Go-side function both the
endpoint and any terminal route call, so the capability is not panel-only.

On the panel, **Delete transcript** carries the same confirmation shape **Clear
records** carries *at the time*: today a single dialogue whose text says the
removal cannot be undone and that the transcript switch is left as it is; the
in-page two-step once itd-2609100519003748 has removed modal confirmation. A
test asserts the two buttons go through the same confirmation mechanism, so the
panel's redesign moves both together rather than leaving one behind.

Switching the transcript off stops new records and leaves the existing ones
where they are, and the pane says so in its own words beside the switch — the
sentence the statistics fieldset already teaches, in the transcript's own
vocabulary.

### 7. Bridged turns, and the channel notice

A bridged message already runs the same admission, merge and observation as an
HTTP request through `gateway.Ask`, so the retainer sits on that path too and
writes the turn with `source: bridge` under the model that served it. There is
no response object on that path and so no header; the notice goes into the
message text, where there is no JSON mode and no tool-call shape to corrupt
(adr-2609201008470380, decision 8).

`conversation` gains a `noticed` flag under the lock that already guards its
model and turns. When an answer is about to be recorded for a channel whose flag
is false, the bridge's editor opens the first message it writes for that channel
with the notice and sets the flag. The flag is set only when a notice was
actually written, so a channel whose first answer went out with the transcript
off is told when the next answer is recorded. `/reset` clears it with the rest
of the channel's state, and so does stopping the bridge: it is per-run state, as
every other per-channel fact the bridge keeps is, and the cost of that is one
repeated notice after a restart rather than a missing one.

The notice, in the product's own words: "This server keeps a transcript. What
you send here and what the model answers are written down on the Mac that runs
it, where its operator can read them."

### 8. Gropius's own client

Two things, and they are different promises.

**The notice, before Bob types.** The composer's `VStack` gains a caption
`Label` in the shape the existing `cannotSend` line already uses, above the
input row rather than between the text field and its modifier chain — the
proportions test matches that chain and an insertion inside it would break it.
The rule for *which* sentence is shown, and when, lives in a new
Foundation-free `client/GropiusChat/TranscriptNotice.swift` so that it can be
checked without the app: notice when the selected answerer is a server model the
server reports as recorded, no notice when it is not, and **never** for
`Answerer.builtIn`, whose whole promise is that nothing leaves the device
(cond-2609201007369668).

**The icon, where Bob picks a model.** The picker's per-model `Label` gains a
glyph drawn from the models-list field. It is a third distinct symbol, never
`lock` or `lock.shield.fill`: `TestTheLockForAPairingIsNotTheLockForAnAPIKey` is
the standing precedent that one glyph means one claim, and this one gains its
twin test.

Held by `internal/archtest/chat_client_transcript_test.go`, in the shape
`chat_client_load_state_test.go` uses (the client's sources read as one string,
each client-side pattern paired with the gateway-side one it must match), plus a
case in the client's own checks under `client/tests/`. That target is three
`swiftc` mains run from Go, not an XCTest target, so the case is a fourth pair —
`client/tests/transcript-notice.sh` with its `TranscriptNoticeTests.swift`,
compiled against the Foundation-free source above and run by the Go test with
the loud `xcrun` skip the others use. The criterion's phrase "the client's Swift
unit target" means this, because nothing else exists to mean.

### 9. What is never written, and how the store says so

Nothing Gropius itself asked a model is in the transcript. The pool's readiness
probe, the self-test and the context probe each compose their own conversation
and post it straight to a model server on loopback, never through the gateway
(cond-2609201007367156, on the shipped cond-2609061822382803); the client's
built-in on-device model never reaches the server at all
(cond-2609201007369668); the control plane's own traffic is not a conversation.

The store says so by construction rather than by filter: every record carries
`source`, one of two fixed classes — `http` and `bridge` — and there is no class
for a request Gropius made of itself, because nothing Gropius makes of itself
goes through the retainer's seam. The reference page states this as its own
section, in the shape `docs/request-statistics.md` states what is never
recorded.

### 10. The docs

Two new pages, because the house rule is one Diátaxis type per page and the
criterion asks for both a reference and a how-to:

- `docs/transcript.md` — the how-to: turning it on, what appears on the panel,
  the cap, deleting the records, and why the switch is unavailable on a shared
  install.
- `docs/transcript-store-reference.md` — the reference, in the shape
  `docs/statistics-store-reference.md` takes: where the files are, their mode,
  the record's fields one table row each, the cap and what rotation drops, what
  removes the files, and what is never written.

Four pages change. `docs/response-headers.md` gains a row for the recording
header and its two values, and its blanket sentence — that the headers appear
only on an install with a key — is narrowed to the two wait headers it was
written about, since the new one is unconditional. `docs/models-list.md` gains
the `recording` row in `## Fields`. `docs/discord-bridge.md` states the
bridged-turn rule and the channel notice under `## What is recorded`.
`docs/statistics-store-reference.md` gains one line distinguishing the two
stores, so a reader who arrives at the wrong one is sent to the right one. The
README names the transcript as off by default and not the statistics store, and
`internal/ui/static/index.html` links the how-to where it links the bridge's.
British English, present tense, the word "transcript" throughout.

## How each acceptance criterion is held

The intent's criteria carry no minted ids of their own — only its seven scope
conditions do — so each row is keyed by the criterion's ordinal in
`## Acceptance Criteria` and its own words. Every test named is written and
watched to fail before the change and to pass after; a hand check is recorded in
the shipping decision line with what was done and what was seen.

| # | Criterion | Held by |
| --- | --- | --- |
| 1 | Off by default, and nothing turns it on | A test over `config.Default()` asserting the switch false and the cap at its default; and a test that drives a full request under a default config and fails if any file appears in `config.TranscriptDir` |
| 2 | The header on every answer, including composed refusals | `TestEveryExitPathCarriesTheRecordingHeader`, a table naming each exit the API surface has — streamed, non-streamed, 400, 401, 403, 404, 408, 413, 500, 502, 503 (each of the three), 504 — so a path added later without the header fails |
| 3 | Written on a keyless install too | One test asserting all three facts together on one keyless LAN response: the recording header present, `X-Gropius-State` and `X-Gropius-Queue-Time` absent — so the deliberate break of the `setWaitHeaders` symmetry is the thing under test |
| 4 | An upstream can neither forge nor shadow it | A test in the shape of `TestAModelServerCannotAddASecondValueToTheWaitHeaders`; the folded name in `gropiusHeaders`; the entry in `internal/archtest/repo_id_fold_test.go` |
| 5 | `recording` on every models-list entry, in the unentitled half | A test asserting the field present on a listing that `assertNoResidency` also passes, with one model true and one false; `TestModelsListReferenceDocumentsEveryFieldServed` and `TestChatClientReadsTheCategoryTheGatewayPublishes` both widened |
| 6 | Its own file, 0600, on the existing rotator, never the statistics store | A test asserting the file's mode and that its path is outside `config.StatsDir`; `TestEveryBoundedWriterRefusesAPlantedFile` extended; `TestTheTreeHoldsNoThirdSizeRotatingWriter`; the retainer named in both lists of `prompt_content_test.go`; and `TestNothingFromTheRequestReachesTheRecordOrTheLog`, `TestNothingFromTheRequestReachesTheStore`, `TestNothingFromTheRequestReachesTheSummary` passing **unchanged** — an edit to any of the three fails this criterion rather than satisfying it |
| 7 | The cap in Settings, the oldest record's date beside it | A round-trip test in `internal/ui` over the cap (markup ids and posted keys, in the shape of `TestSettingsOffersTheRetentionFiguresAndClear`); a test that the oldest-record line is rendered from `transcript_store` rather than a constant |
| 8 | Switching off keeps; **Delete transcript** removes | Two tests in `internal/ui` over the pane's words and the posted route; a Go test on the endpoint (200 on loopback, 403 from the LAN, the switch unchanged, only the store's own file names touched); a markup test that the two buttons share one confirmation mechanism and that the text says the removal cannot be undone and the switch is left as it is |
| 9 | Three surfaces; a save never refused over an untouched setting | `TestEverySettingHasAPanelControlOrAnExemption` and its reverse half, with no new exemption; a repair test over an out-of-range cap; and a save-path test that loads the panel, hand-edits `config.json` underneath it, saves an unrelated setting and asserts the hand-edited per-model exception survives — **red until the sibling's merge lands** (§2) |
| 10 | Refused under the shared-cache install, with the reason on the panel | One test asserting both halves: the fieldset cannot render the control without its reason (the `bridge_egress_test.go` fieldset walk), and a server started under a shared root with the switch on opens no store, writes nothing and says so once in the log |
| 11 | Bridged turns recorded, the channel told once | A test on the bridge's editor: the notice in the text of the first edited message for a channel, absent from the second, and present for a channel whose first answer was sent while the transcript was off |
| 12 | Gropius's own client shows the person before they type | `internal/archtest/chat_client_transcript_test.go` (the notice in the composer, the picker's third glyph, no notice for `Answerer.builtIn`) plus `client/tests/transcript-notice.sh`; the drawn result on Mac and iPad is a hand check recorded in the shipping line |
| 13 | Two superseding ADRs, linked both ways, nothing else changed | Already on the record and **not this spec's to write**: adr-2609201008470380 supersedes adr-2609061610102325, adr-2609201008476813 supersedes adr-2609061503319212. Held by the cross-link check `abcd lint` runs over the ADR store, plus a hand check that each superseded file's diff touches its status fields and nothing else |
| 14 | The docs and the README say it | `TestTheResponseHeaderReferenceDescribesEveryHeaderServed` extended to the new header and its values; `TestModelsListReferenceDocumentsEveryFieldServed` for the field; a docs test over the two new pages in the shape `TestTheStatisticsPageNamesEveryFieldThatIsRecorded` uses, driving off the record's own field list; `TestTheStatisticsPagesAreOneTypeEach`'s rule applied to the transcript pair; `abcd docs lint` |

**Three criteria need saying plainly rather than building around.**

- Criterion 12 names "the client's Swift unit target". There is no XCTest
  target; the client's checks are `swiftc` mains under `client/tests/` run from
  Go. The criterion is buildable in that shape and in no other, and the price is
  that the rule under test must live in a source file that imports nothing.
- Criterion 14 asks for a reference page *and* a how-to. One page cannot be
  both under this repository's own Diátaxis rule, so it is two pages (§10).
- Criterion 9's second half depends on a change this spec does not own (§2).
  It is red until itd-2609091715089488 lands the per-model merge, and saying so
  now is cheaper than discovering it at the gate.

## Security

The store is the highest-value file this product writes: with the switch on it
holds, in plain text, everything anybody asked this server. The review before
landing is not optional — `internal/gateway`, `internal/config` and a new writer
under the account directory are all declared trust boundaries, and
adr-2609181004167097 already obliges an adversarial pass on a change of this
kind. **The security-reviewer agent reviews the diff before it is presented.**

- **Prompt content on disk.** The file is 0600 in a 0700 directory, opened
  through `applog.OpenIn`, which refuses a symlink, a pipe, a non-regular file
  and a file carrying permission bits outside 0600 — the names are predictable
  and the directory is reachable by anything running as this account. Nothing
  weakens for the transcript what the log and the statistics store already
  hold. What this does not defend against, stated rather than designed around:
  anything running as Alice can read Alice's transcript, which is why the switch
  is off by default and why the delete is a first-class act.
- **The shared-cache install.** Two gates, because one is a pane and panes are
  not the only way in: the control is unavailable with its reason where
  `config.IsSharedRoot` is true, and a server started under a shared root with a
  hand-edited `transcript: true` opens no store at all. A single gate on the
  panel would be a promise a text editor can break.
- **The header's forging path.** The one way a client could be told the wrong
  thing is a model server emitting the header itself. The folded name in
  `gropiusHeaders` makes `copyResponseHeaders` drop it before merging, and the
  merge-rather-than-replace behaviour is why dropping is the right verb: a
  second value beside ours would be read first by some clients. Tested directly.
- **Delete on loopback.** The route is destructive and unauthenticated in the
  same sense `POST /api/stats/clear` is: loopback only, behind the control
  plane's own Host and Origin guards, so a page in Alice's browser and a
  DNS-rebound one cannot reach it. It takes no path from the request and removes
  only names the store itself writes, so it can never be steered at another
  file. Its 500 does not name the directory.
- **Nothing is served back.** No endpoint returns a record, on any listener. The
  panel receives figures only. A read-back route would put every prompt on the
  loopback control plane, and the day a browser page defeats one of its guards
  the exposure would be the whole store rather than a setting.
- **The notice is Gropius's own word.** No client can suppress the header, ask
  for it to be omitted, or make the bridge's channel notice not be written; the
  value is composed from the server's own state and the resolved model, never
  from anything the request carried.

## Verification gates

- `make test` green, `gofmt -l .` empty, `go vet ./...` clean.
- Every new behaviour has a test watched to fail before the change and pass
  after; each of the three byte-scanning tests passes **unchanged**, and an edit
  to any of them fails criterion 6 instead of satisfying it.
- `abcd docs lint` and `abcd lint` clean, the second covering the ADR
  cross-links criterion 13 rests on.
- The client's `swiftc` checks run where `xcrun` is on PATH and skip loudly
  where it is not.
- Hand checks, recorded in the shipping decision line with what was seen: the
  refusal and its sentence on a genuinely shared install; a real Discord channel
  told once and not twice; the composer notice and the picker glyph on a Mac and
  on an iPad; a switched-off transcript leaving its records in place and
  **Delete transcript** removing them.
- The security review above, before the change is presented.

## What is deliberately not built

- No read, search or export route for the transcript, and no view of its content
  in the panel.
- No per-account transcript under the shared root: the singleton does not run as
  the requesting account, the shared key attributes nothing, and a loopback
  address attributes less.
- No days horizon, no summary fold — a summary of a transcript is a transcript —
  and no migration: pre-1.0, a hand-edited figure out of range is repaired.
- No per-model exception, no `/model` omission, no panel model-card icon: the
  sibling's, and this spec ships with the field they need already published.
- No notice to a person using a client Gropius did not write. The server tells
  every program on every path; whether a program tells anybody is its own
  choice, and the heading is narrowed to say so.

## Falsifiers

- Any of the three byte-scanning tests needing an edit: the store was not
  separate after all.
- A relayed answer in which a client can read a recording value the gateway did
  not write.
- A reachable state in the chat client with a recording server selected and no
  notice drawn, or a notice drawn over the on-device model.
- 512 MB turning out to be a window of minutes under real agent traffic: then
  the number is wrong, not the design (cond-2609201007361219).
- Testers using "recording" and "transcript" interchangeably anyway: then the
  pane needs more than a word.
