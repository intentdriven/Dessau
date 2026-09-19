---
schema_version: 1
id: "iss-2609190106410918"
slug: "internal-bridge-discord-session-go-s-resumeurl-checks-only-t"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/session.go"
resolution: "resumeURL parses the value and requires its host to be the gateway the bridge was pointed at or a host under Discord's own domains, as well as refusing a downgrade. TestAResumeMayNotDowngradeTheTransport covers both."
impact: fix
---

internal/bridge/discord/session.go's resumeURL checks only the SCHEME of the resume_gateway_url the READY payload names. A value of wss://somewhere-else is accepted, stored, dialled on the next attempt, and sent an opResume carrying the bot token — so the one field in the protocol that can redirect the credential is checked for everything except where it points. The peer that would have to supply it is the TLS-verified Discord gateway, which makes it hardening rather than a live hole, but it is cheap to close: parse the URL and require its host to match the gateway the bridge was pointed at, or a discord.gg / discord.com suffix.

## Grounds

- pursued: we expect the fix to hold because a test watched to fail covers it; wrong if the same class of defect appears at a seam this change did not touch
