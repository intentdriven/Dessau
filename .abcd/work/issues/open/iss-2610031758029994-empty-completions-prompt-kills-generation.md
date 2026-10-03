---
schema_version: 1
id: "iss-2610031758029994"
slug: "empty-completions-prompt-kills-generation"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse a /v1/completions prompt that is empty or only whitespace, and a chat request whose messages are empty."
---

An empty prompt on /v1/completions, with a tokenizer that adds no BOS, tokenizes to nothing, and insert_segments (generate.py:1669, from server.py:742) raises on an empty prompt on the generation thread (medium confidence, tokenizer dependent).
