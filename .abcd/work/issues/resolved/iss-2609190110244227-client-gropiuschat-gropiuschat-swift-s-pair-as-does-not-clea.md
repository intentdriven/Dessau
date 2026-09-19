---
schema_version: 1
id: "iss-2609190110244227"
slug: "client-gropiuschat-gropiuschat-swift-s-pair-as-does-not-clea"
severity: "major"
category: "security"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "pair(as:) forgets the last handshake before the probe it learns its pin from, and requires that probe to have reached the server, so a probe that never connects cannot be mistaken for one that did. Held by the extended TestTheClientPinsWhatTheHandshakePresented."
impact: fix
---

client/GropiusChat/GropiusChat.swift's pair(as:) does not clear pinning.lastPresentedSPKI before the probe handshake it learns the pin from, and discards the probe's error. If the probe never completes a handshake — the TLS port unreachable, or the wrong port — the guard succeeds with the fingerprint of the previous server this app run handshook with, and the client writes a pairing for one server pinning another's key. Settings then shows that value under 'That server's key', which is the one human comparison the whole design rests on.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
