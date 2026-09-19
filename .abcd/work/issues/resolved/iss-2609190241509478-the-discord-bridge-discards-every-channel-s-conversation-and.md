---
schema_version: 1
id: "iss-2609190241509478"
slug: "the-discord-bridge-discards-every-channel-s-conversation-and"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session.go"
resolution: "The per-channel conversations now live on the Bridge rather than on one gateway session: the store is made when the bridge starts, borrowed by every session, and dropped when the bridge stops, so a resume or a fresh identify after a drop keeps the channel's history and the model /model chose. A fake-gateway test drives a pick, a message, a forced reconnect and a second message, and a stop-then-start asserts the conversation is gone."
impact: fix
---

The Discord bridge discards every channel's conversation and its /model choice on every reconnect, not only when the bridge is switched off. internal/bridge/discord/session.go:176 builds newConversations() inside runSession, so the conversations belong to one gateway connection; the comment at session.go:191 acknowledges it as the bound the intent asks for. But the intent's scope condition cond-2609181018490128 says a conversation is gone when the bridge stops, and acceptance criterion ac-10 says the bridge resumes by itself after Alice's Mac sleeps — on that exact path the bot reconnects, says nothing, and has silently forgotten the channel's history and the model Bob chose with /model. Hoisting the conversations onto Bridge, or passing them into runSession, would hold them across a resume without touching the protocol; the disk promise is unaffected because they stay in memory and still go when the bridge stops.

## Grounds

- pursued: we expect a channel's history and model to survive a reconnect because the intent's scope condition bounds a conversation by the bridge stopping, not by the socket dropping; wrong if a maintainer decides a dropped session should start the conversation again.
