---
schema_version: 1
id: "iss-2610031758018556"
slug: "logit-bias-out-of-range"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse logit_bias with more than 300 entries, a key that is not a decimal integer in [0, 2^31), or a value outside [-100, 100]; a key past a padded vocabulary needs the runtime's own guard (left to the health watch)."
resolution: "logit_bias is held to 300 int32 token ids and values in [-100, 100]; a key past a padded vocabulary is left to the health watch."
impact: fix
resolved_by:
  commit: "9086fb03a99800b50bec772e36835c3eb8407e25"
---

logit_bias is forwarded unchecked: a key beyond int64 raises building the index array (setup, speculative); a key at or beyond the vocabulary, or negative, indexes past the logits (step, speculative), and on a padded SentencePiece vocabulary a biased key past the tokenizer's length raises in the detokenizer (tokenizer_utils.py:165); and a dict of millions of keys costs a scatter per token.
