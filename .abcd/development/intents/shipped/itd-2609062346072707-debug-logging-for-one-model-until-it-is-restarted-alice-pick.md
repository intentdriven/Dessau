---
id: itd-2609062346072707
slug: debug-logging-for-one-model-until-it-is-restarted-alice-pick
spec_id: spc-2609201007359229
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091412177263]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Debug logging for one model, until it is restarted: Alice picks a model in the control panel and asks Dessau to run it with its own logging turned all the way up, in a panel that says in plain words what that writes down. From then until that model's server is restarted, its log holds every request sent to it and every answer it produced, prompts and completions included, so Alice can see what a client is actually sending when a model answers strangely. It is per model, it is never on for a model she did not choose, and it is a separate action from recording request statistics: turning statistics on never raises a log level, and this never turns statistics on. Bob's clients see no change, and nothing leaves the Mac either way.

> The title is the quoted capture and is left as captured — a record's id is not
> a description. Two of its sentences do not survive the review below and are
> corrected in "Why This Matters" rather than rewritten here: the log cannot
> start recording "from then" (the level is a launch flag, so it starts at the
> next launch), and "Bob's clients see no change" is a disclosure question, not
> a reassurance.

## Press Release

> _Seeded from a quoted-text intent capture. Expanded briefly once the three
> open questions were answered on 2026-09-20; the narrative below is the
> answers, not a substitute for them._

A model answers strangely and Alice wants to see the traffic. She opens the
control panel, finds the model, and arms debug logging for it. The panel says
in plain words what that means: from the model's next start until the one
after, its own log holds every request sent to it and every answer it
produced — prompts and completions, whoever sent them, exactly as the model
server writes them. Arming it changes nothing about the run that is going on
now; the panel says that too, and offers the Unload that starts the next run.
While a model is armed or running at debug, the panel shows it.

The run after that is back at INFO, because the level is a launch flag and the
mark is spent at launch. When the run ends, its log is kept rather than
emptied, so the restart that ends the mode leaves Alice the evidence. The file
is bounded, because a client on the network chooses the bytes in it.

This is one model, chosen by the operator, never on by default, and never
reachable from the statistics switch or from `log_level`. Bob receives exactly
what he received before and is told nothing about it — the honest cost, stated
in the ADR that narrows adr-2609061503319212 rather than designed around,
because a stateless server has nowhere to tell him. A model whose transcript
exception says nothing of it is ever written down refuses the arm and says
why. Nothing is sent anywhere either way.

## Why This Matters

Alice runs a model and it answers strangely. The one thing that would tell her
why is what the client actually sent — the merged system message, the
parameters the client chose, the answer as the model produced it — and today
nothing on this Mac holds that. The gateway is deliberately blind to prompt
content (adr-2609061610102325), the server's own log never writes a prompt or
an answer at either level (itd-2609091412177263), and the statistics store is
content-free by construction (adr-2609061503319212). Every one of those is a
promise worth keeping; together they mean the operator of a local inference
server cannot see her own traffic when it is the traffic that is wrong.

The model server underneath already has the capability. Run at its debug level
it writes each request body and each response to its own per-model log. Dessau
runs every model server at INFO instead, in one place, and an architecture test
holds it there, because that level is one switch away from turning a
content-free statistics toggle into a prompt recorder — the coupling a
comparable local-LLM tool shipped and has an open security issue about.

So the capability exists, the wiring is one flag, and the reason it is not
wired is a decision rather than an omission. This record is the case for
making a narrow, deliberate exception for one model at a time, and the honest
account of what that exception costs.

What the title claims, corrected:

- The log does not fill "from then". A running model server's level cannot be
  changed; it is a command-line flag. Turning this on arms the next launch, so
  Alice arms it, the model restarts, and the run after that is the one she can
  read. Finding 2.
- "Until that model's server is restarted" is a lifetime Alice does not
  control: the pool restarts a model when memory pressure evicts it, and any
  client on the network can cause that. Finding 3.
- "Bob's clients see no change" is true of what Bob receives and false of what
  it costs him: his prompts are written to Alice's disk and he is told nothing.
  That is the promise the sibling recording draft was held on. Finding 10,
  Question 1.

## Mechanism

We expect a launch flag to be the only level switch we need, because the
pinned model server sets its level once — `logging.basicConfig(level=...)` from
`--log-level`, at `main()` — and exposes no runtime control over it. Falsified
if a request, a signal or an environment variable can change a running model
server's level, in which case arm-then-restart is a self-inflicted constraint
and the mode could start where the title says it does.

We expect arming to be safe because the mark is consumed at launch, not read
per request: the launcher composes `launchArgs` once, takes the level from the
mark, and clears it in the same step, so the process that starts is at DEBUG
and the next one is at INFO with nothing to remember. Falsified if a launch can
consume the mark without starting — a failed spawn, a provisioning error, a
port already taken — and leave Alice thinking she is armed when she is not, or
armed twice; the acceptance criteria name the case.

We expect the separation from the statistics switch to hold because a readers'
list plus a liveness rule is the instrument that already holds it in one
direction, and this adds the mirror: the mark is its own field, derived from
neither `Statistics` nor `log_level`, and its only reader in
`internal/runtime` is the launcher's level argument. Falsified the moment the
mark's value can reach the launcher under another name — which the scan cannot
see and a review of the diff must, which is why the list is a file that has to
be edited rather than a comment.

We expect the rename at launch to keep the evidence because truncation is the
only thing that destroys it: every launch opens the per-model log
`O_CREATE|O_TRUNC`, so a launch that renames the previous file first leaves
exactly one run's record behind it. Falsified if the rename can lose the file
instead of moving it — a rename across the account boundary the shared-cache
install creates, or a second launch racing the first — and the acceptance
criterion is the rename, not the intention.

We expect a size bound to hold against a looping client because the bound is
ours and applies to bytes written, not to requests served: the server itself
has no `maxBytes`, no `FileHandler` and no truncation of either body, so a
client that repeats one request writes until the bound stops the writing. It
is the same instrument `dessau.log` already uses. Falsified if a single
request can exceed the bound before it is checked — one 200,000-character
context probe body is a single `logging.debug` call — so the bound has to hold
per write and not only per file.

