---
schema_version: 1
id: "iss-2609181213199613"
slug: "the-thoughts-row-s-own-textselection-was-removed-and-selecti"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The Thoughts row's own textSelection was removed and selection is now per block, so a drag across two thought blocks is no longer one selection, while the row's doc comment still promises that dragging selects the text.
