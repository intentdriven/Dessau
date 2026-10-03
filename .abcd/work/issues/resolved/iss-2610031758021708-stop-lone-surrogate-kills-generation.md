---
schema_version: 1
id: "iss-2610031758021708"
slug: "stop-lone-surrogate-kills-generation"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse a stop string whose raw JSON carries a surrogate escape that is not part of a valid pair, or that decodes to U+FFFD."
resolution: "A stop carrying a surrogate escape that is not half of a pair is refused, read from the raw bytes."
impact: fix
resolved_by:
  commit: "9086fb03a99800b50bec772e36835c3eb8407e25"
---

A stop string carrying a lone surrogate escape (\\ud800) passes the gateway's badStop, whose Go decode turns it into U+FFFD, but reaches the server as a lone surrogate, which tokenizer.encode cannot encode: it raises in _make_state_machine (server.py:711) on the generation thread.
