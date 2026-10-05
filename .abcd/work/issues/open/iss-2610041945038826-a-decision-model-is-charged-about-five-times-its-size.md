---
schema_version: 1
id: "iss-2610041945038826"
slug: "a-decision-model-is-charged-about-five-times-its-size"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "tracing an unexplained model load after a peer's run, 2026-10-04, v0.9.3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
remedy: "Check what chargeLocked charges for a decision model; if it reserves a KV cache for a served chat window that the decision server never uses, charge decision models by their own measured peak instead, before itd-2610030656210408 ships."
---

A decision model is charged about five times its size against the memory budget. While the self-test loaded mlx-community/clef-4bit on 2026-10-04, /api/state showed bytes 16330982843 (about 16 GB on disk) and charge_bytes 82463242771 (about 82 GB). The charge probably includes a KV cache sized for a served chat window, which a decision model answering short typed questions never fills. An over-charge like this makes the pool refuse or delay other models for memory that is never used.
