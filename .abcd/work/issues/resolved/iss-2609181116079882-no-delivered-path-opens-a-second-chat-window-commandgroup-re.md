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
resolution: "The WindowGroup carries an id and the File commands carry New Window on Cmd-Shift-N, opening another window of that group through @Environment(\\.openWindow); New Chat keeps Cmd-N. macOS only: the iPad has the one window and no such item to restore. TestChatClientOpensASecondWindow holds it."
impact: fix
---

No delivered path opens a second chat window: CommandGroup(replacing: .newItem) removes the standard New Window item and nothing in the client calls openWindow, so the trunk intent's multi-window restoration criterion has nothing behind it.

## Grounds

- pursued: a second window onto the same chats is one shortcut away again, as it was before .newItem was replaced. Wrong if the item opens nothing, or the iPad build loses a command it had.
