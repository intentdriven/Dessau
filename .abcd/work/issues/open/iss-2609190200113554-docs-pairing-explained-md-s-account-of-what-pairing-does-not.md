---
schema_version: 1
id: "iss-2609190200113554"
slug: "docs-pairing-explained-md-s-account-of-what-pairing-does-not"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/pairing-explained.md"
---

docs/pairing-explained.md's account of what pairing does not protect against writes the compromise item about the device holding the client key. The acceptance criterion asks that the reader be told neither end is protected from a Mac that is already compromised, and the server's Mac — which holds the server key, the paired set and the shared API key — is the end the page does not mention. One clause would close it.
