# Reference: fields a completion request may not carry

`POST /v1/chat/completions` and `POST /v1/completions` pass a request's body on
to the model server, changing only the fields listed under
[What Dessau changes in a request it passes on](#what-dessau-changes-in-a-request-it-passes-on).
Two fields are refused instead, and a request carrying either goes no further;
so is a request that asks for an empty answer or whose `stop` is not text.

## The refused fields

| Field | What the model server would do with it | The error message |
| --- | --- | --- |
| `draft_model` | load a second model, from the path given, for speculative decoding | `"draft_model" is not accepted: Dessau does not load a second model for a request` |
| `adapters` | load adapter weights from the path given, and reload the served model to apply them | `"adapters" is not accepted: Dessau does not load adapter weights a request names` |

Dessau loads only the model a request names in `model`, from files it has
checked before the model server starts. A model's own `config.json` can name a
Python file that loading runs, which is why Dessau refuses such a model (see
[Troubleshooting](getting-started.md#troubleshooting)); a second model or
adapter named in a request would be loaded without that check. Speculative
decoding and per-request adapters are therefore not available through Dessau.

## How a request is matched

- The field's presence is enough, whatever its value: `null`, an empty string,
  a number, a list and an object are all refused.
- Only a field at the top level of the body counts. The same name inside a
  message, or written in another case (`Draft_Model`), is not one the model
  server reads, and the request is passed on as usual.
- A name written with JSON escapes (`"draft\u005fmodel"`) is the name it
  decodes to, and is refused. So is a field given more than once.
- A request carrying both is told about `draft_model`.

## The answer

The status is **400**, in the error shape every other refusal uses:

```json
{
  "error": {
    "message": "\"draft_model\" is not accepted: Dessau does not load a second model for a request",
    "type": "invalid_request_error",
    "code": 400
  }
}
```

The request is refused before a model is chosen: no model is loaded and none
is evicted for it, the model server never receives it, and the answer carries
none of the [response headers](response-headers.md). Every client receives
the same message, with or without the API key.

## An empty answer budget

A request whose `max_tokens` or `max_completion_tokens` is below 1 — `0`, a
negative number, a fraction under 1, or `false` — is refused in the same way,
with **400** and the message `"max_tokens" must be at least 1` (or
`"max_completion_tokens" must be at least 1`). The model server accepts such a
budget and then fails on it, and on a model serving batched requests the
failure stops that model answering anyone until it is restarted. A budget
written as a string is passed on, and the model server refuses it itself.

## A stop that is not text

A request whose `stop` is anything but a string, an array of non-empty strings
or `null` — a number, an array holding a number, `null` or an empty string, a
nested array, an object — is refused with **400** and the message
`"stop" must be a string or an array of strings`.
The model server takes such a value and then fails on it, in the same way as an
empty answer budget.

## What Dessau changes in a request it passes on

| Field | When | What the model server receives |
| --- | --- | --- |
| `model` | every request | the model server's own name for the model; the answer carries the name the request sent |
| `stream`, `stream_options` | `stream` is absent or `false`, and the request asks for neither `logprobs` nor `top_logprobs` | `"stream": true` and `"stream_options": {"include_usage": true}`, in place of any `stream_options` the request carried; the answer is assembled, as below |
| `stream_options` | `stream` is `true` and [request statistics](request-statistics.md#what-dessau-asks-the-model-server-for) are on | `include_usage` set inside it; the extra chunk is removed from the answer unless the request asked for it |
| the system messages | a chat request to a model with [Merge system messages](system-message-merging.md) switched on | the system messages gathered into the first one |

### A request that asks for no stream

The model server sends nothing of an unstreamed answer, not even its status,
until the whole answer is written, and while it writes one it does not notice
a client that has gone. Dessau therefore asks it for a stream and returns the
answer as the one JSON object the request asked for, built field for field as
the model server builds its unstreamed answer: the same `id`, `object`
(`chat.completion` or `text_completion`), `created`, `system_fingerprint` and
`model`; each choice's `index`, `finish_reason`, and its `message` — `role`,
`content`, `reasoning` and `tool_calls` — or its `text`; and the `usage`
counts with their `prompt_tokens_details`.

What follows from it:

- The [wait for the model server](getting-started.md#upstream_header_timeout_sec--how-long-a-model-may-take-to-start-answering)
  ends when the model server starts answering, so a long answer is not cut
  off by it. Nothing in Dessau bounds the answer after that: a model server
  that stops sending part-way without closing the connection holds the
  request until the client hangs up, as it holds a streamed one. A client
  with no timeout of its own waits for as long as that lasts.
- A client that hangs up stops the answer being generated.
- An answer the model server stops part-way through is a **502** with an
  error message, never part of an object.
- An answer larger than 64 MiB is a **502** whose message says to ask for it
  streamed.

A request that asks for `logprobs` or `top_logprobs` is passed on unstreamed
as it came, because the model server returns those only in an unstreamed
answer. For it, the wait covers the whole answer.
