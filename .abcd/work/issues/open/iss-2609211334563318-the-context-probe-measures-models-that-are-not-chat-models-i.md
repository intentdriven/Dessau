---
schema_version: 1
id: "iss-2609211334563318"
slug: "the-context-probe-measures-models-that-are-not-chat-models-i"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "live server v0.9.1 on 2026-09-21, 503 for every chat request"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/contextprobe.go"
---

The context probe measures models that are not chat models: it takes every Ready() model as a candidate (internal/app/contextprobe.go Candidates), so on the live server it picked mlx-community/GLM-OCR-bf16 (pipeline image-to-text, chat: false in the models list), spawned an mlx_lm server for it and POSTed /v1/chat/completions, which never answered (the child idle at 0% CPU answering /health in under a millisecond, in_flight 1 for over ten minutes at 'calibrating at 1024 tokens'). A served window is only meaningful for a model the server offers to chat; the probe should skip chat: false models, and a step whose child answers /health but not a completion should fail fast rather than wait for the step timeout.
