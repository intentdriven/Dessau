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

## Decision 2026-10-03

Maintainer's decision: REPRODUCE FIRST, ON THE MAC. The finding is medium-confidence and needs a real model, so it is a hand step on the Mac; no code change until then. If the exact prompt-cache hit crashes the server, the maintainer reports it to mlx-lm and the pool's health watch stays the mitigation. If it does not reproduce, this record is closed.

## Reproduction 2026-10-03

Not reproduced. On the Mac, on Dessau v0.9.3 with mlx-lm 0.32.0 and a 30B mixture-of-experts 4-bit model, the same `/v1/completions` request (`max_tokens` 8, temperature 0) was sent three times; the second and third were exact prompt-cache hits. All three answered with HTTP 200 in about 0.3 s, identically, and the model server kept the same pid. That is one model and one prompt, so it is evidence rather than proof. Closed on the maintainer's decision of 2026-10-03 ("If it does not reproduce, this record is closed"); the pool's health watch stays the net under it, and a later sighting is a new capture.
