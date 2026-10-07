---
schema_version: 1
id: "iss-2610071035138788"
slug: "the-pool-admits-twice-as-many-requests-per-model-as-it"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "SOTA research on MLX serving performance, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
remedy: "Make the charge and the admission agree: either size the semaphore to DecodeConcurrency or charge 2 x DecodeConcurrency sequences, choosing by measurement of what the batching gains; add a test that the charge covers every request the semaphore admits."
---

The pool admits twice as many requests per model as it charges memory for. chargeLocked in internal/runtime/pool.go charges KV for DecodeConcurrency sequences, but each loaded model's semaphore admits 2 x DecodeConcurrency requests at once (the sem in startLocked), and at the default of 1 mlx-lm is launched without --decode-concurrency, so its own batch limit (32) lets both admitted requests decode together. Two long requests on one model can therefore use about twice the KV the budget reserved.
