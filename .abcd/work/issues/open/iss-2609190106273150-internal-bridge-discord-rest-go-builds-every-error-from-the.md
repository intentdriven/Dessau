---
schema_version: 1
id: "iss-2609190106273150"
slug: "internal-bridge-discord-rest-go-builds-every-error-from-the"
severity: "major"
category: "security"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/rest.go"
---

internal/bridge/discord/rest.go builds every error from the method and the PATH it called, and respondToInteraction's path contains the interaction token. A Discord interaction token is a fifteen-minute bearer credential that lets its holder post and edit messages AS THE BOT without the bot token. commands.go logs that error at Debug on any failure — an expired interaction, a 429, a network blip — so the credential lands in the rotated log file. It also falsifies the file's own header comment, which says every error is built so that a failure reaching the operator's log cannot carry a credential. The error's path must be a template naming the fields rather than their values.
