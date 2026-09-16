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

<!-- abcd-review: INGESTED receipt=rcp-750863424b6e -->
Fidelity review — receipt rcp-750863424b6e (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:eb908bef96cb2d5fd1d0578539f908a8b949ad34cab5ddf0fd2bf94ba1f19d59 · prompt_hash sha256:aea9537e163e4693114b510cd8258b876a674cefc64a43f5b1d72b730f99cd45
Input attestations: tree:main at acacd07b9fb29a2dfc94a8aa58da22d67e8d8b36, range 6b1f600..acacd07 (git tree c2a07327a90111d5c97947dfd19381231ade212f; PR 52 also carried three other intents, only this intent's criteria were judged; `go test ./internal/selftest ./internal/app ./internal/config ./internal/ui ./internal/archtest` all passing at this tree)@sha256:unknown; request:.abcd/.work.local/reviews/rcp-750863424b6e.request.md@sha256:eb908bef96cb2d5fd1d0578539f908a8b949ad34cab5ddf0fd2bf94ba1f19d59; intent:.abcd/development/intents/shipped/itd-2609100457007827-gropius-tests-its-own-models-while-nobody-is-using-it-alice.md@sha256:aea9537e163e4693114b510cd8258b876a674cefc64a43f5b1d72b730f99cd45;

Acceptance rollup: MET 10 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The switch defaults off and is omitempty; with it off the loop's tick never reaches next()/run() (the only writer of the results file) because the self-test branch is gated on the SelfTest() callback, and the package test asserts no acquisition and no file while the app test asserts a fresh install has no selftest directory and no running loop.
  evidence: internal/config/config.go:636 — "SelfTest bool `json:"self_test,omitempty"`"
  evidence: internal/selftest/selftest.go:392 — "if job == nil && (r.opts.SelfTest == nil || r.opts.SelfTest()) {"
  evidence: internal/selftest/selftest_test.go:214 — "func TestOffLoadsNothingAndWritesNothing(t *testing.T) {"
  evidence: internal/app/selftest_test.go:20 — "if _, err := os.Stat(paths.SelfTest); !os.IsNotExist(err) {"
  evidence: internal/config/selftest_test.go:13 — "func TestTheSelfTestIsOffByDefaultAndSurvivesARoundTrip(t *testing.T) {"
- ac-2 — MET: next() picks the ready model with no result inside the re-test window, run() acquires it through Server.Acquire, which the app adapter answers with Pool.Acquire tagged with its own source, runs standardSet and writes one Run line; the package test sees both models acquired once, given pp512,tg128,tg128x2 with the server's counts, and one line each.
  evidence: internal/selftest/selftest.go:508 — "if at, ok := r.tested[config.FoldRepoID(id)]; ok && now.Sub(at) < r.opts.Retest {"
  evidence: internal/selftest/selftest.go:571 — "up, release, err := r.opts.Server.Acquire(loadCtx, model)"
  evidence: internal/app/selftest.go:36 — "up, release, err := s.a.Pool.Acquire(runtime.WithSource(ctx, selfTestSource), repoID)"
  evidence: internal/selftest/selftest.go:597 — "for _, spec := range standardSet(r.opts.Server.Concurrency()) {"
  evidence: internal/selftest/selftest_test.go:234 — "func TestAnIdleMacGetsEveryModelMeasuredAndLeftAsFound(t *testing.T) {"
- ac-3 — MET_WITH_CONCERNS: run()'s deferred release unloads the model when wasResident is false and leaves it when true, and tests hold both halves — but the same defer also skips the unload when the run yielded, so a model the self-test cold-loaded stays resident after a yield (until the pool's own eviction or idle reaper takes it). The spec sanctions this ('the client's request decides what stays'), the criterion does not carve it out, and both docs pages state the unload unconditionally (self-test.md:55-56, self-test-reference.md:28 for cold_load).
  evidence: internal/selftest/selftest.go:588 — "if wasResident || res.Outcome == OutcomeYielded {"
  evidence: internal/selftest/selftest.go:591 — "if err := r.opts.Server.Unload(model); err != nil {"
  evidence: internal/selftest/selftest_test.go:282 — "func TestAModelThatWasResidentIsLeftResident(t *testing.T) {"
  evidence: internal/selftest/selftest_test.go:244 — "if len(acquired) != 2 || len(unloaded) != 2 {"
  evidence: docs/self-test.md:55 — "When the set is done Gropius writes one line to the results file and, if the"
- ac-4 — MET: A watcher started before Acquire polls the pool every Poll and cancels the run's context when busy() sees an in-flight request on any model beyond the self-test's own holds, a load waiting, or a rise in the pool's new Residency.Refusals counter; the run is written as yielded and the hold released, and three tests bound the yield (mid-request, mid-load, and on a refused client) at under 500 ms.
  evidence: internal/selftest/selftest.go:466 — "func busy(act Activity, h hold) bool {"
  evidence: internal/selftest/selftest.go:674 — "if busy(r.opts.Server.Activity(), h) {"
  evidence: internal/runtime/pool.go:799 — "p.refusals++"
  evidence: internal/selftest/selftest_test.go:301 — "func TestAClientRequestYieldsTheRun(t *testing.T) {"
  evidence: internal/selftest/selftest_test.go:642 — "func TestAClientRequestDuringTheLoadYieldsToo(t *testing.T) {"
  evidence: internal/selftest/selftest_test.go:687 — "func TestAClientRefusedRoomYieldsTheRun(t *testing.T) {"
- ac-5 — MET: next() skips any model for which Server.Fits is false without charging it a day, and the app answers Fits from the residency snapshot plus chargeOf against the pool's budget; the package test sees the no-fit model left unloaded and measured once room appears, and the app test holds Fits for small, vast, unsized and absent models.
  evidence: internal/selftest/selftest.go:513 — "if !r.opts.Server.Fits(id) {"
  evidence: internal/app/selftest.go:87 — "return used+s.a.chargeOf(m, size) <= s.a.Pool.MemoryBudget()"
  evidence: internal/selftest/selftest_test.go:712 — "func TestAModelThatWouldNeedAnEvictionIsLeftForLater(t *testing.T) {"
  evidence: internal/app/selftest_test.go:94 — "func TestFitsAnswersFromTheChargeAndTheBudget(t *testing.T) {"
- ac-6 — MET: heldBy() returns a reason (waiting, downloading, in_flight, recent) before any run starts, and the four-case table test holds a one-hour quiet period against each condition with nothing acquired; the app adapter feeds Waiting, Downloading and per-model InFlight from the pool and the download map.
  evidence: internal/selftest/selftest.go:413 — "func (r *Runner) heldBy(act Activity, now time.Time) string {"
  evidence: internal/selftest/selftest.go:399 — "if held := r.heldBy(act, now); held != "" {"
  evidence: internal/app/selftest.go:43 — "func (s selfTestServer) Activity() selftest.Activity {"
  evidence: internal/selftest/selftest_test.go:339 — "func TestNothingRunsWhileTheMacIsBusy(t *testing.T) {"
- ac-7 — MET: next() returns "" when every ready model has a result younger than Retest (24 h by default), seeded from the file at start; the tail of the idle-Mac test sees no third acquisition once both models have results, and the day-window test measures only the two stale models, stalest first.
  evidence: internal/selftest/selftest.go:165 — "DefaultRetest = 24 * time.Hour"
  evidence: internal/selftest/selftest.go:521 — "if len(due) == 0 {"
  evidence: internal/selftest/selftest_test.go:275 — "if acquired, _, _ := srv.snapshot(); len(acquired) != 2 {"
  evidence: internal/selftest/selftest_test.go:370 — "func TestAResultStandsForADayAndTheStalestModelGoesFirst(t *testing.T) {"
- ac-8 — MET: Run carries model, at, outcome, cold_load, load_ms and tests; Test carries name, prompt_tokens, completion_tokens, first_token_ms, total_ms, tokens_per_sec (plus parallel, prompt_tokens_per_sec, counted); the reader keeps only timing and counts from the stream, a failed run carries a class not the error text, and the test plants an answer and prompt words and asserts none reach the file while every Run field is present.
  evidence: internal/selftest/results.go:61 — "type Run struct {"
  evidence: internal/selftest/results.go:80 — "type Test struct {"
  evidence: internal/selftest/request.go:216 — "if len(ev.Choices) > 0 && ev.Choices[0].Delta.Content != "" {"
  evidence: internal/selftest/selftest_test.go:491 — "func TestAResultCarriesNoPromptAndNoAnswer(t *testing.T) {"
  evidence: internal/selftest/selftest_test.go:449 — "if strings.Contains(string(raw), "simulated") {"
- ac-9 — MET_WITH_CONCERNS: SetEnabled(false) cancels the loop's context, waits for the run to drain, and the run is written as stopped with its hold released (package test), and SetConfig applies the switch live through applyIdleJobs (app test) — but applyIdleJobs calls SetEnabled(c.SelfTest || c.ContextProbe || probe queued), so when the context probe is on, or a 'Measure now' is queued, turning self_test off leaves the loop up and a self-test run in progress completes its set instead of being cancelled; the SelfTest() callback is only consulted at the next tick. No test covers that case, and docs/self-test.md:26-27 promises the cancel unconditionally.
  evidence: internal/selftest/selftest.go:282 — "if !on {"
  evidence: internal/selftest/selftest_test.go:401 — "func TestSwitchingOffStopsARunAndReleasesTheModel(t *testing.T) {"
  evidence: internal/app/contextprobe.go:102 — "a.SelfTest.SetEnabled(c.SelfTest || c.ContextProbe || len(a.Probe.Queued()) > 0)"
  evidence: internal/app/app.go:497 — "a.applyIdleJobs(c)"
  evidence: internal/app/selftest_test.go:15 — "func TestTheSelfTestFollowsItsSwitch(t *testing.T) {"
- ac-10 — MET: write() stats the file before appending and truncates to zero when the next line would pass maxBytes, then writes the line; the bounded-file test writes ten runs under a 300-byte cap and asserts the size never exceeds the cap, the mode is 0600 in a 0700 directory, and the newest run (At 9) is what remains.
  evidence: internal/selftest/results.go:168 — "if st, err := h.Stat(); err == nil && st.Size()+int64(len(line)) > f.maxBytes {"
  evidence: internal/selftest/results.go:25 — "DefaultMaxBytes int64 = 4 << 20"
  evidence: internal/selftest/selftest_test.go:456 — "func TestTheResultsFileIsPrivateAndBounded(t *testing.T) {"
  evidence: internal/selftest/selftest_test.go:484 — "if len(runs) == 0 || runs[len(runs)-1].At != 9 {"
- ac-11 — MET: Settings has a Self-test fieldset with the setSelfTest checkbox, the panel fills it from c.self_test and posts it as self_test — the key Config.SelfTest reads — and the ui sync test holds all three regexes.
  evidence: internal/ui/static/index.html:465 — "< input id="setSelfTest" type="checkbox">"
  evidence: internal/ui/static/app.js:1024 — "$('setSelfTest').checked = !!c.self_test;"
  evidence: internal/ui/static/app.js:1636 — "self_test: $('setSelfTest').checked,"
  evidence: internal/config/config.go:636 — "SelfTest bool `json:"self_test,omitempty"`"
  evidence: internal/ui/selftest_test.go:12 — "func TestThePaneIsWiredToTheSelfTestSwitch(t *testing.T) {"
- ac-12 — MET: docs/self-test.md has 'Switch it on' (panel steps and the config.json key) and 'What it costs, and what it never does'; docs/self-test-reference.md names the file's location and every Run and Test field in tables, and an architecture test holds the reference page to RunFields()+TestFields(), the test names, the outcomes, the file name and the cap, while a second holds the how-to's never-records wording and the README link.
  evidence: docs/self-test.md:17 — "## Switch it on"
  evidence: docs/self-test.md:60 — "## What it costs, and what it never does"
  evidence: docs/self-test-reference.md:7 — "`selftest/results.jsonl` under this account's Gropius data folder: the"
  evidence: docs/self-test-reference.md:21 — "| Field | Type | Meaning |"
  evidence: internal/archtest/selftest_docs_test.go:16 — "func TestTheSelfTestReferenceNamesEveryField(t *testing.T) {"
  evidence: internal/archtest/selftest_docs_test.go:42 — "func TestTheSelfTestPagesAreLinkedAndSayWhatIsNeverWritten(t *testing.T) {"

Gap audit:
- honoured:
  - One switch on three surfaces: `self_test` in config.json, a Settings checkbox posted under that key, and the Go loop applied live at save and at construction.
    evidence: internal/config/config.go:636 — "SelfTest bool `json:"self_test,omitempty"`"
    evidence: internal/ui/static/app.js:1636 — "self_test: $('setSelfTest').checked,"
    evidence: internal/app/app.go:334 — "SelfTest: func() bool { return a.Config().SelfTest },"
    evidence: internal/app/selftest_test.go:52 — "func TestASavedSwitchStartsTheSelfTestAtConstruction(t *testing.T) {"
  - Loads go through the pool's ordinary Acquire, never a second launcher, and the self-test never evicts: a model that does not fit beside what is resident is left for a later tick.
    evidence: internal/app/selftest.go:36 — "up, release, err := s.a.Pool.Acquire(runtime.WithSource(ctx, selfTestSource), repoID)"
    evidence: internal/app/selftest.go:68 — "func (s selfTestServer) Fits(repoID string) bool {"
    evidence: internal/selftest/selftest_test.go:712 — "func TestAModelThatWouldNeedAnEvictionIsLeftForLater(t *testing.T) {"
  - A real request is never made to wait: the run is abandoned the moment a client appears, during the load as much as during a test, and on a client refused room; the run is recorded as yielded.
    evidence: internal/selftest/selftest.go:552 — "w := r.startWatch(ctx, model, true)"
    evidence: internal/runtime/pool.go:1888 — "Refusals uint64"
    evidence: internal/selftest/selftest_test.go:319 — "if took := time.Since(arrived); took > 500*time.Millisecond {"
    evidence: internal/runtime/selftest_refusals_test.go:11 — "func TestResidencyCountsTheRefusalsAnEvictionCouldHaveCured(t *testing.T) {"
  - The same short set for every model, named after llama-bench (pp512, tg128, tg128xN at the decode concurrency) plus the load time, with the parallel test holding its places in the pool.
    evidence: internal/selftest/request.go:57 — "func standardSet(concurrency int) []testSpec {"
    evidence: internal/selftest/selftest_test.go:570 — "func TestTheStandardSetIsFixedAndNamedAfterLlamaBench(t *testing.T) {"
    evidence: internal/selftest/selftest_test.go:732 — "func TestTheParallelTestHoldsItsPlacesInThePool(t *testing.T) {"
  - One line of figures per run in a bounded, private JSON Lines file under the account's data directory, with a planted link, FIFO or open directory refused, and the cycle surviving a restart by seeding from the file.
    evidence: internal/config/config.go:239 — "SelfTest: filepath.Join(acct, "selftest"),"
    evidence: internal/selftest/results.go:132 — "const filePerm os.FileMode = 0o600"
    evidence: internal/selftest/selftest_test.go:755 — "func TestAPlantedResultsLocationIsRefused(t *testing.T) {"
    evidence: internal/selftest/selftest.go:334 — "func (r *Runner) seedLocked() {"
  - Nothing in the file is a prompt or an answer and nothing leaves the Mac: the prompts are two constants, the reader keeps only timing and counts, the client follows no redirect, and request.go is allow-listed on both sides of the archtest content boundary.
    evidence: internal/selftest/request.go:36 — "const generationPrompt = "Write a long story about a lighthouse keeper who finds a message in a bottle.""
    evidence: internal/selftest/selftest.go:256 — "CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },"
    evidence: internal/archtest/prompt_content_test.go:31 — ""internal/selftest/request.go": "builds the self-test's own requests from two constants in that file; reads nothing from a client (itd-2609100457007827)","
    evidence: internal/selftest/selftest_test.go:491 — "func TestAResultCarriesNoPromptAndNoAnswer(t *testing.T) {"
  - A how-to and a reference page, linked from the README, with the reference held to the record's fields by an architecture test.
    evidence: README.md:82 — "once ([how to switch it on] (docs/self-test.md), and"
    evidence: internal/archtest/selftest_docs_test.go:18 — "for _, field := range append(selftest.RunFields(), selftest.TestFields()...) {"
- diverged:
  - Press release and ac-3: a model the self-test loaded is unloaded when the run ends. Delivered: unloaded unless the run yielded — a cold-loaded model stays resident after a yield and the client's request (or the pool's eviction/idle reaper) decides what happens to it. Sanctioned in the spec's Approach, not in the intent, and both docs pages state the unload without the exception.
    evidence: internal/selftest/selftest.go:588 — "if wasResident || res.Outcome == OutcomeYielded {"
    evidence: docs/self-test-reference.md:28 — "| `cold_load` | boolean | `true` when the model was not in memory before the run, so `load_ms` is a load from disk and the run unloaded the model afterwards. |"
  - ac-9 and docs/self-test.md: switching off stops a run in progress at once. Delivered: the loop is shared with the context probe, and applyIdleJobs keeps it up whenever the probe is on or a probe is queued, so in that state a save that clears self_test does not cancel a self-test run in progress; the switch is only re-read at the next tick.
    evidence: internal/app/contextprobe.go:102 — "a.SelfTest.SetEnabled(c.SelfTest || c.ContextProbe || len(a.Probe.Queued()) > 0)"
    evidence: internal/selftest/selftest.go:392 — "if job == nil && (r.opts.SelfTest == nil || r.opts.SelfTest()) {"
    evidence: docs/self-test.md:26 — "Clear the box and save to switch it off again. A run in progress is stopped"
  - The intent resolved 'no panel view of the results in this intent; the file is the surface'. Delivered beyond the record: GET /api/selftest and a block at the head of the Statistics tab (recorded in DECISIONS.md on 2026-09-11 against iss-2609100515475279). Its `enabled` flag reads Runner.Enabled(), which is the shared idle loop and not the self_test switch, so with the probe on and self_test off the tab says the self-test is 'On'.
    evidence: internal/gateway/control_selftest.go:55 — "writeJSON(w, http.StatusOK, selfTestView(c.App.SelfTest.Enabled(), runs))"
    evidence: internal/selftest/selftest.go:299 — "func (r *Runner) Enabled() bool {"
    evidence: internal/ui/static/app.js:1740 — "? (latest.length ? 'On. Gropius measures a model whenever this Mac has been idle for five minutes.'"
  - Press release and scope condition: the quiet period is 'a few minutes' / five minutes. Delivered: a setting, idle_threshold_sec (60–3600 s, default 300), shared with the context probe and applied live through SetQuiet; the Settings hint and the Statistics-tab hint still say 'five minutes' unconditionally.
    evidence: internal/config/config.go:652 — "IdleThresholdSec int `json:"idle_threshold_sec,omitempty"`"
    evidence: internal/app/contextprobe.go:100 — "a.SelfTest.SetQuiet(c.EffectiveIdleThreshold())"
    evidence: internal/ui/static/index.html:469 — "for a model for five minutes and nothing is downloading, Gropius loads one of your"
  - Press release: 'the moment a real request arrives the run in progress is abandoned ... and the model is his'. Delivered for a request on a resident model; a request whose model needs the memory the run holds is refused once (503) and served on its retry, the run ending at that refusal — a pool-side preemptible hold is a recorded follow-up.
    evidence: docs/self-test.md:74 — "whose model would need the memory the run is holding is refused once, the"
    evidence: internal/runtime/pool.go:799 — "p.refusals++"
- missing:
  - A test that the switch going off cancels a self-test run and releases the model while the context probe is also on (the state applyIdleJobs leaves the loop running in). The package test drops the whole loop and the app test switches off with the probe off, so the promise in docs/self-test.md:26-27 is unarmed for that state.
    evidence: internal/app/contextprobe.go:102 — "a.SelfTest.SetEnabled(c.SelfTest || c.ContextProbe || len(a.Probe.Queued()) > 0)"
    evidence: internal/app/selftest_test.go:32 — "if err := a.SetConfig(config.Default()); err != nil {"

Scope-condition dispositions:
- cond-2609100502039780 — survived: The switch is off by default and omitempty, a configuration written before the field loads with it off, and with it off the loop's self-test branch is never taken and the results directory is never created — off is the state before this change.
  evidence: internal/config/config.go:636 — "SelfTest bool `json:"self_test,omitempty"`"
  evidence: internal/config/selftest_test.go:13 — "func TestTheSelfTestIsOffByDefaultAndSurvivesARoundTrip(t *testing.T) {"
  evidence: internal/app/selftest_test.go:20 — "if _, err := os.Stat(paths.SelfTest); !os.IsNotExist(err) {"
- cond-2609100502036756 — survived: heldBy() is exactly the condition's four clauses — nothing in flight, nobody waiting, no download, last use older than the quiet period — with the refinement that the self-test's own release is not a request; the quiet period became the idle_threshold_sec setting (default five minutes) rather than a constant, which is still 'a quiet period'.
  evidence: internal/selftest/selftest.go:413 — "func (r *Runner) heldBy(act Activity, now time.Time) string {"
  evidence: internal/selftest/selftest.go:429 — "if own, ok := r.touched[config.FoldRepoID(m.RepoID)]; ok && !m.LastUsed.After(own) {"
  evidence: internal/config/config.go:1323 — "DefaultIdleThresholdSec = 300"
  evidence: internal/selftest/selftest_test.go:339 — "func TestNothingRunsWhileTheMacIsBusy(t *testing.T) {"
- cond-2609100502036640 — survived: The adapter's Acquire is Pool.Acquire with a source tag and no launcher of its own, and next() refuses any model Fits says would need room made, Fits being the app's charge arithmetic against the pool's budget; Fits is a snapshot and the pool decides at the load, but the watcher runs from before Acquire so a client loading in between yields the run rather than letting it evict.
  evidence: internal/app/selftest.go:36 — "up, release, err := s.a.Pool.Acquire(runtime.WithSource(ctx, selfTestSource), repoID)"
  evidence: internal/app/selftest.go:87 — "return used+s.a.chargeOf(m, size) <= s.a.Pool.MemoryBudget()"
  evidence: internal/selftest/selftest.go:513 — "if !r.opts.Server.Fits(id) {"
  evidence: internal/app/selftest_test.go:94 — "func TestFitsAnswersFromTheChargeAndTheBudget(t *testing.T) {"
- cond-2609100502034205 — narrowed: Yielding is the self-test cancelling its own request or load on a 250 ms poll and the pool gained no preemption, as assumed; but a client whose model needs the memory the run holds does not wait for that cancel — with eviction grace off it is refused once, the refusal count is what yields the run, and the client is served on its retry.
  narrowing: Holds for a request on a model that is already resident, which waits in the pool for the self-test's cancel (bounded by the poll interval); a request whose model would need the run's memory is refused once with a 503 rather than waiting, and is served on its retry.
  evidence: internal/selftest/selftest.go:19 — "// A real request always wins. The pool has no preemption, so the self-test"
  evidence: internal/runtime/pool.go:1888 — "Refusals uint64"
  evidence: internal/selftest/selftest_test.go:687 — "func TestAClientRefusedRoomYieldsTheRun(t *testing.T) {"
  evidence: docs/self-test.md:74 — "whose model would need the memory the run is holding is refused once, the"
- cond-2609100502035912 — survived: The two prompts are constants in request.go, the stream reader keeps only whether a chunk carried text and the usage counts, a failed run carries a class rather than the error's text, the file is asserted free of prompt words and a planted answer, and archtest allow-lists request.go as the package's only reader and writer of a conversation.
  evidence: internal/selftest/request.go:42 — "const promptSentence = "The lighthouse keeper climbed the spiral stairs every evening to light the lamp before the fishing boats came home. ""
  evidence: internal/selftest/request.go:216 — "if len(ev.Choices) > 0 && ev.Choices[0].Delta.Content != "" {"
  evidence: internal/selftest/selftest_test.go:501 — "for _, text := range []string{"SECRET", "lighthouse", "keeper"} {"
  evidence: internal/archtest/prompt_content_test.go:62 — ""internal/selftest/request.go": "times the first chunk of the self-test's own answer and counts the chunks; keeps nothing of what they say (itd-2609100457007827)","
- cond-2609100502031008 — survived: Paths.SelfTest is derived from accountDir(root), the same per-account directory the config and registry use, the file is created 0600 in a 0700 directory the writer refuses if owned or writable by another account, and Ready() lists the shared registry's models; no test runs as a second account, so the shared-install half rests on the path derivation and the docs rather than on an exercised scenario.
  evidence: internal/config/config.go:239 — "SelfTest: filepath.Join(acct, "selftest"),"
  evidence: internal/selftest/results.go:233 — "if sys, ok := st.Sys().(*syscall.Stat_t); ok && int(sys.Uid) != os.Getuid() {"
  evidence: internal/selftest/selftest_test.go:467 — "if st.Mode().Perm() != 0o600 {"
  evidence: docs/self-test-reference.md:9 — "account's own Application Support directory under a shared install, beside"
## Grounds

- pursued: we expect a per-model figure measured on the operator's own Mac, taken automatically, to replace the one-evening hand-run campaign as the source the settings are sized from; shown wrong if after a cycle the figures separate no setting choice, or if a serving Mac is never idle long enough for a cycle to complete. Planning delegated to the agent by the maintainer on 2026-09-10 as an experiment.
