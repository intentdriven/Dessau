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

## Note 2026-10-03

Left open on purpose by the branch that closed the rest of the audit: refusing an empty prompt or empty messages means reading the conversation, which adr-2609061610102325 grants to the merge alone, and a medium-confidence raiser does not earn an exception to it. The pool's health watch (iss-2610031444343397) restarts a model server whose generation thread dies on it. Re-check at the next runtime upgrade.

## Decision 2026-10-03

Maintainer's decision: REPRODUCE FIRST, ON THE MAC. The finding is medium-confidence and needs a real model, so it is a hand step on the Mac; no code change and no ADR until then. If the empty prompt crashes the server, the gateway gets a narrow exception — it checks only whether a `/v1/completions` prompt is empty or whitespace-only, or a chat request's messages are empty, and refuses those, reading nothing else and keeping nothing — recorded as a new ADR that partly supersedes adr-2609061610102325 and adr-2609201008470380, since a ratified ADR is never edited. If it does not reproduce, this record is closed.
