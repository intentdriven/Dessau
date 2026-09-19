---
schema_version: 1
id: "iss-2609190106402401"
slug: "internal-bridge-discord-session-go-assigns-the-bot-s-own-use"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session.go"
resolution: "The bot's own user id and the application id are carried on the resume state rather than on the connection, so they survive a RESUMED that does not carry them. TestAMentionIsStillAnsweredAfterAResume holds it; with the identity scoped to the connection it reports no POST to the channel arriving."
impact: fix
---

internal/bridge/discord/session.go assigns the bot's own user id and the application id only in the READY branch, and a session object is built fresh per connection. Discord does not send READY on a resume — it sends RESUMED — so after the first resume botID is empty, mentionsUs returns false for every message, and onMessage drops every guild message. The bridge goes on answering direct messages and silently stops answering every mention in every channel, with nothing in the log and the panel reading connected. A resume is the normal path after a Wi-Fi blip, a lid close or an op-7 Reconnect, so this is the steady state rather than an edge. The identifiers have to survive a resume, and a test has to send a guild mention after RESUMED.

## Grounds

- pursued: we expect the fix to hold because a test watched to fail covers it; wrong if the same class of defect appears at a seam this change did not touch