## Scope Conditions

- **The pinned model server, and what it writes at DEBUG.** `mlx-lm==0.31.3` <!-- cond: cond-2609201007358534 -->
  (`internal/runtime/mlx-requirements.txt`, `mlxLMVersion` in
  `internal/runtime/provision.go`). At DEBUG that version writes the whole
  request body including every message, every generation step's text, and the
  whole non-streaming response object — verified against the pinned wheel in
  `.abcd/development/research/notes/2026-09-20-mlx-lm-0.31.3-debug-logging.md`,
  which is the precondition finding 1 named. Nothing in it is redacted,
  sampled or truncated, so an API key a client puts in a message is written
  verbatim. This scope condition is void at the next pin bump: what the level
  writes is the server's to change, and the note is the fixture to re-read.
- **Per model, one process lifetime.** The mark is armed against one model, <!-- cond: cond-2609201007350925 -->
  and the run it buys ends when that model server next stops, for whatever
  reason. The reasons are not Alice's to choose: the pool evicts under memory
  pressure and the idle reaper unloads where an idle timeout is set, so any
  client on the network can end the run at a moment she cannot predict. The
  only operator-driven restart is the panel's existing per-model Unload.
- **Operator-invoked only.** Never on by default, never on for a model Alice <!-- cond: cond-2609201007357647 -->
  did not arm, never derived from the statistics switch and never from
  `log_level`. An arm is an action on one model in a panel Alice is already
  authenticated to; there is no client-facing surface that arms it, and no
  client-facing surface that reveals it.
- **The shared-cache install's exposure, as a cost.** Where the file lives and <!-- cond: cond-2609201007353261 -->
  what mode it has are settled and unchanged (`internal/runtime/launcher.go`:
  the logs folder through the account directory, 0600, `O_NOFOLLOW`,
  `O_NONBLOCK`, a regular-file check on the handle). Under
  `make install-shared` the added exposure is not another local account reading
  the file — it is that network clients' prompts and the models' answers land
  in the serving account's directory tree, under the deliberate `3775`
  semantics the Makefile explains. That is the cost, stated; "readable by
  nobody else" is the phrase that failed in the sibling and is not used here.
- **Nothing leaves the Mac, by the records that already own it.** <!-- cond: cond-2609201007354397 -->
  adr-2609061503319212 (no public telemetry, local telemetry strictly opt-in),
  the shipped statistics intents, and itd-2609091412177263 (the server's own
  log) own that invariant between them. This record does not re-promise it; it
  links it, and the ADR that narrows adr-2609061503319212 is where the one
  exception this makes is written down.
- **A no-transcript model refuses the arm.** itd-2609091715089488's 2026-09-20 <!-- cond: cond-2609201007351874 -->
  answer: a model carrying the transcript exception refuses a debug arm with
  the reason, and both panels say so — the exception means no prompts on disk,
  not "not in this one file". This record holds that refusal, not a carve-out
  from it.
- **The log also holds the traffic Dessau's own probes send.** The context <!-- cond: cond-2609201007352211 -->
  probe posts a generated filler prompt sized to the window under test
  (`internal/contextprobe/probe.go`) and the self-test and readiness paths send
  their own requests. At DEBUG those bodies are written like any other, and the
  filler is the largest single body the server ever sees — which is why the
  size bound has to hold per write. The panel's plain words must not say the
  file holds only what clients sent.

## Acceptance Criteria

- **Arming alone changes no running process.** Given a model server that is
  running at INFO, when Alice arms debug logging for that model, then no
  signal, no argument and no file reaches that process, its log gains no line
  of prompt content, and the panel says the mode begins at the model's next
  start. *Held by:* a runtime test that arms the mark against a running fake
  and asserts the process was neither restarted nor written to.
- **The next launch runs at DEBUG and the one after at INFO.** Given a model
  armed and then unloaded, when the pool launches it, then `launchArgs` carries
  `--log-level DEBUG` exactly once and the mark is cleared in the same step, so
  that when the model is launched again it carries `--log-level INFO`. *Held
  by:* a launcher test over two consecutive launches asserting the two
  argument lists, and a third asserting that a launch which fails to spawn
  leaves the mark armed rather than spent.
- **The panel says in plain words what this writes down.** Given the control
  panel's card for a model, when Alice is about to arm debug logging, then the
  panel states that the model's own log will hold every request sent to it and
  every answer it produced, prompts and completions included, whoever sent
  them, and that this begins at the model's next start. *Held by:* an
  architecture test over `internal/ui/static/` asserting the sentence is
  present at the control, in the shape the logging docs test already uses for
  `docs/logging.md`.
- **A model at debug is visibly at debug.** Given a model that is armed or
  running at DEBUG, when Alice looks at the panel, then that model is shown as
  such — armed and running distinguished, since they are different runs — and a
  model that is neither shows nothing. *Held by:* a control-endpoint test that
  the per-model state is served, plus the static-surface test that it is drawn.
  *Hand check:* arm a model in the web panel, confirm the indication appears
  without a reload, unload it, confirm the indication moves from armed to
  running.
- **The mark never comes from the statistics switch, in either direction.**
  Given the tree, when the architecture tests run, then no file outside the
  new readers' list names the debug mark, the mark's value is derived from
  neither `Statistics` nor `log_level` anywhere, and arming it writes nothing
  to `Statistics`. *Held by:* the mirror of `statistics_switch_test.go` — a
  `debugMarkReaders` list with the reason per entry, a scan that fails a build
  which adds a reader, and a liveness rule in the shape of
  `TestStatisticsSwitchReadersAllExist` so a renamed reader cannot leave its
  exemption behind for the next file to inherit.
- **`TestTheModelServerIsAlwaysLaunchedAtInfo` is amended, not deleted.**
  Given the amended test, when the launcher names DEBUG for any reason other
  than the per-model mark — from `log_level`, from the statistics switch, or as
  a second spelling of the level — then the test fails; and it still fails if
  the level is named in more than the one place, or if an unarmed launch is not
  at INFO. *Held by:* the amended test itself, renamed to say what it now
  protects, with the comment block carrying why the exception exists and the id
  of the ADR that narrows adr-2609061503319212.
