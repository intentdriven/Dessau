---
schema_version: 1
id: "iss-2609190110118690"
slug: "internal-gateway-pairing-go-s-pairhandler-decodes-the-body-w"
severity: "major"
category: "security"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
resolution: "readPairRequest requires Content-Type application/json and refuses any request carrying an Origin, so a POST a browser will send without a preflight is not a pairing. Held by TestThePairingEndpointIsNotAFormAPageCanPost over the four shapes a page can produce."
impact: fix
---

internal/gateway/pairing.go's PairHandler decodes the body whatever the Content-Type is and checks no Origin, so POST with Content-Type text/plain is CORS-simple, sends no preflight, and any web page a person on the LAN visits can enrol an attacker's public key into config.json. The attacker never needs to read the answer, because the paired lookup is on the key and they can sign their own certificate for it. That widens the first-come window adr-2609182357322050 accepted from hosts on the LAN to any website anybody on the LAN visits, which is not what was accepted and is the attack class withAuth and loopbackOnly already defend against elsewhere in this tree.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
