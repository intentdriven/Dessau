---
schema_version: 1
id: "iss-2610071035141725"
slug: "multi-token-prediction-mtp-speculative-decoding-is-waiting"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "SOTA research on MLX serving performance, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/launcher.go"
remedy: "Once an mlx-lm release includes PR 990 with its automatic switch-off under load, offer MTP as an operator setting per model on all three surfaces, after measuring acceptance on this Mac's 4-bit models; do nothing before that release."
---

Multi-token prediction (MTP) speculative decoding is waiting on upstream. ml-explore/mlx-lm PR 990 (open, last pushed 2026-09-19) reports about 1.57x decode speed for a dense Qwen3.5-27B 4-bit model on an M4 Pro at 88 percent acceptance, but only 1.09x to 1.11x on mixture-of-experts models, and it switches itself off when several requests arrive at once. MTP uses the model's own head, so no second model is loaded and Dessau's refusal of draft_model in requests can stand.
