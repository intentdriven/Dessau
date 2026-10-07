---
schema_version: 1
id: "iss-2610071228050036"
slug: "testsavingthesamesettingsrestartsabridgediscordstopped"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "full test runs of the 2026-10-07 fix lanes"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord"
remedy: "Run it with -race -count=200 beside a CPU-bound load until it fails, keep the message, and root-cause it; a failing test is not a flake until shown to be one."
---

TestSavingTheSameSettingsRestartsABridgeDiscordStopped (internal/bridge/discord) failed once during a full make test run on 2026-10-07 and passed on its own with -count=5 and on the full re-run. It was seen on a base without any of that day's bridge changes, so it predates them. The failure message was not kept.
