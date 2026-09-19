---
schema_version: 1
id: "iss-2609190312064731"
slug: "every-channel-s-conversation-survives-a-fatal-stop-of-the-di"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "adversarial security review of the conversation-store change"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/bridge.go"
---

Every channel's conversation survives a fatal stop of the Discord bridge, indefinitely, and the operator cannot clear it. internal/bridge/discord/bridge.go drops the conversation store only in stopLocked, which is reached from Apply and Close; the fatal exit in run() — Discord refusing the token (4004), the intents (4014), 4010-4013, or a READY over the read limit — logs, sets StateStopped and returns without going near it. The goroutine is gone, the panel reads stopped, and every stranger's message text stays on the Bridge for the life of the process. A token being revoked or regenerated is the ordinary case. It is also a regression of the change that moved the store off the session (iss-2609190241509478), where the fatal path collected it with the session object, and it falsifies the package doc's own sentence that a channel's history goes when the bridge stops. The store's life should be exactly the run goroutine's, which covers every way the bridge stops.
