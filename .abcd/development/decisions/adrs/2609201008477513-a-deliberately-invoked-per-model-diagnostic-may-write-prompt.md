---
id: adr-2609201008477513
slug: a-deliberately-invoked-per-model-diagnostic-may-write-prompt
status: accepted
date: 2026-09-20
supersedes: null
superseded_by: null
related_intents: [itd-2609062346072707, itd-2609091715089488]
related_rfcs: []
related_adrs: [adr-2609061503319212, adr-2609111126115848, adr-2609201008476813, adr-2609201008470380, adr-2609061610102325]
---

# ADR-2609201008477513: A deliberately invoked per-model diagnostic may write prompts and answers to the operator's own log, and the client is not told

**Narrows:** [adr-2609061503319212](2609061503319212-no-public-telemetry-local-telemetry-only-as-a-strict-opt-in.md),
on the per-model diagnostic only. It does not supersede that record; the record
that does is
[adr-2609201008476813](2609201008476813-local-telemetry-may-record-prompt-text-and-completions-only.md),
which carries this exception by reference.

## Context

itd-2609062346072707 promises Alice one thing: for one model, for one run of
that model's server, she can see what a client is *actually* sending and what
the model is *actually* answering. That is the failure the draft opens on — an
agent client whose requests are refused, or answered oddly, for a reason no
count and no timing can show. Token totals, latencies and status codes are
what the statistics store holds, and none of them settle a malformed tool block
or a system message in the wrong place.

The pinned model server already has that capability and it costs nothing to
reach. A verification made a precondition of planning this draft, and carried
out on 2026-09-20 against the pinned wheel itself
(`.abcd/development/research/notes/2026-09-20-mlx-lm-0.31.3-debug-logging.md`),
establishes the facts this decision rests on:

- At `--log-level DEBUG` mlx-lm 0.31.3 writes the whole request body, including
  every message, as indented JSON; every generation step's text; and the whole
  non-streaming response object. Both sides of the conversation, in full.
- Nothing in it is redacted, sampled, truncated or rate-limited. An API key a
  client puts inside a message is written verbatim.
- The level is set once by a bare `basicConfig` at start-up from a launch flag.
  A running process has no way to change it, and there is no size bound of any
  kind — no handler, no `maxBytes`. The bound is Gropius's to add.
- Content from the network already reaches the per-model log *today*, at INFO:
  an unparseable request body, and a body that parses but is not an object, are
  logged at ERROR with the raw bytes.

So the mechanism is a launch flag, and the record that has an opinion about it
is adr-2609061503319212. That record's clause is unqualified: "Even when on,
local telemetry never records prompt text, completions, API keys, or anything
the existing error redaction removes." Its own definition of local telemetry —
"recorded on that Mac, for the operator of that Mac, and shown in Gropius's own
control panel or logs" — is exactly what a per-model log at DEBUG would be.
Shipping the diagnostic without settling that would be routing around a
ratified record rather than amending it.

Two things make the tension narrower than it first looks, and one makes it
sharper.

**Gropius reads nothing.** The shape chosen at the 2026-09-20 interview is
shape (i): raise the child's own level and let the child write its own log. The
gateway parses no message, retains no message and never sees the file's
contents. adr-2609061610102325's boundary — the one reader, for the one merge
purpose — is untouched, the readers' list in
`internal/archtest/prompt_content_test.go` gains no entry, and the alternative
shape in which the gateway composes the log (shape (ii)) is the one that would
have needed a wider decision. It is not needed at the pin.

**This is a diagnostic, and the house already knows what that means.**
adr-2609111126115848 met a ratified rule that had a deliberately invoked
diagnostic in its scope, and narrowed it rather than routing round it: one
class of verb, invoked by a person, at the moment they are already debugging,
gating nothing, under conditions that are armed by tests rather than asserted
in prose. The same reasoning applies here and the same shape is used. An arm is
an action Alice takes in a panel she is already authenticated to, on one model
she names, at a moment she has chosen.

**And the sharp edge: Bob is not told.** A client whose prompts are written to
that file receives no notice, because there is nothing that could carry one
honestly. The server holds no session; every request stands alone; and the file
is the child's, written by a process the gateway does not speak to about
logging. The transcript store decided in
[adr-2609201008470380](2609201008470380-the-gateway-may-retain-both-sides-of-a-conversation-in-a-tra.md)
can disclose on every answer precisely because Gropius composes it. This cannot.
Three options were put to the maintainer at the 2026-09-20 interview and the
answer was (a): narrow the record, write the diagnostic, and state the cost in
the record rather than designing around it.

## Decision

We narrow adr-2609061503319212 on one point only: **its bar on recording prompt
text and completions does not reach a per-model diagnostic that the operator
deliberately invokes, under the conditions below.** Every other decision in
that record — no public telemetry, ever; local telemetry strictly opt-in; the
statistics store content-free — stands unchanged and is read from there and
from adr-2609201008476813, which restates it.

All seven conditions hold together. A diagnostic that drops one is not covered
by this record.

**Condition 1 — operator-invoked, per model, and never derived.** Never on by
default; never on for a model Alice did not arm; never reachable from the
statistics switch, and never from `log_level`. The mark is its own field, read
in one place — the launcher's level argument — and derived from no other
setting. There is no client-facing surface that arms it, and no client-facing
surface that reveals it.

**Condition 2 — one process lifetime, and arming alone changes nothing.**
Arming marks the model. The next launch of that model's server consumes the
mark and runs at DEBUG; the launch after it is back at INFO with nothing to
remember. A running process is not raised by an arm, and the run that is bought
ends when that model's server next stops — for a reason Alice does not choose,
since the pool evicts under memory pressure and the idle reaper unloads. A
switch that survives restarts was declined at the interview, as was a time box:
the first is the shape that leaves debug on for a month.

