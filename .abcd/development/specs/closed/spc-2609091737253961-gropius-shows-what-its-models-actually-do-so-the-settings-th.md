---
id: spc-2609091737253961
slug: gropius-shows-what-its-models-actually-do-so-the-settings-th
intent: itd-2609091712141073
origin: researcher-authored
production_mode: dictated-and-formatted
---
# Usage measurement: what each request did, against the settings in force

## Summary

The request statistics record gains the facts that decide a served window and
a concurrency — the prompt's size against the window the model declares and
the window it serves, how many requests were in flight when it was admitted,
which sampling values were in force and whether the client overrode them, and
the model server's memory footprint at completion — and the load event gains
the sampling values the server was launched with. A periodic sampler reads
each model server's footprint. The Statistics tab shows, per model, the
prompt-size distribution against both windows, the override rate and the
footprint over time. Nothing in it is a prompt or an answer, nothing is
exported, and the statistics reference page names every new field.

Every fact already passes through the gateway or the pool where recording it
costs one lookup; that is the intent's mechanism claim, and this design adds
no new source of truth for any of them.

## Scope

In scope: new fields on `stats.Record` and `stats.Event`; a footprint sampler
in `internal/runtime` (one reader of a process's resident memory, per launched
server, at a fixed cadence, reported through the existing `Observer`); the
gateway's `observation` filling the new request fields; the dashboard
aggregates in `internal/stats` and their rendering on the Statistics tab;
`docs/statistics-store-reference.md` and `docs/statistics-explained.md`; the
re-measured `ApproxRecordBytes`; the changelog entry.

Out of scope, each for its reason: an export button (resolved at interview —
the aggregates-only rule of the statistics endpoint stands, and the reference
page says where the files are); per-request memory attribution (the footprint
is a periodic per-model sample joined by time, and the record says so by its
name); storing any client-supplied sampling number (a per-parameter override
flag is the one field derived from a client body, argued on the docs page);
recording for the self-test's or the probe's own requests beyond what the
pool's load and removal events already carry (they go to the model server
directly and land in no request record).

## Approach

### The record

`stats.Record` gains:

| Field | Type | Where it comes from |
| --- | --- | --- |
| `declared_context` | int64 | the registry's `ContextLength` for the resolved model, at admission |
| `served_context` | int64 | `config.ServedContext(model, declared)` at admission — the figure the refusal was judged against |
| `estimated_prompt_tokens` | int | the gateway's own estimate (`estimatedTokens(bodyBytes)`), written for every request including one refused for size, so a refusal still carries how big the prompt was |
| `requested_tokens` | int | the figure the served-window check judged: the estimate plus the answer asked for |
| `in_flight` | int | the model's in-flight count at admission, reported by the pool on `AcquireStats` as the request takes its slot |
| `overrides` | []string | the names of the sampling parameters the client set in its body, from the same parse the gateway already does for `max_tokens`; the values are never stored |
| `footprint_bytes` | int64 | the model server's latest sampled footprint at completion, named as a sample, zero when the sampler has none yet |

`stats.Event` gains, on a load:

| Field | Type | Where it comes from |
| --- | --- | --- |
| `sampling` | the launch flags as `config.Sampling` | the `Spec` the launcher was handed (`SamplingFor` at launch) |

And a third event kind, `footprint`, carrying `model`, `at` and `bytes`: the
sampler's readings, so the footprint over time is in the store as a series
rather than only as the value stitched onto a request. It is a fifth record
kind, ratified by adr-2609121450000000, which supersedes
adr-2609090716413337.

The bounds every persisted numeric takes apply: a window beyond
`MaxContextLength`, a negative count, an override name not in the closed set
of sampling parameter names, or a footprint beyond the Mac's memory is
dropped at the recorder, never written.

### The sampler

