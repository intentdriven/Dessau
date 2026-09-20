---
id: spc-2609201451069488
slug: alice-s-server-knows-which-models-can-call-tools-a-one-time
intent: itd-2609201445423499
origin: researcher-authored
production_mode: hand-written
---

# The tool-call probe: one question per model, recorded and published

## Summary

This spec delivers itd-2609201445423499: a one-shot probe that asks a served
model a fixed question with one small function tool declared, records whether
it answered with a tool call, and publishes the answer as a plain field on
the models list and on the model's card in the control panel. It runs once
per model per runtime version, directly against the model's own server so it
lands in no request statistic and no gateway log line, and the field it
writes is informative only: nothing on the completions path reads it, and a
client may still send tools to a model recorded as unable
(cond-2609201451067098). Impact: additive.

## Scope

In: a new package `internal/toolprobe`; `internal/registry` (the stored
field); `internal/gateway/gateway.go` (`handleListModels` only);
`internal/ui` (the model card and its test); `internal/app` (wiring the job
to the pool); `internal/archtest/prompt_content_test.go` (two admitted
readers); `docs/models-list.md`. Out: the on-device model and the Discord
bridge (cond-2609201451067566), the completions path, `config.json` — the
probe adds no setting — and any enforcement anywhere.

## Approach

**Why it is not a client of the gateway.** The context probe deliberately
goes through Dessau's own OpenAI endpoint, because what it measures is what
the gateway will actually serve. What this probe measures is a property of
the model's chat template and weights on this runtime, which the gateway does
not touch, and criterion 5 requires that it appear in no request statistic
and no request log line. So it takes the self-test's route instead: one
request to the model server's own loopback `BaseURL` with the exact
`ModelArg` that server expects (`runtime.Upstream`). Nothing counts it
because nothing on that path counts anything — the same reason the self-test's
runs are absent from statistics — rather than because a counter was taught to
skip it.

**The package.** `internal/toolprobe` is shaped like `internal/contextprobe`:
a `Sources` interface for what it needs from the app (`Acquire` a ready
model and release it, `Save` a verdict, the runtime version in force), an
`Options` struct with a logger, an HTTP client, a clock and a request
timeout, and one exported `Run` that probes one model. It owns the method,
the prompt and the verdict; it owns no idea of idleness, because it never
loads a model.

**When it runs.** A `runtime.PoolObserver` watches for a model reaching
`loaded`. If that model has no current verdict for the runtime in force, the
probe is queued — one at a time, on the app's own goroutine, after the
request that caused the load has been served. It never loads a model, never
evicts one, and never runs on a client's request (cond-2609201451063083): it
takes the same acquisition the self-test takes so nothing evicts the model
underneath it, and if the model has gone by the time the queue reaches it,
the probe is dropped and will be queued again the next time it is served.

**The request.** Non-streaming, `temperature: 0`, a small `max_tokens`, one
fixed user message that is a package constant, and one `tools` entry: a
function `get_time` taking `{"city": {"type": "string"}}`, with the message
asking for the time in a named city. The prompt carries no person's text and
is generated from nothing (cond-2609201451062229). Both
`internal/toolprobe/probe.go` entries are added to
`internal/archtest/prompt_content_test.go`: to `promptContentReaders`,
because it builds its own one-line conversation from a constant in that file
and reads nothing from a client; and to `generatedContentReaders`, because it
reads whether the choice carries a tool call and keeps nothing of what the
answer says.

**The verdict.** A choice carrying `tool_calls`, or `finish_reason:
"tool_calls"`, records `can`. Text, an empty message or a choice with
neither records `cannot` — the pinned runtime returns an empty message for
some families, and that is a real "cannot" on this runtime, which is what the
field claims. A transport error, a non-200 answer or a timeout records
nothing at all and leaves the model unprobed, because "the server did not
answer" is not evidence about the model, and a stored `cannot` on that basis
would be a lie a client acts on.

**Storage.** `registry.Model` gains `ToolCalling *ToolCalling
json:"tool_calling,omitempty"`, beside `Measured`:

```go
type ToolCalling struct {
    Can     bool   `json:"can"`
    At      int64  `json:"at"`
    Runtime string `json:"runtime"`
}
```

