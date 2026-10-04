---
schema_version: 1
id: "iss-2610041945020793"
slug: "discord-bridge-answers-stop-silently-at-a-fixed-1-024-token"
severity: "minor"
category: "bug"
source: "manual-test"
found_during: "live Discord checks (iss-2609190242198542), 2026-10-04, v0.9.3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/conversation.go"
remedy: "Size the bridge's max_tokens from the model's served window, as conversation.go already does for the history, instead of a fixed 1024; when an answer still stops on length, say so in the reply rather than ending silently or reporting that the model answered with nothing."
---

Discord bridge answers stop silently at a fixed 1,024-token cap. Every bridged request asks for max_tokens = answerTokens (1024, internal/bridge/discord/conversation.go, the same on main), whatever the model's served window. When the model reaches it the reply simply ends: mid-sentence, sometimes with markdown left open (an unclosed **). With a reasoning model the hidden reasoning spends the budget first. Nemotron-3.5-Lightning used 404 tokens to say a one-line hello, and one reply was 'The model answered with nothing.' (1,024 tokens, all reasoning). Seen 2026-10-04: three Nemotron answers and one Qwen3-Coder-Next answer each stopped at exactly 1,024 tokens.
