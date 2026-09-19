---
schema_version: 1
id: "iss-2609190110110994"
slug: "internal-gateway-pairing-go-writes-the-new-client-into-the-l"
severity: "critical"
category: "security"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
resolution: "The pairing is written into a clone, so a save that fails leaves the live registry untouched and a 500 admits nobody. Held by TestAPairingThatCouldNotBeSavedIsNotAdmitted, which makes the settings directory unwritable and asserts the registry does not know the key afterwards."
impact: fix
---

internal/gateway/pairing.go writes the new client into the live configuration map before App.SetConfig is called, so a save that fails answers 500 while the client stays admitted by the live registry until the process restarts. The attacker does not need the minted certificate to use it: the paired lookup is on the key, so any certificate carrying that key is accepted. A 500 on the pairing endpoint is therefore full access to the TLS port with no API key, for the lifetime of the process — fail-open on the error path of a trust boundary.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
