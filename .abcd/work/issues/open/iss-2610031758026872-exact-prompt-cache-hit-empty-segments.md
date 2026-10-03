---
schema_version: 1
id: "iss-2610031758026872"
slug: "exact-prompt-cache-hit-empty-segments"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
remedy: "Rely on the pool's health watch to restart a server whose generation thread died; report the empty-segment case upstream to mlx-lm and re-check at the next runtime upgrade."
---

A prompt that exactly matches a cached one leaves fetch_nearest_cache with nothing left to process (models/cache.py:1656); server.py:723-730 then pops every segment and insert_segments raises on an empty prompt on the generation thread. A /v1/completions request replaying a cached prompt reaches it (medium confidence). The gateway cannot see the cache.
