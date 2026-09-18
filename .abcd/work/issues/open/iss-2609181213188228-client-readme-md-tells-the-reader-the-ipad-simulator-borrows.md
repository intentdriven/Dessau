---
schema_version: 1
id: "iss-2609181213188228"
slug: "client-readme-md-tells-the-reader-the-ipad-simulator-borrows"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/README.md"
---

client/README.md tells the reader the iPad simulator borrows the Mac's own language model so the on-device answer can be tried there, but no such answer was ever obtained: the recorded simulator result is launch-and-stayed-running only.
