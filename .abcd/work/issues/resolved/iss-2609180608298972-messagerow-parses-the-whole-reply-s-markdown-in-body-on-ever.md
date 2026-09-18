---
schema_version: 1
id: "iss-2609180608298972"
slug: "messagerow-parses-the-whole-reply-s-markdown-in-body-on-ever"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The row keeps its blocks in state, parses on text change, and throttles the streaming row to five parses a second."
impact: fix
---

MessageRow parses the whole reply's markdown in body on every token of every visible row, against the formatted-replies spec's cost condition; the blocks should be state refreshed on text change and throttled for the streaming row.
