# Reference: the server's log

Dessau keeps a log of what it does. This page says where it is, what each of
its two levels writes, and what it never writes at either.

## Where the file is

In the `logs` folder of your own Dessau data folder, as `dessau.log`, beside
the model servers' own logs. That folder belongs to your macOS account: on a
Mac where several people use Dessau, each account has its own log and no
account can read another's. The file is created owner-only.

The first line of every run says where it is, so if you are not sure, start
Dessau from a terminal and read the `log=` field on the `dessau starting`
line — or open Settings, which says which folder it is in.

Dessau writes to standard error as well, with the same lines. That is what you
see when you run the server from a terminal. It is not what you see when you
launch the app from the Finder: macOS discards a Finder-launched app's standard
error, and the file is what this page exists for.

## The two levels

Set the level in Settings, under **Server log**, or as `log_level` in
`config.json`. Changing it takes effect on the next line — nothing needs
restarting. Sparse is the default.

| Level | What it writes |
| --- | --- |
| `sparse` | One line for each thing that mattered: the server starting and stopping, a request refused — which model, which refusal it was, and whether the client was one this server tells the reason to — a model loading, a model leaving memory and why, a model server that would not start, and a save in Settings. |
| `detailed` | Every sparse line, and the figures they leave out: how long a load took, how many requests were already in flight, the memory budget in bytes, how long a request waited, the error behind a launch that failed, and the drain behind an eviction. |

Sparse is enough to answer "why was my client refused". Detailed is what to
turn on when the answer is "because there was no room" and you want to know how
much room there was.

A client on your network is told what Dessau could not do and never why:
"cannot serve this model right now". A client on this Mac, or one holding the
API key, still gets the fuller answer. Either way the reason is in this log —
sparsely, as which refusal it was, and in full at the detailed level.

The figures are on the detailed level rather than the sparse one for a reason
beyond noise. The memory budget in bytes is roughly how much memory this Mac
has, and the number of requests in flight is how busy it is. A client on your
network can cause a refusal every time it sends a request, so at the sparse
level a stranger cannot make Dessau write a detailed description of your Mac
to your disk as fast as it can ask.

## What is never written

At either level, and whatever else is switched on:

- No prompt, and no part of one.
- No answer, and no part of one.
- No API key, and no HuggingFace token.
- No client address — not the address a request came from, not its port, and
  not a host name for it.

A line describes the call, never the caller: the method, the path, the status
and how long it took.

## This is not the model servers' log level

The model servers keep their own logs, one per model, in the same folder, as
`<org>@<name>.log`. They run at their INFO level, and `log_level` here is
Dessau's own and reaches nothing else: neither it nor the statistics switch
changes what a model server writes.

One action does. **Debug logging**, on a model's card in the control panel,
arms that one model so that its next start runs the model server at its
debug level. From that start until the start after it, the model server's own
log holds every request sent to it and every answer it produced — the prompts
and the completions, whoever sent them, and the requests Dessau's own probes
and self-test send as well. Nothing in it is redacted, so an API key a client
puts in a message is written as sent. It is per model and off unless you arm
it; arming changes nothing about the run going on now, so unload the model to
start the run that is logged, and the run after that is back at INFO. A model
server also restarts when memory pressure evicts it or the idle timeout
unloads it, which any client's request can cause, so the logged run can end at
a moment you did not choose. While a model is armed or running at its debug
level its card says so, and the Posture tab names it. Nothing is sent
anywhere, and clients are not told. A model that keeps no transcript refuses
the arm.

A run at the debug level writes bytes chosen by whoever is sending requests,
so its log stops at 64 MB: what fits is written, one final line says the log
stopped there and the model is still serving, and the rest is dropped while
the model keeps answering. A run at INFO has no bound of its own.

Those logs are not rotated the way this one is. Each model server's start
keeps the previous run's file beside the new one, as
`<org>@<name>.previous.log`, and replaces the one before that — so the run
you armed is still there to read after the restart that ends it, and one
model's logs take at most two files.

## How big it gets

`dessau.log` grows to 5 MB and is then rolled over: the current file becomes
`dessau.1.log`, that becomes `dessau.2.log`, and so on. Five files are kept
in all — the current one and four previous — so the log takes at most 25 MB and
the oldest is removed as new lines arrive. Nothing has to be tidied up by hand,
and nothing is sent anywhere: the file stays on this Mac.

## Where the setting is kept

In `config.json`, as `log_level`:

```json
{
  "log_level": "detailed"
}
```

The two values are `"sparse"` and `"detailed"`. A missing field, or an empty
one, means sparse — so a settings file written before this field existed keeps
the default. A word that is neither is repaired to sparse when the file loads,
and Settings says it was repaired; saving one from Settings is refused instead,
because that is the moment you are there to read why.
