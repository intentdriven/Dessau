---
schema_version: 1
id: "iss-2610031757373541"
slug: "self-test-loads-non-chat-models"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031010371709"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/selftest.go"
remedy: "Skip a model that cannot chat in selfTestServer.Ready, as the context probe already does."
resolution: "selfTestServer.Ready skips a model that cannot chat."
impact: fix
resolved_by:
  commit: "a0264236"
---

The idle self-test's ready list skipped only models whose load failed, so with the self-test on it loaded a model the chat rule marks not chat-capable and sent it a chat request — the nonsense answers and evictions the refusal exists to close.
