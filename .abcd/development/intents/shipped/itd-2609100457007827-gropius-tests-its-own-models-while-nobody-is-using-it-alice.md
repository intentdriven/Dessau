---
id: itd-2609100457007827
slug: gropius-tests-its-own-models-while-nobody-is-using-it-alice
spec_id: spc-2609100502029941
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091301112705, itd-2609061521082551]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Gropius tests its own models while nobody is using it. Alice switches on the self-test in Settings; whenever her Mac is idle, Gropius loads each downloaded model in turn, runs the same standard set of tests against it, unloads it, and records what it measured, so that the numbers she sizes a model's settings from come from her own Mac rather than from someone else's benchmark. The runs never delay a real request and nothing leaves the machine.

## Press Release

Gropius now measures its own models, on your Mac, while nobody is using it.
Alice ticks **Self-test when idle** in Settings and saves. That evening,
once the last request is a few minutes old and nothing is downloading,
Gropius loads the first of her models, runs the same short set of tests every
model gets — how long a load takes, how fast a long prompt is read, how fast
tokens come out, and what happens when four requests arrive at once — writes
one line of figures to a file in her Gropius folder, and unloads the model if
it was not resident before. Then the next model, and the next. In the morning
the file holds one line per model, measured on her machine, and the number she
sizes the served window and the concurrency from is hers rather than a
stranger's. Bob's agent sending a request at midnight is never made to wait:
the moment a real request arrives the run in progress is abandoned, marked as
yielded, and the model is his. Nothing in the file is a prompt or an answer,
and nothing in it leaves the Mac.

## Why This Matters

The 2026-09-06 model-bench campaign produced the numbers the memory budget and
the served window are now designed from — and it was one person running a
script by hand for an evening on one Mac, a record that goes stale with the
next model, the next runtime or the next machine. The context-probe draft
(itd-2609091301112705) automates one of those measurements; this is the
harness that draft is one test of, and it refines it: the idle detection, the
yielding to a real request and the rule against evicting a pinned model are
the same costs, carried once. The usage dashboard (itd-2609091712141073)
measures what real requests did; this measures what a model can do when asked
the same thing as every other model, which is the comparison usage traffic
never gives. The held draft itd-2609091712142715, settings that fit by
themselves, is what these figures exist to feed; it is not part of this intent.

## Mechanism

We expect one line of figures per model per day to be enough to compare models
and size their settings, because the four measurements are the ones the
2026-09-06 campaign actually used to decide the budget and the window, and
llama-bench's prompt-processing and text-generation pair has been the shared
vocabulary for local inference numbers since 2023. Shown wrong if a setting the
figures were meant to decide turns out to need a measurement not in the set, or
if idle time on a serving Mac is too rare for a cycle to complete.

## Scope Conditions

- Apple Silicon macOS, one Mac, with the self-test switch on; off is the default and the same state as today. <!-- cond: cond-2609100502039780 -->
- Idle means: no request in flight on any model, nobody waiting for a load, no download running, and the last request older than a quiet period. <!-- cond: cond-2609100502036756 -->
- The self-test loads through the pool's ordinary Acquire path and never a second launcher, so the budget, the pins and the served window are the pool's own; and it never evicts, loading a model only when it fits beside what is resident. <!-- cond: cond-2609100502036640 -->
- The pool has no preemption: yielding is the self-test cancelling its own request, not the pool taking the model away, so a real request can wait for one in-flight self-test request to be cancelled. <!-- cond: cond-2609100502034205 -->
- The results file holds figures only, never a prompt or an answer; the test prompts are constants in the code, and no client's request is ever part of a run. <!-- cond: cond-2609100502035912 -->
- Shared-cache mode records under the serving account's own data directory, at 0600; the models measured are every account's, since they are one set. <!-- cond: cond-2609100502031008 -->

## Acceptance Criteria

- Given the switch is off, when Gropius runs for any length of time, then no model is loaded by the self-test and no results file exists.
- Given the switch is on and the Mac is idle, when a ready model has not been tested within the re-test window, then it is acquired through the pool, the standard set runs against it, and one line is appended to the results file.
- Given a model was not resident before its run, when the run ends, then the model is unloaded; given it was resident, then it is left where it was.
- Given a run is in progress, when a request from a client arrives on any model, or a client's load is refused for want of room, then the self-test's request or load is cancelled, the run is recorded as yielded, and the model is released.
- Given a ready model that would need another evicted to load, when the tick fires, then it is not loaded, and it is measured once there is room.
- Given a request is in flight, a load is waiting, or a download is running, when the tick fires, then nothing is loaded and nothing runs.
- Given every ready model has been tested within the re-test window, when the tick fires, then nothing runs.
- Given a results line, when it is read back, then it carries the model, when, the outcome, whether the load was cold, the load time, and for each test its name, token counts, time to first token, total time and tokens per second, and no field carries prompt or answer text.
- Given the switch is turned off while a run is in progress, when the save applies, then the run is cancelled and the model released.
- Given the results file reaches its size cap, when the next line is written, then the file is started again and the newest line is kept.
- Given the switch, when Settings is rendered, then the panel offers it and posts it under the key config.json reads.
- Given the docs, when the feature ships, then a how-to page says how to switch it on and what it costs, and a reference page names every field of a results line and where the file is.

## Open Questions

- Resolved 2026-09-10 at planning (delegated to the agent by the maintainer): the standard set is load, pp512, tg128 and tg128 at the decode concurrency; the names follow llama-bench so the figures can be compared with published ones.
- Resolved 2026-09-10 at planning: results go to a single bounded JSON Lines file under the account's data directory rather than into the statistics store, because the store's schema is per-request and its writer is in another lane; folding it in is a follow-up.
- Resolved 2026-09-10 at planning: no panel view of the results in this intent; the file is the surface, the panel has the switch. A results view is a follow-up.
- Deferred: the re-test cadence is fixed at a day; whether it should be a setting is decided when someone wants a different one.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-750863424b6e -->
Fidelity review OWED (receipt rcp-750863424b6e).

## Grounds

- pursued: we expect a per-model figure measured on the operator's own Mac, taken automatically, to replace the one-evening hand-run campaign as the source the settings are sized from; shown wrong if after a cycle the figures separate no setting choice, or if a serving Mac is never idle long enough for a cycle to complete. Planning delegated to the agent by the maintainer on 2026-09-10 as an experiment.
