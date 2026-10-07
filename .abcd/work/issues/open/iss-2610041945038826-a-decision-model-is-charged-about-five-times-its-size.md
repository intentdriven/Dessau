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
remedy: "Waits on itd-2610030656210408: spc-2610030846273729 steps 2 (recognise a decision model) and 4 (charge it at weights plus its measured prefill peak) are the fix, by the maintainer's decision of 2026-10-07; no interim code, and the issue is resolved in the change that lands those steps."
---

A decision model is charged about five times its size against the memory budget. While the self-test loaded mlx-community/clef-4bit on 2026-10-04, /api/state showed bytes 16330982843 (about 16 GB on disk) and charge_bytes 82463242771 (about 82 GB). The charge probably includes a KV cache sized for a served chat window, which a decision model answering short typed questions never fills. An over-charge like this makes the pool refuse or delay other models for memory that is never used.

## Triage 2026-10-07

The cause is not a decision-model KV reservation. Nothing in `internal/` recognises a decision model yet, so clef-4bit was loaded and charged as an ordinary chat model. With no `served_context` set, `App.ServedWindow` derives the largest window the budget allows, (budget − 1.2 × weights) ÷ (per-token charge × batched requests), and `chargeLocked` charges that window back. So every model with an automatic window is charged about the whole budget. The figures fit: 60% of 128 GiB is 82,463,372,083 bytes, and the observed charge is 129,312 bytes below it, less than one token's charge. This is the property the 2026-09-21 decision line notes and leaves unchanged. Since v0.10.0, the self-test, the probe, chat and the panel's Load skip non-chat models, so the observed path is mostly closed; `/v1/completions` can still load one.

Decided by the maintainer: no stopgap. The spec for itd-2610030656210408 already commits to the right charge, so steps 2 and 4 of spc-2610030846273729 are the remedy. Revisiting how automatic windows are derived for every model was considered and not taken now, because it reopens the 2026-09-20 decision.
