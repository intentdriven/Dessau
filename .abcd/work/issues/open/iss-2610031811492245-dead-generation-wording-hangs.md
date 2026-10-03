---
schema_version: 1
id: "iss-2610031811492245"
slug: "dead-generation-wording-hangs"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
remedy: "Say every request fails until the model restarts."
---

The CHANGELOG entry, the HealthInterval and watchHealth comments and the DECISIONS line say requests to a server whose generation thread died wait for an answer that never comes; mlx-lm 0.32.0 answers them with RuntimeError('generation thread died') (server.py 1020, 1030), so every request fails rather than hangs.
