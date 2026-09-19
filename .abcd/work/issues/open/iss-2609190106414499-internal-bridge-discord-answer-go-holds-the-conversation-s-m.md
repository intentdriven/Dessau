---
schema_version: 1
id: "iss-2609190106414499"
slug: "internal-bridge-discord-answer-go-holds-the-conversation-s-m"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/answer.go"
---

internal/bridge/discord/answer.go holds the conversation's mutex for the whole of a generation, and runCommand's /model and /reset take the same mutex through modelOf and reset. A slash command run in a channel that is being answered therefore blocks for the whole generation — minutes, against Discord's three-second interaction deadline — so the command reports a failure to the person who ran it, the reply is posted into a token that has expired, and with only two answer workers the blocked command occupies one of them. The turns and the model need a short lock of their own, separate from the lock that says a channel is being answered.
