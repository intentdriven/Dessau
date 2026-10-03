---
schema_version: 1
id: "iss-2610031444343397"
slug: "batched-generation-thread-killable"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "adversarial review of spc-2610030846273729 step 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Audit each unguarded call in _generate against the request fields that reach it and refuse at the gateway what would raise; treat a 503 from the child's /health as a crash so the pool restarts it."
---

The pinned mlx-lm server's batched generation loop (mlx_lm/server.py _generate) has no try around the per-request setup, so any request value that raises there kills the generation thread for every client until the model restarts. The runtime upgrade refuses the two found (max_tokens below 1, a stop that is not text); the other unguarded calls (_make_sampler, _make_logits_processors, insert_segments) are not yet audited, and a child whose /health answers 503 after the thread dies is not yet treated as crashed.
