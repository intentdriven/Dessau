---
schema_version: 1
id: "iss-2610030656102623"
slug: "client-rename-orphans-conversation-history"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "peer-session report while consuming the gateway"
origin: researcher-authored
production_mode: hand-written
found_at: "client/DessauChat/DessauChat.swift"
resolution: "Maintainer's decision 2026-10-03, NO MIGRATION: the pre-1.0 no-migration rule covers the client's conversation history; no code, and a tester moves the file by hand if they want it."
impact: internal
resolved_by:
  commit: "d8c621d20552a6306504c8789d735f7f2aa6ca2d"
---

The chat client's rename orphans its saved conversation history. The client now saves to ~/Library/Application Support/DessauChat/conversations.json (client/DessauChat/DessauChat.swift saveURL), but a tester's Mac still holds conversations.json in the client's pre-rename Application Support folder (named after the superseded product name), and DessauChat does not exist there, so an updated client starts with an empty history while the old one sits unread on disk. The server-side pre-1.0 no-migration rule names config.json, the registry and the statistics files; whether it covers the client's conversation history, or whether one move-on-first-launch is warranted, is a maintainer call. An empty leftover server folder under the superseded name sits beside it. Reported by a peer session on 2026-10-03; the file was not opened.
