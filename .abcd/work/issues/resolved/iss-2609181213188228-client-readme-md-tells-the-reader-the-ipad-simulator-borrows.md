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
resolution: "Checked on the iPad Pro 13-inch (M5) simulator running iOS 27.0, 2026-09-19: SIM=1 client/build-ipad.sh built, installed and launched the client, which stayed running, and a probe app installed into the same booted simulator — using the same FoundationModels API client/GropiusChat/Backends.swift calls — reported SystemLanguageModel.default as available and returned an answer from it. The simulator does borrow the Mac's model, so the README sentence stands and is left unchanged; the app's own send path was not driven, because this session has no touch automation for the simulator."
impact: internal
---

client/README.md tells the reader the iPad simulator borrows the Mac's own language model so the on-device answer can be tried there, but no such answer was ever obtained: the recorded simulator result is launch-and-stayed-running only.

## Grounds

- pursued: we expect the README claim to stand because the built-in model answered inside the booted simulator through the same framework the client's built-in backend uses; wrong if the client's own send path fails there where a bare framework call succeeds, or if a Mac without Apple Intelligence makes the sentence untrue as written
