# Reference: fields a completion request may not carry

`POST /v1/chat/completions` and `POST /v1/completions` pass a request's body on
to the model server. Two fields are refused instead, and a request carrying
either goes no further.

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
