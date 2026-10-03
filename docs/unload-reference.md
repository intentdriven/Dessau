# Unloading a model

Dessau unloads a model on its own when it goes unused for the idle timeout, or
when another model needs its memory. A program that has finished with a model
can also ask for it to be unloaded now, through the model API. The control
panel's **Unload** button is the operator's own: it unloads a pinned model, or
one inside its eviction grace, which a program's request does not.

| Route | Who can use it | What it unloads |
| --- | --- | --- |
| `POST /v1/dessau/unload` | A program on this Mac, a program that sends the API key, a paired client — unless the operator turns it off | A model nobody is relying on |
| `POST /api/models/unload` | A program on this Mac, through the control panel's own routes | Any loaded model that is not answering a request |

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

### Turning it off

**Settings → Unloading from a program** turns the route off, or
`"api_unload_off": true` in `config.json`. While it is off, every caller
entitled to the route is refused with `403` and nothing is unloaded; a caller
the route would refuse anyway gets the refusal it always gets, and is told
nothing about the setting. The key absent means on. The control
panel's **Unload** button, and its route below, are not affected.

### What is recorded

An unload through this route is recorded in the request statistics, when they
are on, as a removal with the reason `released` and the kind of caller that
asked: a program on this Mac (whether or not it sent the API key), a program
elsewhere that sent the API key, or a paired client. The statistics record the
kind only, never the key and never which client
([statistics reference](statistics-store-reference.md)). The server log's
`model unloaded` line says the same, and names a paired client by the name it
was paired under and the short fingerprint of its key.

## `POST /api/models/unload`

The control panel's own route, which its **Unload** button sends. A script on
this Mac can send it too:

```sh
curl -X POST http://localhost:11535/api/models/unload \
  -H 'Content-Type: application/json' \
  -d '{"model": "mlx-community/Qwen3-8B-4bit"}'
```

| | |
| --- | --- |
| Body | `{"model": "<repo id>"}`, the model's full id. |
| Answer | `200` with `{"status": "unloaded", "model": "<repo id>"}`. |
| Who can use it | A program on this Mac, with no key, as for every route the control panel uses. Any other device, and a web page in this Mac's browser, is refused with `403`. |

It is the operator's unload: it unloads a pinned model, which stays pinned, and
one inside its eviction grace. When the server's own idle work (the self-test,
the context probe) is using the model, that work is asked to let go and the
route waits briefly for it. It is refused with `409` when the model is
answering a request or is not loaded. It is recorded with the reason
`unloaded`.

## `POST /api/models/load`

The control panel's **Load** button, which a script on this Mac can send to
bring a model into memory ahead of its first request.

| | |
| --- | --- |
| Body | `{"model": "<repo id>"}`, the model's full id. |
| Answer | `202` with `{"status": "loading", "model": "<repo id>"}` at once; the load carries on in the background. A second request while it is loading gets the same answer and starts nothing. |
| Who can use it | The same as `POST /api/models/unload`. |

The answer does not wait for the load. The models list says when the model is
in memory, in its `state` field for a program on this Mac
([models list](models-list.md)).
