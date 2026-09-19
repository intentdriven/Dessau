# Test your models while the Mac is idle

Gropius can measure its own models, on your Mac, while nobody is using them:
how long each takes to load, how fast it reads a long prompt, how fast it
generates, and what that becomes when several requests arrive at once. The
figures come from your machine rather than from someone else's benchmark, so
they are the ones to size a model's
[served context](memory-budget.md) and the decode concurrency from, and to
compare two quantisations of the same model by numbers instead of by feel.

The self-test is off until you turn it on. Nothing it records leaves this Mac
([the decision that settles it](../.abcd/development/decisions/adrs/2609061503319212-no-public-telemetry-local-telemetry-only-as-a-strict-opt-in.md)),
and nothing it records is a prompt or an answer: the file holds counts and
timings, field by field in
[Reference: the self-test results](self-test-reference.md).

## Switch it on

1. Open the control panel and go to **Settings → Self-test**.
2. Tick **Test the models when this Mac is idle**.
3. **Save settings**.

It applies at once; there is nothing to restart. In `config.json` the same
switch is `"self_test": true`.

Clear the box and save to switch it off again. A run in progress is stopped
at once and its model released; the results already written stay where they
are.

## What happens while it is on

Once a minute Gropius asks whether the Mac is idle: no request in flight on
any model, no request waiting for a model to load, no download running, and
the last request older than the idle threshold — five minutes unless you
change it under **Settings → Context probe**, `idle_threshold_sec` in
`config.json`; the self-test and the [context probe](context-probe.md) share
it. When it is, Gropius picks the
model measured longest ago — a model never measured comes first, and a model
measured within the last day is left alone — and runs it through the same
short set every model gets:

- **load** — how long the model takes to load, when it was not already in
  memory.
- **pp512** — a prompt of about 512 tokens, asking for one token back: how
  fast the model reads.
- **tg128** — a short prompt, asking for 128 tokens back: how long the first
  token takes and how fast the rest come.
- **tg128×N** — the same, sent N times at once, where N is the decode
  concurrency: what the machine delivers when several clients arrive
  together.

The names are [llama-bench](https://github.com/ggml-org/llama.cpp/tree/master/tools/llama-bench)'s,
so a figure here can be set beside a published one.

When the set is done Gropius writes one line to the results file and, if the
self-test was what loaded the model, unloads it again; a model that was
already in memory is left there. Then it goes back to waiting for the next
idle minute and the next model.

## What it costs, and what it never does

A run holds one model for the length of the set — a few seconds for a small
model, a minute or two for a large one on a cold load — and the memory that
model takes while it is held. The load goes through the same path a request
takes, so the [memory budget](memory-budget.md) and the
[pinned models](pinning-models.md) apply, and the self-test never evicts: a
model that would need another pushed out to make room is left for a time
when there is room, so the model you keep warm stays warm.

A request from anyone ends the run at once, during a load as much as during
a test: the self-test's own request or load is cancelled, the run is recorded
as yielded, and the model is the client's. A request for a model already in
memory waits a moment for that cancellation and for nothing else. A request
whose model needs the memory the run is holding takes it: the run is asked to
let go, it does, and the model it was holding is unloaded for the client's —
so the client waits for the run to stop rather than being refused. Removing a
model that is under test is refused until the run ends, as removing a model
that is serving is.

A run touches the model the way a request does, so when
[request statistics](request-statistics.md) are on, the self-test's loads and
unloads appear in the load and eviction views there, and a model the
self-test measured counts as recently used for the idle timeout.

The self-test never runs a model the Mac does not already have, never sends
anything anywhere but to the model server on this Mac, and never records a
prompt, an answer, a key or an address.

## Where the results are

The **Statistics** tab of the control panel shows the latest run for each
model — when it ran, how it ended, the load time, and the figures of note
from each test — and the most recent runs beneath. It shows them whether or
not request statistics are being recorded; the two switches are separate.

The file is `selftest/results.jsonl` in your Gropius data folder, beside the
[statistics store](statistics-store-reference.md): one line of JSON per run,
readable with any tool that reads JSON Lines. It is this account's own file,
at mode 0600, and it is bounded — when the next line would take it past
4 MiB it is started again, which is thousands of runs away. The
[reference page](self-test-reference.md) names every field.

On a Mac several people log into, the models are one set, so the self-test
measures every account's models, and the switch and the file are the serving
account's own.

## Related

- [Reference: the self-test results](self-test-reference.md) — every field of
  a results line.
- [The memory budget](memory-budget.md) — what a loaded model is charged, which
  is what the load figure and the concurrency figure are for.
- [Record request statistics on this Mac](request-statistics.md) — what real
  requests did, as opposed to what a model can do when asked the same thing as
  every other.
