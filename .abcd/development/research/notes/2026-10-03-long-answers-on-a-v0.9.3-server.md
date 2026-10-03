# Long answers on a v0.9.3 server: run lines (2026-10-03)

Measured by a peer session against a v0.9.3 Dessau server on loopback, at
temperature 0, while it consumed the gateway from a local classification tool.
The task is a classification over a list of items: 16 items (about 0.8k
prompt tokens) and 44 items (about 2.5k). Evidence for iss-2610030652514762
(an omitted `max_tokens` capped at 512) and iss-2610030919536329 (a non-streamed
answer ending in a 504 after ten minutes). "None known" means the panel was not
checked at the time.

Fields: model · stream · max_tokens · prompt tokens · completion tokens ·
finish_reason · HTTP · elapsed s · served_context · other requests in flight.

## 16 items

| model | stream | max_tokens | prompt | completion | finish | HTTP | s | served | in flight |
|---|---|---|---|---|---|---|---|---|---|
| GLM-4.7-Flash-8bit | false | 4000 | 789 | 3324 | stop | 200 | 67.4 | 29190 | none known |
| Nemotron-3.5-Lightning-30B-A3B-4bit | false | 4000 | 857 | 4000 | length | 200 | 32.9 | 57384 | none known |
| Nemotron-3.5-Lightning-30B-A3B-4bit | false | absent | 857 | 512 | length | 200 | 6.2 | 57384 | none known |
| Nemotron-3.5-Lightning-30B-A3B-4bit | false | 55336 | 857 | 7886 | stop | 200 | 67.3 | 57384 | none known |
| Qwen3.8-27B-8bit | false | 33829 | 892 | 3094 | stop | 200 | 210.5 | 35877 | none known |
| Ternary-Bonsai-27B-mlx-2bit | false | 53065 | 850 | 4438 | stop | 200 | 122.2 | 55113 | none known |

## 44 items

| model | stream | max_tokens | prompt | completion | finish | HTTP | s | served | in flight |
|---|---|---|---|---|---|---|---|---|---|
| GLM-4.7-Flash-8bit | false | 25094 | 2550 | 8872 | stop | 200 | 181.0 | 29190 | none known |
| Nemotron-3.5-Lightning-30B-A3B-4bit | false | 53288 | ~2.6k | n/a | n/a | 504 | not timed (≥600 by the gateway's floor) | 57384 | none known |
| Qwen3.8-27B-8bit | false | 31781 | ~2.6k | n/a | n/a | 504 | not timed | 35877 | likely Nemotron's abandoned decode |
| Nemotron-3.5-Lightning-30B-A3B-4bit | true | 53288 | ~2.6k | pending | pending | 200, stream open | >2,700 so far | 57384 | likely Qwen's abandoned decode at the start |

A request whose prompt the server counted at about 29,326 tokens against GLM's
served 29,190 was refused with a 400 naming the served window, in under five
seconds: the served-window check behaving as documented.

## What the lines show

- The 512 cap on an omitted `max_tokens` (row 3 of the first table), the fix
  for which launches with the served window.
- Two non-streamed 504s for reasoning models at large `max_tokens`, and a
  streamed run of the same request past 45 minutes with no 504: the header
  wait bounds the whole of a non-streamed answer.
- A 504'd request's decode plausibly overlapping the next request.
- A reasoning model given a context-sized limit can reason for a very long
  time (the streamed run); a bound on long reasoning runs, if any, is a product
  decision, not recorded here.
