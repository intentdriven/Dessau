# Reference: the self-test results

What the [self-test](self-test.md) writes, where, and field by field.

## The file

`selftest/results.jsonl` under this account's Dessau data folder: the
data root for a per-user install or a root set with `DESSAU_ROOT`, and this
account's own Application Support directory under a shared install, beside
the statistics store and under the same rule. Its directory is created, at
mode 0700, on the first write, and the file is mode 0600.

The format is JSON Lines: one object per line, one run per object, appended
in the order the runs happened. The file is bounded at 4 MiB. When the next
line would take it past that, the file is started again and the line is the
first in it, so the newest run is always kept and the history is at most the
cap. There is one file: no numbered predecessors and nothing beside it.

It is written by the same writer as Dessau's own log and its
[request statistics](statistics-store-reference.md#kind-run--one-self-test-run),
and held to the same rules. A name in this folder standing for something that
is not this account's own plain file — a link, a named pipe, a folder, or a
file any other account could read — is refused rather than written to or
waited on, and the run is dropped with a line in the log.

## A run

| Field | Type | Meaning |
| --- | --- | --- |
| `kind` | string | Always `run`. |
| `model` | string | The model's repo id, as the models list spells it. |
| `at` | integer | When the run started, Unix seconds, UTC. |
| `outcome` | string | `ok`, `yielded`, `stopped` or `failed`; see below. |
| `reason` | string | For a failed run, `load` or `request`; absent otherwise. |
| `cold_load` | boolean | `true` when the model was not in memory before the run, so `load_ms` is a load from disk and the run unloaded the model afterwards. |
| `load_ms` | integer | How long acquiring the model took, in milliseconds. For a model that was already in memory this is close to zero. |
| `tests` | array | The tests that completed, in the order they ran; see below. A yielded, stopped or failed run carries those that finished before it ended. |

### Outcomes

- `ok` — every test in the set completed.
- `yielded` — a request from a client arrived, or a client's load needed the
  memory the run was holding; the self-test cancelled its own request and
  released the model.
- `stopped` — the switch went off, or Dessau quit, during the run.
- `failed` — the model could not be loaded (`reason` is `load`), or a request
  to it failed or timed out (`reason` is `request`). The reason is a class, not
  the error's text; Dessau's own log has the text.

A model with a failed or yielded run is not retried until its turn comes round
again the next day. A stopped run does not count: the model is measured
again when the self-test is next on and the Mac idle.

## A test

| Field | Type | Meaning |
| --- | --- | --- |
| `name` | string | `pp512`, `tg128`, or `tg128xN` where N is the number of requests sent at once. |
| `parallel` | integer | How many requests were sent at once: 1, or N for the concurrent test. |
| `prompt_tokens` | integer | The model server's own count of the prompt's tokens, summed across a parallel test. |
| `completion_tokens` | integer | The model server's own count of the tokens generated, summed across a parallel test. |
| `first_token_ms` | integer | How long the first piece of the answer took, in milliseconds; the mean across a parallel test. |
| `total_ms` | integer | How long the whole request took; for a parallel test, the wall time of the batch. |
| `prompt_tokens_per_sec` | number | The prompt's tokens over the time to the first token: how fast the model read. Zero for a parallel test. |
| `tokens_per_sec` | number | The tokens after the first over the time after it. For a parallel test, every token the batch generated over the batch's wall time. |
| `counted` | string | `usage` when the counts are the model server's own, from its usage event; `chunks` when the server sent none and the streamed chunks were counted instead, one a token. |

The prompts are fixed constants and are the same for every model and every
run: `pp512` is an English passage built to about 512 tokens and asks for one
token back; `tg128` is one sentence asking for a story and asks for 128 tokens
back. Neither is written to the file, and nothing of an answer is.

## The set

| Test | What it measures | The figure of note |
| --- | --- | --- |
| load | A load from disk, when `cold_load` is true | `load_ms` |
| `pp512` | Prompt processing: how fast a long prompt is read | `prompt_tokens_per_sec` |
| `tg128` | Text generation: the first token, and the rate after it | `first_token_ms`, `tokens_per_sec` |
| `tg128xN` | Generation under N concurrent requests, N being the decode concurrency | `tokens_per_sec` (the batch's), `first_token_ms` (the mean) |

The set runs when the decode concurrency is above one; at one, the concurrent
test is omitted.

`pp512` is measured through the server: the time to the first token includes
one decode step and the round trip as well as the prompt's processing, so
`prompt_tokens_per_sec` runs a little below the figure llama-bench reports
for the same model, which times the prompt alone.

## The cadence

| | |
| --- | --- |
| The idle check | once a minute |
| Idle means | no request in flight, none waiting for a load, no download running, and the last request older than the idle threshold (`idle_threshold_sec`, five minutes unless set; shared with the context probe) |
| A model is loaded only when | it fits beside what is in memory; the self-test never evicts |
| A run in progress ends when | a request is in flight on any model, a load is waiting for room, a load has been refused room since the run began, or a client's load needs the memory the run is holding |
| A run in progress checks for a client | every quarter of a second |
| A model is measured again after | one day |
| One load, and one request, may take at most | ten minutes; longer is a failed run |

## Related

- [Test your models while the Mac is idle](self-test.md) — how to switch it on
  and what it costs.
- [Reference: the request statistics store](statistics-store-reference.md) —
  the file beside this one.
