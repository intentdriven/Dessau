---
schema_version: 1
id: "iss-2609181124292809"
slug: "the-commit-announcing-the-appearance-feature-carried-the-arc"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The commit announcing the appearance feature carried the architecture test, the README and the changelog but no Swift change; the setting itself arrived in a follow-up fix commit, so the promise was true of the merged range and not of the commit that announced it.
