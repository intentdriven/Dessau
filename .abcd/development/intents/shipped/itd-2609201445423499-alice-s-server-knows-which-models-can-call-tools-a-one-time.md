---
id: itd-2609201445423499
slug: alice-s-server-knows-which-models-can-call-tools-a-one-time
spec_id: spc-2609201451069488
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091301112705]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Alice's server knows which models can call tools: a one-time probe per model, recorded server-side and shown on the models list

## Press Release

Alice downloads a model and, the first time it is served, Dessau Server
asks it one question with one small tool declared, the way it already
measures a new model's servable context. If the model answers with a tool
call, the server records that it can call tools; if it answers with text or
nothing, it records that it cannot. The answer lives beside the model's
other measurements, shows on the models list as a plain capability field
that any client can read, and appears on the model's card in the control
panel. Bob's Dessau Chat reads it and knows, before he asks, whether search
will come from the model's own tool calls or from the search-first switch;
Carol's coding tool reads the same field. Nothing is asked twice: the probe
runs once per model, and again only when the model or the runtime changes.

## Why This Matters

The pinned model server renders tools into the chat template and returns a
tool call when the model makes one, but ignores `tool_choice` and cannot
force it; whether a given model ever makes one depends on its template and
its training, and on this runtime some families answer empty. Every client
that wants tools would otherwise have to find that out for itself, once per
client per model. The server already measures a model once and tells every
client (the context probe); this is the same shape for one more fact, and it
is what the two search intents build on.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect one probe request per
model to settle whether it calls tools because the answer is a property of
the model's chat template and weights on this runtime, not of the prompt,
so a single well-formed request is as informative as a hundred. What would
show this wrong: a model that calls tools for some prompts and not others
on the same runtime, or the answer changing between runs with nothing
updated.

## Scope Conditions

- Served models on this Mac only; the on-device model and the bridge are <!-- cond: cond-2609201451067566 -->
  not probed, and a remote server's field is that server's.
- One probe per model per runtime version, run when the model is first <!-- cond: cond-2609201451063083 -->
  served, never on a client's request; the result is stored with the
  model's measurements and re-run only when the model or the runtime
  changes.
- The probe's prompt is fixed, short and the server's own; it carries no <!-- cond: cond-2609201451062229 -->
  person's text and is not logged as a request.
- The field is informative, never enforced: a client may still send tools <!-- cond: cond-2609201451067098 -->
  to a model recorded as unable, and the server relays them as it does today.

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked and accepted:

- Given a model is served for the first time, when its context has been
  measured, then one tool-bearing request is sent to it and its answer is
  recorded as can-call-tools or cannot, with the runtime version it was
  measured under.
- Given the models list, when a client reads a model entry, then it carries
  the recorded capability in a plain field, and a model not yet probed says
  so rather than saying no.
- Given the control panel, when Alice opens a model's card, then the field
  is shown beside the measured context.
- Given the runtime is updated, when a model is next served, then it is
  probed again and the field updated.
- Given the probe, when it runs, then it appears in no request statistic
  and in no request log line, like the self-test's own runs.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-89e6e831c5c0 -->
Fidelity review — receipt rcp-89e6e831c5c0 (verifier intent-auditor claude-opus-5).

Provenance: intent-auditor@claude-opus-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:68162d0120dd65a694800d6e4fb9163d1c4cdfb33b7986283e4e32fb3869f6b9
Input attestations: diff:1880fe35..9b94bba1@sha256:46af3538c5a53045e5b24fda1d3037f98a26e6b3cb8d8f9255f08c28a3b9823b;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: One tool-bearing request is sent at first serve and the verdict is saved with the runtime in force (probe.go:328, app test asserts Runtime == MLXLMVersion and Stats.Requests empty); concern: the trigger is the pool's LoadFinished observer once the loading request is served, not 'when its context has been measured' — the hand check (DECISIONS.md:376) records the probe running before any context measurement, as the spec's Approach designed.
  evidence: internal/toolprobe/probe.go:328 — "tc := &registry.ToolCalling{Can: can, At: p.opts.Now().Unix(), Runtime: p.opts.Sources.Runtime()}"
  evidence: internal/toolprobe/probe.go:352 — ""tools": []map[string]any{{"
  evidence: internal/app/app.go:970 — "if o.loaded != nil {"
  evidence: internal/app/toolprobe_test.go:93 — "func TestAModelReachingLoadedWithoutACurrentVerdictQueuesOneProbe"
  evidence: internal/toolprobe/probe_test.go:130 — "func TestTheVerdictIsReadFromTheAnswer"
  evidence: .abcd/work/DECISIONS.md:376 — "the probe ran before any context measurement"
