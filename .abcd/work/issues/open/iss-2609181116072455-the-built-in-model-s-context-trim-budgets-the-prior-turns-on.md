---
schema_version: 1
id: "iss-2609181116072455"
slug: "the-built-in-model-s-context-trim-budgets-the-prior-turns-on"
severity: "major"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Backends.swift"
---

The built-in model's context trim budgets the prior turns only and not the new prompt's own tokens, so a prompt that overflows the window on its own is left to the single retry to absorb.
