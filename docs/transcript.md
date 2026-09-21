# Keep some models out of the transcript

A Dessau server's transcript, while it is on, writes down every prompt and
every answer the server serves. Some of what passes through a server is
nobody's business but the person typing it, so the transcript can be switched
off for one model at a time: that model answers exactly as it did before, and
nothing it is asked and nothing it replies is written down — not while it is
excepted, and not afterwards either, because what was never written cannot be
produced later. Every other model on the server goes on as it was.

## Except a model

1. Open the control panel and go to **Settings**.
2. Under **Transcript**, tick each model that is to keep no transcript.
3. Save. The exception holds from the next request that model serves.

The box is off for every model until you tick it. It is one of the settings
Dessau keeps per model, beside pinning, merging and the served window, so it
lives in the same three places they do: the panel, `config.json` under
`models` as `no_transcript`, and the terminal's `dessau config show`. A model
is matched whichever way its repository id is spelled — `ORG/Model` and
`org/model` are the same model — so an exception typed by hand in
`config.json` bites under the registry's spelling.

You can except a model that is not on this Mac yet. Add it to `models` in
`config.json` with `"no_transcript": true` and the exception is in force from
the first request the model ever serves after it is downloaded; the panel
draws a row for it, marked *not on this Mac*, so it can be cleared from the
form that shows it. A request naming a model the server does not hold is
refused before anything could be written, so there is no gap beside the
exception.

## What a client is told

Every client that asks the server for its models is told, for each one,
whether a conversation with it is recorded: the `recording` field on every
entry of the [models list](models-list.md), present with or without an API
key, on the LAN as on this Mac. Dessau Chat reads it and shows an icon beside
each model in its picker — a pencil while a conversation with the model is
written down, struck through while it is not — labelled in words for a screen
reader, so the state is visible before the model is chosen. The control
panel's model cards carry the same icon.

An ordinary OpenAI-compatible client reads a model's name and displays none of
this. Dessau cannot make software show something it was not written to show;
what it can do is publish the fact where every client can read it, and show it
in its own client and panel.

## A conversation that changes model

The exception is a fact about the model that answers a request, not about text
that has passed through another one. Dessau has requests, not conversations:
it cannot tell which model produced a turn already sitting in the message
array a client sends back. So a request answered by a recorded model is
written down whole, including earlier turns from an excepted model that the
client carried back as prior context. To keep a conversation out of the
transcript, keep it on an excepted model from its first turn to its last.

## Over the Discord bridge

An excepted model is not offered over the [Discord bridge](discord-bridge.md).
Discord keeps every message and every answer under its own terms, so a
conversation there would leave no transcript on this Mac and a full one on
Discord's servers — the opposite of what the box promises. The bridge's
`/model` listing leaves excepted models out, naming one is refused in the
channel with the reason, and a channel already on a model that is excepted
afterwards is refused at its next message.

## Debug logging

Per-model [debug logging](logging.md) writes every prompt and every answer a
model server sees into that model's own log, which is a transcript by another
name. Arming it on an excepted model is refused with the reason, and the
paragraph at the control says so; ticking the transcript box for a model takes
it off the debug-logging list.

## What the exception does not cover

It is a promise about the transcript this server writes on this Mac. It does
not reach a platform on the far side of a bridge, which is why an excepted
model is not offered there; and it does not reach a model server's own output
at a level set by hand outside Dessau's arming path.