- **The file is bounded.** Given a model running at DEBUG, when a client loops
  a request, or when the context probe sends one filler body larger than the
  bound, then the per-model log stops at its bound rather than filling the
  disk, and the bound is stated in `docs/logging.md`. *Held by:* a test in the
  shape of `rotating_writer_test.go` over the per-model log's writer, with a
  case for a single write larger than the bound.
- **The ending restart leaves Alice the file.** Given a model server that ran
  at DEBUG and has stopped, when it is launched again, then the previous run's
  log is present under its own name and the new run's log is a new, empty file
  — the truncation that the docs describe today no longer destroys the run
  Alice armed. *Held by:* a launcher test asserting the rename happens before
  the `O_TRUNC` open and that the renamed file's content survives, plus a case
  for a second consecutive launch not accumulating without bound.
- **The documentation sentence is corrected.** Given `docs/logging.md`, when
  the docs tests run, then the page no longer says the model servers' debug
  level is something "no setting in Dessau asks for it", and instead says
  which per-model action asks for it, what it writes, what bounds it, and that
  the previous run's file is kept. *Held by:* the shipped
  `internal/archtest/logging_docs_test.go`, extended with the new claims.
- **A no-transcript model refuses the arm.** Given a model carrying
  itd-2609091715089488's transcript exception, when Alice arms debug logging
  for it, then the arm is refused, the reason is returned in the refusal, and
  both panels say so at the control rather than only on the failure. *Held by:*
  a control-endpoint test for the refusal and its reason, and the
  static-surface test for the sentence.
- **The exception is written down before it ships.** Given the shipped change,
  when the record is read, then the ADR that narrows adr-2609061503319212
  exists, is ratified, states the narrowing in the terms of the maintainer's
  2026-09-20 answer — one model, one process lifetime, operator-invoked, never
  on by default, never reachable from the statistics switch or `log_level`,
  named on the panel while it is on, bounded in size — and states plainly that
  the client is not told; adr-2609061503319212 links forward to it and
  adr-2609061610102325 is untouched, because shape (i) makes Dessau no reader
  of prompt content. *Held by:* the ADR link-integrity check, and a hand check
  that the amended architecture test and this record both cite that ADR's id.

## Review 2026-09-19

