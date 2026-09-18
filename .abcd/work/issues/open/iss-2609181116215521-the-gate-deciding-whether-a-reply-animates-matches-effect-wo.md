---
schema_version: 1
id: "iss-2609181116215521"
slug: "the-gate-deciding-whether-a-reply-animates-matches-effect-wo"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Effects.swift"
---

The gate deciding whether a reply animates matches effect words over the whole reply's raw markdown text while the renderer matches per rendered block over the block's plain text, so the two matches can disagree.
