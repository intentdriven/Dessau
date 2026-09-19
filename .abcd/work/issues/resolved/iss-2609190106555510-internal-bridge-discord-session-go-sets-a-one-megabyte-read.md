---
schema_version: 1
id: "iss-2609190106555510"
slug: "internal-bridge-discord-session-go-sets-a-one-megabyte-read"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session.go"
resolution: "Identify asks for a large_threshold of 50, the gateway read limit is four megabytes, and a close for a message too big stops the bridge with the reason on the panel instead of being retried forever."
impact: fix
---

internal/bridge/discord/session.go sets a one-megabyte read limit on the gateway connection and asserts in a comment that a READY payload is comfortably inside it. For a bot in many guilds it is not — the guild array alone can exceed a megabyte — and exceeding the limit closes the connection, which the bridge treats as weather and retries forever with the same limit, leaving a permanent reconnect loop with one Debug line to show for it. Identify asks for no large_threshold, which is the field that bounds what READY carries. Ask for a small large_threshold, raise the limit, and treat a close for a message too big as a reason to stop with something on the panel rather than as weather.

## Grounds

- pursued: we expect the fix to hold because a test watched to fail covers it; wrong if the same class of defect appears at a seam this change did not touch
