---
schema_version: 1
id: "iss-2610031811486498"
slug: "accept-logit-bias-against-vocab"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/registry/registry.go"
remedy: "Record the vocabulary size among a model's facts and check logit_bias keys against it after resolution; accept logit_bias again."
---

With logit_bias refused outright, a client that relies on it (banning a token, forcing a choice) loses it. The gateway could accept it again if the registry recorded each model's vocabulary size (config.json vocab_size, or text_config's) and the refusal held keys below it after the model is resolved, minding a padded vocabulary larger than the tokenizer.

## Decision 2026-10-03

Maintainer's decision: NOT NOW. `logit_bias` stays refused. The record stays open as future work; the vocabulary-size check is built only when a client needs `logit_bias`.
