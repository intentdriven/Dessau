---
schema_version: 1
id: "iss-2609180608286789"
slug: "opening-settings-silently-re-binds-the-stored-api-key-to-wha"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "SettingsView's change handler ignores a value equal to the stored key, so the initial load no longer re-binds it; TestChatClientSendsTheKeyToOneHostOnly holds the guard."
impact: fix
---

Opening Settings silently re-binds the stored API key to whatever server the client is pointed at. SettingsView loads the key in onAppear, which fires the onChange that calls saveAPIKey, which sets apiKeyHost to the current server; after picking a found server the key is then sent to it on the next request. The change handler must ignore the initial load, and the architecture test must hold that.
