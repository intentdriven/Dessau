---
id: itd-2609062346072707
slug: debug-logging-for-one-model-until-it-is-restarted-alice-pick
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091412177263]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Debug logging for one model, until it is restarted: Alice picks a model in the control panel and asks Gropius to run it with its own logging turned all the way up, in a panel that says in plain words what that writes down. From then until that model's server is restarted, its log holds every request sent to it and every answer it produced, prompts and completions included, so Alice can see what a client is actually sending when a model answers strangely. It is per model, it is never on for a model she did not choose, and it is a separate action from recording request statistics: turning statistics on never raises a log level, and this never turns statistics on. Bob's clients see no change, and nothing leaves the Mac either way.

> The title is the quoted capture and is left as captured — a record's id is not
> a description. Two of its sentences do not survive the review below and are
> corrected in "Why This Matters" rather than rewritten here: the log cannot
> start recording "from then" (the level is a launch flag, so it starts at the
> next launch), and "Bob's clients see no change" is a disclosure question, not
> a reassurance.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

Not expanded. The narrative turns on the three open questions below: whether
this may record a client's prompts at all, which of the two shapes writes the
log, and what ends it. Writing the press release now would settle them by
prose.

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
it writes each request body and each response to its own per-model log. Gropius
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

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

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
   prompts and answers "so no setting in Gropius asks for it". Both are
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
   the child process's level: Gropius reads no prompt content, so the
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
    setting in Gropius asks for the model servers' debug level. If this ships
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
credible shapes, and a product choice about what ends the mode. The draft is
held until they are answered.

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

- (i) The child's level. No new reader of prompt content anywhere in Gropius,
  so the prompt-content record is untouched. The content, the format and the
  size are the model server's to choose; there is nothing to redact with and
  no panel view of it; and it is unverified at the pinned version (finding 1).
- (ii) The gateway writes it. Bounded, rotatable, one model only, the format
  ours, a panel view possible, secrets redactable. It makes the network-facing
  process a second reader of prompt content and the first writer of it to
  disk, which the prompt-content record says is a new decision superseding it.

*Recommendation: (i), conditional on finding 1.* The question Alice is asking
is "what is the client actually sending", and the upstream request body is
exactly that; keeping Gropius's own blindness to prompt content intact is worth
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

_Empty. Populated by intent-auditor when intent moves to shipped/._
