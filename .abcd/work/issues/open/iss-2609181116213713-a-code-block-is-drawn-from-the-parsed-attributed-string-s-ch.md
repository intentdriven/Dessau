---
schema_version: 1
id: "iss-2609181116213713"
slug: "a-code-block-is-drawn-from-the-parsed-attributed-string-s-ch"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
---

A code block is drawn from the parsed attributed string's characters rather than from its raw source range as the mechanism promised, so its line breaks surviving is unproven.
