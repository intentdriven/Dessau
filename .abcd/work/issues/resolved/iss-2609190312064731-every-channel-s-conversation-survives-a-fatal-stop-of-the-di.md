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
resolution: "The conversation store's life is now exactly the life of the bridge's goroutine: made before it starts and dropped in a defer as it ends, which covers the switch, a token change, the app closing, and the fatal exit the run loop takes when Discord refuses the credentials — the path that reached no teardown at all. The defer is registered after the one that closes the done channel so it runs before it, which is what keeps a dying goroutine from dropping the store the next start has just made. Test watched to fail: a 4004 close on a live session left one channel's conversation held."
impact: fix
---

Every channel's conversation survives a fatal stop of the Discord bridge, indefinitely, and the operator cannot clear it. internal/bridge/discord/bridge.go drops the conversation store only in stopLocked, which is reached from Apply and Close; the fatal exit in run() — Discord refusing the token (4004), the intents (4014), 4010-4013, or a READY over the read limit — logs, sets StateStopped and returns without going near it. The goroutine is gone, the panel reads stopped, and every stranger's message text stays on the Bridge for the life of the process. A token being revoked or regenerated is the ordinary case. It is also a regression of the change that moved the store off the session (iss-2609190241509478), where the fatal path collected it with the session object, and it falsifies the package doc's own sentence that a channel's history goes when the bridge stops. The store's life should be exactly the run goroutine's, which covers every way the bridge stops.

## Grounds

- pursued: we expect a store tied to the goroutine to cover every stop because the goroutine is the bridge running; wrong if a future path were to want the conversations to outlive a stop, which nothing in the intent asks for.
