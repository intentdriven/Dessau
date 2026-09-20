---
schema_version: 1
id: "iss-2609200815308397"
slug: "switching-appearance-from-system-to-dark-and-back-to-system"
severity: "major"
category: "bug"
source: "manual-test"
found_during: "maintainer manual test of the chat client, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

Switching Appearance from System to Dark and back to System leaves the Settings window dark and blanks the main window's chat content: the main window turns light but shows no messages until the Settings window is closed, at which point the conversation reappears. Seen on macOS with the Settings window open beside the main window; the Appearance picker reads System while the Settings window is still rendered dark. Every window applies preferredColorScheme from the same AppStorage value, so a nil scheme (System) after an explicit one is not re-evaluated the same way in each scene.
