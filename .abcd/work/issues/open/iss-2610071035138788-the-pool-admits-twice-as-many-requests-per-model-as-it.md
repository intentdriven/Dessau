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
remedy: "Maintainer's decision 2026-10-07: admit what is charged. Size each model's semaphore to DecodeConcurrency, not twice it, so the budget's charge covers every request admitted; at the default of 1 a second request to the same model waits. Add a test that the charge covers every request the semaphore admits."
---

The pool admits twice as many requests per model as it charges memory for. chargeLocked in internal/runtime/pool.go charges KV for DecodeConcurrency sequences, but each loaded model's semaphore admits 2 x DecodeConcurrency requests at once (the sem in startLocked), and at the default of 1 mlx-lm is launched without --decode-concurrency, so its own batch limit (32) lets both admitted requests decode together. Two long requests on one model can therefore use about twice the KV the budget reserved.
