---
schema_version: 1
id: "iss-2610032215521663"
slug: "pushed-past-a-failing-gate"
severity: "minor"
category: "lapse"
source: "agent-finding"
found_during: "driving iss-2610032212269524"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest/debug_mark_test.go"
remedy: "Run the gates as their own step and push only after reading a clean result; never chain a push after a gate run."
resolution: "Corrected in the next commit on the same branch, before CI finished; gates now run as their own step before any push."
impact: internal
resolved_by:
  commit: "e7ca992"
---

A commit was pushed and its PR opened although the gates had just failed (internal/archtest TestTheDebugMarkIsNamedOnlyByItsReaders): the gate run and the commit, push and PR were chained in one command that did not stop on the failure. The fix was pushed to the same unarmed PR before CI finished.
