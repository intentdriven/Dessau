---
schema_version: 1
id: "iss-2609181116079882"
slug: "no-delivered-path-opens-a-second-chat-window-commandgroup-re"
severity: "major"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

No delivered path opens a second chat window: CommandGroup(replacing: .newItem) removes the standard New Window item and nothing in the client calls openWindow, so the trunk intent's multi-window restoration criterion has nothing behind it.
