---
schema_version: 1
id: "iss-2610031758029994"
slug: "empty-completions-prompt-kills-generation"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse a /v1/completions prompt that is empty or only whitespace, and a chat request whose messages are empty."
resolution: "Refused at the gateway before a model is resolved: an empty or whitespace-only prompt, an empty prompt array, and empty messages (adr-2610040749545010)."
impact: breaking
resolved_by:
  commit: "7399d35c19b5f3b5775af92438b96c5edbce63ff"
---

An empty prompt on /v1/completions, with a tokenizer that adds no BOS, tokenizes to nothing, and insert_segments (generate.py:1669, from server.py:742) raises on an empty prompt on the generation thread (medium confidence, tokenizer dependent).

## Note 2026-10-03

Left open on purpose by the branch that closed the rest of the audit: refusing an empty prompt or empty messages means reading the conversation, which adr-2609061610102325 grants to the merge alone, and a medium-confidence raiser does not earn an exception to it. The pool's health watch (iss-2610031444343397) restarts a model server whose generation thread dies on it. Re-check at the next runtime upgrade.

## Decision 2026-10-03

Maintainer's decision: REPRODUCE FIRST, ON THE MAC. The finding is medium-confidence and needs a real model, so it is a hand step on the Mac; no code change and no ADR until then. If the empty prompt crashes the server, the gateway gets a narrow exception — it checks only whether a `/v1/completions` prompt is empty or whitespace-only, or a chat request's messages are empty, and refuses those, reading nothing else and keeping nothing — recorded as a new ADR that partly supersedes adr-2609061610102325 and adr-2609201008470380, since a ratified ADR is never edited. If it does not reproduce, this record is closed.

## Reproduction 2026-10-03

Reproduced by the maintainer on the Mac, on Dessau v0.9.3 with mlx-lm 0.32.0 and a 30B mixture-of-experts 4-bit model, every request through Dessau with `max_tokens` 8, temperature 0 and a 120-second client timeout. An ordinary `/v1/completions` prompt answered in about 0.3 s, three times. Then `"prompt": ""` got no answer within 120 s, and so did a whitespace prompt, an empty chat `messages`, and an ordinary prompt after them. The model server's process stayed alive with the same pid throughout, and Dessau reported the model loaded with 0 requests in flight. An unload ended the process and cleared it. Severity raised from minor to major on the maintainer's proposal: any client on the network can freeze a model for everyone with one request. Whether a whitespace prompt or empty `messages` freezes the server on its own is not known, because the server was already frozen when they were sent.
