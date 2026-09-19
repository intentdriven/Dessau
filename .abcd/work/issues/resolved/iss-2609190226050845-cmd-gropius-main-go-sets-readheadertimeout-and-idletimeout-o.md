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
resolution: "Both listeners are now built from one listenerServer constructor in cmd/gropius/listeners.go that sets a 60s ReadTimeout alongside the existing ReadHeaderTimeout and IdleTimeout, so a peer that sends headers and then stalls the body is cut off instead of holding a goroutine and a connection. WriteTimeout stays unset: the answer is not what is bounded."
impact: fix
---

cmd/gropius/main.go sets ReadHeaderTimeout and IdleTimeout on the listeners but no ReadTimeout, so the body of a request is bounded in bytes and not in time. An unauthenticated peer posting to /pair can trickle a pairing body a byte at a time and hold a goroutine inside the read indefinitely, and the same holds for any other body the gateway reads. It no longer wedges pairings or settings saves, since the read was moved out from under settingsMu, so the cost is a goroutine and a connection per stalled request rather than the whole control plane. The pairing endpoint is the sharp case because it asks for no credential.

## Grounds

- pursued: we expect a whole-request read bound to cost streaming nothing because net/http clears the connection read deadline when it starts its background read, which happens as soon as the body is read to EOF and immediately for a bodiless request; wrong if a handler streams a long answer while its request body is still unread, or if 60s is short enough to refuse a legitimate 32 MiB body on a slow link
