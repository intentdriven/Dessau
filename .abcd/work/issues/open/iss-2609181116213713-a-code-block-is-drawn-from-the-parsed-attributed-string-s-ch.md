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

## Triage 2026-09-18

Left open for the maintainer: drawing a code block from its raw source range means keeping the range through the block split, which changes the parser's shape; the line breaks that are unproven today want a rendered-appearance check (iss-2609181116225273) to prove either way.

## Deferral 2026-09-19

Waits on the maintainer: drawing a code block from its raw source range changes the parser's shape, and the proof wants the rendered-appearance check of iss-2609181116225273.