- ac-2 — MET: handleListModels writes tool_calling as yes/no/unknown on every ready entry, unknown for the unprobed and the stale; the gateway test asserts all four states from a network client.
  evidence: internal/gateway/gateway.go:487 — "case !tc.Current(): entry["tool_calling"] = "unknown""
  evidence: internal/gateway/toolcalling_test.go:24 — "func TestModelsListCarriesTheToolCallVerdictOnEveryReadyEntry"
  evidence: internal/gateway/toolcalling_test.go:52 — ""org/unprobed": "unknown", "org/stale": "unknown", "org/yes": "yes", "org/no": "no""
- ac-3 — MET: renderModels places the toolcalls line directly after the measured line on a ready model's card, rendered by toolCallText for the three states; two ui tests hold the function and its placement in the markup.
  evidence: internal/ui/static/app.js:377 — "${measured ? `<div class="info measured">${escapeHtml(measured)}</div>` : ''} ${tools ? `<div class="info toolcalls">${escapeHtml(tools)}</div>` : ''}"
  evidence: internal/ui/static/app.js:476 — "function toolCallText(m) {"
  evidence: internal/ui/toolcalling_test.go:12 — "func TestTheCardSaysWhetherTheModelCallsTools"
  evidence: internal/ui/toolcalling_test.go:34 — "func TestTheCardIsBuiltFromTheToolCallLine"
- ac-4 — MET: RefreshStaleness at start (app.go:355) marks a verdict under another runtime StaleRuntime, Current() then reads false, modelLoaded enqueues the model at its next serve and the probe overwrites; registry, toolprobe and app tests hold each step.
  evidence: internal/registry/toolcalling.go:33 — "func (tc *ToolCalling) StaleAgainst(runtime string) string {"
  evidence: internal/app/toolprobe.go:80 — "if m.ToolCalling.Current() { return } a.ToolProbe.Enqueue(m.RepoID)"
  evidence: internal/app/app.go:355 — "a.refreshStaleness()"
  evidence: internal/registry/toolcalling_test.go:64 — "func TestAToolCallVerdictRecordedUnderOneRuntimeIsNotPublishedUnderAnother"
  evidence: internal/toolprobe/probe_test.go:266 — "func TestAQueuedProbeRunsAfterTheLoadingRequestAndOnlyOnce"
  evidence: internal/app/toolprobe_test.go:132 — "// A stale verdict is re-probed. a.modelLoaded("org/c")"
- ac-5 — MET: The request goes to the model server's own BaseURL, the package imports neither gateway nor stats (archtest), the app test asserts Stats.View().Requests is empty after a probe, and the hand check on the built binary found one request in the store and one request log line, the person's.
  evidence: internal/toolprobe/probe.go:370 — "up.BaseURL+"/v1/chat/completions""
  evidence: internal/archtest/toolprobe_test.go:18 — "func TestTheToolCallProbeIsNotAClientOfTheGatewayAndCountsNothing"
  evidence: internal/app/toolprobe_test.go:158 — "if reqs := a.Stats.View().Requests; len(reqs) != 0 {"
  evidence: .abcd/work/DECISIONS.md:376 — "The operational log held one `POST /v1/chat/completions` request line, the person's, and no request line for the probe."

Gap audit:
- honoured:
  - one probe per model, recorded server-side beside the model's other measurements
    evidence: internal/registry/registry.go:96 — "ToolCalling *ToolCalling `json:"tool_calling,omitempty"`"
    evidence: internal/toolprobe/probe.go:177 — "func (p *Probe) Enqueue(repoID string) bool {"
  - a plain capability field on the models list any client can read
    evidence: internal/gateway/gateway.go:487 — "entry["tool_calling"] = "unknown""
    evidence: docs/models-list.md:48 — "| `tool_calling` |"
  - shown on the model's card in the control panel
    evidence: internal/ui/static/app.js:476 — "function toolCallText(m) {"
    evidence: docs/context-probe.md:86 — "**Tool calls: yes**, **no**, or **not measured**"
  - asked again only when the model or the runtime changes
    evidence: internal/registry/toolcalling_test.go:117 — "func TestARedownloadClearsTheToolCallVerdict"
    evidence: internal/registry/toolcalling_test.go:64 — "func TestAToolCallVerdictRecordedUnderOneRuntimeIsNotPublishedUnderAnother"
  - informative, never enforced: tools for a model recorded unable are relayed unchanged
    evidence: internal/gateway/toolcalling_test.go:67 — "func TestARequestWithToolsForAModelRecordedUnableIsRelayedUnchanged"
    evidence: internal/gateway/toolcalling_test.go:117 — "func TestTheToolCallVerdictIsNamedNowhereOutsideTheModelsList"
  - the probe's answer is 'cannot' for text or an empty message, and nothing for a server that did not answer
    evidence: internal/toolprobe/probe.go:380 — "if resp.StatusCode != http.StatusOK {"
    evidence: internal/toolprobe/probe_test.go:215 — "func TestAServerThatDidNotAnswerRecordsNothing"
