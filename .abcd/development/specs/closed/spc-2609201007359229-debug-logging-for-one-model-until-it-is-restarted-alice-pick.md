---
id: spc-2609201007359229
slug: debug-logging-for-one-model-until-it-is-restarted-alice-pick
intent: itd-2609062346072707
origin: researcher-authored
production_mode: hand-written
---

# Debug logging for one model, for one run

## Summary

This spec delivers itd-2609062346072707 as shape (i) of its Question 2: the
model server's own level is raised for one run of one model, and Dessau
reads no prompt content anywhere. Alice arms a model from its card in the
control panel; the arm is a transient per-model mark held in memory by the
pool, consumed by the next launch of that model and cleared in the same step,
so that run is at DEBUG and every run after it is at INFO again. Arming sends
nothing to the process that is running now, and the panel says so and offers
the Unload that starts the logged run. The launch renames the previous run's
log rather than truncating it, so the restart that ends the mode leaves Alice
the evidence; the armed run's output passes through a bounded writer, so a
looping client or a context probe cannot fill the disk. A model carrying
itd-2609091715089488's transcript exception refuses the arm with its reason.
`TestTheModelServerIsAlwaysLaunchedAtInfo` is amended rather than deleted, a
mirror readers' list holds the mark away from the statistics switch and from
`log_level`, and `docs/logging.md` stops saying that no setting in Dessau
asks for the model servers' debug level.

The exception this makes is written down in adr-2609201008477513, which
narrows adr-2609061503319212 (one model, one process lifetime,
operator-invoked, never on by default, never reachable from the statistics
switch or `log_level`, named on the panel while it is on, bounded in size,
and the client is not told). adr-2609061610102325 is untouched: nothing here
reads a message.

Impact: additive. Severity: major. Nothing stored changes shape, so pre-1.0
migration does not arise.

## Scope

