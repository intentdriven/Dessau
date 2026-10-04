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
found_at: "client/DessauChat/DessauChat.swift"
deferred_after: "v0.10.0"
deferral_reason: "Re-deferred at the 0.10.0 cut: the Swift client builds and its 46 script checks pass (2026-10-04), but the appearance switch has not been checked by hand in the built app; it is a client defect outside this server cut's fixes, and the maintainer tests it by hand against this release."
---

Switching Appearance from System to Dark and back to System leaves the Settings window dark and blanks the main window's chat content: the main window turns light but shows no messages until the Settings window is closed, at which point the conversation reappears. Seen on macOS with the Settings window open beside the main window; the Appearance picker reads System while the Settings window is still rendered dark. Every window applies preferredColorScheme from the same AppStorage value, so a nil scheme (System) after an explicit one is not re-evaluated the same way in each scene.

## Deferral 2026-10-04

Re-deferred at the 0.10.0 cut. On 2026-10-04 the Swift client built with `client/build.sh` against the macOS 27 SDK and the four `client/tests` scripts passed (46 checks), but the System to Dark to System switch has not been checked by hand in the built app. It is a client defect, outside this server cut's fixes; the maintainer tests it by hand against this release.