- diverged:
  - the probe fires 'when its context has been measured' (ac-1); delivered: it fires on the pool's LoadFinished once the loading request is served, before and independent of any context measurement
    evidence: internal/app/app.go:970 — "if o.loaded != nil { o.loaded(repoID)"
    evidence: .abcd/work/DECISIONS.md:376 — "the probe ran before any context measurement (the context probe waits for idleness; the observer fires on `loaded`"
  - the press release says the probe runs 'the way it already measures a new model's servable context'; delivered: not an idle-loop job through the gateway like the context probe but a direct request to the model server on a soft, resident-only hold
    evidence: internal/toolprobe/probe.go:8 — "It is not a client of the gateway."
    evidence: internal/app/toolprobe.go:53 — "runtime.WithResidentOnly(runtime.WithSoftHold(runtime.WithSource(ctx, toolProbeSource), selftest.YieldFrom(ctx)))"
  - spec scope named no change in internal/runtime; delivered adds WithResidentOnly/ErrNotResident to the pool (recorded as a departure)
    evidence: internal/runtime/pool.go:2540 — "func WithResidentOnly(ctx context.Context) context.Context {"
    evidence: internal/runtime/softhold_test.go:188 — "func TestAResidentOnlyAcquireNeverLoads"
- missing:
  - no promise of the press release is absent; the one owed check (the make run hand check) was performed and recorded rather than left owed
    evidence: .abcd/work/DECISIONS.md:376 — "done by the pilot's orchestrator on the integration head (080226be) rather than recorded as owed"

Scope-condition dispositions:
- cond-2609201451067566 — survived: The only trigger is the pool observer's LoadFinished for a model server this Mac runs; the diff touches nothing under internal/bridge and the on-device model lives in the Swift client, which the range does not touch, so only served models are probed.
  evidence: internal/app/app.go:318 — "Observer: poolObserver{rec: a.Stats, log: opts.Log, loaded: a.modelLoaded},"
  evidence: internal/app/toolprobe.go:75 — "func (a *App) modelLoaded(repoID string) {"
- cond-2609201451063083 — survived: Enqueue dedups a queued model, modelLoaded skips a current verdict, the pool holds the hold resident-only so the probe never loads, a re-download and a runtime change are the only re-runs, and nothing in internal/gateway names the probe so no client request can trigger it.
  evidence: internal/toolprobe/probe.go:184 — "for _, q := range p.queue { if config.FoldRepoID(q) == key { return false"
  evidence: internal/app/toolprobe.go:80 — "if m.ToolCalling.Current() { return"
  evidence: internal/app/toolprobe_test.go:265 — "func TestTheProbesAcquireNeverLoadsAModelThatHasGone"
- cond-2609201451062229 — survived: The prompt is a package constant, the archtest allow-lists probe.go as reading nothing from a client and keeping nothing of the answer, the decoder reads only tool_calls and finish_reason, and the hand check found no request log line for the probe.
  evidence: internal/toolprobe/probe.go:55 — "const Prompt = "What is the time in Paris right now?""
  evidence: internal/archtest/prompt_content_test.go:33 — ""internal/toolprobe/probe.go": "builds the tool-call probe's own one-line conversation from a constant in that file; reads nothing from a client"
  evidence: internal/toolprobe/probe.go:391 — "ToolCalls []json.RawMessage `json:"tool_calls"`"
- cond-2609201451067098 — survived: A request carrying tools for a model recorded no is relayed unchanged and answered, and a scan test refuses any file in internal/gateway other than handleListModels naming the field.
  evidence: internal/gateway/toolcalling_test.go:67 — "func TestARequestWithToolsForAModelRecordedUnableIsRelayedUnchanged"
  evidence: internal/gateway/toolcalling_test.go:117 — "func TestTheToolCallVerdictIsNamedNowhereOutsideTheModelsList"
## Grounds

- pursued: every client that wants tools would otherwise discover this per client per model; wrong if the answer varies by prompt
