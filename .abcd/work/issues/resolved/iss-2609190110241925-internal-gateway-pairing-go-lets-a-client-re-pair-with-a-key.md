---
schema_version: 1
id: "iss-2609190110241925"
slug: "internal-gateway-pairing-go-lets-a-client-re-pair-with-a-key"
severity: "major"
category: "security"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
resolution: "A pairing whose stored row would be identical answers without saving, so re-pairing under a key already held writes nothing. Held by TestRepairingWithAnUnchangedRowSavesNothing, which asserts the settings file's modification time does not move over five repeats and does move when the name changes."
impact: fix
---

internal/gateway/pairing.go lets a client re-pair with a key it is already paired under, which skips the ceiling check and performs a full config.Save each time — temp file, fsync, rename — with no rate limit and no authentication. Sustained, that is a remote fsync loop on the operator's settings file, holding settingsMu throughout. A pairing whose stored row would be unchanged should not be saved at all.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
