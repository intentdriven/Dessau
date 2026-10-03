---
schema_version: 1
id: "iss-2610031811483739"
slug: "logit-bias-key-past-vocab-scatter"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/generation_refusal.go"
remedy: "Refuse logit_bias outright until keys can be checked against the model's own vocabulary."
---

logit_bias keys are accepted up to 2^31-1, but the gateway does not know the model's vocabulary: a key past it reaches logits.at[:, indices].add (sample_utils.py:116). If MLX raises, the health watch restarts the server; if it writes out of bounds, as an unchecked GPU scatter may, nothing catches it and every client of that model shares the memory.
