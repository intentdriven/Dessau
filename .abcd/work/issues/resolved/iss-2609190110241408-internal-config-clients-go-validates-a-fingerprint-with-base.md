---
schema_version: 1
id: "iss-2609190110241408"
slug: "internal-config-clients-go-validates-a-fingerprint-with-base"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/config/clients.go"
resolution: "Fingerprints are validated with base64.StdEncoding.Strict(), so only the canonical spelling of a hash is one; and EffectiveTLSPort returns no port for a server on 65535 rather than 65536. Held by TestOnlyACanonicalFingerprintIsAFingerprint and TestThereIsNoPortAfterTheLastOne."
impact: fix
---

internal/config/clients.go validates a fingerprint with base64.StdEncoding, which is not strict: sixteen distinct 44-character strings decode to the same 32-byte hash and all are accepted from a hand-edited config.json. Not exploitable — every lookup key is produced by pairing.Fingerprint and is canonical, so a non-canonical row never matches and fails closed — but base64.StdEncoding.Strict() costs nothing and removes the question. Separately, Config.EffectiveTLSPort returns Port+1 without bounding it, so a server on port 65535 answers and advertises 65536, which is not a port.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
