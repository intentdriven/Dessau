# Mark a model as one that keeps no transcript

Dessau keeps no transcript: nothing any model is asked and nothing it
answers is written down on the Mac that runs the server, and every client is
told so. A model can still be marked as one that keeps no transcript, one at
a time. The mark holds that model apart from the two places a conversation
could leave a record: it is not offered over the Discord bridge, and debug
logging cannot be armed on it. Every other model goes on as it was.

## Mark a model

1. Open the control panel and go to **Settings**.
2. Under **Transcript**, tick each model that is to keep no transcript.
3. Save. The mark holds from the next request that model serves.

The box is off for every model until you tick it. It is one of the settings
Dessau keeps per model, beside pinning, merging and the served window, so it
lives in the same three places they do: the panel, `config.json` under
`models` as `no_transcript`, and the terminal's `dessau config show`. A model
is matched whichever way its repository id is spelled — `ORG/Model` and
`org/model` are the same model — so a mark typed by hand in `config.json`
bites under the registry's spelling.

You can mark a model that is not on this Mac yet. Add it to `models` in
`config.json` with `"no_transcript": true` and the mark is in force from the
first request the model ever serves after it is downloaded; the panel draws a
row for it, marked *not on this Mac*, so it can be cleared from the form that
shows it.

Every client is told whether a conversation with a model is recorded,
through the models list's `recording` field and the icon Dessau Chat and the
control panel draw from it; [the models list](models-list.md#recording)
describes both.

## Over the Discord bridge

A marked model is not offered over the [Discord bridge](discord-bridge.md).
Discord keeps every message and every answer under its own terms, so a
conversation there would leave no record on this Mac and a full one on
Discord's servers — the opposite of what the box promises. The bridge's
`/model` listing leaves marked models out, naming one is refused in the
channel with the reason, and a channel already on a model that is marked
afterwards is refused at its next message.

## Debug logging

Per-model [debug logging](logging.md) writes every prompt and every answer a
model server sees into that model's own log, which is a transcript by another
name. Arming it on a marked model is refused with the reason, and the
paragraph at the control says so; ticking the transcript box for a model takes
it off the debug-logging list. The debug level is set when a model server
starts, so a model already running at debug keeps writing to its log until it
stops: unload it from its card to end that run.

## What the mark does not cover

It is a promise about what Dessau writes on this Mac. It does not reach a
platform on the far side of a bridge, which is why a marked model is not
offered there; and it does not reach a model server's own output at a level
set by hand outside Dessau's arming path.