In: `internal/runtime` (`Spec.DebugLog`, the level argument in `launchArgs`,
the bounded writer, the rename at launch, the pool's mark and its consumption,
`Resident.DebugLog`); `internal/gateway/control.go` (one route, the
per-model state on the snapshot, the refusal seam); `internal/app` (wiring
the refusal seam to the per-model transcript field); `internal/ui/static`
(the card's action and pills, the plain-words paragraph, the posture line);
`internal/archtest` (the amended launcher test, the new `debugMarkReaders`
list and its liveness rule, the static-surface test, the extended logging-docs
test); `docs/logging.md`; the changelog.

Out: everything under "Out of scope" below. `config.json` gains no field, so
`internal/config` is untouched.

## Approach

### The mark is an action, not a setting

The mark is transient per-model state held in memory and spent at the next
launch. It is **not** a json-tagged field on `config.ModelSettings`, and that
is a design choice with three consequences, each of which is the reason:

1. **The settings-surface obligation's own scope excludes it.**
   `TestEverySettingHasAPanelControlOrAnExemption` walks the json-tagged
   settings of `config.Config`; the obligation is that a setting Go holds has a
   panel control or a written exemption. An action is not a setting, so no
   entry is added to `settingsPaneExemptions` — and adding one would be wrong
   twice over, because that table exists for settings the panel cannot reach
   and this one has a control on every card.
2. **`TestConfigHoldsExactlyOnePerModelMap` stays untouched**, because no map
   and no field are added.
3. **The mode cannot be left on for a month.** A stored flag survives a reboot
   and is forgotten; a mark in memory dies with the process that holds it, and
   the run it buys dies with the model server. That is the property option (ii)
   of Question 3 was declined for.

### Where the arm lives, and how a launch consumes it

The pool owns it, because the pool is what composes the `Spec`:

- `Pool.debugArmed map[string]bool`, keyed by `config.FoldRepoID(repoID)` and
  guarded by the pool's existing `p.mu` — the same lock the launch path holds,
  so there is no second lock and no new order to invert.
- `Pool.ArmDebugLog(repoID) error` and `Pool.DisarmDebugLog(repoID) error`
  set and clear it. Arming touches no process: no signal, no write to the
  child's stdin, no restart. A model that is resident stays resident and keeps
  serving at the level it was launched at. Disarm is the inverse of arm and
  exists so a mark Alice set by mistake can be taken back before it is spent;
  it is one map delete beyond the criteria and is called out here rather than
  smuggled in.
- `Spec` gains one field, `DebugLog bool`. It is derived from the mark and
  from nothing else.
- In the pool's start path the mark is **read and cleared under one
  acquisition of `p.mu`**, but the clear is committed only after
  `Launcher.Launch` returns without error. A launch that fails to spawn — a
  provisioning error, a port taken, a missing model directory — leaves the
  mark armed, so Alice is not told she is armed when the run she wanted never
  started. Two concurrent launches of the same model cannot both consume it:
  the pool keys entries by the folded id and holds `p.mu` across the
  read-launch-clear.

`launchArgs` keeps naming `--log-level` exactly once. Its value comes from the
spec and nowhere else:

```go
const debugLogLevel = "DEBUG" // the one place this level is named

level := "INFO"
if spec.DebugLog {
    level = debugLogLevel
}
```

Returning to INFO needs no state: the next launch finds no entry in the map,
`Spec.DebugLog` is false, and the constant default applies.

### The previous run's file is renamed, not truncated

Before the `O_CREATE|O_TRUNC` open, and on **every** launch rather than only an
armed one, the launcher renames the existing per-model log to
`<safe-repo-id>.previous.log` — `logFileName`'s name with `.previous` before
the extension. One generation only: the rename replaces any previous file of
that name, so consecutive launches do not accumulate.

The rename goes through `os.OpenRoot(l.LogDir)` and `(*os.Root).Rename`, which
is the discipline the rest of the tree already applies to these files
(`internal/applog`'s `OpenIn`): the root refuses a name that leaves the
directory, and both names are in the one directory, so the rename cannot cross
a filesystem and cannot cross the account boundary the shared-cache install
creates — the logs folder resolves through the account directory, which is what
makes the open reachable at all. A rename that fails because there is no
previous file is normal and is not an error. A rename that fails for any other
reason refuses the launch with the reason, in the same class as today's refusal
when the log cannot be opened: silently truncating the run the operator armed
is the failure this criterion exists to prevent.

### The size bound

A bounded writer on the child's output, interposed **only when the launch is
armed**. An unarmed launch keeps today's code path byte for byte —
`cmd.Stdout = cmd.Stderr = logFile` — so no pipe, no goroutine and no
behavioural change reaches anyone who does not arm.

When armed, both are set to the same `*boundedWriter` value wrapping the log
file. Because it is the same non-`*os.File` value on both, `os/exec` creates
one pipe and one copying goroutine, and stdout and stderr stay interleaved as
they are today.

- **The figure: 64 MB**, `runtime.DebugLogMaxBytes = 64 << 20`. Large enough to
  hold thousands of ordinary request-and-answer pairs, and many times the
  largest single body the server ever sees — the context probe's generated
  filler, which is the reason the bound holds per write rather than per file.
  With the kept previous file, one model's worst case on disk is two files.
- **At the bound** the writer writes the part of the current write that fits,
  then one final line saying the log stopped at its bound and that the model is
  still serving, then discards everything after it while continuing to report
  every write as accepted (`len(p), nil`). It never returns an error and never
  blocks: a writer that stopped draining would fill the pipe and stall the
  model server, which would turn a diagnostic into an outage.
- It is safe for concurrent use and is exercised under `-race`.

### The panel

**The action.** Each ready model's card gains a `confirmBtn` — the two-click
pattern Delete already uses, because this is a deliberate act — labelled
"Debug logging" when the model is unmarked and "Stop debug logging" when it is
armed. It posts to the control plane and nothing else.

**The plain words, in one place.** One paragraph lives in `index.html` in the
Models pane as `<p class="hint" id="debugLogBlurb">`, and the card's button
takes its `title` from that element's text at render time, so the two surfaces
cannot drift. It says, in British English and plain words:

> Debug logging is a per-model diagnostic and is off unless you arm it. From
> that model's next start until the start after it, the model server's own log
> holds every request sent to it and every answer it produced — the prompts and
> the completions, whoever sent them — and the requests Dessau's own probes
> and self-test send as well. Arming changes nothing about the run going on
> now: unload the model to start the run that is logged. The run after that is
> back to the ordinary level. The file is owner-only, stops at 64 MB, and the
> previous run's file is kept beside it; nothing is sent anywhere, and clients
> are not told. A model that keeps no transcript refuses this.

**The indication.** Armed and running are different runs and are drawn
differently: a `debug armed` pill on a model with a mark waiting, and a
`logging at debug` pill on a model whose running process was launched with it.
A model that is neither draws nothing. The running fact is read from
`Resident.DebugLog`, recorded on the entry at launch from the `Spec`, so it is
a fact about the process rather than about the mark; the armed fact is read
from `State.DebugArmed`. Both arrive on the snapshot the panel already polls,
so the pills move from armed to running without a reload.

**The posture line.** `postureLines` gains a `debug_log` item, present only
when at least one model is armed or running at debug — the conditional shape
the `private` line already uses. It names those models, says what the level
writes, says it ends at that model's next start and that another client's
request can cause that start, names the 64 MB bound and the kept previous file,
and says the clients are not told.

### The control plane

One route, beside the other model actions and under the same loopback guard:

- `POST /api/models/debug-log` with `{"model": "...", "armed": true|false}`,
  decoded through a sibling of `decodeModelRequest` under the same
  `config.MaxConfigBytes` cap.
- 200 `{"status":"armed"|"disarmed","model":...}`; 400 for a malformed body or
  a missing model field; 404 for a model the registry does not hold; 409 with
  the reason for a refusal.
- `State` gains `DebugArmed []string` (`"debug_armed"`), read from the pool the
  way `Pinned` is, and `runtime.Resident` gains `DebugLog bool`
  (`"debug_log"`).

**The refusal.** `Control.TranscriptExcepted func(repoID string) bool`, a
function field the app sets and a nil value reads as "never excepted". The
handler refuses an arm for an excepted model with 409 and the reason: the
exception means no prompts on disk, not "not in this one file"
(itd-2609091715089488's 2026-09-20 answer, `.abcd/work/DECISIONS.md`).

*Sequencing, stated because it is a dependency and not a detail:* the per-model
`recording` field that predicate reads is itd-2609091715089488's to add, and it
is not in the tree today. The seam, its refusal, its message and its tests are
buildable now and are built here; the wiring in `internal/app` that makes the
predicate true for a real model lands with that intent. If this change ships
first, the refusal is live as soon as the sibling supplies the field and the
end-to-end hand check belongs to that intent's shipping line. Nothing else in
this spec depends on it.

### The mirror readers' list

A new `internal/archtest/debug_mark_test.go`, built in the shape of
`statistics_switch_test.go`:

- `debugMarkReaders`, a map from file to the reason it may name the mark:
  `internal/runtime/pool.go` (holds the mark and consumes it at launch),
  `internal/runtime/launcher.go` (the one level argument),
  `internal/gateway/control.go` (the route that arms and disarms, and the
  snapshot), `internal/ui/static/app.js` (the action and the pills),
  `internal/ui/static/index.html` (the paragraph and the markup).
- `debugMarkNames`, the ways it gets named: `DebugLog`, `debugArmed`,
  `ArmDebugLog`, `"debug_log"`, `"debug_armed"`.
- `TestTheDebugMarkIsNamedOnlyByItsReaders` fails a build that adds a reader,
  with the message saying what to do: a genuine new reader needs a decision
  record and an entry here.
- `TestTheDebugMarkIsDerivedFromNothingElse` asserts that the two runtime files
  name neither `.Statistics` nor `LogLevel` nor `"log_level"`, and that the
  arming handler's own body in `control.go` names neither.
- `TestDebugMarkReadersAllExist` is the liveness rule, the mirror of
  `TestStatisticsSwitchReadersAllExist`: a renamed reader cannot leave its
  exemption behind for the next file to inherit.

### The amended launcher test

`TestTheModelServerIsAlwaysLaunchedAtInfo` is renamed to
`TestTheModelServerLevelComesOnlyFromThePerModelDebugMark` and keeps every
assertion it was protecting, with one narrowed: `--log-level` is still named
exactly once; the default is still INFO; `"DEBUG"` appears exactly once and
only as `debugLogLevel`'s value; the only use of `debugLogLevel` is inside the
`if spec.DebugLog` block; and the file names neither the statistics switch nor
`log_level`. Its comment block carries why the exception exists and cites
adr-2609201008477513 as the ADR that narrows adr-2609061503319212.

### The documentation

`docs/logging.md`, the section "This is not the model servers' log level", is
rewritten in present tense: the model servers run at INFO unless the operator
arms debug logging for one model from its card; what that writes; that it
begins at that model's next start and ends at the start after; the 64 MB
bound; and that the previous run's file is kept beside the current one rather
than the current one being emptied. The sentence "so no setting in Dessau asks
for it" goes.

*A trap for whoever writes it:* `TestTheLoggingPageNamesTheLevelsTheCodeAccepts`
fails the page if it contains the literal `` `debug` ``, because that is not a
`log_level` value. The new prose says "their debug level" and "DEBUG"
unbackticked.

## How each criterion is held

The intent's Acceptance Criteria carry no minted ids; the rows below are in the
intent's own order, each keyed by the criterion's leading phrase. Every test
named is watched red before the change and green after.

| # | Criterion | Held by |
| --- | --- | --- |
| 1 | Arming alone changes no running process | `TestArmingTouchesNoRunningProcess` (`internal/runtime`): arms against a running fake, asserts no restart, nothing written to the child, no new argument and no growth of its log |
| 2 | The next launch runs at DEBUG and the one after at INFO | `TestAnArmedLaunchIsAtDebugAndTheNextIsAtInfo` over two consecutive launches, asserting both argument vectors; `TestALaunchThatFailsToSpawnLeavesTheMarkArmed` for the third case (`internal/runtime`) |
| 3 | The panel says in plain words what this writes down | `TestThePanelSaysWhatDebugLoggingWrites` (`internal/archtest`) over `internal/ui/static/`, in the shape `logging_docs_test.go` uses: the paragraph's phrases in `index.html`, and `app.js` sourcing the button's title from that element |
| 4 | A model at debug is visibly at debug | `TestTheSnapshotCarriesTheDebugState` (`internal/gateway`) for `debug_armed` and the resident entry's `debug_log`, and `TestTheCardDrawsTheDebugPills` (`internal/archtest`) for the two pills and for nothing on an unmarked model. *Hand check:* arm a model in the web panel, confirm the pill appears without a reload, unload it, confirm the pill moves from armed to running |
| 5 | The mark never comes from the statistics switch, in either direction | `TestTheDebugMarkIsNamedOnlyByItsReaders`, `TestTheDebugMarkIsDerivedFromNothingElse` and `TestDebugMarkReadersAllExist` (`internal/archtest`), plus `TestTheDebugArmWritesNoSettings` (`internal/gateway`): the configuration after an arm equals the configuration before, in the shape of the untouched-settings test |
| 6 | `TestTheModelServerIsAlwaysLaunchedAtInfo` is amended, not deleted | the amended test itself, renamed `TestTheModelServerLevelComesOnlyFromThePerModelDebugMark`, with the comment block citing adr-2609201008477513 |
| 7 | The file is bounded | `TestTheDebugLogStopsAtItsBound` (`internal/runtime`), in the shape of `rotating_writer_test.go`: a looping write past the bound, a single write larger than the bound, the final line, writes after the bound reported as accepted and discarded, and the whole run under `-race`. The figure is stated in `docs/logging.md` and held there by the extended docs test |
| 8 | The ending restart leaves Alice the file | `TestALaunchKeepsThePreviousRunsLog` (`internal/runtime`): the rename happens before the truncating open and the renamed file's content survives; `TestTwoConsecutiveLaunchesKeepOnlyOnePreviousLog` for the accumulation case |
| 9 | The documentation sentence is corrected | `internal/archtest/logging_docs_test.go` extended: the page must not contain "no setting in Dessau asks for it", and must name the per-model action, what it writes, "64 MB", the kept previous file and that the mode begins at the model's next start |
| 10 | A no-transcript model refuses the arm | `TestAnExceptedModelRefusesTheDebugArm` (`internal/gateway`) for the 409 and its reason, driving `Control.TranscriptExcepted`; the static-surface test of row 3 for the sentence at the control. *Dependency:* the predicate's real source is itd-2609091715089488's per-model field — see the sequencing note in the Approach |
| 11 | The exception is written down before it ships | adr-2609201008477513 (authored in its own lane), held by the ADR link-integrity check that adr-2609061503319212 links forward to it. *Hand check:* the amended launcher test's comment and this spec both cite that ADR's id |

## Security

This change writes third parties' prompt content to the operator's disk. The
diff touches `internal/runtime` (subprocess management), `internal/gateway`
(network input) and the panel, all declared trust boundaries, so **the
security-reviewer agent reviews the diff before it lands** — that is a gate,
not a courtesy.

- **Prompt content on disk, unredacted.** At DEBUG the pinned `mlx-lm==0.31.3`
  writes the whole request body, every generation step's text and the whole
  non-streaming response object, with nothing sampled, truncated or redacted
  (`.abcd/development/research/notes/2026-09-20-mlx-lm-0.31.3-debug-logging.md`).
  An API key a client puts in a message is written verbatim. Dessau cannot
  redact what it does not read, and shape (i) is the choice not to read it.
  The paragraph at the control and the posture line say this in plain words;
  Bob is not told, which is adr-2609201008477513's stated cost.
- **The shared-cache install.** Where the file lives and what mode it has are
  unchanged — the account's own logs folder, 0600, `O_NOFOLLOW`, `O_NONBLOCK`
  and a regular-file check on the handle. The added exposure under
  `make install-shared` is not another local account reading the file; it is
  that network clients' prompts and the models' answers land in the serving
  account's directory tree under the deliberate `3775` semantics the Makefile
  explains. That is the cost, stated. "Readable by nobody else" is the phrase
  that failed in the sibling and is not used.
- **A looping client.** Bytes in the file are chosen by whoever is sending
  requests, so the bound is the control: 64 MB per run, enforced per write, and
  the model keeps serving when it is reached.
- **The loopback control plane answers every local account without a
  password**, by design (`.abcd/work/DECISIONS.md`, 2026-09-07). So another
  account on a shared Mac can arm debug logging for a model. It cannot read the
  resulting file — that is 0600 in the serving account's folder — but it can
  cause the operator's disk to fill to the bound with other people's prompts,
  and the panel's pills and the posture line are what make that visible. This
  is stated rather than gated: gating it would be a new authentication model
  for the whole control plane, which this record does not own.
- **Already true today, and now said out loud:** an unparseable or non-object
  request body is logged by the model server at ERROR, so some network content
  already reaches the per-model log at INFO. The rewritten docs section does
  not claim otherwise.
- **The rename and the open.** Both go through an `os.Root` on the logs
  directory, so a link or a name escaping the directory is refused rather than
  followed, and a failed rename refuses the launch instead of truncating.

## Verification

- `make test` green, `gofmt -l .` empty, `go vet ./...` clean.
- Every test named in the criteria map watched to fail before the change and
  pass after; the bounded writer and the pool's mark run under `-race`.
- `abcd docs lint` clean over the rewritten `docs/logging.md`.
- `abcd intent ready itd-2609062346072707` passing.
- The hand checks in rows 4, 10 and 11, recorded in the shipping decision line.
- The security review above, before presentation.

## Out of scope

- **Shape (ii)** — the gateway writing the log itself: bounded, formatted,
  redactable and visible in the panel, at the cost of making the
  network-facing process a reader and writer of prompt content. Declined at the
  2026-09-20 interview; taking it later supersedes adr-2609061610102325 rather
  than amending this spec.
- **A panel view of what the log holds.** The file is read with a text editor.
- **Redaction, formatting or sampling** of what the model server writes. It is
  the server's output, and reading it to redact it is shape (ii).
- **A switch that survives restarts, or a time box.** Both declined; the
  process lifetime is the boundary.
- **Telling Bob.** No stateless server can; the cost is recorded in
  adr-2609201008477513.
- **Bounding the INFO-level per-model log.** It keeps today's behaviour, and
  now keeps one previous generation as well, so its worst case on disk doubles.
  That is stated rather than fixed: bounding it is a separate change with its
  own record.
- **Rotation** of the per-model log. This is a bound and one kept previous
  file, not a rotation with a generation count.
- **Migration.** Nothing stored changes shape; the mark is never written down.
