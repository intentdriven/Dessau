---
schema_version: 1
id: "iss-2610070546301424"
slug: "kolibri-1-velaia-kolibri-1-mlx-4bit-on-huggingface-aleph"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "abcd:intent assessment of Kolibri 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/mlx-requirements.txt"
remedy: "Once an mlx-lm release includes ml-explore/mlx-lm PR 1945, bump mlx-lm in internal/runtime/mlx-requirements.txt (uv pip compile --generate-hashes), then verify a real Kolibri 1 load on a large-memory Mac; never vendor or run the repo's kolibri1.py."
---

Kolibri 1 (velaia/Kolibri-1-MLX-4bit on HuggingFace, Aleph Alpha's 78B MoE at 4-bit, 44.3 GB) cannot be served: its config.json names model_type kolibri1, which the pinned mlx-lm 0.32.0 does not ship, so the load fails with an unsupported-model-type error. The repo bundles kolibri1.py and a run.py that registers it, but config.json sets no model_file, so CheckModelCode passes and Dessau correctly never runs that file (iss-2610030709283687). Support is upstream in ml-explore/mlx-lm PR 1945, open and unreviewed as of 2026-10-07. Assessed from the model card, config.json and the code; no load was attempted. reasoning_effort already reaches the chat template through chat_template_kwargs. Expect a Mac with 64 GB or more.
