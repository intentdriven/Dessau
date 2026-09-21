# Measure a model's real context window

Dessau can measure how long a prompt each model on your Mac can actually
take, rather than believing the window its configuration declares. It sends
prompts of growing length through its own OpenAI endpoint, the way a client
does, bisects between the last one that came back and the first that did
not, and records the window it verified beside the one the model declares
and the one it is set to serve. The figure comes from your Mac, your runtime
and your settings, not from someone else's benchmark.

The probe is off until you turn it on, and even then it runs only while the
Mac is idle. Nothing it measures leaves this Mac, and nothing it sends is
anything but its own fixed filler text. Because it reaches the model through
the same endpoint a client uses, its requests are counted in the
[request statistics](request-statistics.md) when recording is on — as
counts and timings only, never as text.

## What it costs

Read this before switching it on. A measurement is not free:

- **About forty minutes of GPU time per model**, at full prefill, on the
  2026-09-06 campaign's figures. Six models is an evening, one after another.
- **A single step can take half an hour**: the gateway allows a prompt
  about a second per 150 tokens plus a minute before it gives up, and never
  less than ten minutes, and a 262,144-token prompt is near the top of
  that. The probe's own limit on a step is that allowance plus a minute, so
  no step outlives eleven minutes without an answer at the smallest size,
  nor about thirty at the largest.
- **An unload and a reload between steps**, so a retained prompt cache
  cannot flatter the next reading. Each reload reads the weights from disk
  again.
- **The GPU is nobody else's while a step runs.** The probe never starts
  while anyone is using the Mac and stops the moment anyone does, but a
  request that arrives mid-step waits a moment for the probe to stand down.
- **On a laptop**, that is battery and heat, and the lid must stay open: a
  Mac that sleeps loses the run.
- **The probe never pushes another model out** to make room. A model that
  would need one evicted waits for a time when there is room.

## Switch it on

1. Open the control panel and go to **Settings → Context probe**.
2. Tick **Measure each model's context window when this Mac is idle**.
3. **Save settings**.

From then on, whenever the Mac has been idle for the idle threshold (five
minutes unless you change it, in the same section) and nothing is
downloading, Dessau measures the first model that has no current
measurement, one model at a time. A model you download later is measured
the next time the Mac is idle. In `config.json` the switch is
`"context_probe": true` and the threshold `"idle_threshold_sec"`.

## Which models it measures

Only a model the server offers to chat: one the
[models list](models-list.md) publishes with `"chat": true`. The probe
measures by sending chat completions, so a served window means nothing for
a model that cannot hold a conversation — a speech model, an OCR model — and
its server never answers the request that would measure it. Such a model is
never picked, and **Measure now** on its card says so.

Nor is a model whose last load failed. A model server that starts and never
becomes ready — its own log says the model type is not supported or a
module is missing, it exits, or ten minutes pass without an answer — is
marked on its card: **did not load**, with the reason. While that mark
stands, neither the probe nor the [self-test](self-test.md) picks the model
again, a queued measurement of it is dropped, and a request for it is
refused at once with the same reason rather than waiting through another
load. The mark is lifted when the runtime, the memory budget or the model's
served window changes — the load may go differently under them — when the
model is downloaded again, and when you press **Load** or **Measure now**
on its card, which is how to try it once more by hand. A mark whose reason
is Dessau's own bound rather than the model server's verdict — the ten
minutes ran out, or the server was ended by a signal — also goes when
Dessau restarts: a slow load on a busy Mac says nothing about the next one.

To measure one model without switching the probe on, open the **My Models** tab
and press **Measure now** on its card. The run starts at the next idle
minute.

## Read the result

The model's card on the My Models tab shows the measured window beside the
declared and the served ones, with what stopped the step above it:

- **the model** — the server refused or crashed above this size. The figure
  is the model's own limit on this Mac.
- **the prefill deadline**, **the served window**, or **the memory guard** —
  Dessau's own limit stopped the probe first. The figure is a verified
  floor: the model takes at least this much, and its own limit is not known.
  On the 2026-09-06 evidence that is the common case.

While a run is in progress the card says which step it is on and the bounds
so far; if a run is due but held back, it says what held it: a request in
flight, a caller waiting for a model, a download, or a recent request. It also
says so when the model does not fit the memory budget beside what is already
loaded: the measurement stays queued and waits for the memory to fall free,
because a measurement never evicts a model to make room for itself.

A measurement is marked **stale** when the runtime, the memory budget, the
decode concurrency or the model's served window has changed since it was
taken, and the card says which. A stale figure is not published and cannot
be adopted; measure again.

The figure is also published on the models list as `measured_context` and
`measured_bound`, beside `context_length` and `served_context`:
[The models list](models-list.md) says what each means.

## Whether the model calls tools

Beside the measured window, the card carries one more line Dessau found
for itself: **Tool calls: yes**, **no**, or **not measured**. The first time
a model is served, once the request that loaded it has been answered, Dessau
asks the model one fixed question with one small tool declared and records
whether it answered with a tool call. The question is Dessau's own and goes
to the model's own server rather than through the OpenAI endpoint, so it
costs one short answer, runs without the idle threshold, needs no switch,
and appears in no request statistic and no request log line.

The line reads **not measured** while the model has not been asked under
the runtime in force: a model downloaded again, or a Dessau update that
changes the runtime, is asked again the next time it is served. It refuses
nothing — a client may still send tools to a model marked **no**, and Dessau
relays them as it does today. The same answer is published on the models
list as `tool_calling`.

## Adopt the figure

A measurement changes nothing on its own: no memory charge, no refusal.
Press **Use this window** on the card to make it the model's served window.
That is the same setting as **Settings → Served context**, so the next load
is charged at it and a request above it is refused, exactly as if you had
typed the figure yourself. Setting the served window to the measured figure
does not make the measurement stale: the floor was verified under a window at
least that large, and still stands. Setting it to any other figure does.

## Stop it

Clear the box and save, or quit Dessau. A run in progress stops at once,
writes no figure, leaves the model unloaded, and the card says the probe was
incomplete. It is not retried on its own; press **Measure now** to run it
again.

While a run holds a model, the memory that model is charged is not free for
anyone else, and a client whose model would need it is refused. The card's
pill says **loading for the context probe** or **held by the context
probe** rather than only that the model is in memory, and a client on this
Mac, or one holding the API key, is told in the refusal which model the
probe holds and for how long. **Unload** on that card releases it at once,
even while the probe's own request is in flight: the run stands down the
way it does for a client's request, keeps its bounds, and carries on at the
next idle minute.

## Related

- [The models list](models-list.md) — the three windows a client reads.
- [The memory budget, explained](memory-budget-explained.md) — what a served
  window costs, and why adopting a measurement is a separate act.
- [Test your models while the Mac is idle](self-test.md) — the other idle
  job, which shares the idle threshold.