`internal/runtime` gains a footprint reader: the physical footprint as
Activity Monitor and `top` report it — what a unified-memory Mac actually
spends, and what the 2026-09-06 campaign sampled — read through `top -l 1`
on the server's pid with a three-second deadline, because no unprivileged
pure-Go reader of that figure exists (`proc_pid_rusage` needs cgo, and `ps`
reports resident size, a different and smaller figure). The reader is an
optional `Footprinter` interface on the launcher's process, so a process that
cannot report one is simply never sampled. The pool samples every ready
server every `FootprintInterval` (thirty seconds) off its own lock, reports
each reading through an optional `FootprintObserver`, and holds the latest
per entry for `Pool.Footprint`, which the gateway stitches onto a completing
request. The sampler is the last thing the pool's close waits for, after the
servers have been told to stop. The self-test's memory guard and the probe's
may read the same latest figure later; nothing here depends on that.

### The gateway

The served-window check becomes `judgeServedContext`, which returns the
figures it judged — the two windows, the prompt's estimate, and the judged
figure (estimate plus the answer asked for) — so the record, the refusal and
the dashboard's bands rest on one number; `observation.judged` records
them, and `overrides(names []string)` the parameter names from the existing
body parse. The in-flight count at admission comes from the pool itself, on
`AcquireStats.InFlight`, taken at the moment the request takes its slot,
rather than from a residency snapshot before the request is admitted.
`footprint(bytes int64)` at completion takes the pool's latest sample. Every
one of these runs only while recording is on: the path with the switch off
is the path it was. No new parse of the body: only the presence of each
sampling key is read.

### The dashboard

`internal/stats` gains three aggregates over the range the existing views use:
per model, the distribution of `requested_tokens` — the figure the refusal
is judged on — in bands of the served window, beside the declared one, with
how often the served-window refusal bit; per model, the override
rate per parameter; per model, the footprint series from the `footprint`
events, downsampled to the range. The Statistics tab draws them under the
existing historical views, on the existing range control, with the page's
existing "what the views cannot show" section extended: a footprint is a
sample of the process, not a cost attributable to a request.

### Three surfaces

Go carries the whole of it. `config.json` gains nothing: there is no new
setting, the statistics switch already gates all of it. The panel gains the
three views. The sync obligation is the existing statistics docs test
(`internal/archtest/statistics_docs_test.go`), which holds the reference page
to `RecordFields()` and so fails the build the moment a field ships unnamed.

## How each acceptance criterion is tested

| Criterion | Test |
| --- | --- |
| A completing request carries declared, served and in-flight at admission | `internal/gateway`: `TestARecordCarriesTheWindowsAndTheInFlightAtAdmission` |
| A request refused for size still carries the estimate | `internal/gateway`: `TestARefusedRequestStillCarriesItsEstimatedSize` |
| A client-set parameter is marked overridden, no client number stored | `internal/gateway`: `TestOverridesAreNamedAndNeverValued` (the record's JSON carries no number the client sent) |
| A load event carries the launch sampling values | `internal/runtime` + `internal/app`: `TestALoadEventCarriesTheLaunchSampling` |
| Footprint sampled periodically; a record carries the footprint at completion, named as such | `internal/runtime`: `TestTheSamplerReportsEachServersFootprint` (fake launcher with a fake footprint); `internal/gateway`: `TestARecordCarriesTheLatestFootprintSample` |
| The dashboard shows prompt-size distribution against both windows, the override rate, the footprint over time | `internal/stats`: `TestPromptSizesAreBucketedAgainstBothWindows`, `TestOverrideRatesPerParameter`, `TestFootprintSeriesIsDownsampledToTheRange`; `internal/ui`: the view functions as pure functions |
| A new field is named on the reference page, the record-size constant re-measured, the docs say where the files are | `internal/archtest/statistics_docs_test.go` (existing, extended by the field list); `internal/stats/store_test.go`'s `ApproxRecordBytes` measurement |

## Grounds

- pursued: fields on the existing records plus a periodic footprint sample
  show which settings matter, because each fact already passes through the
  gateway or the pool; shown wrong if deciding a window or a concurrency turns
  out to need per-request memory attribution the sample cannot give — the
  intent's own mechanism claim, carried unchanged.
