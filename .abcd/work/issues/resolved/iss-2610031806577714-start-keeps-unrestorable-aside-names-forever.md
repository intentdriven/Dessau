---
schema_version: 1
id: "iss-2610031806577714"
slug: "start-keeps-unrestorable-aside-names-forever"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of fbf4fa6 on the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Remove an aside whose name does not give a valid repo id, since nothing can put it back."
resolution: "An aside whose name gives no valid repo id is removed at start."
impact: fix
resolved_by:
  commit: "f4f28f2c17f7fb71283a23723db1da26a821b7e2"
---

An aside whose name is not a valid repo id with the aside suffix (a planted '+oldx', or 'a b+old1') can never be put back, yet the start's clear keeps it for ever and logs each start that something else stands in its model's folder, which is not what happened.
