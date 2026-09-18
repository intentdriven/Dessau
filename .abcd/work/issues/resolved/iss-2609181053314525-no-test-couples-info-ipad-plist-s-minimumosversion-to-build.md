---
schema_version: 1
id: "iss-2609181053314525"
slug: "no-test-couples-info-ipad-plist-s-minimumosversion-to-build"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest/chat_client_ipad_test.go"
resolution: "TestChatClientIPadDeclaresOneFloor holds Info-iPad.plist's MinimumOSVersion to build-ipad.sh's DEPLOYMENT_TARGET and to every -apple-ios triple, as the macOS floor guard does for the Mac pair; watched failing on a changed target."
impact: internal
resolved_by:
  intent: "itd-2609180943290800"
---

No test couples Info-iPad.plist's MinimumOSVersion to build-ipad.sh's deployment target, though both files promise it and the Mac pair is held by the floor guard; add the same comparison.
