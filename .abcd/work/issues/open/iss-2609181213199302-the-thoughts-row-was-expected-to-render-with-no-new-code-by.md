---
schema_version: 1
id: "iss-2609181213199302"
slug: "the-thoughts-row-was-expected-to-render-with-no-new-code-by"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The Thoughts row was expected to render with no new code by reusing the reply block parser; it landed with roughly thirty lines including three new state properties and a duplicated parse-and-throttle twin rather than reuse of the existing scheduler.
