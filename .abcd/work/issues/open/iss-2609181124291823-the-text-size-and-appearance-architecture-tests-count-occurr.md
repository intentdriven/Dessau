---
schema_version: 1
id: "iss-2609181124291823"
slug: "the-text-size-and-appearance-architecture-tests-count-occurr"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest"
---

The text-size and appearance architecture tests count occurrences of the modifier anywhere in the file rather than holding it at the window and Settings scene roots, so both modifiers inside one scene would still pass; the appearance enum regex also pins only the first case line, so a fourth case on a following line slips past.
