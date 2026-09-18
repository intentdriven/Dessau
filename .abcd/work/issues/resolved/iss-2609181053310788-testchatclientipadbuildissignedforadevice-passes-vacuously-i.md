---
schema_version: 1
id: "iss-2609181053310788"
slug: "testchatclientipadbuildissignedforadevice-passes-vacuously-i"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ruthless review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest/chat_client_ipad_test.go"
resolution: "TestChatClientIPadBuildIsSignedForADevice anchors on the guard's own test expression and on that guard reaching exit 1; watched failing with the refusal deleted."
impact: internal
resolved_by:
  intent: "itd-2609180943290800"
---

TestChatClientIPadBuildIsSignedForADevice passes vacuously: its regex matches a mention of the two env vars plus any later 'exit 1', so deleting the whole refusal guard leaves it green; anchor it on the guard's own test expression.
