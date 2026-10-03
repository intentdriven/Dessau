# Unloading a model

Dessau unloads a model on its own when it goes unused for the idle timeout, or
when another model needs its memory. A program that has finished with a model
can also ask for it to be unloaded now, through the model API. The control
panel's **Unload** button is the operator's own: it unloads a pinned model, or
one inside its eviction grace, which a program's request does not.

## `POST /v1/dessau/unload`

The model API's one request that changes what the server holds.

```sh
curl -X POST http://localhost:11535/v1/dessau/unload \
  -H 'Content-Type: application/json' \
  -d '{"model": "mlx-community/Qwen3-8B-4bit"}'
```

| | |
| --- | --- |
| Body | `{"model": "<id>"}`, sent as `Content-Type: application/json`. The model is named the way a chat request names it: its full id, or its short name when that is unambiguous. |
| Answer | `200` with `{"status": "unloaded", "model": "<id>"}`. |

### Who can use it

The callers the server already trusts with what it is doing:

- **A program on the Mac that runs the server.** A web page in that Mac's
  browser is not one: a request a page sends is refused.
- **A program elsewhere that sends the API key**, on a server that has one.
  On a server without an API key, no other device on the network can use it.
- **A paired client**, over its own connection.

A program elsewhere that does not send the key, on a server that has one, is
refused with `401`; anyone else is refused with `403`. Either way the answer is
the same whatever model the request names, so the caller learns nothing about
what is loaded.

### When it is refused

The model stays loaded, and the answer is `409` with the reason, at once —
nothing waits:

| Reason | What it means |
| --- | --- |
| `model is not loaded` | The model is not in memory. |
| `the model is pinned` | The operator has pinned it; a pin is never undone by a program. |
| `model is busy` | It is answering a request, or the server's own idle work (the self-test, the context probe) is using it. |
| `the model is still loading` | It is being loaded for a request. |
| `the model is inside its eviction grace` | It answered a request within the eviction grace, which protects it for that long. With the grace on, a program that unloads a model straight after using it is refused until the grace has passed; ask again then. |

A model the server does not serve is `404`; a body that is not
`{"model": "<id>"}` is `400`; a request whose `Content-Type` is not
`application/json` is `415`.

### What it never does

It never deletes a model. There is no `DELETE` anywhere under `/v1`, and no
request to the model API removes a model's files.
