---
schema_version: 1
id: "iss-2610041338330838"
slug: "discord-answers-fill-in-in-jumps-not-word-by-word-the-bridge"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "live Discord checks (iss-2609190242198542), 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/editor.go"
remedy: "Measure the bridge's edit cadence against Discord's per-channel rate limit and tighten it to the fastest rate that holds, so the answer fills in as smoothly as Discord allows."
---

Discord answers fill in in jumps, not word by word. The bridge posts a placeholder and edits in the text at most every 2 s (minEditInterval in internal/bridge/discord/editor.go), so a reply appears as a few large chunks. The maintainer wants it to read like a live stream, word by word. Discord has no streaming API and rate-limits message edits per channel, so word-by-word can only be approximated by a tighter edit cadence. Seen during the live Discord checks on 2026-10-04 (v0.9.3).
