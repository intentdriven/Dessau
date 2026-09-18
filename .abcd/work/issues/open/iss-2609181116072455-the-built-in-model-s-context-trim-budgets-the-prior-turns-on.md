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
deferred_after: "v0.7.1"
deferral_reason: "Left open at the 0.7.1 cut: an observation from the 2026-09-18 fidelity audits that needs the maintainer's decision; recorded here rather than stepped over."
---

The built-in model's context trim budgets the prior turns only and not the new prompt's own tokens, so a prompt that overflows the window on its own is left to the single retry to absorb.

## Triage 2026-09-18

Left open for the maintainer: budgeting the new prompt's own tokens changes what the window holds and when a turn is refused outright rather than retried — a policy call on the built-in model, not a mechanical correction.
