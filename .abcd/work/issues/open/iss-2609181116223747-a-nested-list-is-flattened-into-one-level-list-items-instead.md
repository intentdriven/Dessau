---
schema_version: 1
id: "iss-2609181116223747"
slug: "a-nested-list-is-flattened-into-one-level-list-items-instead"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
---

A nested list is flattened into one-level list items instead of falling back to the model's own text; only tables fall back, against the promise that an undrawable construct is shown as the model wrote it.
