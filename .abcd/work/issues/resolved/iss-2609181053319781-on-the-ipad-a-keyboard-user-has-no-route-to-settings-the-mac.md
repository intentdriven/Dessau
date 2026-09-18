---
schema_version: 1
id: "iss-2609181053319781"
slug: "on-the-ipad-a-keyboard-user-has-no-route-to-settings-the-mac"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The iPad's Chat menu carries a Settings command on Cmd-, driving the same sheet as the toolbar's gear; the sheet's state moved up to the App, which owns the one window."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

On the iPad a keyboard user has no route to Settings: the Mac's Cmd-, comes from the Settings scene, which is guarded out, and the iPad offers only the gear tap; the menu bar must carry a Settings command with the shortcut driving the same sheet.
