---
schema_version: 1
id: "iss-2610031757384526"
slug: "precheck-dangling-link-untested"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031317475284"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/launcher.go"
remedy: "Wrap the stat error and add a dangling-link case to the test."
resolution: "The stat error is wrapped and a dangling interpreter link is tested as not installed."
impact: fix
resolved_by:
  commit: "42c1fe0"
---

Telling a runtime that is not installed from an untrusted one rested on the interpreter's stat error, which was not wrapped into the refusal, and no test covered a dangling interpreter link, which must read as not installed rather than untrusted.
