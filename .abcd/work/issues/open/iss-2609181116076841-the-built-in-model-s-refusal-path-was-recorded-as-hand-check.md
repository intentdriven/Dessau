---
schema_version: 1
id: "iss-2609181116076841"
slug: "the-built-in-model-s-refusal-path-was-recorded-as-hand-check"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Backends.swift"
---

The built-in model's refusal path was recorded as hand-checked, but what the record names is the guardrailViolation mapping, a different branch of the error switch from the .refusal explanation the acceptance criterion asks for.
