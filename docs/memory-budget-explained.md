# Why there is a memory budget

Apple Silicon has unified memory: the weights a model loads come out of the same
pool as the window server, the browser and everything else. Nothing in macOS
stops a process filling that pool, and a model server that does takes the
machine down with it. So Dessau decides for itself how much of the Mac it is
willing to fill, and holds every load to that figure. The
[models list reference](models-list.md) states the figures and the rules; this
page is about what they mean.

## Why the default is a share, worked out per machine

The default is a share of the memory this Mac has, rather than a number in the
settings file. That is deliberate: a settings file copied to a smaller Mac would
otherwise carry a figure chosen for a machine that no longer exists. Leave the
field blank and each Mac answers for itself.

The share is chosen to leave the rest of the machine usable, which is the right
default for a Mac someone also works on and a poor one for a Mac that does
nothing else. That is the whole reason the figure is a setting.

## Why it is bounded by the machine

A budget larger than the memory that exists cannot be honoured, so Dessau does
not pretend to: whatever the settings file says, models are held to what this
Mac has. The stored figure is left alone — it is yours, and it may have been
written for a bigger machine you will move the file back to — but nothing is
admitted on the strength of memory that is not there.

If this Mac's memory cannot be read at all, there is nothing to bound the figure
to. Dessau falls back to a conservative default, shows no percentage, and takes
any figure you type at its word.

## What a model is charged, and why the cache is most of it

A loaded model is charged three things: its weights, a fifth of them again for
the working set a running model needs whatever the prompt is, and the attention
cache the window it is served at costs — once for every sequence its server may
decode at once.

The cache is the term that decides which models can share this Mac. It grows
with the prompt, and what it costs per token is a property of the architecture
rather than of the model's size: measured on a 128 GB Mac, four models between
18 GB and 45 GB of weights ranged from 12 KB to 353 KB per token, a thirtyfold
spread, and the largest of them was not the most expensive. The model server
also keeps a prompt's cache after the answer is sent, and caches stack across
requests, so the charge is what the model may come to hold rather than what it
holds the moment it loads.

The figure per token is read from the model's own configuration: how many of
its layers attend over the whole prompt, and what one layer's entry costs. That
arithmetic is a floor rather than an estimate — every model measured held more
than it, because a server keeps more than the raw cache — so Dessau multiplies
it: five times for a model that caches keys and values per head, seven for one
that caches a compressed latent, each the top of what its own kind was measured
holding. A model whose configuration cannot be read is charged the flat figure
instead: its weights plus a fifth.

## The window is the operator's, and it is the same window twice

The window in that charge is the one **Settings → Served context** holds for
the model. When that field is blank, the window is the largest one that fits
the memory budget: what is left of the budget after the model's weights and
their headroom, divided by what a token of cache costs across the batched
requests in force, capped at the window the model's own configuration declares
and never below 4,096 tokens. The card and the field both show the figure and
say it is the default. It is worked out, never stored, so it follows the budget
and the concurrency as they change, and a settings file carried to another Mac
carries no window chosen for a machine that no longer exists. It is one figure
doing two jobs, and that is what makes it trustworthy: the memory budget
charges the cache this window costs, and the gateway refuses a request
estimated to be larger than it. Dessau never budgets for one window and then
serves another.

The default is the budget's answer, not the model's. A model declaring 262,144
tokens can cost more than a whole Mac at that window, and every current
long-context model declares a window of that order; served at what fits, it is
a model this Mac can hold beside another rather than one it cannot hold at all.
A figure of your own in the field is honoured as typed, up to the declared
window: type one to serve a model longer than the default and pay the memory
for it, or shorter to make room beside it. **Batched requests (decode
concurrency)** is the other term: each sequence that may run at once holds its
own cache, so the default window is what the budget has room for per sequence.
It is one by default, which gives each model its widest window; raising it to
serve several clients at once narrows the default window in proportion. That
one is not immediate — the model servers take the concurrency when Dessau
starts, so a change to it is charged from the next start, and every figure
Dessau shows in the meantime is worked out from the concurrency in force.

A model whose charge does not fit the budget even at 4,096 tokens is refused
rather than loaded on a smaller figure. The refusal names the window and the
concurrency it was charged at and the largest of each that would fit, so what
to change is in the message. Dessau will not quietly charge a model less than
it costs: a budget that lies by a factor is worse than one that says no.

## The measured window is offered, never assumed

The [context probe](context-probe.md) can find the largest prompt a model
will actually take on this Mac, and that figure sits on the model's card
beside the served window. It is not charged. The budget charges the served
window, and only the served window, because that is the figure the operator
chose; a measurement is evidence for a choice, not the choice. Pressing
**Use this window** writes the measured figure into the served window, and
from then on it is charged and enforced exactly as a typed figure would be.
The two acts are kept apart so that a measurement can never lower or raise
what a model is charged without someone deciding it should.

## Why a change applies to the next load

Lowering the budget unloads nothing. A model in memory is one somebody is
probably using, and taking it away at the moment the operator pressed Save is
the one thing they were not asking for. The new figure governs the next load
instead, and the panel reports the machine as over its budget until the models
resident at the time go by the usual rules.

A swap does not lift the ceiling either. A model being replaced keeps its share
of the budget until its server process has actually gone, so the request that
needed the room waits out those seconds rather than starting a second server on
top of the first. That wait is bounded: a server that will not go — one the
system has stopped and then killed, and which is still there — leaves the
request refused rather than held, and the panel says how much memory is being
held that way and by how many servers.

Nothing waits on that memory any more. Dessau has already asked the server to
stop and then killed it; there is nothing further it can do, so it stops
counting on the memory coming back and works around the loss instead — models
go on loading and being swapped inside what is left, rather than every request
for room being refused from then on. If the system does let go of the server
later, the memory is credited then, and the panel stops reporting it. If it
never does, restarting Dessau is the remedy; the log names the server on the
way out, so the next start can finish the job.

## Choosing a figure

Alice's Mac Studio serves models and does nothing else, so she raises the budget
to most of the machine and keeps her writer and her reviewer in memory together.
Bob works on the laptop he serves from, so he leaves the default alone.

Add up the charged sizes of the models you want resident at the same time and
leave room for the ones your clients ask for occasionally. The charge already
counts the long prompts, so a set that fits is one this Mac can serve at those
lengths rather than one it can merely load. [Pinning](pinning-models.md) is the
other half of this decision: it decides which models keep their place in the
budget when something else needs the room.
