---
schema_version: 1
id: "iss-2609190226050845"
slug: "cmd-gropius-main-go-sets-readheadertimeout-and-idletimeout-o"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial security review of the pairing audit fixes branch"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/gropius/main.go"
---

cmd/gropius/main.go sets ReadHeaderTimeout and IdleTimeout on the listeners but no ReadTimeout, so the body of a request is bounded in bytes and not in time. An unauthenticated peer posting to /pair can trickle a pairing body a byte at a time and hold a goroutine inside the read indefinitely, and the same holds for any other body the gateway reads. It no longer wedges pairings or settings saves, since the read was moved out from under settingsMu, so the cost is a goroutine and a connection per stalled request rather than the whole control plane. The pairing endpoint is the sharp case because it asks for no credential.
