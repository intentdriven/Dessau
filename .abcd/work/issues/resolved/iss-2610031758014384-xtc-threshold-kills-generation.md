---
schema_version: 1
id: "iss-2610031758014384"
slug: "xtc-threshold-kills-generation"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse xtc_threshold outside [0, 0.5] and xtc_probability outside [0, 1] at the gateway."
resolution: "xtc_threshold outside [0, 0.5] and xtc_probability outside [0, 1] are refused at the gateway."
impact: fix
resolved_by:
  commit: "9086fb03a99800b50bec772e36835c3eb8407e25"
---

An xtc_threshold above 0.5 (the server validates it only to 1.0) with xtc_probability and temperature above zero raises in apply_xtc (sample_utils.py:248) on the batched generation thread, which dies for every client until the model restarts.
