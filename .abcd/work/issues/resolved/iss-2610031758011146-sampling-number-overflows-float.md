---
schema_version: 1
id: "iss-2610031758011146"
slug: "sampling-number-overflows-float"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse temperature outside [0, 100], the penalties outside [-100, 100] (repetition [0, 100]), and the context sizes outside integers in [0, 1048576]."
resolution: "temperature, the penalties and the context sizes are held to bounds, and a number too large for a float64 is refused."
impact: fix
resolved_by:
  commit: "9086fb03a99800b50bec772e36835c3eb8407e25"
---

temperature is validated only as non-negative and accepts an integer: one of 10**309 or more raises OverflowError when the sampler divides by it (sample_utils.py:273) on the generation thread. The penalties and the context sizes are likewise unbounded integers mlx may fail to convert (speculative).
