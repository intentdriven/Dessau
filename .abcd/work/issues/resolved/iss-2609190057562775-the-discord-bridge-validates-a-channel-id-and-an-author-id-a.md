---
schema_version: 1
id: "iss-2609190057562775"
slug: "the-discord-bridge-validates-a-channel-id-and-an-author-id-a"
severity: "major"
category: "security"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/commands.go"
resolution: "Every identifier the bridge puts in a REST URL now goes through validID: the application id from READY (no commands are registered without one, and the log says so), the interaction id, and the message id Discord returns for a placeholder (the answer continues in new messages rather than editing something that is not a message). The interaction token, which is not a snowflake, is percent-escaped where it is used."
impact: fix
---

The Discord bridge validates a channel id and an author id as snowflakes before putting them in a REST URL, but not the three other identifiers it also puts in a URL path: the interaction id and the interaction token in internal/bridge/discord/commands.go (POST /interactions/{id}/{token}/callback), the application id taken from the READY payload in internal/bridge/discord/session.go (PUT /applications/{app_id}/commands), and the message id read back from Discord's own create-message response in internal/bridge/discord/editor.go (PATCH /channels/{c}/messages/{id}). All three arrive over the network from the other end of the connection. An id of '../../applications/X' would traverse out of the path the bridge meant to call and make an authenticated request, as the bot, against an endpoint it never intended — bulk-overwriting the application's commands, for instance. Every identifier this bridge puts in a URL should go through validID, and the interaction token, which is not a snowflake, should be percent-escaped.

## Grounds

- pursued: we expect a shape check at every point a network-supplied value enters a path to be enough because the bridge builds every URL from a fixed prefix plus these values; wrong if a later call interpolates a value this list does not cover
