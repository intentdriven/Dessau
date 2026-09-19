---
schema_version: 1
id: "iss-2609190114530255"
slug: "adr-2609182357322050-decision-3-and-docs-pairing-explained-m"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/pairing-explained.md"
resolution: "Said out loud in internal/pairing/registry.go and in docs/pairing-explained.md, and recorded as a decision line: the key is the credential and the minted leaf is the wrapper, so a paired key is admitted under a certificate the client signed itself and revoking on the panel is the only answer to a key that escapes."
impact: internal
---

adr-2609182357322050 decision 3 and docs/pairing-explained.md do not say that the server-minted certificate is not what admits a client. The paired lookup is on the key's SPKI and no chain is built, so anyone holding a paired private key is admitted under a certificate they signed themselves, with any name and any serial they like. That is sound — possession is proved by the handshake — but it is what makes a leaked or wrongly enrolled key immediately usable without the minted leaf, and TestASelfSignedLeafCarryingAPairedNameIsRefused reads as though it covered the case when it covers only an UNPAIRED key.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