**Condition 3 — Gropius reads no prompt content, and this record grants none.**
The level is the child's, and the content and the format of what it writes are
the model server's. The gateway parses nothing, retains nothing and reads
nothing back. adr-2609061610102325 and its successor adr-2609201008470380 are
untouched by this record, and no entry is added to the readers' list on account
of it. Should a future pin stop writing the bodies at DEBUG, the shape in which
the gateway writes the log is a *new* decision that would supersede the
prompt-content record, not an extension of this one.

**Condition 4 — the panel names it while it is on.** For as long as a model is
armed or running at DEBUG, the control panel says so beside that model, in
plain words that say the file holds the prompts and answers of every client
that reached it — and of Gropius's own probes, which is the next condition.

**Condition 5 — the file is bounded, and the previous run is kept.** The size
bound is ours, since the server has none, and it holds per write as well as per
file: one context-probe body is a single write of the largest size the server
ever sees. A launch renames the previous run's file rather than truncating it,
so the restart that *ends* the diagnostic leaves Alice the evidence it
collected instead of destroying it.

**Condition 6 — a model that keeps no transcript refuses the arm.** Under
itd-2609091715089488 a model carrying the transcript exception refuses a debug
arm, with the reason, on both panels. The exception means no prompts on disk,
not "not in this one file". This record holds that refusal; it does not carve
anything out of it.

**Condition 7 — the client is not told, and the record says so plainly.** Bob
receives no notice that the model he is talking to is running at DEBUG. That is
not an oversight to be closed later: no stateless server can tell him. Gropius
answers requests, not sessions; the disclosure carriers that the transcript
store uses are Gropius's own words about a record Gropius writes, and this
record is written by a child process on the operator's disk. The honest
statements are the ones this record makes to the *operator*: the panel says
what the file holds, the docs say it, and nobody writes a sentence claiming the
client knows.

## Alternatives Considered

1. **A new ADR narrowing the telemetry record, with the client not told
   (chosen).** The diagnostic ships, the exception is written where a future
   reader looks for it, and the one cost is stated rather than hidden. It
   follows the shape adr-2609111126115848 set for exactly this kind of
   amendment.
2. **Require disclosure before any content may be written.** Honest, and it
   would keep the telemetry record's clause intact in spirit. Rejected: there
   is no carrier. Holding the draft until one exists holds it indefinitely,
   because the missing thing is a session the server does not have.
3. **Decline the content: the diagnostic reports metadata only.** No amendment
   needed, and the statistics store already answers most questions. Rejected:
   the failures this draft exists for — a malformed tool block, a system
   message in the wrong place, a client sending what it did not mean to — are
   invisible in metadata, and the person debugging would reach for the flag
   themselves with none of the conditions above around it.
4. **Shape (ii): the gateway composes the diagnostic log itself.** It would
   give Gropius control of format, redaction and bound. Rejected as
   unnecessary at the pin, and expensive in the record: it would make the
   gateway a second reader and a retainer of prompt content, superseding
   adr-2609061610102325 for a diagnostic that the child already performs. It
   is the shape to revisit only if a pin bump takes the DEBUG bodies away.
5. **A switch that survives restarts, or a time box.** Both were put and both
   declined. A persistent switch is how debug stays on for a month; a time box
   adds a clock to a bound that the process lifetime already provides, and the
   lifetime is the thing Alice can observe.

## Consequences

- adr-2609061503319212 records this narrowing in its status fields alongside
  its supersession by adr-2609201008476813, which is the record that restates
  the telemetry posture with this exception and the transcript store's named in
  it. A reader who finds the old record is sent to both.
- itd-2609062346072707's mechanism and scope conditions are admissible as
  drafted, and its spec carries the arming field's separation, the per-write
  size bound, the rename at launch and the refusal on an excepted model as
  criteria rather than intentions.
- The readers' list is unchanged, and that is a load-bearing fact rather than
  an absence: if a diff on this intent adds an entry to it, the shape has
  drifted from (i) to (ii) and needs a different decision.
- **What this costs, stated plainly.** Two things. First, a rule that was
  unqualified now has exceptions, and this is one of two written on the same
  day — two look like a pattern where one looked like a case. The boundary is
  deliberately narrow: one model, one process lifetime, armed by the operator
  in an authenticated panel, gating nothing, with a file that is bounded and
  named. Anything wider is a new decision. Second, and worse: Bob's prompts,
  and any key he put in one, can be on Alice's disk without his knowing. The
  product's answer is that Alice knows, that she armed it on purpose, that the
  run ends at the next restart, and that nothing leaves the Mac — not that Bob
  was told, because he was not.
- The shared-cache install is where this costs most. Under `make install-shared`
  the added exposure is not another local account reading the file; it is that
  network clients' prompts and the models' answers land in the serving
  account's directory tree, under the deliberate `3775` semantics the Makefile
  explains. "Readable by nobody else" is the phrase that failed the sibling
  draft's review and it is not used here.
- Nothing leaves the Mac. The records that own that invariant — the telemetry
  record and its successor, the shipped statistics intents, and the server's own
  log intent — continue to own it; this record makes no promise of its own about
  egress and adds no path for it.
- If a future pin stops writing both bodies at DEBUG, this record does not
  repeal itself: it simply has no mechanism, and the question of whether the
  gateway should write the log instead is asked again, as a new decision.
