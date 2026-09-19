---
schema_version: 1
id: "iss-2609190110110353"
slug: "internal-gateway-pairing-go-s-pairhandler-takes-settingsmu-b"
severity: "major"
category: "security"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
resolution: "Both handlers read and validate the request before they take settingsMu, so a connection that sends headers and stalls blocks nothing but itself. handleRevoke already did this; PairHandler now does too."
impact: fix
---

internal/gateway/pairing.go's PairHandler takes settingsMu before pairInto reads the request body, and the plain server sets ReadHeaderTimeout but no ReadTimeout. One unauthenticated connection from the LAN that sends headers and then stalls blocks the handler inside Decode while holding the lock, which wedges every future pairing and every settings save the operator makes, for as long as the socket is open. handleRevoke beside it does this correctly: it decodes before it locks.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
