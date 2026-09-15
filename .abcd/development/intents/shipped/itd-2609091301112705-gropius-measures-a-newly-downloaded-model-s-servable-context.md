---
id: itd-2609091301112705
slug: gropius-measures-a-newly-downloaded-model-s-servable-context
spec_id: spc-2609112108142027
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609061431481936, itd-2609061431463108]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Gropius measures a model's real context window

## Press Release

Gropius now tells you how long a prompt each model on your Mac can actually
take. When a download finishes, Gropius measures the model rather than
believing its configuration: it sends prompts of growing length, bisects
between the last one that came back and the first that did not, and records
the window it verified beside the one the model declares.

Alice downloads a 262,144-token model in the evening. In the morning the
models list tells her that her Mac serves it to about 91,000 tokens — the
declared window was never reachable here — so her editor's long-file prompts
are sized to what works instead of failing halfway through the afternoon. The
memory budget charges the window the machine can serve rather than the one the
file advertises, and the figure comes from her own Mac rather than from someone
else's benchmark.

The measurement is not free, and the design has to carry that. Probing one
model took about forty minutes of GPU time at low load in the 2026-09-06
campaign, with an unload and a reload between steps so a retained prompt cache
cannot flatter the next reading, and it needs the machine not to be serving
anyone: a probe that runs while clients are asking for models measures the
queue rather than the model. So this runs when the Mac is idle, is
interruptible, and yields the machine to a real request the moment one arrives.

## Why This Matters

The declared window is a claim about the architecture, not about this Mac. The
2026-09-06 campaign found three of four models bounded by the gateway rather
than by the model, and one that reached only a third of its declared cap here;
that record exists because someone ran a script by hand for an evening, and it
goes stale the moment a model, a runtime or the machine changes. A window
nobody has measured is a number a client sizes its prompts from and then loses
an afternoon to.

## Mechanism

We expect an automated sweep-and-bisect to recover the same servable window a
person recovered by hand, because the campaign's method is mechanical — unique
prompts generated from a seed so no two share a prefix and no prefix is cached,
one output token at temperature 0, an unload and a reload before each long step,
the window read from the gateway's own `usage.prompt_tokens` — and every input
it needs is already in the process: the registry knows the declared cap, the
pool knows what is resident and what is idle, and the gateway already derives a
prefill deadline from prompt size. We are wrong if the figure is not stable
enough to publish — a repeat probe on the same Mac, the same models and the same
runtime returning a materially different window — or if what the probe finds is
the gateway's bound rather than the model's, in which case the number belongs to
Gropius's configuration and moves whenever `upstream_header_timeout_sec` does.

The second failure is not hypothetical: the 2026-09-06 campaign found three of
four models bounded by the gateway rather than by the model. So a reading that a
gateway bound stopped is published as a floor under the model's window and names
the bound that stopped it, never as the model's limit.

## Scope Conditions

- Apple Silicon Macs with unified memory; the probe's memory guard is arithmetic over one shared pool, not over a discrete GPU's. <!-- cond: cond-2609112108142038 -->
- The pinned runtime, mlx-lm 0.31.3 today. The prefill rate, the retained prompt cache and the per-token cost are properties of the runtime as much as of the model, so a figure measured under one runtime is not a figure under another. <!-- cond: cond-2609112108146425 -->
- One serving process per Mac. The singleton election means the probe competes with every account's requests, not only with its own. <!-- cond: cond-2609112108146321 -->
- Models the registry holds as ready, with a declared context length read from their own `config.json`. A model that declares no window has nothing to bisect between. <!-- cond: cond-2609112108149020 -->
- Text models. The campaign's recall check assumes a prompt of text filler. <!-- cond: cond-2609112108143953 -->
- A Mac that stays awake for the duration. A probe does not survive sleep. <!-- cond: cond-2609112108142632 -->
- The probe reaches the model the way a client does, over this Mac's own OpenAI endpoint, so what it measures is the window a client can actually get. A step that the gateway's prefill deadline or its served-window check stopped bounds Gropius's configuration, not the model, and is recorded as such. <!-- cond: cond-2609112108146949 -->

## Acceptance Criteria

**Idle detection and yielding**

- Given the probe switch is off, when a download finishes, then nothing is measured and nothing is queued: a probe begins only from the global switch, which is off by default, or from a per-model "Measure now".
- Given a probe queued, when any model has served a request inside the idle threshold, or the pool reports a waiting caller, or anything is in flight, then the probe does not start and the panel says which of those held it back.
- Given a probe running, when a real request arrives for any model, then the probe cancels its own in-flight request within the eviction grace, asks the pool to unload the model it was driving so the abandoned prefill neither keeps the GPU nor leaves its cache resident, and records the step as yielded rather than as a failure.
- Given a probe whose unload is refused because a request is now in flight on that model, then the reading that step would have produced is discarded rather than kept, because a server that kept an abandoned cache measures the cache and not the model.
- Given a probe that has yielded, when the Mac is idle again, then it resumes from the bisection bounds already established, not from the beginning.

