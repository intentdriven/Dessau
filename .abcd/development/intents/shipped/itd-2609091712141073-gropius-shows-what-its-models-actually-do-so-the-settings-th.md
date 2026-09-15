---
id: itd-2609091712141073
slug: gropius-shows-what-its-models-actually-do-so-the-settings-th
spec_id: spc-2609091737253961
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609061521082551, itd-2609061431481936]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Gropius shows what its models actually do, so the settings that matter can be seen before they are chosen. For every request the usage dashboard records the prompt's size against the window the model declares and the window it serves, the time to the first token, the memory the model server held while answering, how many requests were in flight, the sampling values in force and whether the client overrode them, and the class of any refusal; per model it shows their distributions and how often a limit bit. Alice reads it to decide the served window and the concurrency; nothing in it is a prompt or an answer, and it can be exported as a file.

## Press Release

Alice opens the usage dashboard and sees, per model, what its requests did:
how large the prompts were against the window the model declares and the one
it is set to serve, how long the first token took, how many requests were in
flight, how the model server's footprint moved, which sampling values each
server was launched with and how often clients set their own, and which
refusals happened. Nothing in it is a prompt or an answer, and the files are
the plain JSON Lines the reference page names.

## Why This Matters

The request record today holds the class, the token counts and four
durations; it cannot say how a prompt sat against a window, what was in flight,
or whether a client set its own sampling. Those are the facts that decide a
served window and a concurrency, and today they are guessed. Refines the
context-probe draft (itd-2609091301112705), which measures a model once at
download, by measuring every request in use. No export: the statistics
endpoint keeps handing the panel aggregates only, and the reference page says
where the files are.

## Mechanism

We expect fields on the existing request and load records, plus a periodic
footprint sample, to show which settings matter, because each fact already
passes through the gateway or the pool where recording it costs a lookup.
Shown wrong if deciding a window or a concurrency turns out to need
per-request memory attribution the sample cannot give.

## Scope Conditions

- Apple Silicon, one Mac, with the statistics opt-in on. <!-- cond: cond-2609091737259918 -->
- The served window is the per-model setting. <!-- cond: cond-2609091737257664 -->
- Memory is the model server's footprint at completion from a periodic sample, not an attributable per-request cost. <!-- cond: cond-2609091737256053 -->
- Shared-cache mode records every local account's traffic under the serving account's opt-in. <!-- cond: cond-2609091737259099 -->
- The sampling parameters are mlx-lm's launch-flag set. <!-- cond: cond-2609091737254282 -->

## Acceptance Criteria

- Given a request completes, when its record is written, then it carries the model's declared window, its served window and the in-flight count at admission.
- Given a request is refused for prompt size, when its record is written, then it still carries the gateway's estimated size.
- Given a client sets a sampling parameter, when the request is recorded, then that parameter is marked overridden and no client-supplied number is stored.
- Given a model server is launched, when the load event is written, then it carries the sampling values the server was launched with.
- Given a model server is running, when the sampler runs, then its footprint is recorded periodically and a request record carries the footprint at completion, named as such.
- Given the usage dashboard, when a model is shown, then it shows the prompt-size distribution against both windows, the override rate and the footprint over time.
- Given a new field, when it ships, then the statistics reference page names it, the record-size constant is re-measured, and the docs say where the files are.

## Open Questions

