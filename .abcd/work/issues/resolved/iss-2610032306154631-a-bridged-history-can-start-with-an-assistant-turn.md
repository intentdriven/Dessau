---
schema_version: 1
id: "iss-2610032306154631"
slug: "a-bridged-history-can-start-with-an-assistant-turn"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of the conversation byte budget (iss-2609190312188937)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/conversation.go"
remedy: "Drop turns in user/assistant pairs, or drop any leading assistant turns after every trim, so a history always opens on the user."
resolution: "buildRequest, the one place every trimmed history passes through, starts a cut only on a user turn and its single-turn fallback sends the newest user turn, so the history sent always opens on the user"
impact: fix
resolved_by:
  commit: "852e98010705f87c5b62b8b2e2f101474276904e"
---

A bridged request's history can start with an assistant turn: the turn count, the store's byte budget and buildRequest's window trim all drop turns one at a time from the front, so after a drop the oldest kept turn may be the model's answer. Chat templates that require strict user/assistant alternation (Gemma- and Mistral-style) raise on that, and the channel's next message fails. Pre-existing through buildRequest; more frequent now that a channel keeps 32 turns rather than 64 and the budget trims across channels.