**Interruptibility and safety**

- Given a probe running, when the operator presses Stop, or Gropius is quit, or the machine sleeps, then no partial figure is written, the model is left unloaded, and the next start reports the probe incomplete rather than silently retrying it.
- Given a step whose projected peak footprint comes within the campaign's margin of this Mac's available memory, then the step is skipped and recorded as skipped, together with the projection that skipped it.
- Given a probe between steps, when the next step begins, then the model server has been unloaded and started again and the step's prompt carries its own nonce, so no retained prompt cache flatters the reading.
- Given a pinned model, when the probe needs room, then nothing pinned is evicted for it: the probe acquires only when the room is already free, and it waits in the queue every other caller waits in and is counted by the pool's waiting count, so it cannot hide from the queue caps.

**The result, published beside the declared and the served window, and charged only when adopted**

- Given a completed probe, when a client reads `GET /v1/models`, then the entry carries the measured window beside `context_length` and `served_context`, and the models-list reference says what each of the three means.
- Given a completed probe, when the operator adopts the measured figure, then that figure becomes the model's served window, the next load is charged at it, and the panel shows the figure it is charged at; until it is adopted the measurement changes no charge and refuses no request.
- Given a probe whose largest step was stopped by the gateway's prefill deadline or by its served-window check rather than by the model, then the result names the bound that stopped it, the figure is published as a floor, and the panel and the docs say plainly that the model's own window is not known.
- Given the global switch and the per-model "Measure now", when the Go settings, `config.json` and the control panel are compared, then all three carry them and the progress and the result, and a save that names neither leaves both alone.

**Invalidation**

- Given a stored measurement, when the runtime version changes, the model is re-downloaded, or the memory budget, the decode concurrency or the served window changes, then the measurement is marked stale, is neither charged nor published as current, and the record says which of those changed.
- Given a stored measurement, when Gropius starts, then the provenance stored with it — runtime version, budget, decode concurrency, served window, and what stopped the largest step — is compared with what is in force now, so staleness is a stored fact rather than an assumption.

## Open Questions

Settled on 2026-09-10 at a planning interview delegated to the agent on the
maintainer's instruction "implement fully autonomously", adopting the
recommendations of the planning brief prepared on 2026-09-09: the probe is a
global switch defaulting off with a per-model "Measure now" (1); it runs at the
first sustained idle and never at download time (2); a result is never reused
between machines (3); it never evicts a pinned model, acquires only when the
room is already free, and is counted by the pool's waiting count like any other
caller (4); the gateway's prefill timeout is not relaxed for it, and the probe
drives the same OpenAI path a client drives, so its largest step is bounded by
the served window and by the gateway's own limits and a reading so bounded is
published as a floor (5).

Clarification (2026-09-10, from the same interview; the press release stands as
written and this says what was found true of it): the press release's opening —
"when a download finishes, Gropius measures the model" — is delivered as the
morning after rather than the moment after. A download finishing arms the
measurement; the first sustained idle window runs it, and only when the operator
has switched the probe on or asked for this model by name. Alice's story is
unchanged: she downloads in the evening and reads the figure in the morning.

What that choice cannot measure, stated plainly: the model's own window above
whatever the gateway stops first. On the 2026-09-06 evidence that is three
models of four, so for most models this intent publishes a verified floor and a
named bound rather than the architecture's limit.

