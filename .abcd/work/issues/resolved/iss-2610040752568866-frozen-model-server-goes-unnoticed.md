---
schema_version: 1
id: "iss-2610040752568866"
slug: "frozen-model-server-goes-unnoticed"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "the maintainer's Mac reproduction of iss-2610031758029994 on 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
remedy: "First measure on the Mac what /health answers while a server is frozen this way. If it answers 503, the health watch already covers it and this closes. If not, decide how Dessau notices a server that accepts requests and never answers, such as a bound on the time to the first byte from the model server, and what it does then."
resolution: "Measured on the Mac on 2026-10-04: a frozen mlx-lm 0.32.0 server answers /health with 503, and main's health watch (1ff92cc9) stops it within one tick, so the next request starts a fresh server. The hang seen on v0.9.3 predates both the watch and the internal streaming."
impact: internal
resolved_by:
  commit: "1ff92cc9"
---

A model server that stops answering while its process stays up goes unnoticed. In the Mac reproduction of iss-2610031758029994, after one empty prompt the mlx-lm 0.32.0 server process stayed alive with the same pid, Dessau reported the model loaded with 0 requests in flight, and every later request to that model hung until the client gave up at 120 s; only an unload recovered it. Nothing in Dessau marked the model unhealthy or bounded the wait. The health watch on main (iss-2610031444343397) stops a server only when its /health answers 503, and whether /health answered 503 in this state was not measured, so it is not known to catch it. The empty prompt itself is refused by the gateway on its own branch; this is the general case, for any other cause of the same freeze.

## Note 2026-10-04 — the symptom and the source disagree

The adversarial review of the empty-prompt fix read mlx-lm 0.32.0's server and found that an empty prompt should raise in `insert_segments` outside the generation thread's `try`, mark generation failed, turn `/health` to 503, and answer later requests at once with a "generation thread died" error rather than leaving them hanging. The reproduction saw requests hang instead, on v0.9.3, which predates the health watch. Until the measurement explains the difference, the hang may sit on Dessau's side of the relay rather than in the model server, and then other upstream failures could cause it too. The same applies to the exact prompt-cache hit (iss-2610031758026872), whose net is the health watch. The measurement should therefore also read the child's own answer to a request sent to it directly while Dessau shows the hang.

## Measurement 2026-10-04 — /health answers 503; the health watch catches it

Measured on the maintainer's Mac against a scratch build of main at 9e5cad9d
(v0.9.3-313-g9e5cad9d), with its own data root on loopback, the same mlx-lm
0.32.0 pin and Nemotron-3.5-Lightning-30B-A3B-4bit:

- The empty prompt went straight to the child. The gateway's refusal (#196)
  had not landed in that build. The client got HTTP 502 after 1.2 s ("the
  model server stopped part-way through the answer"), not a hang.
- About 7 s later the log read "model server stopped: its generation thread
  has died, so it would answer no request; the next request starts it
  again", then `model unloaded reason=crashed`. That line is written only
  when the child's `/health` answers 503 (`internal/runtime/pool.go`, the
  health watch). So `/health` did answer 503 in this state.
- The next normal request answered with HTTP 200 in 2.1 s, from a fresh server
  process.

Not measured:
- A request sent to the child directly. The installed v0.9.3 serves from
  another account, whose socket this account cannot open.
- A normal request sent inside the up-to-10 s window before the watch fires.

The hang seen on 2026-10-03 was on v0.9.3, which has neither the health watch
nor the internal streaming. On main the failing request is answered at once
and the server is replaced within one health tick.

Follow-up runs on v0.9.3 the same day narrowed the cause:
- A whitespace-only prompt answers normally.
- Empty messages are refused by mlx-lm itself with 404 (the gateway now
  refuses them with 400).
- Array prompts `["hi"]` and `[""]` freeze the server exactly as the empty
  prompt did (iss-2610040805353721).
- A token array `[72,73]` is refused by mlx-lm with 404 and leaves the server
  healthy.
