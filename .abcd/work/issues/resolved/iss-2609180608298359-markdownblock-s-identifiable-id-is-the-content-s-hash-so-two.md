---
schema_version: 1
id: "iss-2609180608298359"
slug: "markdownblock-s-identifiable-id-is-the-content-s-hash-so-two"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
resolution: "Blocks are laid out by position (enumerated offset); MarkdownBlock is no longer Identifiable."
impact: fix
---

MarkdownBlock's Identifiable id is the content's hash, so two identical blocks in one reply collide in ForEach and rows are dropped or mis-rendered; blocks should be identified by position.
