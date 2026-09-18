---
schema_version: 1
id: "iss-2609180608295712"
slug: "tablesegments-cuts-any-line-beginning-with-a-pipe-out-of-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
resolution: "tableSegments tracks fence state and never cuts a line inside a fence."
impact: fix
---

tableSegments cuts any line beginning with a pipe out of the prose before parsing, including lines inside a fenced code block, which splits the fence and mis-renders the rest of the reply; fence state must be tracked.
