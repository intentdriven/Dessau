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
---

A model server that stops answering while its process stays up goes unnoticed. In the Mac reproduction of iss-2610031758029994, after one empty prompt the mlx-lm 0.32.0 server process stayed alive with the same pid, Dessau reported the model loaded with 0 requests in flight, and every later request to that model hung until the client gave up at 120 s; only an unload recovered it. Nothing in Dessau marked the model unhealthy or bounded the wait. The health watch on main (iss-2610031444343397) stops a server only when its /health answers 503, and whether /health answered 503 in this state was not measured, so it is not known to catch it. The empty prompt itself is refused by the gateway on its own branch; this is the general case, for any other cause of the same freeze.

## Note 2026-10-04 — the symptom and the source disagree

The adversarial review of the empty-prompt fix read mlx-lm 0.32.0's server and found that an empty prompt should raise in `insert_segments` outside the generation thread's `try`, mark generation failed, turn `/health` to 503, and answer later requests at once with a "generation thread died" error rather than leaving them hanging. The reproduction saw requests hang instead, on v0.9.3, which predates the health watch. Until the measurement explains the difference, the hang may sit on Dessau's side of the relay rather than in the model server, and then other upstream failures could cause it too. The same applies to the exact prompt-cache hit (iss-2610031758026872), whose net is the health watch. The measurement should therefore also read the child's own answer to a request sent to it directly while Dessau shows the hang.
