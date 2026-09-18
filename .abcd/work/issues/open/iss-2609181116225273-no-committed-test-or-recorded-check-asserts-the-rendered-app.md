---
schema_version: 1
id: "iss-2609181116225273"
slug: "no-committed-test-or-recorded-check-asserts-the-rendered-app"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
---

No committed test or recorded check asserts the rendered appearance of the five markdown constructs: the record evidences only the harness's block split, and nothing holds italic, bold, monospaced, a clickable link or a kept line break as drawn.
