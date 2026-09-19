---
schema_version: 1
id: "iss-2609190200098392"
slug: "client-gropiuschat-pairing-swift-passes-nil-for-the-cferror"
severity: "major"
category: "security"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Pairing.swift"
resolution: "makeKey now captures the CFError from the Secure Enclave attempt, reads its code, and falls back to a permanent Keychain key only for errSecMissingEntitlement (-34018), the one status the 2026-09-19 spike measured; any other refusal throws PairingError.keyRefused carrying the framework's reason inside the client's own sentence, which the pairing sheet already shows, and the kind of key actually made is logged through os.Logger. An archtest pins the shape. The Enclave path itself still needs a provisioning-profile build to observe, which the record's triage line already owes to the maintainer."
impact: fix
---

client/GropiusChat/Pairing.swift passes nil for the CFError out-parameter of the Secure-Enclave SecKeyCreateRandomKey attempt and falls back to a permanent non-Enclave Keychain key whenever that call returns nil. The recorded decision line of 2026-09-19 and the spec both say the client falls back on errSecMissingEntitlement, the one condition the spike measured. As written, any other Enclave failure — a different OSStatus, a policy change, a future signing arrangement — silently downgrades the device's key to a non-Enclave one with nothing shown to Bob and nothing logged. The error should be read and the fallback taken only on the status the decision names, with any other failure surfaced.

## Triage 2026-09-19

The code change is doable in the repository — read the CFError, take the
fallback only on errSecMissingEntitlement, surface anything else — but
confirming that the Enclave path then works needs the maintainer's hand: a
build signed from a provisioning profile, which this Mac's ad-hoc signature
cannot have, on a device that grants the Enclave. Until then the corrected
branch is reasoned rather than observed.

## Grounds

- pursued: we expect a non-Enclave pairing key to appear only where the signature cannot have the Enclave, because that is the one condition the spike measured; wrong if a device is found that refuses the Enclave with some other status where a software key would have been the right answer, which would now show as a refused pairing rather than a quiet downgrade