Two hostile passes — design and feasibility, then record discipline — against
the shipped record: itd-2609091412177263 (the server's own log),
itd-2609081259493890 (the three surfaces and the settings obligation),
adr-2609061503319212, adr-2609061610102325, adr-2609111126115848, the held
sibling itd-2609091707499248, and the code the draft would change.

### Design and feasibility

1. **The mechanism is unverified at the version this build pins.** The claim
   that the model server's debug level writes full request bodies and full
   responses was read off the project's main branch, and the research note
   carrying it says in as many words to verify it against the pinned 0.31.3
   before relying on it. The build pins 0.31.3. If that version does not log
   bodies at debug, the whole of shape (i) in Question 2 delivers nothing.
   *Changes:* a precondition on planning — read the pinned wheel's server
   source and record what it logs at debug — and Question 2's recommendation
   is conditional on the answer.

2. **"From then" is not available.** The level is a launch argument
   (`internal/runtime/launcher.go`, the one `--log-level` in `launchArgs`), and
   a running Python process has no level switch. The only semantics that work
   are arm-then-restart: arming marks the model, the next launch consumes the
   mark and runs at debug, and the launch after that is back at INFO.
   *Changes:* the press-release sentence (corrected above), and the acceptance
   criteria must say that arming alone changes no running process.

3. **The lifetime is not Alice's to control, and a stranger can end it.** A
   model server restarts when the pool evicts it under memory pressure, or when
   the idle reaper unloads it where an idle timeout is set. The only
   operator-driven restart is the panel's existing per-model Unload. So "until
   it is restarted" is an expiry another client's request can cause at a moment
   Alice cannot predict.
   *Changes:* Question 3 — the end condition is a product choice, not a
   mechanism detail.

4. **The restart that ends the mode also destroys the evidence.** Every launch
   opens the per-model log with `O_TRUNC`; that truncation is the only thing
   bounding those logs, and the docs say each is emptied when its model server
   next starts. The debug run Alice wanted to read survives exactly until that
   model next starts, so an unattended reload overnight leaves her an empty
   file.
   *Changes:* any shape that promises Alice a readable record has to keep the
   file past the next launch — a copy or a rename at exit — which is new
   behaviour the draft never asks for.

5. **Nothing bounds the file within a run.** The server's own log is
   size-rotated at 5 MB with five files kept. The per-model logs are not
   rotated at all. At debug, with full bodies, a looping client on the network
   writes bytes of its own choosing to Alice's disk for as long as the model
   stays up. The logging page already reasons about a stranger causing detailed
   lines "as fast as it can ask" at the sparse level; this is the same
   adversary with a far bigger pen.
   *Changes:* a size bound is not optional — it belongs in the scope conditions
   and in an acceptance criterion, whichever shape wins.

6. **Two shipped architecture tests and one shipped documentation sentence are
   broken by design.** `TestTheModelServerIsAlwaysLaunchedAtInfo` asserts the
   launcher names `--log-level` exactly once, as INFO, and never names DEBUG.
   The logging page tells the reader that the model servers' debug level writes
   prompts and answers "so no setting in Dessau asks for it". Both are
   deliberate and both must be amended in the same change, by a record that
   says why.
   *Changes:* the amendment is in scope and must preserve what the test was
   protecting — the level comes only from this per-model action, never from the
   statistics switch and never from `log_level` — rather than deleting the
   assertion.

7. **The readers' list is the right instrument and the wrong list.** The
   statistics-switch guard names the files allowed to read that switch and
   fails a build that adds another. The new action needs the mirror of it: the
   debug mark is its own field, it is never derived from `Statistics` or
   `log_level`, and its only reader in the runtime package is the launcher's
   level argument. Without that test, the draft's best sentence — "turning
   statistics on never raises a log level, and this never turns statistics on"
   — is prose.
   *Changes:* one acceptance criterion per direction, both held by an
   architecture test, plus a liveness rule on the new list the way
   `TestStatisticsSwitchReadersAllExist` holds its own.

8. **There are two credible shapes and they cost different things.** (i) Raise
   the child process's level: Dessau reads no prompt content, so the
   prompt-content decision record is untouched, but the content is whatever the
   model server chooses to write — unbounded, unredacted, unverified at the
   pin, and not available to any panel view. (ii) Have the gateway write the
   request-and-answer log itself: bounded, rotatable, formatted, scoped to one
   model — but it makes the gateway a second reader of prompt content and a
   writer of it to disk, and the prompt-content record grants reading for
   exactly one purpose and says any further reading is a new decision that
   supersedes it. The held sibling's review already named this cost as "two
   readers and one writer".
   *Changes:* Question 2. Nothing below the shape can be specified until it is
   picked.

9. **The overlap with the recording draft is real, was adjudicated, and is
   recorded nowhere in this file.** The decomposition note of 2026-09-06 records
   the recording draft as *proposing* to supersede this one; the maintainer kept
   this record separate at the interview of 2026-09-09, and the shipped server
   log intent's own mechanism defers prompts and answers to this draft by id.
   A reader of this file alone finds none of that: `builds_on` was empty.
   *Changes:* `builds_on` now names the shipped server-log intent; the
   relationship to the held sibling and to both decision records is stated in
   this section. It is not a duplicate: the sibling is a server-wide recording
   mode with a client-facing notice, this is a per-model diagnostic with no
   client-facing surface at all.

10. **"Bob's clients see no change" is the promise the sibling was held on.**
    Alice's diagnostic writes Bob's prompts and the model's answers to Alice's
    disk, and Bob is told nothing. The telemetry record says local telemetry
    never records prompt text or completions even when it is switched on. The
    diagnostic carve-out record is the precedent for narrowing a ratified rule
    for something a person deliberately invokes — but it narrows *reporting an
    observed signal*, not writing a third party's content to disk.
    *Changes:* Question 1. This record cannot be planned until it is answered,
    and the answer is a new decision record either way.

11. **The panel sentence is the one unambiguously buildable part, and it is
    missing its other half.** If the mark is a json-tagged field of the
    configuration, the settings-surface obligation applies and it needs a
    control or a written exemption. If it is transient state, it is an action
    rather than a setting, which the settings obligation's own scope conditions
    exclude — and then nothing in the draft asks for a visible indication of
    which model is at debug right now. A mode with no visible "on" is how debug
    logging stays on for a month.
    *Changes:* whichever it is, an acceptance criterion for the plain-words
    sentence *and* one for the "this model is at debug" indication.

12. **Where the file lives and what mode it has are already settled.** The logs
    folder resolves through the account directory, the per-model log is opened
    0600 with `O_NOFOLLOW` and `O_NONBLOCK` and a regular-file check on the
    handle. Nothing here needs a new location or a new mode. Under the
    shared-cache install the added exposure is not another local account reading
    the file — it is network clients' content landing in the serving account's.
    *Changes:* the scope conditions link these rather than restate them, and
    the shared-cache exposure is stated as a cost rather than left to "readable
    by nobody else", which is the phrase that failed in the sibling.

### Record discipline

13. **There is no press release.** The section carries its seed placeholder and
    "Why This Matters" was the title pasted verbatim, so the record said the
    same thing twice and explained nothing.
    *Changes:* "Why This Matters" is now an argument rather than a copy; the
    press release stays unexpanded, and says why.

14. **The itd-1 discipline is unmet three times over.** Mechanism, Scope
    Conditions and Acceptance Criteria are all still generated prompts, so
    `abcd intent ready` cannot pass and planning is refused.
    *Changes:* they stay unfilled deliberately while the open questions stand,
    and this section is the record of why the draft is not simply unattended.

15. **The strongest sentence in the record is the one with an instrument
    waiting for it.** The separation from the statistics switch is stated as
    prose where a shipped test already holds the same rule in one direction.
    *Changes:* it becomes acceptance criteria (finding 7), not a claim.

16. **A claim that duplicates shipped work.** "Nothing leaves the Mac either
    way" restates the no-public-telemetry record and the shipped statistics
    intents. An invariant with several owners is an invariant nobody maintains.
    *Changes:* it is linked in the scope conditions when they are written, not
    re-promised.

17. **A claim that contradicts shipped prose.** The logging reference page,
    delivered under the shipped server-log intent, tells the reader that no
    setting in Dessau asks for the model servers' debug level. If this ships
    that sentence is false — while the same intent's mechanism names this draft
    as where prompts and answers live. The records agree; the delivered page
    does not.
    *Changes:* the documentation change is inside this record's scope and gets
    an acceptance criterion, rather than being discovered by whoever ships it.

18. **The frontmatter understates the change.** `severity: minor` and
    `impact: additive` describe a change that amends a ratified decision
    record's reach, amends two architecture tests, and writes third parties'
    prompt content to disk.
    *Changes:* left as captured for now, because Questions 1 and 2 decide what
    it actually is; the maintainer's answer sets them at the interview.

19. **Personas are correct.** Alice operates the server, Bob is a client, Carol
    is unused, no other names appear, and no artefact refers to the maintainer.
    Nothing to change.

*Before any planning, and needing nobody's decision:* verify what the pinned
model-server version writes at its debug level (finding 1), and record it as a
research note. The answer changes which shape is even available.

## Open Questions

Three, and each is the maintainer's: a privacy posture, a choice between two
credible shapes, and a product choice about what ends the mode. All three were
answered at the 2026-09-20 interview and the draft is no longer held; the
questions and their options stay below as the record of what was weighed, each
with the line saying where its answer lives.

- **Question 1 is answered:** (a). `.abcd/work/DECISIONS.md`, 2026-09-20; the
  narrowing ADR is named in the Acceptance Criteria as the ADR that narrows
  adr-2609061503319212.
- **Question 2 is answered:** (i), and its condition is met — the pinned
  0.31.3 does write both bodies at DEBUG, verified in
  `.abcd/development/research/notes/2026-09-20-mlx-lm-0.31.3-debug-logging.md`.
  `.abcd/work/DECISIONS.md`, 2026-09-20.
- **Question 3 is answered:** (i), plus a size bound, plus keeping the previous
  run's file. `.abcd/work/DECISIONS.md`, 2026-09-20; all three are acceptance
  criteria above.
- **Finding 18 is answered by the answers:** `severity` is now `major` — the
  change amends a ratified ADR's reach and writes third parties' content to
  disk — and `impact` stays `additive`, since nothing existing changes
  behaviour for anyone who does not arm it.

**Question 1 — May an operator-invoked diagnostic record prompts and answers a
client never consented to, and under which record?**

- (a) A new decision record narrowing the no-public-telemetry record the way
  the diagnostic carve-out narrowed its own predecessor: a deliberately
  invoked, per-model, single-process-lifetime diagnostic may write request and
  response content to the operator's own log, with no notice to clients, and
  the record states plainly that the client is not told.
- (b) Require the disclosure the sibling recording draft could not deliver.
  That holds this draft too, on the same reasoning and probably for as long.
- (c) Decline the content: build the panel action and raise the level, but bound
  what is written to what is not prompt content — which, at this level, is
  approximately nothing. This is a decline of the intent wearing a feature's
  clothes.

*Recommendation: (a).* A diagnostic Alice types while she is already debugging
changes no behaviour, admits no request and gates nothing — the exact
distinction the diagnostic carve-out already drew — and the exception can be
written as narrowly as that one was: one model, one process lifetime,
operator-invoked, never on by default, never reachable from the statistics
switch, named on the panel while it is on, and bounded in size. The cost, that
Bob is not told, is stated in the record rather than designed around, because
no stateless server can tell him; the sibling's hold is the evidence for that.

**Question 2 — Which shape: raise the model server's own level, or have the
gateway write the log?**

- (i) The child's level. No new reader of prompt content anywhere in Dessau,
  so the prompt-content record is untouched. The content, the format and the
  size are the model server's to choose; there is nothing to redact with and
  no panel view of it; and it is unverified at the pinned version (finding 1).
- (ii) The gateway writes it. Bounded, rotatable, one model only, the format
  ours, a panel view possible, secrets redactable. It makes the network-facing
  process a second reader of prompt content and the first writer of it to
  disk, which the prompt-content record says is a new decision superseding it.

*Recommendation: (i), conditional on finding 1.* The question Alice is asking
is "what is the client actually sending", and the upstream request body is
exactly that; keeping Dessau's own blindness to prompt content intact is worth
more than a prettier file. If the maintainer wants the log bounded by us,
redacted, or visible in the panel, the answer is (ii) and the decision record
in Question 1 becomes the larger one — it then supersedes the prompt-content
record as well as narrowing the telemetry one.

**Question 3 — What ends it, and what bounds it?**

- (i) As drafted: the model server's next restart — which another client's
  request can cause, and which Alice cannot predict (finding 3), and which
  erases the log (finding 4).
- (ii) A switch Alice turns off: survives restarts until she does.
- (iii) A time box, or a request count, whichever comes first.

Independently of the choice: the file needs a size bound it does not have
today (finding 5).

*Recommendation: (i) plus a size bound, plus keeping the previous run's file.*
The process lifetime is the only boundary that needs no new persistent state
anywhere, cannot outlive a reboot, and cannot be forgotten; the title already
chose it. Option (ii) is the shape that leaves debug logging on for a month,
which is the failure this class of feature always has. If (i) is taken, finding
4 must be answered — the ending restart has to leave Alice the file, not an
empty one.

## Audit Notes



**Answer 2026-09-20 (Question 1):** (a) — a new ADR narrowing the telemetry ADR; the client is not told, and the record says so. Recorded in `.abcd/work/DECISIONS.md`.

**Answer 2026-09-20 (Question 2):** (i) the child's level, conditional on the 0.31.3 verification. **(Question 3):** (i) next restart, plus a size bound, plus keeping the previous run's file. Recorded in `.abcd/work/DECISIONS.md`.

<!-- abcd-review: INGESTED receipt=rcp-adb86bbdc558 -->
Fidelity review — receipt rcp-adb86bbdc558 (verifier intent-auditor claude-opus-5).

Provenance: intent-auditor@claude-opus-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:fafa96a88f16f4296d99b98ece2cb9e2e9d496e87d3d12c02ce2fb042bec32ad
Input attestations: diff:12a42cfb4ba0fdef1b18b0712090fa0e3e0d8c8e..638ea61c981e083859ac5a7ed83390040ac0616a@sha256:c29dc967cd6cf5a705eb1ca7b49962a1dad1c662a193fb8e2acd7ebcda71f69b;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Pool.ArmDebugLog only writes a map entry under p.mu; TestArmingTouchesNoRunningProcess arms against a running fake and asserts one launch, same process, not stopped, no request sent, spec unchanged; the panel blurb says the run begins at the model's next start.
  evidence: internal/runtime/pool.go:721 — "func (p *Pool) ArmDebugLog(repoID string) error {"
  evidence: internal/runtime/debuglog_test.go:14 — "func TestArmingTouchesNoRunningProcess(t *testing.T) {"
  evidence: internal/runtime/debuglog_test.go:35 — "t.Error("arming replaced the running process")"
  evidence: internal/ui/static/index.html:71 — "From that model's next start until the start after it"
- ac-2 — MET: launchArgs sets level from Spec.DebugLog; startLocked reads the mark and deletes it only after Launch returns nil; TestAnArmedLaunchIsAtDebugAndTheNextIsAtInfo asserts INFO/DEBUG/INFO with --log-level counted once per launch, and TestALaunchThatFailsToSpawnLeavesTheMarkArmed asserts the mark survives a failed spawn and is spent by the next success.
  evidence: internal/runtime/launcher.go:256 — "level := "INFO""
  evidence: internal/runtime/pool.go:1354 — "delete(p.debugArmed, key)"
  evidence: internal/runtime/debuglog_test.go:96 — "for i, want := range []string{"INFO", "DEBUG", "INFO"} {"
  evidence: internal/runtime/debuglog_test.go:140 — "func TestALaunchThatFailsToSpawnLeavesTheMarkArmed(t *testing.T) {"
- ac-3 — MET: index.html carries the debugLogBlurb paragraph with every promised phrase; app.js sets the card button's title from that paragraph at render time; TestThePanelSaysWhatDebugLoggingWrites holds the phrases with the same containsAll shape logging_docs_test uses.
  evidence: internal/ui/static/index.html:70 — "< p class="hint" id="debugLogBlurb">Debug logging is a per-model diagnostic and is off unless you arm it."
  evidence: internal/ui/static/index.html:72 — "request sent to it and every answer it produced — the prompts and the completions, whoever sent"
  evidence: internal/archtest/debug_mark_test.go:140 — "func TestThePanelSaysWhatDebugLoggingWrites(t *testing.T) {"
  evidence: internal/archtest/debug_mark_test.go:170 — "if !containsAll(para, want) {"
- ac-4 — MET_WITH_CONCERNS: State.DebugArmed and Resident.DebugLog are served (TestTheSnapshotCarriesTheDebugState) and the card draws two distinct pills from those two fields with nothing for a model that is neither (TestTheCardDrawsTheDebugPills); concern: the recorded hand check went through /api/state and resident[].debug_log rather than the web panel, so 'appears without a reload' in the DOM was not the check performed.
  evidence: internal/gateway/control.go:352 — "DebugArmed []string `json:"debug_armed"`"
  evidence: internal/runtime/pool.go:123 — "DebugLog bool `json:"debug_log"`"
  evidence: internal/ui/static/app.js:406 — "pill += '< span class="pill debug">debug armed< /span>'"
  evidence: internal/ui/static/app.js:407 — "pill += '< span class="pill debug">logging at debug< /span>'"
  evidence: internal/gateway/debuglog_test.go:65 — "func TestTheSnapshotCarriesTheDebugState(t *testing.T) {"
  evidence: internal/archtest/debug_mark_test.go:195 — "func TestTheCardDrawsTheDebugPills(t *testing.T) {"
  evidence: .abcd/work/DECISIONS.md:383 — "Row 4 checked by hand on the scratch root with the small model: arming appears in `/api/state` on a fresh read"
- ac-5 — MET: debugMarkReaders lists five files with a reason each; TestTheDebugMarkIsNamedOnlyByItsReaders walks the tree and fails on any other file naming the mark; TestTheDebugMarkIsDerivedFromNothingElse forbids .Statistics/.LogLevel/"log_level" in pool.go, launcher.go and the handleDebugLog body; TestDebugMarkReadersAllExist is the liveness rule; TestTheDebugArmWritesNoSettings shows arming leaves Config and config.json byte-identical with Statistics on.
  evidence: internal/archtest/debug_mark_test.go:25 — "var debugMarkReaders = map[string]string{"
  evidence: internal/archtest/debug_mark_test.go:39 — "func TestTheDebugMarkIsNamedOnlyByItsReaders(t *testing.T) {"
  evidence: internal/archtest/debug_mark_test.go:90 — "forbidden := []string{".Statistics", "Statistics bool", ".LogLevel", `"log_level"`}"
  evidence: internal/archtest/debug_mark_test.go:122 — "func TestDebugMarkReadersAllExist(t *testing.T) {"
  evidence: internal/gateway/debuglog_test.go:159 — "func TestTheDebugArmWritesNoSettings(t *testing.T) {"
- ac-6 — MET: The archtest is renamed TestTheModelServerLevelComesOnlyFromThePerModelDebugMark with a comment block naming adr-2609201008477513 and why the exception exists; it still counts --log-level exactly once, requires level := "INFO" for the unarmed path, allows exactly one "DEBUG" spelling as debugLogLevel used once inside the `if spec.DebugLog` block, and forbids .Statistics/.LogLevel/"log_level" in launcher.go.
  evidence: internal/archtest/statistics_switch_test.go:108 — "func TestTheModelServerLevelComesOnlyFromThePerModelDebugMark(t *testing.T) {"
  evidence: internal/archtest/statistics_switch_test.go:97 — "// Why there is an exception at all. adr-2609201008477513 narrows"
  evidence: internal/archtest/statistics_switch_test.go:118 — "if n := strings.Count(src, `"--log-level"`); n != 1 {"
  evidence: internal/archtest/statistics_switch_test.go:124 — "if n := strings.Count(src, `"DEBUG"`); n != 1 {"
  evidence: internal/runtime/launcher_test.go:177 — "func TestAnUnarmedModelServerIsLaunchedAtInfoAndAnArmedOneAtDebug(t *testing.T) {"
- ac-7 — MET: An armed launch routes both streams through boundedWriter with DebugLogMaxBytes = 64 MiB; TestTheDebugLogStopsAtItsBound has a looping-client case, a single-write-larger-than-the-bound case and a concurrency case, TestAnArmedLaunchStopsItsLogAtTheBound exercises it through the real Launch pipe, and docs/logging.md states the 64 MB bound (figure tied to the constant by the docs test); the writer swallowing write errors is captured as iss-2609210903529294.
  evidence: internal/runtime/launcher.go:55 — "const DebugLogMaxBytes = 64 << 20"
  evidence: internal/runtime/launcher.go:355 — "bounded := newBoundedWriter(logFile, max)"
  evidence: internal/runtime/boundedwriter_test.go:43 — "t.Run("a single write larger than the bound is cut at the bound", func(t *testing.T) {"
  evidence: internal/runtime/launcher_test.go:318 — "func TestAnArmedLaunchStopsItsLogAtTheBound(t *testing.T) {"
  evidence: docs/logging.md:86 — "so its log stops at 64 MB: what fits is written, one final line says the log"
- ac-8 — MET: keepPreviousLog runs at launcher.go:309 before the O_CREATE|O_TRUNC open at :331, renaming through an os.Root to < name>.previous.log; TestALaunchKeepsThePreviousRunsLog asserts the old content survives under the previous name and the new file is fresh, TestTwoConsecutiveLaunchesKeepOnlyOnePreviousLog asserts exactly two files after two launches.
  evidence: internal/runtime/launcher.go:309 — "if err := keepPreviousLog(l.LogDir, spec.RepoID); err != nil {"
  evidence: internal/runtime/launcher.go:332 — "os.O_CREATE|os.O_WRONLY|os.O_TRUNC|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0o600)"
  evidence: internal/runtime/launcher.go:443 — "return root.Rename(name, previousLogFileName(repoID))"
  evidence: internal/runtime/launcher_test.go:258 — "func TestALaunchKeepsThePreviousRunsLog(t *testing.T) {"
  evidence: internal/runtime/launcher_test.go:282 — "func TestTwoConsecutiveLaunchesKeepOnlyOnePreviousLog(t *testing.T) {"
- ac-9 — MET: docs/logging.md no longer says 'no setting in Dessau asks for it' and now names the per-model Debug logging action, what it writes, the 64 MB bound and the kept .previous.log; TestTheLoggingPageDescribesThePerModelDebugAction in the shipped logging_docs_test.go refuses the old sentence and requires each new claim.
  evidence: docs/logging.md:69 — "One action does. **Debug logging**, on a model's card in the control panel,"
  evidence: docs/logging.md:92 — "`<org>@<name>.previous.log`, and replaces the one before that — so the run"
  evidence: internal/archtest/logging_docs_test.go:118 — "func TestTheLoggingPageDescribesThePerModelDebugAction(t *testing.T) {"
  evidence: internal/archtest/logging_docs_test.go:121 — "if containsAll(page, "no setting in Dessau asks for it") {"
- ac-10 — MET_WITH_CONCERNS: handleDebugLog refuses with 409 and the reason when Control.TranscriptExcepted says so, TestAnExceptedModelRefusesTheDebugArm drives it through an injected predicate, and the Models-pane blurb and docs say a no-transcript model refuses; concern: nothing in internal/app sets TranscriptExcepted (only the test does), so in the shipped tree the predicate is nil and no model is refused — itd-2609091715089488 is still in planned/ and the spec records the wiring as owed to it; and only the web panel's blurb carries the sentence at the control, no second panel surface was found.
  evidence: internal/gateway/control.go:1546 — "if c.TranscriptExcepted != nil && c.TranscriptExcepted(req.Model) {"
  evidence: internal/gateway/control.go:1548 — ""this model keeps no transcript, so debug logging is refused: at the model server's debug level its log would hold every prompt sent to it")"
  evidence: internal/gateway/debuglog_test.go:120 — "func TestAnExceptedModelRefusesTheDebugArm(t *testing.T) {"
  evidence: internal/gateway/debuglog_test.go:43 — "ctrl := &Control{App: a, TranscriptExcepted: excepted}"
  evidence: internal/ui/static/index.html:76 — "A model that keeps no"
  evidence: .abcd/development/specs/closed/spc-2609201007359229-debug-logging-for-one-model-until-it-is-restarted-alice-pick.md:378 — "**`internal/app` is untouched and `Control.TranscriptExcepted` is not"
- ac-11 — MET_WITH_CONCERNS: adr-2609201008477513 exists with status accepted, states all seven conditions in the maintainer's terms and Condition 7 says plainly the client is not told; adr-2609061503319212's status field links forward to it and adr-2609061610102325 is untouched in the range; the amended archtest cites the id. Concerns: the intent record itself cites the ADR only as 'the ADR that narrows adr-2609061503319212' and never by id (0 occurrences), so the hand check's second half fails; the ADR landed in 44a72c12 before the delivered range; the forward link lives in the old ADR's status prose while its related_adrs stays [], and abcd lint surfaces no ADR link-integrity result to cite.
  evidence: .abcd/development/decisions/adrs/2609201008477513-a-deliberately-invoked-per-model-diagnostic-may-write-prompt.md:4 — "status: accepted"
  evidence: .abcd/development/decisions/adrs/2609201008477513-a-deliberately-invoked-per-model-diagnostic-may-write-prompt.md:145 — "**Condition 7 — the client is not told, and the record says so plainly.**"
  evidence: .abcd/development/decisions/adrs/2609061503319212-no-public-telemetry-local-telemetry-only-as-a-strict-opt-in.md:4 — "narrowed in part by adr-2609201008477513, on the per-model diagnostic"
  evidence: .abcd/development/decisions/adrs/2609061503319212-no-public-telemetry-local-telemetry-only-as-a-strict-opt-in.md:10 — "related_adrs: []"
  evidence: internal/archtest/statistics_switch_test.go:97 — "// Why there is an exception at all. adr-2609201008477513 narrows"
  evidence: .abcd/development/intents/shipped/itd-2609062346072707-debug-logging-for-one-model-until-it-is-restarted-alice-pick.md:258 — "when the record is read, then the ADR that narrows adr-2609061503319212"

Gap audit:
- honoured:
  - arming is an in-memory mark that touches no running process and is spent by the next successful launch only
    evidence: internal/runtime/pool.go:1342 — "_, debugLog := p.debugArmed[key]"
    evidence: internal/runtime/debuglog_test.go:14 — "func TestArmingTouchesNoRunningProcess(t *testing.T) {"
  - the level is derived from the mark alone, held by the mirror readers' list, the liveness rule and the amended launcher archtest
    evidence: internal/archtest/debug_mark_test.go:25 — "var debugMarkReaders = map[string]string{"
    evidence: internal/archtest/statistics_switch_test.go:108 — "func TestTheModelServerLevelComesOnlyFromThePerModelDebugMark(t *testing.T) {"
  - an armed run's log is bounded per write at 64 MB and the previous run's file is kept as .previous.log
    evidence: internal/runtime/boundedwriter.go:32 — "func (b *boundedWriter) Write(p []byte) (int, error) {"
    evidence: internal/runtime/launcher.go:426 — "func keepPreviousLog(dir, repoID string) error {"
  - the panel, the posture line and docs/logging.md say in plain words what the file holds, that probes' traffic is in it, and that clients are not told
    evidence: internal/ui/static/index.html:70 — "id="debugLogBlurb""
    evidence: internal/ui/static/app.js:1006 — "lines.push({ id: 'debug_log', heading: 'Debug logging', text,"
    evidence: docs/logging.md:82 — "anywhere, and clients are not told. A model that keeps no transcript refuses"
  - a deleted model takes its mark with it so a re-download is not launched at debug (review fix beyond the press release)
    evidence: internal/app/app.go:1862 — "if err := a.Pool.Remove(repoID); err != nil {"
    evidence: internal/runtime/pool.go:2280 — "delete(p.debugArmed, key)"
- diverged:
  - a no-transcript model refuses the arm: delivered as a seam (Control.TranscriptExcepted) that the app never sets, so the refusal is not live until itd-2609091715089488 lands
    evidence: internal/gateway/control.go:84 — "TranscriptExcepted func(repoID string) bool"
    evidence: .abcd/development/specs/closed/spc-2609201007359229-debug-logging-for-one-model-until-it-is-restarted-alice-pick.md:378 — "**`internal/app` is untouched and `Control.TranscriptExcepted` is not"
  - the row-4 hand check was performed through /api/state rather than in the web panel without a reload
    evidence: .abcd/work/DECISIONS.md:383 — "arming appears in `/api/state` on a fresh read"
  - the two named tests were renamed: TestTheModelServerIsAlwaysLaunchedAtInfo → TestTheModelServerLevelComesOnlyFromThePerModelDebugMark, and the runtime sibling TestEveryModelServerIsLaunchedAtInfo → TestAnUnarmedModelServerIsLaunchedAtInfoAndAnArmedOneAtDebug (amended, not deleted, as promised)
    evidence: internal/archtest/statistics_switch_test.go:108 — "func TestTheModelServerLevelComesOnlyFromThePerModelDebugMark(t *testing.T) {"
    evidence: internal/runtime/launcher_test.go:177 — "func TestAnUnarmedModelServerIsLaunchedAtInfoAndAnArmedOneAtDebug(t *testing.T) {"
- missing:
  - the intent record citing adr-2609201008477513 by id (the hand check in ac-11); the record names the ADR only by description
    evidence: .abcd/development/intents/shipped/itd-2609062346072707-debug-logging-for-one-model-until-it-is-restarted-alice-pick.md:258 — "then the ADR that narrows adr-2609061503319212"
  - the wiring in internal/app that makes TranscriptExcepted true for a real model (owed to itd-2609091715089488, still in planned/)
    evidence: internal/gateway/debuglog_test.go:147 — "// the per-model field the predicate reads exists."
    evidence: .abcd/development/intents/planned/itd-2609091715089488-some-models-keep-no-transcript-even-while-recording-is-on-al.md:1 — "---"

Scope-condition dispositions:
- cond-2609201007358534 — survived: the pin is still mlx-lm==0.31.3 (requirements and provision.go untouched in the range) and the recorded hand check saw the prompt in the model's log at DEBUG, as the note says that version writes
  evidence: internal/runtime/mlx-requirements.txt:211 — "mlx-lm==0.31.3 \"
  evidence: .abcd/work/DECISIONS.md:383 — "the model's log at DEBUG with the prompt in it"
- cond-2609201007350925 — survived: the mark is keyed per folded repo id, spent at the one launch that carries it, and the level is a launch flag so the run ends with the process however it stops
  evidence: internal/runtime/pool.go:1354 — "delete(p.debugArmed, key)"
  evidence: internal/runtime/debuglog_test.go:63 — "func TestAnArmedLaunchIsAtDebugAndTheNextIsAtInfo(t *testing.T) {"
- cond-2609201007357647 — survived: the only arming surface is the control-plane route POST /api/models/debug-log; the archtests forbid deriving the mark from .Statistics/.LogLevel/"log_level" and the arm writes nothing to config
  evidence: internal/gateway/control.go:149 — "mux.HandleFunc("POST /api/models/debug-log", c.handleDebugLog)"
  evidence: internal/archtest/debug_mark_test.go:82 — "func TestTheDebugMarkIsDerivedFromNothingElse(t *testing.T) {"
  evidence: internal/gateway/debuglog_test.go:159 — "func TestTheDebugArmWritesNoSettings(t *testing.T) {"
- cond-2609201007353261 — survived: the open is unchanged (0600, O_NOFOLLOW, O_NONBLOCK, regular-file check) and the rename stays inside the same logs directory through an os.Root, so the file's place and mode are as the condition assumed; the ADR states the shared-cache cost in the condition's own words
  evidence: internal/runtime/launcher.go:332 — "os.O_CREATE|os.O_WRONLY|os.O_TRUNC|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0o600)"
  evidence: internal/runtime/launcher.go:427 — "root, err := os.OpenRoot(dir)"
  evidence: .abcd/development/decisions/adrs/2609201008477513-a-deliberately-invoked-per-model-diagnostic-may-write-prompt.md:206 — "- The shared-cache install is where this costs most."
- cond-2609201007354397 — survived: the delivery adds no egress path — the armed run's bytes go to the local log file only — and the narrowing ADR re-links the invariant to the records that own it rather than re-promising it
  evidence: internal/runtime/launcher.go:355 — "bounded := newBoundedWriter(logFile, max)"
  evidence: .abcd/development/decisions/adrs/2609201008477513-a-deliberately-invoked-per-model-diagnostic-may-write-prompt.md:212 — "- Nothing leaves the Mac. The records that own that invariant"
- cond-2609201007351874 — narrowed: the refusal, its 409 and its reason exist behind Control.TranscriptExcepted, but nothing sets that predicate and no model in the tree can carry the transcript exception, so the refusal holds only for models a caller names through the seam
  narrowing: holds only once internal/app wires Control.TranscriptExcepted to itd-2609091715089488's per-model field; in the shipped tree the predicate is nil and no model is refused
  evidence: internal/gateway/control.go:1546 — "if c.TranscriptExcepted != nil && c.TranscriptExcepted(req.Model) {"
  evidence: internal/gateway/debuglog_test.go:148 — "func TestANilTranscriptSeamRefusesNothing(t *testing.T) {"
- cond-2609201007352211 — survived: the bound holds per write (a single write larger than the bound is cut at it) and the panel blurb, posture line and docs name Dessau's own probes and self-test as writers
  evidence: internal/runtime/boundedwriter_test.go:43 — "t.Run("a single write larger than the bound is cut at the bound", func(t *testing.T) {"
  evidence: internal/ui/static/index.html:73 — "them — and the requests Dessau's own probes and self-test send as well."
## Grounds

- pursued: the maintainer answered every open question at the 2026-09-20 interview and the itd-1 sections were written from the answers; what would show this wrong is a criterion that cannot be held by the test it names
