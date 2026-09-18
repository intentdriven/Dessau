---
schema_version: 1
id: "iss-2609180608291870"
slug: "a-served-model-id-of-a-bare-slash-crashes-the-client-split-s"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Backends.swift"
resolution: "Both call sites map the split's last component with a fallback to the id."
impact: fix
---

A served model id of a bare slash crashes the client: split(separator:).last! on remote input in Answerer.displayName and MessageRow.speaker.
