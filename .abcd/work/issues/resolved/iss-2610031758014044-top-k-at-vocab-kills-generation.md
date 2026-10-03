---
schema_version: 1
id: "iss-2610031758014044"
slug: "top-k-at-vocab-kills-generation"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse a top_k that is not an integer in [0, 1024] (config.MaxTopK), far below every shipped vocabulary."
duplicates: [iss-2610031758014384]
resolution: "top_k outside [0, 1024] is refused at the gateway."
impact: fix
resolved_by:
  commit: "9086fb03a99800b50bec772e36835c3eb8407e25"
---

A top_k at or above the model's vocabulary size, with temperature above zero, raises in apply_top_k (sample_utils.py:150) on the batched generation thread, which dies for every client. The gateway does not know the vocabulary.
