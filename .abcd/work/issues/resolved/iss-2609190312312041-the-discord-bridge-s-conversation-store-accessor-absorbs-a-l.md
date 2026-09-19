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
resolution: "Bridge.conversations() still answers a nil store with a throwaway one — the safe direction for a message in flight — but now writes an Info line saying the lifetime is broken, so a later change that drops the store under a live session is visible in the operator's log rather than absorbed in silence."
impact: internal
---

The Discord bridge's conversation-store accessor absorbs a lifetime bug in silence. Bridge.conversations() in internal/bridge/discord/bridge.go answers a nil store with a throwaway one, on the reasoning that only a session being built as the bridge stops can observe nil. The reasoning holds today, but the fallback means any later path that drops the store while a session is live degrades into a session writing a history nobody owns — no log line, no test, nothing an operator or a suite would notice. The condition deserves to say something rather than be asserted only in a comment.

## Grounds

- pursued: we expect a log line to be the right weight because taking the process down from a goroutine a stranger's message started is what this package refuses to do elsewhere; wrong if the condition is ever reached in normal operation, which would make the line noise.
