---
schema_version: 1
id: "iss-2609190312312041"
slug: "the-discord-bridge-s-conversation-store-accessor-absorbs-a-l"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial security review of the conversation-store change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/bridge.go"
---

The Discord bridge's conversation-store accessor absorbs a lifetime bug in silence. Bridge.conversations() in internal/bridge/discord/bridge.go answers a nil store with a throwaway one, on the reasoning that only a session being built as the bridge stops can observe nil. The reasoning holds today, but the fallback means any later path that drops the store while a session is live degrades into a session writing a history nobody owns — no log line, no test, nothing an operator or a suite would notice. The condition deserves to say something rather than be asserted only in a comment.
