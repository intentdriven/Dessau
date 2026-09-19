---
schema_version: 1
id: "iss-2609190207534043"
slug: "client-gropiuschat-pairing-swift-s-makekey-falls-through-to"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "narrowing the Secure Enclave fallback for iss-2609190200098392"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Pairing.swift"
resolution: "makeKey now captures and reads the CFError from SecAccessControlCreateWithFlags and throws PairingError.keyRefused with its reason; a nil access control is an Enclave attempt that failed for something other than the missing entitlement, so it refuses the pairing rather than falling through to a software key. PR 110's archtest was extended to hold the second way out."
impact: fix
---

client/GropiusChat/Pairing.swift's makeKey falls through to the non-Enclave key when SecAccessControlCreateWithFlags returns nil, and that call is passed nil for its own CFError out-parameter. It is the same silent-downgrade shape as the SecKeyCreateRandomKey fallback that iss-2609190200098392 narrowed: a device whose access control cannot be built gets a software key with nothing said and nothing logged. It was left out of that fix deliberately, because failing hard there would deny pairing on a device where the ordinary path works and the condition has never been observed. It should either read the CFError and surface it, or say in the source why a fall-through is right here.

## Grounds

- pursued: we expect a refusal here to be vanishingly rare because the spike of 2026-09-19 reached SecKeyCreateRandomKey on the ad-hoc build, which means the access control was built; wrong if some signing configuration makes SecAccessControlCreateWithFlags return nil while the ordinary Keychain path still works, which would deny pairing where it used to succeed
