---
schema_version: 1
id: "iss-2609181053323585"
slug: "nothing-asserts-that-the-macos-only-calls-the-appkit-import"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest/chat_client_ipad_test.go"
resolution: "TestChatClientGuardsTheMacOnlyCalls walks each client source's conditional-compilation blocks and fails on import AppKit, NSPasteboard, NSApp or controlActiveState outside an os(macOS) branch; watched failing with one guard removed."
impact: internal
resolved_by:
  intent: "itd-2609180943290800"
---

Nothing asserts that the macOS-only calls (the AppKit import, NSPasteboard, NSApp, controlActiveState) stay inside an os(macOS) guard; the iPad build would only find out at compile time. One loop over the client sources closes the other half of acceptance criterion 6.