- **Reuse between accounts on one Mac — deferred, not declined.** Each account keeps its own `registry.json`, so Carol pays another forty minutes of GPU for a measurement Alice already made on the same machine and the same runtime. A reading keyed on the machine and the runtime version and stored in the shared root would avoid that, and it is a new writeable file in a group-writable directory: a trust-boundary change needing its own adversarial security review, which is not on this intent's critical path.
- **How the model's own window is ever learned.** A coordinated pool-side path exempt from the gateway's bounds, a raised `upstream_header_timeout_sec` for the duration, or the manual campaign script staying the only answer. Nothing here decides it.
- **Whether the probe also samples memory**, which would keep `capability`'s safety factors of 5 and 7 under continuous evidence rather than one evening's. Out of scope here — that package belongs to another lane — and worth a follow-on.
- **The idle threshold's value**, and how many models one idle window may take: six models at forty minutes each is an evening, and the models load one at a time.
- **Arbitration with the model self-test** — resolved 2026-09-12 at implementation: the self-test landed first and its loop is the one idle primitive; the probe is a `selftest.Job` it schedules, so idleness, yielding, the never-evict check, the held-back reason and the idle threshold (`idle_threshold_sec`) are decided once, in `internal/selftest`, for both.
- **Whether the yielding seam becomes real preemption in the pool.** The pool has no preemption and no priority today; this intent reads the pool and cancels its own request instead. Any change to the pool itself is a coordinated follow-up with the lane that owns `internal/runtime`.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4431dce599cc -->
Fidelity review — receipt rcp-4431dce599cc (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:aa4fe267783f640add3e6b427ce1f835f9476a56a3c81707492e0a7ef56022f8 · prompt_hash sha256:e406c4ec363428971b4b363a1f9d7f58d69b2a3b6b271893f42c6dd51b51a369
Input attestations: tree:main at acacd07b9fb29a2dfc94a8aa58da22d67e8d8b36, range 6b1f600..acacd07 (PR 52; go test ./internal/contextprobe ./internal/registry ./internal/gateway ./internal/selftest ./internal/app ./internal/config ./internal/runtime ./internal/ui ./internal/archtest all ok)@sha256:unknown; request:.abcd/.work.local/reviews/rcp-4431dce599cc.request.md@sha256:aa4fe267783f640add3e6b427ce1f835f9476a56a3c81707492e0a7ef56022f8; intent:.abcd/development/intents/shipped/itd-2609091301112705-gropius-measures-a-newly-downloaded-model-s-servable-context.md@sha256:e406c4ec363428971b4b363a1f9d7f58d69b2a3b6b271893f42c6dd51b51a369;

Acceptance rollup: MET 12 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Due returns nothing while Enabled is false and only a MeasureNow entry in the queue is offered ahead of the switch; config.Default sets no context_probe, so the switch is off on a fresh install; the app test asserts a ready model is not due and the idle loop is not running until Measure now is pressed.
  evidence: internal/contextprobe/probe.go:240 — "if p.opts.Enabled == nil || !p.opts.Enabled() {"
  evidence: internal/contextprobe/probe.go:201 — "func (p *Probe) MeasureNow(repoID string) {"
  evidence: internal/config/config.go:646 — "ContextProbe bool `json:"context_probe,omitempty"`"
  evidence: internal/app/contextprobe_test.go:26 — "func TestADownloadDoesNotStartAProbe(t *testing.T) {"
  evidence: internal/contextprobe/probe_test.go:236 — "func TestNothingIsMeasuredWhileTheSwitchIsOff(t *testing.T) {"
- ac-2 — MET: The loop asks heldBy before starting a due job and stores the reason (in_flight, waiting, downloading, recent) with the model that was due; the panel's card renders it as 'Measurement waiting: ...'; the probe test drives each condition and asserts no request is sent and the reason is named.
  evidence: internal/selftest/selftest.go:399 — "if held := r.heldBy(act, now); held != "" {"
  evidence: internal/selftest/selftest.go:413 — "func (r *Runner) heldBy(act Activity, now time.Time) string {"
  evidence: internal/ui/static/app.js:388 — "Measurement waiting: ${held}"
  evidence: internal/contextprobe/probe_test.go:478 — "func TestAProbeWaitsForIdleAndNamesWhatHeldItBack(t *testing.T) {"
- ac-3 — MET: The loop's watcher polls the pool every 250 ms (well inside the 120 s default eviction grace) and cancels the probe's request context the moment busy() sees a foreign in-flight request, a waiter or a refusal; on a yield the probe calls unloadWaiting on the model it was driving and files the step as stepYielded, keeping its bounds and logging 'paused' rather than a failure; verified against the fake and through the real gateway and pool.
  evidence: internal/selftest/selftest.go:674 — "if busy(r.opts.Server.Activity(), h) {"
  evidence: internal/selftest/selftest.go:167 — "DefaultPoll = 250 * time.Millisecond"
  evidence: internal/config/config.go:1141 — "DefaultEvictionGraceSec = 120"
  evidence: internal/contextprobe/probe.go:511 — "p.unloadWaiting(context.Background(), c.RepoID)"
  evidence: internal/contextprobe/probe.go:307 — "p.opts.Log.Info("context probe paused""
  evidence: internal/contextprobe/probe_test.go:368 — "func TestAProbeYieldsToARealRequestAndResumesFromItsBounds(t *testing.T) {"
  evidence: internal/gateway/probe_integration_test.go:154 — "func TestAClientRequestThroughTheRealGatewayYieldsTheProbeAndItResumes(t *testing.T) {"
- ac-4 — MET: unloadWaiting returns false when ErrBusy outlasts UnloadWait (a client has the model); on the yield path the step returns an empty stepResult so no bound advances, and an unload refused before a step is itself treated as a yield with no reading; the test plants ErrBusy plus a client in flight and asserts no figure and no advanced bounds.
  evidence: internal/contextprobe/probe.go:440 — "the model stayed busy; a client has it"
  evidence: internal/contextprobe/probe.go:513 — "return out, stepResult{}"
  evidence: internal/contextprobe/probe.go:472 — "if !p.unloadWaiting(s.Ctx, c.RepoID) {"
  evidence: internal/contextprobe/probe_test.go:419 — "func TestAReadingIsDiscardedWhenTheServerCouldNotBeStopped(t *testing.T) {"
- ac-5 — MET: Bounds are kept per model across runs; Run reuses the stored bounds and the sweep resumes from lo (or bisects if already swept); the yield test asserts every post-resume prompt is above what had been verified, and the real-stack test asserts the probe completes after a client's request.
  evidence: internal/contextprobe/probe.go:136 — "bounds map[string]*bounds"
  evidence: internal/contextprobe/probe.go:288 — "b, ok := p.bounds[key]"
  evidence: internal/contextprobe/probe.go:351 — "for !b.swept {"
  evidence: internal/contextprobe/probe_test.go:412 — "after resuming the probe sent a %d-character prompt, below what it had verified"
  evidence: internal/gateway/probe_integration_test.go:185 — "// The probe resumes and finishes."
- ac-6 — MET_WITH_CONCERNS: Stop (switch off) and quit are realised: SetEnabled(false)/Close cancel the run and wait for it, the stepStopped path unloads the model and marks it incomplete without saving, Due skips an incomplete model, and the card says 'Measurement incomplete'. Concern: the sleep trigger is not delivered — no sleep/wake handling exists anywhere in the tree, the how-to only says a sleeping Mac loses the run, and after a wake the run simply continues; a step that then outlasts its timer is filed as a prefill_deadline floor rather than the probe being reported incomplete.
  evidence: internal/selftest/selftest.go:274 — "func (r *Runner) SetEnabled(on bool) {"
  evidence: internal/contextprobe/probe.go:311 — "_ = p.opts.Sources.MarkIncomplete(model, true)"
  evidence: internal/contextprobe/probe.go:245 — "if !ok || c.Declared <= 0 || c.Incomplete {"
  evidence: internal/app/app.go:1727 — "a.SelfTest.Close()"
  evidence: internal/contextprobe/probe_test.go:449 — "func TestAnInterruptedProbeWritesNoFigure(t *testing.T) {"
  evidence: internal/app/contextprobe_test.go:53 — "func TestTheProbeSwitchStartsTheIdleLoopAndQuitStopsIt(t *testing.T) {"
  evidence: internal/ui/static/app.js:396 — "Measurement incomplete: the last probe was interrupted"
  evidence: docs/context-probe.md:34 — "Mac that sleeps loses the run."
  evidence: internal/contextprobe/probe.go:496 — "if errors.Is(err, context.DeadlineExceeded) && s.Ctx.Err() == nil {"
- ac-7 — MET: Before each request the step projects flat charge plus KV charge for the size against Available() less MemoryMargin and returns stepBounded with BoundMemoryGuard and the projection; the projection is persisted as guard_bytes when the guard is what bounded the run, and the test asserts the bound and a non-zero projection. (A guard skip later superseded by a smaller model/deadline bound leaves no persisted trace, which is consistent since it no longer bounds the figure.)
  evidence: internal/contextprobe/probe.go:484 — "projected := c.Bytes + c.Bytes/5 + c.KVChargePerToken*size"
  evidence: internal/contextprobe/probe.go:486 — "return stepBounded, stepResult{bound: registry.BoundMemoryGuard, guardBytes: projected}"
  evidence: internal/registry/measurement.go:31 — "GuardBytes int64 `json:"guard_bytes,omitempty"`"
  evidence: internal/contextprobe/probe_test.go:345 — "func TestAStepTooLargeForThisMacIsSkippedWithItsProjection(t *testing.T) {"
- ac-8 — MET: Every step begins with unloadWaiting and the next request reloads the server through the gateway's ordinary Pool.Acquire; each prompt is fresh filler that opens with an 8-byte random nonce and a nonce-derived marker order; tests assert at least one unload per step and that no two prompts share a 64-character prefix.
  evidence: internal/contextprobe/probe.go:472 — "if !p.unloadWaiting(s.Ctx, c.RepoID) {"
  evidence: internal/gateway/gateway.go:628 — "up, release, err := g.pool.Acquire(r.Context(), model)"
  evidence: internal/contextprobe/filler.go:25 — "nonce := make([]byte, 8)"
  evidence: internal/contextprobe/filler.go:33 — "b.WriteString("[" + tag + "] ")"
  evidence: internal/contextprobe/probe_test.go:286 — "every step unloads first"
  evidence: internal/contextprobe/probe_test.go:292 — "func TestEveryStepReloadsAndSharesNoPrefix(t *testing.T) {"
- ac-9 — MET_WITH_CONCERNS: Nothing pinned is evicted: the pool's own never-evict rule stands and the probe additionally asks Fits before every step, sending a request only when the room is already free; its request travels the gateway's ordinary Pool.Acquire so it is under the same queue and caps, and a 503 holds the step for a later tick. Concern: the probe deliberately never parks (Parks() is false) — it does not 'wait in the queue every other caller waits in'; it declines to start, and if its request does end up queued the loop reads that waiter as a client and yields. Fits is also a snapshot, so an unpinned LRU model could still be evicted by the pool between Fits and Acquire; the never-evict test drives a fake pool's fits flag only.
  evidence: internal/contextprobe/probe.go:198 — "func (p *Probe) Parks() bool { return false }"
  evidence: internal/contextprobe/probe.go:478 — "if !p.fitsWaiting(s, c.RepoID) {"
  evidence: internal/app/selftest.go:65 — "The figures are a snapshot"
  evidence: internal/contextprobe/probe.go:624 — "return 0, answer{kind: answerNoRoom}, nil"
  evidence: internal/runtime/pool.go:701 — "func (p *Pool) Acquire(ctx context.Context, repoID string) (*Upstream, func(), error) {"
  evidence: internal/runtime/pool_test.go:1065 — "func TestPinnedModelsAreNeverEvictedAndTheRefusalNamesNoModel(t *testing.T) {"
  evidence: internal/contextprobe/probe_test.go:510 — "func TestAPinnedModelIsNeverEvictedForAProbe(t *testing.T) {"
- ac-10 — MET: The models-list handler writes measured_context and measured_bound in the same block as context_length and served_context, for a current measurement only; docs/models-list.md carries both fields in the table and a section saying what the three figures mean; the gateway test asserts the current entry carries them and a stale or unmeasured one does not.
  evidence: internal/gateway/gateway.go:429 — "entry["measured_context"] = mm.Window"
  evidence: internal/gateway/gateway.go:430 — "entry["measured_bound"] = mm.Bound"
  evidence: docs/models-list.md:50 — "The largest prompt, in tokens, that the model's server on this Mac verifiably accepted"
  evidence: docs/models-list.md:183 — "the three mean different things:"
  evidence: internal/gateway/control_probe_test.go:31 — "func TestModelsListCarriesTheMeasuredContext(t *testing.T) {"
- ac-11 — MET: AdoptMeasurement writes served_context through SetConfig, the setting the budget already charges and the gateway already enforces; the app test asserts an unadopted measurement leaves charge and served window unchanged and adoption lowers the charge; the card offers 'Use this window' for a current measurement, the panel charges modelCharge at servedContext, and Settings shows the served figure in force.
  evidence: internal/app/contextprobe.go:157 — "ms.ServedContext = m.Measured.Window"
  evidence: internal/app/contextprobe.go:160 — "return a.SetConfig(c)"
  evidence: internal/app/contextprobe_test.go:77 — "func TestAdoptingAMeasurementSetsTheServedContextAndAnUnadoptedOneChangesNoCharge(t *testing.T) {"
  evidence: internal/app/contextprobe_test.go:95 — "an unadopted measurement changed the charge from %d to %d"
  evidence: internal/gateway/gateway.go:418 — "if served := cfg.ServedContext(m.RepoID, m.ContextLength); served > 0 {"
  evidence: internal/ui/static/app.js:340 — "actions.append(btn('Use this window', 'ghost', () =>"
  evidence: internal/ui/static/app.js:1144 — "const window = servedContext(config, m.repo_id, m.context_length || 0);"
  evidence: internal/ui/static/app.js:1320 — "input.value = set ? String(set) : '';"
  evidence: internal/gateway/control_probe_test.go:65 — "func TestMeasureNowQueuesAndAdoptWritesTheServedWindow(t *testing.T) {"
- ac-12 — MET: A 504 is filed as prefill_deadline and a 400 as served_window, a sweep that reaches the cap without a refusal is bounded served_window by construction, and only BoundModel is a limit; the how-to and the models-list reference both say the model's own limit is not known for those bounds, and the card renders 'a floor: the prefill deadline stopped the probe first'; tests cover the deadline floor and the served-window cap.
  evidence: internal/contextprobe/probe.go:294 — "b = &bounds{hi: min(cand.Declared, MaxProbeWindow), bound: registry.BoundServedWindow}"
  evidence: internal/contextprobe/probe.go:620 — "return 0, answer{kind: answerBounded, bound: registry.BoundPrefillDeadline}, nil"
  evidence: internal/registry/measurement.go:76 — "func (m *Measurement) IsLimit() bool { return m != nil && m.Bound == BoundModel }"
  evidence: docs/context-probe.md:64 — "its own limit is not known."
  evidence: docs/models-list.md:192 — "the model takes at least that much, and its own limit is not known."
  evidence: internal/ui/static/app.js:357 — "case 'prefill_deadline': return 'a floor: the prefill deadline stopped the probe first';"
  evidence: internal/contextprobe/probe_test.go:314 — "func TestAStepStoppedByTheDeadlineIsAFloor(t *testing.T) {"
  evidence: internal/contextprobe/probe_test.go:328 — "func TestAServedWindowCapsTheSweep(t *testing.T) {"
- ac-13 — MET: Go carries ContextProbe and IdleThresholdSec, config.json carries them as context_probe and idle_threshold_sec, the panel has the Context probe pane, the per-card Measure now, and reads progress and the result from the state snapshot's idle_jobs and probe_queue; the UI test pins the pane to the file's keys, and both an app test and a control-plane test show a save naming neither setting leaves both alone.
  evidence: internal/config/config.go:652 — "IdleThresholdSec int `json:"idle_threshold_sec,omitempty"`"
  evidence: internal/ui/static/index.html:441 — "< input id="setContextProbe" type="checkbox">"
  evidence: internal/ui/static/index.html:456 — "< input id="setIdleThreshold" type="number" step="1" min="60" max="3600" placeholder="300">"
  evidence: internal/ui/static/app.js:1633 — "context_probe: $('setContextProbe').checked,"
  evidence: internal/ui/static/app.js:336 — "actions.append(btn('Measure now', 'ghost', () =>"
  evidence: internal/gateway/control.go:317 — "IdleJobs selftest.Status `json:"idle_jobs"`"
  evidence: internal/ui/contextprobe_test.go:10 — "func TestThePaneIsWiredToTheProbeSettings(t *testing.T) {"
  evidence: internal/gateway/control_probe_test.go:107 — "func TestASaveThatNamesNeitherProbeSettingLeavesBothAlone(t *testing.T) {"
  evidence: internal/app/contextprobe_test.go:128 — "func TestASaveRejudgesMeasurementsAndAnUnrelatedSaveKeepsTheProbeSettings(t *testing.T) {"
- ac-14 — MET_WITH_CONCERNS: StaleAgainst names the first provenance that moved (runtime, budget, decode_concurrency, served_context), the verdict is stored on the record, refreshed after every save, a stale figure is not published and cannot be adopted, and the card says which changed. Concerns: (1) a re-download does not mark the measurement stale with a reason — the download's Put clears it outright, so the record cannot say 'redownloaded'; (2) 'neither charged' holds for the measurement itself, but a figure already adopted stays in force as served_context after its measurement goes stale, so the charge follows the setting and is not reverted.
  evidence: internal/registry/measurement.go:89 — "func (m *Measurement) StaleAgainst(p Provenance) string {"
  evidence: internal/registry/measurement.go:35 — "Stale string `json:"stale,omitempty"`"
  evidence: internal/app/app.go:498 — "a.refreshStaleness()"
  evidence: internal/gateway/gateway.go:428 — "if mm := m.Measured; mm != nil && mm.Stale == "" && mm.Window > 0"
  evidence: internal/app/contextprobe.go:147 — "if m.Measured == nil || m.Measured.Stale != "" {"
  evidence: internal/ui/static/app.js:393 — "now stale: ${staleText(mm.stale)}; measure again"
  evidence: internal/registry/measurement_test.go:30 — "func TestAMeasurementGoesStaleWhenItsProvenanceMoves(t *testing.T) {"
  evidence: internal/registry/measurement_test.go:98 — "func TestARescanClearsAMeasurementForARedownloadedModel(t *testing.T) {"
  evidence: internal/registry/measurement_test.go:118 — "if m.Measured != nil || m.ProbeIncomplete {"
- ac-15 — MET: App construction calls refreshStaleness before anything reads a measurement; RefreshStaleness compares each stored measurement's runtime, budget, concurrency and served window with the provenance in force per model and writes the verdict to registry.json; the bound that stopped the largest step is stored on the record; the registry test asserts the file carries the stale reason after a refresh and the app test asserts the provenance is the pool's and the runtime's own figures.
  evidence: internal/app/app.go:341 — "a.refreshStaleness()"
  evidence: internal/registry/measurement.go:174 — "func (r *Registry) RefreshStaleness(inForce func(m Model) Provenance) []string {"
  evidence: internal/app/contextprobe.go:59 — "Runtime: runtime.MLXLMVersion(),"
  evidence: internal/registry/measurement.go:19 — "Bound string `json:"bound"`"
  evidence: internal/registry/measurement_test.go:55 — "func TestStalenessIsRecordedAtLoad(t *testing.T) {"
  evidence: internal/app/contextprobe_test.go:167 — "func TestTheProvenanceIsWhatThePoolAndTheRuntimeSay(t *testing.T) {"

Gap audit:
- honoured:
  - Gropius sends prompts of growing length, bisects between the last that came back and the first that did not, and records the verified window beside the declared one.
    evidence: internal/contextprobe/probe.go:351 — "for !b.swept {"
    evidence: internal/contextprobe/probe.go:371 — "for b.hi-b.lo > max(1024, b.lo/8) {"
    evidence: internal/gateway/gateway.go:429 — "entry["measured_context"] = mm.Window"
  - The probe runs when the Mac is idle, is interruptible, and yields the machine to a real request the moment one arrives.
    evidence: internal/selftest/selftest.go:399 — "if held := r.heldBy(act, now); held != "" {"
    evidence: internal/selftest/selftest.go:674 — "if busy(r.opts.Server.Activity(), h) {"
    evidence: internal/contextprobe/probe.go:511 — "p.unloadWaiting(context.Background(), c.RepoID)"
  - An unload and a reload between steps so a retained prompt cache cannot flatter the next reading, with unique prompts that share no prefix.
    evidence: internal/contextprobe/probe.go:472 — "if !p.unloadWaiting(s.Ctx, c.RepoID) {"
    evidence: internal/contextprobe/filler.go:33 — "b.WriteString("[" + tag + "] ")"
    evidence: internal/contextprobe/probe_test.go:292 — "func TestEveryStepReloadsAndSharesNoPrefix(t *testing.T) {"
  - The figure comes from this Mac and is never reused elsewhere; it goes with the files it described.
    evidence: internal/registry/measurement.go:10 — "it is a fact about these files on this"
    evidence: internal/registry/measurement_test.go:98 — "func TestARescanClearsAMeasurementForARedownloadedModel(t *testing.T) {"
  - A reading a gateway bound stopped is published as a floor naming the bound, never as the model's limit, and the docs say so plainly.
    evidence: internal/registry/measurement.go:76 — "func (m *Measurement) IsLimit() bool { return m != nil && m.Bound == BoundModel }"
    evidence: docs/models-list.md:192 — "the model takes at least that much, and its own limit is not known."
    evidence: README.md:87 — "A figure one of Gropius's own limits stopped"
  - Off by default, a global switch and a per-model Measure now on all three surfaces, and the idle threshold shared with the self-test as one setting.
    evidence: internal/config/config.go:646 — "ContextProbe bool `json:"context_probe,omitempty"`"
    evidence: internal/ui/static/index.html:439 — "< legend>Context probe< /legend>"
    evidence: internal/gateway/control.go:129 — "mux.HandleFunc("POST /api/models/measure", c.handleMeasure)"
    evidence: CHANGELOG.md:52 — "the probe and the self-test share is now a setting, `idle_threshold_sec`"
  - The measurement is stamped with its provenance and judged stale against what is in force at start and after every save.
    evidence: internal/registry/measurement.go:174 — "func (r *Registry) RefreshStaleness(inForce func(m Model) Provenance) []string {"
    evidence: internal/app/app.go:341 — "a.refreshStaleness()"
- diverged:
  - Press release: 'When a download finishes, Gropius measures the model.' Delivered: a finished download arms nothing; the switch or Measure now arms it and the first sustained idle runs it — the morning after, as the intent's 2026-09-10 clarification signed off.
    evidence: internal/app/contextprobe_test.go:26 — "func TestADownloadDoesNotStartAProbe(t *testing.T) {"
    evidence: internal/contextprobe/probe.go:240 — "if p.opts.Enabled == nil || !p.opts.Enabled() {"
  - Press release: 'The memory budget charges the window the machine can serve rather than the one the file advertises.' Delivered: the charge moves only when the operator adopts the figure as served_context; an unadopted measurement changes no charge (ac-11 as written, but narrower than the press release's sentence).
    evidence: internal/app/contextprobe.go:157 — "ms.ServedContext = m.Measured.Window"
    evidence: internal/app/contextprobe_test.go:95 — "an unadopted measurement changed the charge from %d to %d"
  - Alice's story reads the figure as what her Mac serves the model to; delivered, for most models the published figure is a floor bounded by the gateway's own limits, and a sweep that reaches the served window without a refusal is bounded served_window by construction.
    evidence: internal/contextprobe/probe.go:294 — "b = &bounds{hi: min(cand.Declared, MaxProbeWindow), bound: registry.BoundServedWindow}"
    evidence: docs/context-probe.md:65 — "On the 2026-09-06 evidence that is the common case."
  - ac-9 as written: the probe 'waits in the queue every other caller waits in'. Delivered: the probe never parks — it starts a step only when Fits says the room is free and a 503 holds the step for a later tick; it is under the queue's caps because it uses the ordinary gateway path, but it does not wait in the queue.
    evidence: internal/contextprobe/probe.go:198 — "func (p *Probe) Parks() bool { return false }"
    evidence: internal/contextprobe/probe.go:624 — "return 0, answer{kind: answerNoRoom}, nil"
  - ac-14 as written: a re-download marks the measurement stale and the record says which changed. Delivered: the download's Put replaces the record and the measurement is cleared, so nothing records the re-download as a reason.
    evidence: internal/registry/measurement_test.go:118 — "if m.Measured != nil || m.ProbeIncomplete {"
  - Spec: the three-surfaces sync is armed as an internal/archtest test in the per-model-settings pattern. Delivered: the pane wiring is pinned by a regexp test in internal/ui instead; internal/archtest carries the package only in its layering and prompt-content entries.
    evidence: internal/ui/contextprobe_test.go:10 — "func TestThePaneIsWiredToTheProbeSettings(t *testing.T) {"
    evidence: internal/archtest/prompt_content_test.go:32 — "internal/contextprobe/probe.go"
- missing:
  - ac-6's sleep trigger: on sleep no partial figure is written, the model is left unloaded, and the next start reports the probe incomplete. Nothing in the tree detects sleep or wake; the how-to only warns that a sleeping Mac loses the run, and a step that outlasts its timer after a wake is filed as a prefill_deadline floor.
    evidence: docs/context-probe.md:34 — "Mac that sleeps loses the run."
    evidence: internal/contextprobe/probe.go:496 — "if errors.Is(err, context.DeadlineExceeded) && s.Ctx.Err() == nil {"
  - Spec: 'No measured figure is published from a run that has not been checked this way at least once' — one Go-probe run compared by hand with the campaign script's figure on a real model. The decision record says the comparison has not been run, and the code publishes a figure the moment a run completes.
    evidence: .abcd/work/DECISIONS.md:267 — "The manual comparison against the campaign script on at least one real model has NOT been run in this session"
    evidence: internal/contextprobe/probe.go:393 — "if err := p.opts.Sources.Save(model, m); err != nil {"

Scope-condition dispositions:
- cond-2609112108142038 — survived: The memory guard is arithmetic over one pool: the model's flat charge plus the KV charge for the step, against the pool's budget less every resident charge and the memory still exiting, with a margin — no discrete-GPU figure appears anywhere.
  evidence: internal/contextprobe/probe.go:485 — "if projected > p.opts.Sources.Available()-p.opts.MemoryMargin {"
  evidence: internal/app/contextprobe.go:74 — "return s.a.Pool.MemoryBudget() - used"
- cond-2609112108146425 — survived: A measurement is stamped with the pinned runtime's version and is judged stale the moment the runtime in force differs, so a figure measured under one runtime is never presented as a figure under another.
  evidence: internal/app/contextprobe.go:59 — "Runtime: runtime.MLXLMVersion(),"
  evidence: internal/registry/measurement.go:91 — "case m.Runtime != p.Runtime:"
  evidence: internal/registry/measurement_test.go:39 — "{"runtime", func(p *Provenance) { p.Runtime = "0.32.0" }, StaleRuntime},"
- cond-2609112108146321 — untested: The probe reads idleness from this process's pool alone, which is consistent with one serving process per Mac, but nothing in the delivery exercises the singleton election or a second account's requests.
- cond-2609112108149020 — survived: Candidates are the registry's ready models; a candidate with no declared window is never due, Measure now refuses it with a message saying there is nothing to measure between, and the card offers no Measure now button for it.
  evidence: internal/contextprobe/probe.go:245 — "if !ok || c.Declared <= 0 || c.Incomplete {"
  evidence: internal/app/contextprobe.go:127 — "declares no context window; there is nothing to measure between"
  evidence: internal/ui/static/app.js:335 — "if (m.context_length > 0) {"
- cond-2609112108143953 — narrowed: The campaign's recall check the condition names was not delivered (a recorded planning decision), so the text-model assumption now rests on the filler the probe sends; that filler is English prose sent to every ready model with a declared window, with no filter on the model's kind.
  narrowing: holds for the text filler the probe generates and sends; no recall check exists for it to bear on, and nothing excludes a non-text model from being probed with that filler
  evidence: internal/contextprobe/filler.go:13 — "const block = "The lighthouse keeper counted the waves"
  evidence: internal/app/contextprobe.go:26 — "models := s.a.Registry.Ready()"
  evidence: .abcd/work/DECISIONS.md:229 — "the automated probe does NOT run the campaign's needle-recall check"
- cond-2609112108142632 — untested: No code observes sleep or wake; the how-to restates the assumption and nothing in the delivery exercises or contradicts what a probe does across a sleep.
- cond-2609112108146949 — survived: The probe POSTs to this Mac's own loopback /v1/chat/completions with the configured key, reads the server's usage.prompt_tokens, and files a 504 as the prefill deadline and a 400 as the served-window check; the real-stack test drives it through the actual gateway and pool.
  evidence: internal/contextprobe/probe.go:602 — "baseURL+"/v1/chat/completions""
  evidence: internal/app/contextprobe.go:92 — "return "http://" + host, s.a.Config().APIKey"
  evidence: internal/contextprobe/probe.go:620 — "return 0, answer{kind: answerBounded, bound: registry.BoundPrefillDeadline}, nil"
  evidence: internal/gateway/probe_integration_test.go:123 — "func TestTheProbeMeasuresThroughTheRealGatewayAndPool(t *testing.T) {"
## Grounds

- pursued: an automated sweep-and-bisect recovers the same servable window a person recovered by hand, because the campaign's method is mechanical and every input it needs — the declared cap, what is resident and idle, a prefill deadline derived from prompt size — is already in the process; shown wrong if a repeat probe on the same Mac, models and runtime returns a materially different window, or if what it finds is the gateway's bound rather than the model's, in which case the figure belongs to Gropius's configuration and moves whenever upstream_header_timeout_sec does — planned autonomously on the maintainer's instruction of 2026-09-10, adopting the brief's recommendations
