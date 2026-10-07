---
schema_version: 1
id: "iss-2610071035147427"
slug: "batching-beyond-the-default-has-never-been-measured-on-this"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "SOTA research on MLX serving performance, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/config/config.go"
remedy: "Run the parallel self-test at decode concurrency 1, 2 and 4 on the models this Mac serves, record total throughput and time to first token per level in a dated research note, and revisit the default only on that evidence."
---

Batching beyond the default has never been measured on this Mac's models. The default decode_concurrency of 1 (decided 2026-09-20) admits two requests per model, while published figures report 2.2x total throughput for 4-way chat on a 27B model (LM Studio mlx-engine, M3 Max) and 2.6x to 3.7x for 8B and 0.6B models (vllm-mlx, M4 Max), with smaller gains for large dense models because decode is bandwidth-bound. The decision has no local evidence either way.
