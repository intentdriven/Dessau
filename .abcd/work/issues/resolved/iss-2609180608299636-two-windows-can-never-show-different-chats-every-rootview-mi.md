---
schema_version: 1
id: "iss-2609180608299636"
slug: "two-windows-can-never-show-different-chats-every-rootview-mi"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "RootView follows only intentSelection, and only as the key window; an intent's chat wins over the restored one within ten seconds of launch."
impact: fix
---

Two windows can never show different chats: every RootView mirrors its selection into model.selectedID and follows model.selectedID back, so a click in one window moves the other. The App Intents need a dedicated signal that only the key window follows, and the intent's chat must win over the restored selection when the intent launched the app.
