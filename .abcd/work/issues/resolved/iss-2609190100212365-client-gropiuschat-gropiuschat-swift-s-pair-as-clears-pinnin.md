---
schema_version: 1
id: "iss-2609190100212365"
slug: "client-gropiuschat-gropiuschat-swift-s-pair-as-clears-pinnin"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "adversarial review of the client-pairing diff before landing"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "pair(as:) remembers the pin in force before clearing it to learn a new one, and puts it back on both ways out that do not set one. Held by TestAFailedPairingRestoresThePinItCleared, which counts the clears against the restores so a third path out cannot be added without one."
impact: fix
---

client/GropiusChat/GropiusChat.swift's pair(as:) clears pinning.pinnedSPKI before the probe handshake that learns the server's key, and does not restore it when that handshake fails. A pairing that gets as far as the certificate and then cannot reach the TLS port leaves the delegate with no pin at all until the app is relaunched, so the next connection to an already-paired server would accept any certificate. The pin should be restored on every path out of pair(as:) that does not set a new one.

## Grounds

- pursued: we expect counting clears against restores in the architecture test to hold this, because the client has no test target and the fault is a path out rather than a value; wrong if the pin ever has to be cleared somewhere the count cannot see
