---
schema_version: 1
id: "iss-2609190100217781"
slug: "client-gropiuschat-pairing-swift-s-pairingstore-forget-delet"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial review of the client-pairing diff before landing"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Pairing.swift"
resolution: "The pairing record now carries the certificate's DER, and PairingStore gains removeLeaf, which deletes by kSecValueRef — the only thing that works, since the macOS file keychain replaces a chosen label with the certificate's common name. forget() removes the certificate first, and a pairing that cannot finish its handshake removes the one it just stored. Held by TestForgettingAPairingRemovesTheCertificateToo."
impact: fix
---

client/GropiusChat/Pairing.swift's PairingStore.forget() deletes the pairing record, the identity and the key by application tag, but the server-minted certificate was added with no tag of its own (store(leaf:) passes only kSecClass and kSecValueRef), so forgetting a pairing leaves the certificate in the user's keychain for ever. Pairing again adds another. The spike of 2026-09-19 established that the macOS file keychain overwrites a certificate's chosen label with its common name, so the certificate has to be deleted by kSecValueRef — which means keeping the DER, or finding the certificate through the identity before the key goes.

## Grounds

- pursued: we expect keeping the DER beside the record to make the certificate removable, because the spike established it cannot be found by a label this app chose; wrong if a keychain that already holds a certificate from an older build leaves one behind, which nothing here can reach