`Registry.SetToolCalling(repoID string, tc *ToolCalling) error` is written
beside `SetMeasurement`, with the same locking, the same snapshot-and-save,
and the same broadcast. A value read from `registry.json` is bounded the way
`plausibleMeasurement` bounds a measurement — `Runtime` at most
`maxProvenanceBytes`, `At` non-negative and below the same ceiling — and a
value outside the bounds is cleared rather than repaired, because in
shared-cache mode another local account can write that file. A re-downloaded
model loses its verdict with its measurement, by the rule that already clears
`Measured`. A verdict whose `Runtime` is not the one in force is stale; it is
not published, and the model is probed again the next time it is served
(criterion 4). Nothing else in the provenance matters: the memory budget and
the served window cannot change what a chat template does.

**Publication.** `handleListModels` adds one always-present string field to
every ready entry:

```go
entry["tool_calling"] = "yes" | "no" | "unknown"
```

Three states, for the reason `chat` is always present: an absent key would
read as an older Dessau that cannot say either way, and a bare boolean would
turn "not yet asked" into "no", which criterion 2 forbids. `unknown` covers
both the unprobed model and the stale verdict. Like `pipeline_tag`, `tags`
and `chat`, it says what a model IS rather than what this Mac is doing, so it
goes to every client, keyed or not, loopback or not, and it is not folded
into the residency projection an open server withholds.

**Informative, never enforced.** Nothing on the completions path reads the
field — the relay does not consult it, does not refuse on it and does not
rewrite `tools` because of it — which is the same rule the `chat` flag lives
under and the same rule adr-2609081118587999 states for third-party state. A
gateway test holds it by asserting that `tool_calling` is named nowhere in
`internal/gateway` outside `handleListModels`.

**The panel.** The model's card in the My Models tab gains one line beside
the measured context: "Tool calls: yes / no / not measured", rendered from
the same three states. No new setting is introduced, so
`internal/archtest/settings_surface_test.go` is unaffected in both
directions; the three-surfaces obligation is met by the card, because a Go
capability reported on the wire with no panel equivalent would be the gap
AGENTS.md names. `docs/models-list.md` gains a row for the field and one
paragraph saying what the probe asks, when it re-runs, and that the field
refuses nothing.

## Acceptance Criteria, and what holds each

- **A first serving is followed by one tool-bearing request whose answer is
  recorded with the runtime it was measured under**: `internal/toolprobe`'s
  own tests against `internal/mlxtest`'s fake server — one that answers with
  a tool call, one that answers with text, one that answers with an empty
  message — asserting the verdict and the recorded runtime; and a pool-observer
  test in `internal/app` asserting a model reaching `loaded` with no current
  verdict queues exactly one probe and a model with one queues none.
- **The models list carries the field, and an unprobed model says so rather
  than saying no**: a gateway test beside the existing models-list tests,
  asserting `tool_calling` is present on every ready entry and reads
  `unknown` for a model with no verdict and for one whose verdict is stale,
  `yes` and `no` for the two recorded verdicts.
- **The control panel shows it beside the measured context**: a test in
  `internal/ui` beside `measurement_test.go`, asserting the card's markup
  carries the line for each of the three states.
- **A runtime update re-probes and updates the field**: a registry test
  asserting a verdict recorded under one runtime string is not published
  under another, and a `internal/toolprobe` test asserting the stale verdict
  is re-queued and overwritten.
- **The probe appears in no request statistic and in no request log line**:
  an architecture test asserting `internal/toolprobe` imports neither
  `internal/stats` nor the gateway, and that its request is built against the
  upstream's own base URL; plus a `make run` hand check recorded in the
  shipping line — statistics on, a fresh model served, and the request table
  and the operational log showing the person's request and nothing else.
- **The probe reads no person's text and keeps nothing of the answer**:
  `TestOnlyTheMergeReadsPromptContent` and `TestPromptContentReadersAllExist`
  in `internal/archtest`, with the two new entries and their reasons — a
  green run with the package present is the evidence.
- **Informative, never enforced**: the gateway test named in the Approach,
  plus a relay test asserting a request carrying `tools` for a model recorded
  as `no` is relayed unchanged and answered.

Before the pull request: the trust-boundary security review, because the
change touches `internal/gateway` and adds a subprocess-facing request path;
and a docs-currency review of `docs/models-list.md`.

## Departures

None at the time of writing.
