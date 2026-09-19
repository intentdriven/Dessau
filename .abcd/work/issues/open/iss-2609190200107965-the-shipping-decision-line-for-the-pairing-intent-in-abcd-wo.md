---
schema_version: 1
id: "iss-2609190200107965"
slug: "the-shipping-decision-line-for-the-pairing-intent-in-abcd-wo"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/DECISIONS.md"
---

The shipping decision line for the pairing intent in .abcd/work/DECISIONS.md reads 'Driven end to end against the built server on one Mac: paired, pinned the fingerprint the handshake presented, answered over mutual TLS, survived a restart, revoked', which reads as though the shipping chat client drove it. The spec's Verification section says the client in that run was a throwaway Swift command-line program. The acceptance criterion asks for the check on Bob's Mac client, so which client drove the run is the whole weight of the evidence, and the decision line should say it.