- Resolved 2026-09-09 at interview: no export button (the aggregates-only rule of the statistics endpoint stands); memory is a periodic per-model sample joined by time; launch sampling values go on the load event and a per-parameter override flag on the request, argued in the docs as the one field derived from a client body.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-916dd6b8bf3f -->
Fidelity review — receipt rcp-916dd6b8bf3f (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:eda32a8f22872c819685a2f0effd603f9fe00c373ca61f6a7752833ad2dfb397 · prompt_hash sha256:e1dfe3f92eaf597527fedcfc436fa0e163cbe261b80ef1f914a8b56585da2409
Input attestations: tree:main at acacd07b9fb29a2dfc94a8aa58da22d67e8d8b36, range 6b1f600..acacd07 (PR 52 carried four intents; only itd-2609091712141073's criteria were judged; `go test ./internal/stats/ ./internal/runtime/ ./internal/gateway/ ./internal/ui/ ./internal/archtest/ ./internal/app/` all passing at that tree)@sha256:unknown; request:.abcd/.work.local/reviews/rcp-916dd6b8bf3f.request.md@sha256:eda32a8f22872c819685a2f0effd603f9fe00c373ca61f6a7752833ad2dfb397; intent:.abcd/development/intents/shipped/itd-2609091712141073-gropius-shows-what-its-models-actually-do-so-the-settings-th.md@sha256:e1dfe3f92eaf597527fedcfc436fa0e163cbe261b80ef1f914a8b56585da2409;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The record declares declared_context, served_context and in_flight; the gateway writes the two windows from judgeServedContext before admission and the in-flight count from the pool's AcquireStats, which the pool takes as the model's count at the moment the request takes its slot (already := e.inFlight); a gateway test asserts 131072/65536 and in_flight 2 on a completed request's record and a pool test asserts 0 then 1 for two admissions.
  evidence: internal/stats/stats.go:149 — "DeclaredContext int64 `json:"declared_context,omitempty"`"
  evidence: internal/gateway/gateway.go:619 — "obs.judged(verdict.declared, verdict.served, verdict.estimate, verdict.judged)"
  evidence: internal/gateway/gateway.go:707 — "obs.waited(up.Waits)"
  evidence: internal/gateway/observe.go:118 — "o.record.InFlight = w.InFlight"
  evidence: internal/runtime/pool.go:907 — "already := e.inFlight"
  evidence: internal/gateway/measurement_test.go:42 — "func TestARecordCarriesTheWindowsAndTheInFlightAtAdmission(t *testing.T)"
  evidence: internal/runtime/observer_test.go:470 — "func TestAcquireReportsTheInFlightCountAtAdmission(t *testing.T)"
- ac-2 — MET: judgeServedContext computes the estimate for every request and obs.judged runs before the `if msg != ""` refusal branch, so a request refused for size still records estimated_prompt_tokens; a gateway test sends a 400,000-byte prompt against a 65,536 window, sees it refused as client_error and asserts the record carries an estimate of at least 100,000 beside served_context 65536.
  evidence: internal/gateway/gateway.go:622 — "if msg != "" {"
  evidence: internal/gateway/gateway.go:1441 — "v.estimate = int64(estimatedTokens(bodyBytes))"
  evidence: internal/gateway/observe.go:86 — "o.record.EstimatedPromptTokens = int(estimate)"
  evidence: internal/gateway/measurement_test.go:77 — "func TestARefusedRequestStillCarriesItsEstimatedSize(t *testing.T)"
  evidence: docs/statistics-store-reference.md:87 — "written for every request including one refused for its size"
- ac-3 — MET_WITH_CONCERNS: overriddenSampling reads only the presence of each key in the closed set (plus max_completion_tokens folded to max_tokens), the recorder drops names outside SamplingParameters, and a gateway test asserts the three names appear and none of the literal client values 0.123456, 777 or 0.5 occur in the marshalled record. Concern: requested_tokens is the estimate plus the client's own max_tokens (capability.AddSaturating(v.estimate, requestedMaxTokens(payload))), and estimated_prompt_tokens sits beside it, so the client's max_tokens value is recoverable by subtraction from two stored fields — a client-supplied number is stored in arithmetic form, and the reference page's claim that overrides is 'the one field derived from a client's body' is not strictly true. The test checks only the literal string, which is why it passes. Not a content leak (a count, not a prompt), but it is a divergence from the criterion's wording.
  evidence: internal/gateway/gateway.go:1530 — "func overriddenSampling(payload map[string]json.RawMessage) []string {"
  evidence: internal/gateway/observe.go:95 — "o.record.Overrides = names"
  evidence: internal/stats/stats.go:172 — "var SamplingParameters = []string{"temperature", "top_p", "top_k", "min_p", "max_tokens"}"
  evidence: internal/gateway/measurement_test.go:96 — "func TestOverridesAreNamedAndNeverValued(t *testing.T)"
  evidence: internal/gateway/gateway.go:1445 — "v.judged = capability.AddSaturating(v.estimate, requestedMaxTokens(payload))"
  evidence: internal/gateway/observe.go:87 — "o.record.RequestedTokens = int(judged)"
  evidence: docs/statistics-store-reference.md:90 — "It is the one field derived from a client's body, and it says only which knob was touched."
- ac-4 — MET: The pool takes SamplingFor(repoID) at launch, keeps it on the entry, and hands it to LoadFinished; the app maps config.Sampling to a by-name map and the recorder writes it as Event.Sampling on the load event, bounded to the closed set; three tests hold the chain — the pool reports the launch temperature, samplingValues maps all five flags, and the recorder's load event carries exactly the in-set parameters.
  evidence: internal/runtime/pool.go:1168 — "sampling = p.opts.SamplingFor(repoID)"
  evidence: internal/runtime/pool.go:1198 — "p.notify(func(o PoolObserver) { o.LoadFinished(e.repoID, took, err, e.sampling) })"
  evidence: internal/app/app.go:924 — "func (o poolObserver) LoadFinished(repoID string, took time.Duration, err error, sampling config.Sampling) {"
  evidence: internal/stats/stats.go:225 — "Sampling map[string]float64 `json:"sampling,omitempty"`"
  evidence: internal/stats/stats.go:549 — "Sampling: boundSampling(sampling),"
  evidence: internal/runtime/observer_test.go:439 — "func TestALoadEventCarriesTheLaunchSampling(t *testing.T)"
  evidence: internal/app/measurement_test.go:13 — "func TestALoadEventCarriesTheLaunchSampling(t *testing.T)"
  evidence: internal/stats/measurement_test.go:67 — "func TestALoadEventCarriesTheLaunchSampling(t *testing.T)"
- ac-5 — MET: sampleFootprints ticks every FootprintInterval (default 30 s), reads each ready server's Footprint off the lock via top -l 1, keeps the newest on the entry and reports it to a FootprintObserver; the app forwards it to Recorder.FootprintSampled, which writes a kind: footprint event; the gateway stitches pool.Footprint(model) onto the record after the relay on both the streamed and non-streamed paths, into a field named footprint_bytes and documented as 'as last sampled'. A pool test sees the sample reported and Footprint return it, a gateway test sees 48 GiB on the record, a recorder test sees the footprint event, and the production top reader is exercised against a live process.
  evidence: internal/runtime/pool.go:1984 — "const DefaultFootprintInterval = 30 * time.Second"
  evidence: internal/runtime/pool.go:1989 — "func (p *Pool) sampleFootprints() {"
  evidence: internal/runtime/pool.go:2031 — "fo.FootprintSampled(t.repoID, bytes)"
  evidence: internal/runtime/launcher.go:372 — "func (p *execProcess) Footprint() int64 {"
  evidence: internal/app/app.go:885 — "func (o poolObserver) FootprintSampled(repoID string, bytes int64) {"
  evidence: internal/stats/stats.go:565 — "ev := Event{At: r.now().UTC().Unix(), Model: model, Kind: EventFootprint, Bytes: bytes}"
  evidence: internal/gateway/gateway.go:834 — "obs.footprint(g.pool.Footprint(model))"
  evidence: internal/stats/stats.go:166 — "FootprintBytes int64 `json:"footprint_bytes,omitempty"`"
  evidence: internal/runtime/observer_test.go:410 — "func TestTheSamplerReportsEachServersFootprint(t *testing.T)"
  evidence: internal/gateway/measurement_test.go:116 — "func TestARecordCarriesTheLatestFootprintSample(t *testing.T)"
  evidence: internal/stats/measurement_test.go:50 — "func TestAFootprintSampleIsAnEvent(t *testing.T)"
  evidence: internal/runtime/footprint_test.go:10 — "func TestExecProcessReportsAFootprintWhileAliveAndNoneAfter(t *testing.T)"
  evidence: docs/statistics-store-reference.md:91 — "The model server's memory footprint as last sampled before the request completed"
- ac-6 — MET_WITH_CONCERNS: History carries prompt_sizes, overrides and footprints; the Statistics tab draws three tables — 'Prompts against the served window' with Declared and Served columns, four bands, refused and largest; 'Sampling overrides' as a percentage per parameter; 'Memory over time' with lowest/highest/latest and a sparkline over at most 200 points — and stats and ui tests hold the buckets, the rates, the downsampling and the row functions. Concern: the criterion says the distribution is shown 'against both windows', but the bands are computed as shares of the served window only (est*4 <= r.ServedContext ...); the declared window is shown as a figure beside them, so the comparison against it is one the reader makes from the Largest prompt column rather than a distribution the panel draws. The spec planned exactly this ('in bands of the served window, beside the declared one'), so it is a signed-off narrowing of the criterion's wording.
  evidence: internal/stats/dashboard.go:238 — "PromptSizes []ModelPromptSizes `json:"prompt_sizes"`"
  evidence: internal/stats/dashboard.go:314 — "Buckets [5]int `json:"buckets"`"
  evidence: internal/stats/dashboard.go:610 — "case est*4 <= r.ServedContext:"
  evidence: internal/stats/dashboard.go:633 — "ov.ByParameter[name]++"
  evidence: internal/stats/dashboard.go:814 — "series.Points = append(series.Points, FootprintPoint{At: at, Bytes: sl.sum / sl.count})"
  evidence: internal/ui/static/index.html:165 — "< h3>Prompts against the served window< /h3>"
  evidence: internal/ui/static/index.html:186 — "< h3>Memory over time< /h3>"
  evidence: internal/ui/static/app.js:2031 — "$('statsSizesRows').innerHTML = (h.prompt_sizes || []).map(sizesRowHtml).join('');"
  evidence: internal/stats/measurement_test.go:104 — "func TestPromptSizesAreBucketedAgainstBothWindows(t *testing.T)"
  evidence: internal/stats/measurement_test.go:128 — "func TestOverrideRatesPerParameter(t *testing.T)"
  evidence: internal/stats/measurement_test.go:150 — "func TestFootprintSeriesIsDownsampledToTheRange(t *testing.T)"
  evidence: internal/ui/measurement_test.go:11 — "func TestTheMeasurementViewsReachThePanel(t *testing.T)"
- ac-7 — MET: The reference page names every new request field, the load event's sampling and the footprint kind, and the archtests hold it to RecordFields() and EventFields() so an unnamed field fails the build; ApproxRecordBytes was re-measured from 230 to 380 by TestARecordLineIsTheSizeTheDocumentationSays over a record carrying every new field, and TestTheRecordSizeIsTheSameFigureEverywhere holds both docs pages and the config comment to that constant; the reference page opens with 'Where the files are' and the how-to points at it.
  evidence: docs/statistics-store-reference.md:7 — "## Where the files are"
  evidence: docs/statistics-store-reference.md:86 — "| `declared_context`, `served_context` | The two windows the request was judged against"
  evidence: docs/statistics-store-reference.md:101 — "| `sampling` | The sampling values the model server was launched with, by parameter name, only the ones set."
  evidence: docs/statistics-store-reference.md:118 — "## `kind: "footprint"` — a reading of a model server's memory"
  evidence: internal/archtest/statistics_docs_test.go:24 — "for _, field := range stats.RecordFields() {"
  evidence: internal/archtest/event_fields_test.go:12 — "func TestTheStatisticsPageNamesEveryEventField(t *testing.T)"
  evidence: internal/stats/store.go:50 — "const ApproxRecordBytes = 380"
  evidence: internal/stats/store_test.go:1397 — "func TestARecordLineIsTheSizeTheDocumentationSays(t *testing.T)"
  evidence: internal/archtest/statistics_docs_test.go:213 — "func TestTheRecordSizeIsTheSameFigureEverywhere(t *testing.T)"
  evidence: docs/request-statistics.md:79 — "A request record measures about 380 bytes, so 200 MB is roughly seven weeks"

Gap audit:
- honoured:
  - Each fact is recorded where it already passes through — the gateway's served-window judgement and the pool's admission — with no new source of truth: one servedVerdict feeds the record, the refusal and the dashboard's bands.
    evidence: internal/gateway/gateway.go:1435 — "func (g *Gateway) judgeServedContext(cfg config.Config, model string, bodyBytes int, payload map[string]json.RawMessage) (string, servedVerdict) {"
    evidence: internal/runtime/pool.go:983 — "InFlight: already,"
  - The footprint is a fifth record kind in the one store, under the one switch and the one retention rule, ratified by adr-2609121450000000 which supersedes adr-2609090716413337.
    evidence: internal/stats/store.go:68 — "KindFootprint = "footprint""
    evidence: internal/stats/summary.go:158 — "return []string{KindRequest, KindLoad, KindRemoved, KindFootprint, KindSettings, KindSummary, KindSummaryIndex}"
    evidence: .abcd/development/decisions/adrs/2609121450000000-the-statistics-store-gains-a-fifth-record-kind-a-footprint.md:45 — "4. **A footprint reading** — `kind: "footprint"`"
  - Nothing is recorded while the switch is off and the switched-off request path is unchanged: every new observation runs under obs.recording(), and a footprint sample is dropped by the recorder when it is not enabled.
    evidence: internal/gateway/gateway.go:618 — "if obs.recording() {"
    evidence: internal/stats/stats.go:569 — "if !on {"
    evidence: internal/stats/measurement_test.go:55 — "t.Fatal("a sample was recorded with the switch off")"
  - No export: the statistics endpoint hands the panel aggregates only, and the reference page says where the files are so any tool can read them.
    evidence: internal/stats/dashboard.go:15 — "The browser never sees a record: it is given the"
    evidence: docs/statistics-store-reference.md:7 — "## Where the files are"
  - Every new numeric and name is bounded at the recorder — windows past MaxContext, counts past any batch, footprints past any Mac, override names outside the closed set — dropped, never repaired, with MaxContext held equal to the registry's bound by an archtest.
    evidence: internal/stats/stats.go:437 — "func bound(rec Record) Record {"
    evidence: internal/stats/measurement_test.go:23 — "func TestARecordCarriesTheMeasurementFieldsAndTheyAreBounded(t *testing.T)"
    evidence: internal/archtest/context_bound_test.go:13 — "func TestTheStatisticsWindowBoundIsTheRegistrys(t *testing.T)"
  - The sampler reads off the pool's lock, a reading is bounded by a three-second deadline, and the sampler is the last thing the pool's close waits for, after the servers have been told to stop.
    evidence: internal/runtime/pool.go:2018 — "// Read off the lock: a reading runs a process listing."
    evidence: internal/runtime/launcher.go:388 — "out, err := exec.CommandContext(ctx, "/usr/bin/top", "-l", "1", "-stats", "mem", "-pid", strconv.Itoa(p.cmd.Process.Pid)).Output()"
    evidence: internal/runtime/pool.go:2138 — "<-p.sampleDone"
  - The three surfaces stay in step: the panel gains the three views, the docs test holds the reference page to the record's fields, and the changelog names the change.
    evidence: internal/ui/measurement_test.go:14 — "regexp.MustCompile(`\(h\.prompt_sizes \|\| \[\]\)\.map\(sizesRowHtml\)`),"
    evidence: internal/archtest/statistics_docs_test.go:24 — "for _, field := range stats.RecordFields() {"
    evidence: CHANGELOG.md:18 — "carries the two windows it was judged against, Gropius's own estimate of"
  - The explanation page extends 'what the views cannot show' with the footprint's limit — a sample of the process, not a cost attributable to a request — and the panel's own hint says the same.
    evidence: docs/statistics-explained.md:197 — "**What one request cost in memory.** The footprint is sampled from the"
    evidence: internal/ui/static/index.html:195 — "< p class="hint">Memory is the model server's own footprint, sampled every thirty seconds and averaged"
- diverged:
  - ac-6 as written: the prompt-size distribution is shown 'against both windows'. Delivered: the bands are shares of the served window only; the declared window is a column beside them and the comparison against it is left to the reader via the Largest prompt figure. The spec planned this shape, so it is a signed-off narrowing rather than an omission.
    evidence: internal/stats/dashboard.go:302 — "// the range. The buckets are shares of the served window: up to a quarter,"
    evidence: internal/ui/static/index.html:165 — "< h3>Prompts against the served window< /h3>"
  - ac-3 as written: 'no client-supplied number is stored'. Delivered: the sampling values are not stored, but requested_tokens = estimated_prompt_tokens + the client's max_tokens, so max_tokens is recoverable by subtraction from two stored fields; the reference page's statement that overrides is 'the one field derived from a client's body' understates this. The test passes because it checks only for the literal string of the value.
    evidence: internal/gateway/gateway.go:1445 — "v.judged = capability.AddSaturating(v.estimate, requestedMaxTokens(payload))"
    evidence: docs/statistics-store-reference.md:88 — "The figure the served-window check judged: the estimate above plus the answer the request asked for."
    evidence: internal/gateway/measurement_test.go:107 — "for _, value := range []string{"0.123456", "777", "0.5"} {"
  - The intent's headline says the dashboard records, for every request, 'the sampling values in force'. Delivered: the values in force live on the load event, joined to requests by model and time; the request record carries only the override names. Resolved at the 2026-09-09 interview and carried by the spec, so a signed-off divergence from the headline.
    evidence: .abcd/development/intents/shipped/itd-2609091712141073-gropius-shows-what-its-models-actually-do-so-the-settings-th.md:66 — "launch sampling values go on the load event and a per-parameter override flag on the request"
    evidence: internal/stats/stats.go:225 — "Sampling map[string]float64 `json:"sampling,omitempty"`"
  - The spec's table says the load event carries 'the launch flags as config.Sampling'. Delivered: a by-name map[string]float64 built in internal/app, because the stats package imports nothing of ours. Same information, different wire shape; the docs describe the delivered one.
    evidence: internal/app/app.go:891 — "func samplingValues(s config.Sampling) map[string]float64 {"
    evidence: docs/statistics-store-reference.md:101 — "| `sampling` | The sampling values the model server was launched with, by parameter name, only the ones set."
  - The footprint reader is top (the physical footprint), as the spec argues; the comment beside DefaultFootprintInterval still says 'a ps per server is nothing'. Cosmetic, but a comment that names the reader the spec rejected.
    evidence: internal/runtime/pool.go:1983 — "// that a ps per server is nothing."
    evidence: internal/runtime/launcher.go:388 — "out, err := exec.CommandContext(ctx, "/usr/bin/top", "-l", "1", "-stats", "mem", "-pid", strconv.Itoa(p.cmd.Process.Pid)).Output()"
- missing:
  - The intent's title promises the dashboard 'can be exported as a file'. No export exists: the control plane's stats routes are the live view, the history aggregate and clear. The intent's own body and the interview resolution withdrew the export, so this is a withdrawn promise the title still carries rather than a delivery gap the maintainer would act on — but the headline and the delivery disagree.
    evidence: .abcd/development/intents/shipped/itd-2609091712141073-gropius-shows-what-its-models-actually-do-so-the-settings-th.md:15 — "nothing in it is a prompt or an answer, and it can be exported as a file."
    evidence: .abcd/development/intents/shipped/itd-2609091712141073-gropius-shows-what-its-models-actually-do-so-the-settings-th.md:34 — "No export: the statistics"
    evidence: internal/gateway/control.go:125 — "mux.HandleFunc("GET /api/stats", c.handleStats)"
  - No single test drives a launch through pool → app observer → recorder → store and reads the sampling back off the written load line; the chain is held by three unit tests in three packages (pool report, app mapping, recorder event). Adequate, but the end-to-end assertion the spec's table implied ('internal/runtime + internal/app') is split rather than joined.
    evidence: internal/runtime/observer_test.go:439 — "func TestALoadEventCarriesTheLaunchSampling(t *testing.T)"
    evidence: internal/app/measurement_test.go:13 — "func TestALoadEventCarriesTheLaunchSampling(t *testing.T)"
    evidence: internal/stats/measurement_test.go:67 — "func TestALoadEventCarriesTheLaunchSampling(t *testing.T)"

Scope-condition dispositions:
- cond-2609091737259918 — survived: Everything new is gated on the statistics opt-in — the gateway's observations run only under obs.recording() and the recorder drops a footprint sample while disabled — and the footprint reader is macOS's own top on this Mac's process table, exercised by a test against a live local process.
  evidence: internal/gateway/gateway.go:618 — "if obs.recording() {"
  evidence: internal/stats/stats.go:569 — "if !on {"
  evidence: internal/runtime/launcher.go:388 — "out, err := exec.CommandContext(ctx, "/usr/bin/top", "-l", "1", "-stats", "mem", "-pid", strconv.Itoa(p.cmd.Process.Pid)).Output()"
- cond-2609091737257664 — survived: The served window the record carries and the refusal is judged against is cfg.ServedContext(model, declared), the per-model setting, and the gateway test sets it through config.Models[testModelID].ServedContext.
  evidence: internal/gateway/gateway.go:1440 — "v.served = cfg.ServedContext(model, v.declared)"
  evidence: internal/gateway/measurement_test.go:33 — "cfg.Models = map[string]config.ModelSettings{testModelID: {ServedContext: 65536}}"
- cond-2609091737256053 — survived: The record's footprint is the pool's newest 30-second sample read at completion, the field's own comment and the reference page call it a sample of the process joined by time, and the explanation page lists per-request memory cost under what the views cannot show.
  evidence: internal/gateway/gateway.go:834 — "obs.footprint(g.pool.Footprint(model))"
  evidence: internal/stats/stats.go:165 — "// not a cost attributable to this request. Zero when nothing was sampled."
  evidence: docs/statistics-explained.md:197 — "**What one request cost in memory.** The footprint is sampled from the"
- cond-2609091737259099 — untested: Nothing in the delivered range runs or asserts shared-cache mode; the store's location text on the reference page predates this intent and no test in the diff exercises a second local account's traffic under the serving account's opt-in.
- cond-2609091737254282 — survived: The closed set the recorder accepts — temperature, top_p, top_k, min_p, max_tokens — is the same five names the launcher maps to mlx-lm's --temp, --top-p, --top-k, --min-p and --max-tokens flags, and samplingValues maps config.Sampling's five fields to exactly those names.
  evidence: internal/runtime/launcher.go:50 — ""temperature": "--temp","
  evidence: internal/stats/stats.go:172 — "var SamplingParameters = []string{"temperature", "top_p", "top_k", "min_p", "max_tokens"}"
  evidence: internal/app/app.go:891 — "func samplingValues(s config.Sampling) map[string]float64 {"
## Grounds

- pursued: we expect the numbers to decide the served window and concurrency defaults instead of guesses; shown wrong if after a month the distributions separate no setting choice, or the added fields push the store's retention below a useful span.
